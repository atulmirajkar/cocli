package session

import (
	"fmt"
	"time"

	copilot "github.com/github/copilot-sdk/go"
)

// ClientInterface defines the interface for copilot client operations
type ClientInterface interface {
	CreateSession(*copilot.SessionConfig) (SessionInterface, error)
	ListModels() ([]copilot.ModelInfo, error)
	Start() error
	Stop() []error
}

// SessionInterface defines the interface for session operations
type SessionInterface interface {
	On(copilot.SessionEventHandler) func()
	SendAndWait(copilot.MessageOptions, time.Duration) (*copilot.SessionEvent, error)
}

// copilotClient wraps the actual copilot.Client to implement ClientInterface
type copilotClient struct {
	*copilot.Client
}

func (r *copilotClient) CreateSession(config *copilot.SessionConfig) (SessionInterface, error) {
	session, err := r.Client.CreateSession(config)
	if err != nil {
		return nil, err
	}
	return &copilotSession{session}, nil
}

// copilotSession wraps the actual copilot.Session to implement SessionInterface
type copilotSession struct {
	*copilot.Session
}

// Manager handles session creation and lifecycle
type Manager struct {
	client            ClientInterface
	session           SessionInterface
	currentTokens     int64
	tokenLimit        int64
	currentModel      string
	currentMultiplier float64
	models            []copilot.ModelInfo
	renderer          *StreamingMarkdownRenderer
}

// NewManager creates a new session manager and initializes the client with default model
func NewManager() (*Manager, error) {
	client := copilot.NewClient(nil)
	if err := client.Start(); err != nil {
		return nil, fmt.Errorf("failed to start client: %w", err)
	}

	renderer, err := NewStreamingMarkdownRenderer()
	if err != nil {
		return nil, fmt.Errorf("failed to create markdown renderer: %w", err)
	}

	mgr := &Manager{
		client:            &copilotClient{client},
		currentModel:      "Claude Sonnet 4.5",
		currentMultiplier: 0,
		models:            []copilot.ModelInfo{},
		renderer:          renderer,
	}

	// Create initial session with default model
	if err := mgr.Create(mgr.currentModel); err != nil {
		return nil, fmt.Errorf("failed to create initial session: %w", err)
	}

	return mgr, nil
}

// NewManagerWithClient creates a manager with a custom client (for testing)
func NewManagerWithClient(client ClientInterface) *Manager {
	return &Manager{
		client:            client,
		currentModel:      "Claude Sonnet 4.5",
		currentMultiplier: 0,
		models:            []copilot.ModelInfo{},
		renderer:          nil, // Will be set up when Create() is called or can be set explicitly
	}
}

// SetRenderer sets a custom renderer (useful for testing)
func (m *Manager) SetRenderer(r *StreamingMarkdownRenderer) {
	m.renderer = r
}

// Create creates a new session with the given model and sets up event handlers.
// The session is configured with a system message that instructs the model to
// always format responses using markdown, ensuring consistent, high-quality output
// that works well with the streaming markdown renderer.
func (m *Manager) Create(model string) error {
	session, err := m.client.CreateSession(&copilot.SessionConfig{
		Model:     model,
		Streaming: true,
		SystemMessage: &copilot.SystemMessageConfig{
			Mode:    "append",
			Content: "Always format responses using markdown with code blocks.",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	m.session = session
	m.currentTokens = 0
	m.tokenLimit = 0

	// Set up event listeners
	m.setupEventHandlers()

	return nil
}

// setupEventHandlers configures the session event listeners
func (m *Manager) setupEventHandlers() {
	m.session.On(func(event copilot.SessionEvent) {
		if event.Type == "assistant.message_delta" {
			if event.Data.DeltaContent != nil {
				if m.renderer != nil {
					m.renderer.ProcessDelta(*event.Data.DeltaContent)
				} else {
					// Fallback to plain text if renderer not available
					fmt.Print(*event.Data.DeltaContent)
				}
			}
		} else if event.Type == "session.idle" {
			if m.renderer != nil {
				m.renderer.Flush()
			}
			fmt.Println()
		}

		// Update token counts from events
		if event.Data.CurrentTokens != nil {
			m.currentTokens = int64(*event.Data.CurrentTokens)
		}
		if event.Data.TokenLimit != nil {
			m.tokenLimit = int64(*event.Data.TokenLimit)
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

// GetModels returns cached models, fetching from server if needed
func (m *Manager) GetModels() ([]copilot.ModelInfo, error) {
	if len(m.models) == 0 {
		fmt.Println("Fetching available models from server...")
		var err error
		m.models, err = m.client.ListModels()
		if err != nil {
			return nil, err
		}
	}
	return m.models, nil
}

// DisplayModels prints the list of available models with billing info
func (m *Manager) DisplayModels() error {
	if len(m.models) == 0 {
		return fmt.Errorf("no models available")
	}

	fmt.Println("\nAvailable models:")
	for i, model := range m.models {
		prefix := "  "
		if model.ID == m.currentModel {
			prefix = "* "
		}
		billingInfo := ""
		if model.Billing != nil {
			billingInfo = fmt.Sprintf(" (%.2fx)", model.Billing.Multiplier)
		}
		fmt.Printf("%s%d. %s (ID: %s)%s\n", prefix, i+1, model.Name, model.ID, billingInfo)
	}
	return nil
}

// ListModels returns available models from the server
func (m *Manager) ListModels() ([]copilot.ModelInfo, error) {
	return m.client.ListModels()
}

// SetModel switches to a new model with the given billing multiplier and creates a new session
func (m *Manager) SetModel(modelID string, multiplier float64) error {
	if err := m.Create(modelID); err != nil {
		return err
	}
	m.currentModel = modelID
	m.currentMultiplier = multiplier
	return nil
}

// GetCurrentModel returns the ID of the currently selected model
func (m *Manager) GetCurrentModel() string {
	return m.currentModel
}

// GetCurrentMultiplier returns the billing multiplier of the currently selected model
func (m *Manager) GetCurrentMultiplier() float64 {
	return m.currentMultiplier
}

// GetTokensLeft returns the number of tokens remaining
func (m *Manager) GetTokensLeft() int64 {
	return m.tokenLimit - m.currentTokens
}

// HasTokenLimit returns whether a token limit is known
func (m *Manager) HasTokenLimit() bool {
	return m.tokenLimit > 0
}

// GetTokenLimit returns the token limit for the current session
func (m *Manager) GetTokenLimit() int64 {
	return m.tokenLimit
}

// Close stops the client and cleans up resources
func (m *Manager) Close() []error {
	return m.client.Stop()
}
