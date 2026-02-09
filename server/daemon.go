package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"regexp"
	"syscall"
	"time"
)

const (
	// DefaultPort is the default port for the daemon
	DefaultPort = 4321

	// startTimeout is how long to wait for daemon to start
	startTimeout = 30 * time.Second

	// stopTimeout is how long to wait for graceful shutdown
	stopTimeout = 5 * time.Second

	// healthCheckTimeout is how long to wait for health check
	healthCheckTimeout = 5 * time.Second
)

var (
	// ErrDaemonAlreadyRunning is returned when trying to start an already running daemon
	ErrDaemonAlreadyRunning = errors.New("daemon is already running")

	// ErrDaemonNotRunning is returned when trying to stop a daemon that isn't running
	ErrDaemonNotRunning = errors.New("daemon is not running")

	// ErrCLINotFound is returned when the copilot CLI cannot be found
	ErrCLINotFound = errors.New("copilot CLI not found")

	// ErrStartTimeout is returned when daemon fails to start within timeout
	ErrStartTimeout = errors.New("daemon failed to start within timeout")
)

// DaemonStatus represents current daemon state
type DaemonStatus struct {
	Running   bool
	PID       int
	Port      int
	StartedAt time.Time
	Uptime    time.Duration
}

// ProcessManager interface for process operations
type ProcessManager interface {
	// IsRunning checks if a process with the given PID is running
	IsRunning(pid int) bool
	// Kill terminates a process with the given PID
	Kill(pid int) error
	// StartProcess starts a new process and returns its PID
	StartProcess(name string, args []string, stdout, stderr io.Writer) (pid int, err error)
}

// HealthChecker interface for server health checks
type HealthChecker interface {
	// Ping checks if the server at host:port is responding
	Ping(host string, port int, timeout time.Duration) error
}

// CLIFinder interface for locating the copilot CLI
type CLIFinder interface {
	// FindCLI returns the path to the copilot CLI executable
	FindCLI() (string, error)
}

// DaemonManager handles daemon lifecycle
type DaemonManager struct {
	config       ConfigStore
	process      ProcessManager
	health       HealthChecker
	cliFinder    CLIFinder
	port         int
	startTimeout time.Duration // Configurable for testing
}

// NewDaemonManager creates a DaemonManager with the given dependencies
func NewDaemonManager(
	config ConfigStore,
	process ProcessManager,
	health HealthChecker,
	cliFinder CLIFinder,
) *DaemonManager {
	return &DaemonManager{
		config:       config,
		process:      process,
		health:       health,
		cliFinder:    cliFinder,
		port:         DefaultPort,
		startTimeout: startTimeout,
	}
}

// DefaultDaemonManager creates a DaemonManager with default implementations
func DefaultDaemonManager() (*DaemonManager, error) {
	config, err := DefaultConfigStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create config store: %w", err)
	}

	return &DaemonManager{
		config:       config,
		process:      &OSProcessManager{},
		health:       &SDKHealthChecker{},
		cliFinder:    &EnvCLIFinder{},
		port:         DefaultPort,
		startTimeout: startTimeout,
	}, nil
}

// SetStartTimeout sets the start timeout (useful for testing)
func (d *DaemonManager) SetStartTimeout(timeout time.Duration) {
	d.startTimeout = timeout
}

// Start starts the daemon
func (d *DaemonManager) Start() error {
	// Check if already running
	if d.IsRunning() {
		return ErrDaemonAlreadyRunning
	}

	// Clean up any stale config
	_ = d.config.Delete()

	// Find CLI
	cliPath, err := d.cliFinder.FindCLI()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCLINotFound, err)
	}

	fmt.Printf("Starting daemon on port %d...\n", d.port)

	// Start process with --server mode (matches SDK's CLI invocation)
	args := []string{
		"--server",
		"--port", fmt.Sprintf("%d", d.port),
		"--no-auto-update",
		"--log-level", "error", // Only log errors to reduce noise
	}

	// Discard stdout/stderr to avoid noise from server health check messages
	pid, err := d.process.StartProcess(cliPath, args, io.Discard, io.Discard)
	if err != nil {
		return fmt.Errorf("failed to start daemon process: %w", err)
	}

	// Wait for health check with timeout
	ctx, cancel := context.WithTimeout(context.Background(), d.startTimeout)
	defer cancel()

	if err := d.waitForHealthy(ctx); err != nil {
		// Kill the process if health check fails
		_ = d.process.Kill(pid)
		return fmt.Errorf("daemon started but health check failed: %w", err)
	}

	// Save config
	config := &DaemonConfig{
		PID:       pid,
		Port:      d.port,
		StartedAt: time.Now(),
	}
	if err := d.config.Save(config); err != nil {
		// Kill the process if we can't save config
		_ = d.process.Kill(pid)
		return fmt.Errorf("failed to save daemon config: %w", err)
	}

	fmt.Printf("Daemon started (PID: %d)\n", pid)
	return nil
}

// waitForHealthy waits for the daemon to become healthy
func (d *DaemonManager) waitForHealthy(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ErrStartTimeout
		case <-ticker.C:
			if err := d.health.Ping("localhost", d.port, healthCheckTimeout); err == nil {
				return nil
			}
		}
	}
}

// Stop stops the daemon
func (d *DaemonManager) Stop() error {
	config, err := d.config.Load()
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			return ErrDaemonNotRunning
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if process is actually running
	if !d.process.IsRunning(config.PID) {
		// Process already dead, just clean up config
		_ = d.config.Delete()
		return ErrDaemonNotRunning
	}

	fmt.Printf("Stopping daemon (PID: %d)...\n", config.PID)

	// Kill the process
	if err := d.process.Kill(config.PID); err != nil {
		return fmt.Errorf("failed to stop daemon: %w", err)
	}

	// Delete config
	if err := d.config.Delete(); err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}

	fmt.Println("Daemon stopped")
	return nil
}

// Status returns the current daemon status
func (d *DaemonManager) Status() (*DaemonStatus, error) {
	config, err := d.config.Load()
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			return &DaemonStatus{Running: false}, nil
		}
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Check if process is running and healthy
	running := d.process.IsRunning(config.PID)
	if running {
		// Verify with health check
		if err := d.health.Ping("localhost", config.Port, healthCheckTimeout); err != nil {
			running = false
		}
	}

	if !running {
		// Clean up stale config
		_ = d.config.Delete()
		return &DaemonStatus{Running: false}, nil
	}

	return &DaemonStatus{
		Running:   true,
		PID:       config.PID,
		Port:      config.Port,
		StartedAt: config.StartedAt,
		Uptime:    time.Since(config.StartedAt),
	}, nil
}

// IsRunning returns true if the daemon is running and healthy
func (d *DaemonManager) IsRunning() bool {
	status, err := d.Status()
	if err != nil {
		return false
	}
	return status.Running
}

// GetPort returns the daemon port if running
func (d *DaemonManager) GetPort() (int, error) {
	status, err := d.Status()
	if err != nil {
		return 0, err
	}
	if !status.Running {
		return 0, ErrDaemonNotRunning
	}
	return status.Port, nil
}

// --- Default Implementations ---

// OSProcessManager implements ProcessManager using os/exec
type OSProcessManager struct{}

// IsRunning checks if a process with the given PID is running
func (p *OSProcessManager) IsRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds, so we send signal 0 to check
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// Kill terminates a process with the given PID
func (p *OSProcessManager) Kill(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	// Try graceful shutdown first (SIGTERM)
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Process might already be dead
		return nil
	}

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case <-done:
		return nil
	case <-time.After(stopTimeout):
		// Force kill
		return process.Kill()
	}
}

// StartProcess starts a new process and returns its PID
func (p *OSProcessManager) StartProcess(name string, args []string, stdout, stderr io.Writer) (int, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// Start as detached process so it survives parent exit
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return 0, err
	}

	// Don't wait for the process - let it run in background
	go func() {
		_ = cmd.Wait()
	}()

	return cmd.Process.Pid, nil
}

// SDKHealthChecker implements HealthChecker using a simple TCP connection check
type SDKHealthChecker struct{}

// Ping checks if the server at host:port is responding by attempting a TCP connection
func (h *SDKHealthChecker) Ping(host string, port int, timeout time.Duration) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

// EnvCLIFinder implements CLIFinder checking env then PATH
type EnvCLIFinder struct{}

// FindCLI returns the path to the copilot CLI executable
func (f *EnvCLIFinder) FindCLI() (string, error) {
	// Check environment variable first
	if path := os.Getenv("COPILOT_CLI_PATH"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Check PATH
	path, err := exec.LookPath("copilot")
	if err != nil {
		return "", fmt.Errorf("copilot not found in PATH: %w", err)
	}
	return path, nil
}

// StreamingProcessManager is an alternative that captures "listening on port" output
type StreamingProcessManager struct{}

// StartProcessWithPortDetection starts process and waits for port announcement
func (p *StreamingProcessManager) StartProcessWithPortDetection(name string, args []string, stdout, stderr io.Writer) (pid int, port int, err error) {
	cmd := exec.Command(name, args...)

	// Create pipes for stdout
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return 0, 0, err
	}

	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return 0, 0, err
	}

	// Read stdout looking for port announcement
	portRegex := regexp.MustCompile(`listening on port (\d+)`)
	scanner := bufio.NewScanner(stdoutPipe)

	portChan := make(chan int, 1)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			// Echo to stdout
			if stdout != nil {
				fmt.Fprintln(stdout, line)
			}
			if matches := portRegex.FindStringSubmatch(line); len(matches) > 1 {
				var p int
				fmt.Sscanf(matches[1], "%d", &p)
				portChan <- p
				return
			}
		}
		close(portChan)
	}()

	// Don't wait for the process - let it run in background
	go func() {
		_ = cmd.Wait()
	}()

	// Wait for port with timeout
	select {
	case port := <-portChan:
		return cmd.Process.Pid, port, nil
	case <-time.After(startTimeout):
		_ = cmd.Process.Kill()
		return 0, 0, ErrStartTimeout
	}
}
