package session

import (
	"fmt"

	copilot "github.com/github/copilot-sdk/go"
)

// Manager handles session creation and lifecycle
type Manager struct {
	client        *copilot.Client
	session       *copilot.Session
	CurrentTokens int64
	TokenLimit    int64
}

// NewManager creates a new session manager and initializes the client
func NewManager() (*Manager, error) {
	client := copilot.NewClient(nil)
	if err := client.Start(); err != nil {
		return nil, fmt.Errorf("failed to start client: %w", err)
	}

	return &Manager{
		client: client,
	}, nil
}

// Create creates a new session with the given model and sets up event handlers
func (m *Manager) Create(model string) error {
	session, err := m.client.CreateSession(&copilot.SessionConfig{
		Model:     model,
		Streaming: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	m.session = session
	m.CurrentTokens = 0
	m.TokenLimit = 0

	// Set up event listeners
	m.setupEventHandlers()

	return nil
}

// setupEventHandlers configures the session event listeners
func (m *Manager) setupEventHandlers() {
	m.session.On(func(event copilot.SessionEvent) {
		if event.Type == "assistant.message_delta" {
			if event.Data.DeltaContent != nil {
				fmt.Print(*event.Data.DeltaContent)
			}
		} else if event.Type == "session.idle" {
			fmt.Println()
		}

		// Update token counts from events
		if event.Data.CurrentTokens != nil {
			m.CurrentTokens = int64(*event.Data.CurrentTokens)
		}
		if event.Data.TokenLimit != nil {
			m.TokenLimit = int64(*event.Data.TokenLimit)
		}
	})
}

// Send sends a message to the current session and waits for response
func (m *Manager) Send(prompt string) error {
	if m.session == nil {
		return fmt.Errorf("no active session")
	}

	_, err := m.session.SendAndWait(copilot.MessageOptions{
		Prompt: prompt,
	}, 0)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// ListModels returns available models from the server
func (m *Manager) ListModels() ([]copilot.ModelInfo, error) {
	return m.client.ListModels()
}

// GetTokensLeft returns the number of tokens remaining
func (m *Manager) GetTokensLeft() int64 {
	return m.TokenLimit - m.CurrentTokens
}

// HasTokenLimit returns whether a token limit is known
func (m *Manager) HasTokenLimit() bool {
	return m.TokenLimit > 0
}

// Close stops the client and cleans up resources
func (m *Manager) Close() []error {
	return m.client.Stop()
}
