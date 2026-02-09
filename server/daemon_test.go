package server

import (
	"errors"
	"io"
	"os"
	"testing"
	"time"
)

// --- Mock Implementations ---

// MockConfigStore implements ConfigStore for testing
type MockConfigStore struct {
	config       *DaemonConfig
	loadErr      error
	saveErr      error
	deleteErr    error
	SaveCalled   bool
	DeleteCalled bool
	SavedConfig  *DaemonConfig
}

func (m *MockConfigStore) Load() (*DaemonConfig, error) {
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	if m.config == nil {
		return nil, ErrConfigNotFound
	}
	return m.config, nil
}

func (m *MockConfigStore) Save(config *DaemonConfig) error {
	m.SaveCalled = true
	m.SavedConfig = config
	if m.saveErr != nil {
		return m.saveErr
	}
	m.config = config
	return nil
}

func (m *MockConfigStore) Delete() error {
	m.DeleteCalled = true
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.config = nil
	return nil
}

func (m *MockConfigStore) GetPath() string {
	return "/mock/path/server.json"
}

// MockProcessManager implements ProcessManager for testing
type MockProcessManager struct {
	runningPIDs map[int]bool
	killErr     error
	startPID    int
	startErr    error
	KillCalled  []int
	StartCalled bool
}

func NewMockProcessManager() *MockProcessManager {
	return &MockProcessManager{
		runningPIDs: make(map[int]bool),
	}
}

func (m *MockProcessManager) IsRunning(pid int) bool {
	return m.runningPIDs[pid]
}

func (m *MockProcessManager) Kill(pid int) error {
	m.KillCalled = append(m.KillCalled, pid)
	if m.killErr != nil {
		return m.killErr
	}
	delete(m.runningPIDs, pid)
	return nil
}

func (m *MockProcessManager) StartProcess(name string, args []string, stdout, stderr io.Writer) (int, error) {
	m.StartCalled = true
	if m.startErr != nil {
		return 0, m.startErr
	}
	m.runningPIDs[m.startPID] = true
	return m.startPID, nil
}

// MockHealthChecker implements HealthChecker for testing
type MockHealthChecker struct {
	healthy bool
	pingErr error
}

func (m *MockHealthChecker) Ping(host string, port int, timeout time.Duration) error {
	if !m.healthy {
		if m.pingErr != nil {
			return m.pingErr
		}
		return errors.New("health check failed")
	}
	return nil
}

// MockCLIFinder implements CLIFinder for testing
type MockCLIFinder struct {
	path string
	err  error
}

func (m *MockCLIFinder) FindCLI() (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.path, nil
}

// --- Tests ---

func TestDaemonManager_IsRunning_NoConfig(t *testing.T) {
	dm := NewDaemonManager(
		&MockConfigStore{loadErr: ErrConfigNotFound},
		NewMockProcessManager(),
		&MockHealthChecker{healthy: true},
		&MockCLIFinder{path: "/usr/bin/copilot"},
	)

	if dm.IsRunning() {
		t.Error("IsRunning() = true, want false when no config")
	}
}

func TestDaemonManager_IsRunning_ProcessDead(t *testing.T) {
	configStore := &MockConfigStore{
		config: &DaemonConfig{PID: 12345, Port: 4321, StartedAt: time.Now()},
	}
	procMgr := NewMockProcessManager()
	// Process NOT in runningPIDs, so IsRunning(12345) returns false

	dm := NewDaemonManager(
		configStore,
		procMgr,
		&MockHealthChecker{healthy: true},
		&MockCLIFinder{path: "/usr/bin/copilot"},
	)

	if dm.IsRunning() {
		t.Error("IsRunning() = true, want false when process is dead")
	}

	// Should clean up stale config
	if !configStore.DeleteCalled {
		t.Error("Expected stale config to be deleted")
	}
}

func TestDaemonManager_IsRunning_HealthCheckFails(t *testing.T) {
	configStore := &MockConfigStore{
		config: &DaemonConfig{PID: 12345, Port: 4321, StartedAt: time.Now()},
	}
	procMgr := NewMockProcessManager()
	procMgr.runningPIDs[12345] = true // Process is running

	dm := NewDaemonManager(
		configStore,
		procMgr,
		&MockHealthChecker{healthy: false, pingErr: errors.New("connection refused")},
		&MockCLIFinder{path: "/usr/bin/copilot"},
	)

	if dm.IsRunning() {
		t.Error("IsRunning() = true, want false when health check fails")
	}

	// Should clean up stale config
	if !configStore.DeleteCalled {
		t.Error("Expected stale config to be deleted when health check fails")
	}
}

func TestDaemonManager_IsRunning_Healthy(t *testing.T) {
	configStore := &MockConfigStore{
		config: &DaemonConfig{PID: 12345, Port: 4321, StartedAt: time.Now()},
	}
	procMgr := NewMockProcessManager()
	procMgr.runningPIDs[12345] = true

	dm := NewDaemonManager(
		configStore,
		procMgr,
		&MockHealthChecker{healthy: true},
		&MockCLIFinder{path: "/usr/bin/copilot"},
	)

	if !dm.IsRunning() {
		t.Error("IsRunning() = false, want true when daemon is healthy")
	}
}

func TestDaemonManager_GetPort_NotRunning(t *testing.T) {
	dm := NewDaemonManager(
		&MockConfigStore{loadErr: ErrConfigNotFound},
		NewMockProcessManager(),
		&MockHealthChecker{healthy: true},
		&MockCLIFinder{path: "/usr/bin/copilot"},
	)

	_, err := dm.GetPort()
	if !errors.Is(err, ErrDaemonNotRunning) {
		t.Errorf("GetPort() error = %v, want ErrDaemonNotRunning", err)
	}
}

func TestDaemonManager_GetPort_Running(t *testing.T) {
	configStore := &MockConfigStore{
		config: &DaemonConfig{PID: 12345, Port: 4321, StartedAt: time.Now()},
	}
	procMgr := NewMockProcessManager()
	procMgr.runningPIDs[12345] = true

	dm := NewDaemonManager(
		configStore,
		procMgr,
		&MockHealthChecker{healthy: true},
		&MockCLIFinder{path: "/usr/bin/copilot"},
	)

	port, err := dm.GetPort()
	if err != nil {
		t.Fatalf("GetPort() error = %v", err)
	}
	if port != 4321 {
		t.Errorf("GetPort() = %d, want 4321", port)
	}
}

func TestDaemonManager_Status_NotRunning(t *testing.T) {
	dm := NewDaemonManager(
		&MockConfigStore{loadErr: ErrConfigNotFound},
		NewMockProcessManager(),
		&MockHealthChecker{healthy: true},
		&MockCLIFinder{path: "/usr/bin/copilot"},
	)

	status, err := dm.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.Running {
		t.Error("Status().Running = true, want false")
	}
}

func TestDaemonManager_Status_Running(t *testing.T) {
	startTime := time.Now().Add(-5 * time.Minute)
	configStore := &MockConfigStore{
		config: &DaemonConfig{PID: 12345, Port: 4321, StartedAt: startTime},
	}
	procMgr := NewMockProcessManager()
	procMgr.runningPIDs[12345] = true

	dm := NewDaemonManager(
		configStore,
		procMgr,
		&MockHealthChecker{healthy: true},
		&MockCLIFinder{path: "/usr/bin/copilot"},
	)

	status, err := dm.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !status.Running {
		t.Error("Status().Running = false, want true")
	}
	if status.PID != 12345 {
		t.Errorf("Status().PID = %d, want 12345", status.PID)
	}
	if status.Port != 4321 {
		t.Errorf("Status().Port = %d, want 4321", status.Port)
	}
	// Uptime should be approximately 5 minutes
	if status.Uptime < 4*time.Minute || status.Uptime > 6*time.Minute {
		t.Errorf("Status().Uptime = %v, want approximately 5 minutes", status.Uptime)
	}
}

func TestDaemonManager_Start_AlreadyRunning(t *testing.T) {
	configStore := &MockConfigStore{
		config: &DaemonConfig{PID: 12345, Port: 4321, StartedAt: time.Now()},
	}
	procMgr := NewMockProcessManager()
	procMgr.runningPIDs[12345] = true

	dm := NewDaemonManager(
		configStore,
		procMgr,
		&MockHealthChecker{healthy: true},
		&MockCLIFinder{path: "/usr/bin/copilot"},
	)

	err := dm.Start()
	if !errors.Is(err, ErrDaemonAlreadyRunning) {
		t.Errorf("Start() error = %v, want ErrDaemonAlreadyRunning", err)
	}
}

func TestDaemonManager_Start_CLINotFound(t *testing.T) {
	dm := NewDaemonManager(
		&MockConfigStore{},
		NewMockProcessManager(),
		&MockHealthChecker{healthy: true},
		&MockCLIFinder{err: errors.New("not found")},
	)

	err := dm.Start()
	if !errors.Is(err, ErrCLINotFound) {
		t.Errorf("Start() error = %v, want ErrCLINotFound", err)
	}
}

func TestDaemonManager_Start_ProcessStartFails(t *testing.T) {
	procMgr := NewMockProcessManager()
	procMgr.startErr = errors.New("failed to start")

	dm := NewDaemonManager(
		&MockConfigStore{},
		procMgr,
		&MockHealthChecker{healthy: true},
		&MockCLIFinder{path: "/usr/bin/copilot"},
	)

	err := dm.Start()
	if err == nil {
		t.Error("Start() error = nil, want error")
	}
}

func TestDaemonManager_Start_HealthCheckFails(t *testing.T) {
	procMgr := NewMockProcessManager()
	procMgr.startPID = 12345

	dm := NewDaemonManager(
		&MockConfigStore{},
		procMgr,
		&MockHealthChecker{healthy: false, pingErr: errors.New("connection refused")},
		&MockCLIFinder{path: "/usr/bin/copilot"},
	)
	dm.SetStartTimeout(1 * time.Second) // Short timeout for test

	err := dm.Start()
	if err == nil {
		t.Error("Start() error = nil, want error when health check fails")
	}

	// Should have killed the process
	if len(procMgr.KillCalled) == 0 {
		t.Error("Expected process to be killed when health check fails")
	}
}

func TestDaemonManager_Start_Success(t *testing.T) {
	configStore := &MockConfigStore{}
	procMgr := NewMockProcessManager()
	procMgr.startPID = 12345

	dm := NewDaemonManager(
		configStore,
		procMgr,
		&MockHealthChecker{healthy: true},
		&MockCLIFinder{path: "/usr/bin/copilot"},
	)

	err := dm.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Should have saved config
	if !configStore.SaveCalled {
		t.Error("Expected config to be saved")
	}
	if configStore.SavedConfig == nil {
		t.Fatal("SavedConfig is nil")
	}
	if configStore.SavedConfig.PID != 12345 {
		t.Errorf("SavedConfig.PID = %d, want 12345", configStore.SavedConfig.PID)
	}
	if configStore.SavedConfig.Port != DefaultPort {
		t.Errorf("SavedConfig.Port = %d, want %d", configStore.SavedConfig.Port, DefaultPort)
	}
}

func TestDaemonManager_Stop_NotRunning(t *testing.T) {
	dm := NewDaemonManager(
		&MockConfigStore{loadErr: ErrConfigNotFound},
		NewMockProcessManager(),
		&MockHealthChecker{healthy: true},
		&MockCLIFinder{path: "/usr/bin/copilot"},
	)

	err := dm.Stop()
	if !errors.Is(err, ErrDaemonNotRunning) {
		t.Errorf("Stop() error = %v, want ErrDaemonNotRunning", err)
	}
}

func TestDaemonManager_Stop_ProcessAlreadyDead(t *testing.T) {
	configStore := &MockConfigStore{
		config: &DaemonConfig{PID: 12345, Port: 4321, StartedAt: time.Now()},
	}
	procMgr := NewMockProcessManager()
	// Process NOT running

	dm := NewDaemonManager(
		configStore,
		procMgr,
		&MockHealthChecker{healthy: true},
		&MockCLIFinder{path: "/usr/bin/copilot"},
	)

	err := dm.Stop()
	if !errors.Is(err, ErrDaemonNotRunning) {
		t.Errorf("Stop() error = %v, want ErrDaemonNotRunning when process already dead", err)
	}

	// Should have cleaned up config
	if !configStore.DeleteCalled {
		t.Error("Expected config to be deleted")
	}
}

func TestDaemonManager_Stop_KillFails(t *testing.T) {
	configStore := &MockConfigStore{
		config: &DaemonConfig{PID: 12345, Port: 4321, StartedAt: time.Now()},
	}
	procMgr := NewMockProcessManager()
	procMgr.runningPIDs[12345] = true
	procMgr.killErr = errors.New("permission denied")

	dm := NewDaemonManager(
		configStore,
		procMgr,
		&MockHealthChecker{healthy: true},
		&MockCLIFinder{path: "/usr/bin/copilot"},
	)

	err := dm.Stop()
	if err == nil {
		t.Error("Stop() error = nil, want error when kill fails")
	}
}

func TestDaemonManager_Stop_Success(t *testing.T) {
	configStore := &MockConfigStore{
		config: &DaemonConfig{PID: 12345, Port: 4321, StartedAt: time.Now()},
	}
	procMgr := NewMockProcessManager()
	procMgr.runningPIDs[12345] = true

	dm := NewDaemonManager(
		configStore,
		procMgr,
		&MockHealthChecker{healthy: true},
		&MockCLIFinder{path: "/usr/bin/copilot"},
	)

	err := dm.Stop()
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Should have killed the process
	if len(procMgr.KillCalled) == 0 {
		t.Error("Expected process to be killed")
	}
	if procMgr.KillCalled[0] != 12345 {
		t.Errorf("Kill called with PID %d, want 12345", procMgr.KillCalled[0])
	}

	// Should have deleted config
	if !configStore.DeleteCalled {
		t.Error("Expected config to be deleted")
	}
}

// --- Tests for default implementations ---

func TestOSProcessManager_IsRunning_InvalidPID(t *testing.T) {
	pm := &OSProcessManager{}

	if pm.IsRunning(0) {
		t.Error("IsRunning(0) = true, want false")
	}
	if pm.IsRunning(-1) {
		t.Error("IsRunning(-1) = true, want false")
	}
}

func TestOSProcessManager_IsRunning_CurrentProcess(t *testing.T) {
	pm := &OSProcessManager{}

	// Current process should be running
	pid := currentPID()
	if !pm.IsRunning(pid) {
		t.Errorf("IsRunning(%d) = false, want true for current process", pid)
	}
}

func TestOSProcessManager_IsRunning_NonExistentPID(t *testing.T) {
	pm := &OSProcessManager{}

	// Very high PID unlikely to exist
	if pm.IsRunning(999999999) {
		t.Error("IsRunning(999999999) = true, want false for non-existent PID")
	}
}

func TestOSProcessManager_Kill_InvalidPID(t *testing.T) {
	pm := &OSProcessManager{}

	err := pm.Kill(0)
	if err == nil {
		t.Error("Kill(0) error = nil, want error for invalid PID")
	}

	err = pm.Kill(-1)
	if err == nil {
		t.Error("Kill(-1) error = nil, want error for invalid PID")
	}
}

func TestEnvCLIFinder_FindCLI_FromEnv(t *testing.T) {
	// Set env var to a known path
	t.Setenv("COPILOT_CLI_PATH", "/bin/sh") // Using /bin/sh as a file that exists

	finder := &EnvCLIFinder{}
	path, err := finder.FindCLI()
	if err != nil {
		t.Fatalf("FindCLI() error = %v", err)
	}
	if path != "/bin/sh" {
		t.Errorf("FindCLI() = %q, want /bin/sh", path)
	}
}

func TestEnvCLIFinder_FindCLI_EnvNotExists(t *testing.T) {
	// Set env var to non-existent path
	t.Setenv("COPILOT_CLI_PATH", "/nonexistent/path/to/copilot")

	finder := &EnvCLIFinder{}
	// Should fall back to PATH lookup
	path, err := finder.FindCLI()

	// If copilot is in PATH, it should find it
	// If not, it should error
	if err != nil && path != "" {
		t.Errorf("FindCLI() returned both path and error: path=%q, err=%v", path, err)
	}
}

// Helper to get current process PID
func currentPID() int {
	return os.Getpid()
}
