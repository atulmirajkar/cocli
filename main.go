package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"atulm/cocli/session"
)

func main() {
	// Create and initialize session manager with client
	sessionMgr, err := session.NewManager()
	if err != nil {
		log.Fatal(err)
	}
	defer sessionMgr.Close()

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
			} else {
				fmt.Println("Unknown command. Use /models or /list to see available models.")
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
