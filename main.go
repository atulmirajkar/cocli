package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"atulm/cocli/client"
	"atulm/cocli/server"
	"atulm/cocli/session"
)

func main() {
	// Create client first (handles daemon connection)
	cli, err := client.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Stop()

	// Create session manager with client
	sessionMgr, err := session.NewManager(cli)
	if err != nil {
		log.Fatal(err)
	}

	// Display connection mode
	if cli.IsUsingDaemon() {
		fmt.Printf("Connected to daemon on port %d\n", server.DefaultPort)
	} else {
		fmt.Println("Using embedded server (consider: /server start)")
	}

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nBye")
		os.Exit(0)
	}()

	reader := bufio.NewReader(os.Stdin)

	// Check if a prompt was provided as a command-line argument
	var initialPrompt string
	if len(os.Args) > 1 {
		initialPrompt = strings.Join(os.Args[1:], " ")
	}

	// Interactive loop
	for {
		var prompt string
		var err error

		// Use initial prompt if provided, otherwise read from stdin
		if initialPrompt != "" {
			prompt = initialPrompt
			initialPrompt = "" // Clear it so we only use it once
		} else {
			// Display prompt with tokens and multiplier if available
			if sessionMgr.HasTokenLimit() {
				fmt.Printf("[%s | %.2fx | %d/%d tokens] > ", sessionMgr.GetCurrentModel(), sessionMgr.GetCurrentMultiplier(), sessionMgr.GetTokensLeft(), sessionMgr.GetTokenLimit())
			} else {
				fmt.Printf("[%s | %.2fx] > ", sessionMgr.GetCurrentModel(), sessionMgr.GetCurrentMultiplier())
			}
			prompt, err = reader.ReadString('\n')
			if err != nil {
				log.Fatal(err)
			}
			prompt = strings.TrimSpace(prompt)
		}

		// Handle slash commands
		if strings.HasPrefix(prompt, "/") {
			if prompt == "/models" || prompt == "/list" {
				if err := promptForModelSelection(sessionMgr, reader); err != nil {
					fmt.Printf("Error: %v\n", err)
				}
			} else if strings.HasPrefix(prompt, "/server") {
				shouldExit, err := handleServerCommand(prompt, cli.IsUsingDaemon())
				if err != nil {
					fmt.Printf("Error: %v\n", err)
				}
				if shouldExit {
					fmt.Println("Bye")
					return
				}
			} else {
				fmt.Println("Unknown command. Available: /models, /list, /server")
			}
			continue
		}

		// Send prompt if not empty
		if prompt != "" {
			if err := sessionMgr.Send(prompt); err != nil {
				log.Fatal(err)
			}
		}
	}
}

func promptForModelSelection(sessionMgr *session.Manager, reader *bufio.Reader) error {
	models, err := sessionMgr.GetModels()
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}
	if len(models) == 0 {
		return fmt.Errorf("no models available from server")
	}

	if err := sessionMgr.DisplayModels(); err != nil {
		return fmt.Errorf("failed to display models: %w", err)
	}

	fmt.Printf("Enter model number (current: %s, press Enter to skip): ", sessionMgr.GetCurrentModel())
	modelInput, _ := reader.ReadString('\n')
	modelInput = strings.TrimSpace(modelInput)

	if modelInput == "" {
		return nil
	}

	var modelIdx int
	_, err = fmt.Sscanf(modelInput, "%d", &modelIdx)
	if err != nil || modelIdx <= 0 || modelIdx > len(models) {
		return fmt.Errorf("invalid model selection")
	}

	model := models[modelIdx-1]
	multiplier := 0.0
	if model.Billing != nil {
		multiplier = model.Billing.Multiplier
	}

	if err := sessionMgr.SetModel(model.ID, multiplier); err != nil {
		return fmt.Errorf("failed to switch model: %w", err)
	}

	fmt.Printf("Switched to: %s (%.2fx)\n\n", model.ID, multiplier)
	return nil
}

// handleServerCommand handles /server subcommands
// Returns (shouldExit, error) - shouldExit is true when daemon is stopped and we were using it
func handleServerCommand(cmd string, usingDaemon bool) (bool, error) {
	parts := strings.Fields(cmd)
	if len(parts) < 2 {
		printServerHelp()
		return false, nil
	}

	dm, err := server.DefaultDaemonManager()
	if err != nil {
		return false, fmt.Errorf("failed to initialize daemon manager: %w", err)
	}

	switch parts[1] {
	case "start":
		err := dm.Start()
		if err == nil && !usingDaemon {
			fmt.Println("\nNote: This session is using an embedded server.")
			fmt.Println("Restart the CLI to connect to the daemon.")
		}
		return false, err
	case "stop":
		err := dm.Stop()
		if err != nil {
			return false, err
		}
		// Exit CLI if we were connected to the daemon we just stopped
		return usingDaemon, nil
	case "status":
		return false, printServerStatus(dm)
	case "help":
		printServerHelp()
		return false, nil
	default:
		printServerHelp()
		return false, nil
	}
}

// printServerStatus displays the current daemon status
func printServerStatus(dm *server.DaemonManager) error {
	status, err := dm.Status()
	if err != nil {
		return err
	}

	if status.Running {
		fmt.Printf("Daemon: running\n")
		fmt.Printf("  PID:     %d\n", status.PID)
		fmt.Printf("  Port:    %d\n", status.Port)
		fmt.Printf("  Uptime:  %s\n", formatDuration(status.Uptime))
	} else {
		fmt.Println("Daemon: not running")
		fmt.Println("\nStart the daemon with: /server start")
	}
	return nil
}

// printServerHelp displays help for server commands
func printServerHelp() {
	fmt.Println("Usage: /server <command>")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Printf("  start   Start the background daemon (port %d)\n", server.DefaultPort)
	fmt.Println("  stop    Stop the daemon")
	fmt.Println("  status  Show daemon status")
	fmt.Println("  help    Show this help")
	fmt.Println("")
	fmt.Println("When the daemon is running, cocli will connect to it")
	fmt.Println("instead of starting a new server, making startup faster.")
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
