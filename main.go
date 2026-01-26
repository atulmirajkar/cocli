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

	copilot "github.com/github/copilot-sdk/go"
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

	// Use default model
	defaultModel := "gpt-4.1"
	selectedModel := defaultModel
	var models []copilot.ModelInfo

	// Create initial session
	if err := sessionMgr.Create(selectedModel); err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(os.Stdin)

	// Interactive loop
	for {
		// Display prompt with tokens if available
		if sessionMgr.HasTokenLimit() {
			fmt.Printf("[%s | %d/%d tokens] > ", selectedModel, sessionMgr.GetTokensLeft(), sessionMgr.TokenLimit)
		} else {
			fmt.Printf("[%s] > ", selectedModel)
		}
		prompt, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		prompt = strings.TrimSpace(prompt)

		// Handle slash commands
		if strings.HasPrefix(prompt, "/") {
			if prompt == "/models" || prompt == "/list" {
				// Fetch models from server only when requested
				if len(models) == 0 {
					fmt.Println("Fetching available models from server...")
					var err error
					models, err = sessionMgr.ListModels()
					if err != nil {
						fmt.Printf("Failed to list models: %v\n", err)
						continue
					}
					if len(models) == 0 {
						fmt.Println("No models available from server")
						continue
					}
				}

				fmt.Println("\nAvailable models:")
				for i, model := range models {
					prefix := "  "
					if model.ID == selectedModel {
						prefix = "* "
					}
					fmt.Printf("%s%d. %s (ID: %s)\n", prefix, i+1, model.Name, model.ID)
				}
				fmt.Printf("Enter model number (current: %s, press Enter to skip): ", selectedModel)
				modelInput, _ := reader.ReadString('\n')
				modelInput = strings.TrimSpace(modelInput)

				if modelInput != "" {
					var modelIdx int
					_, err := fmt.Sscanf(modelInput, "%d", &modelIdx)
					if err == nil && modelIdx > 0 && modelIdx <= len(models) {
						selectedModel = models[modelIdx-1].ID
						fmt.Printf("Switched to: %s\n\n", selectedModel)
						// Recreate session with new model
						if err := sessionMgr.Create(selectedModel); err != nil {
							log.Fatal(err)
						}
					}
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
