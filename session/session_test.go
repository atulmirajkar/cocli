package session

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	copilot "github.com/github/copilot-sdk/go"
)

// Mock implementations for testing
type mockClient struct {
	models      []copilot.ModelInfo
	createError error
	listError   error
}

func (m *mockClient) ListModels() ([]copilot.ModelInfo, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	if m.models == nil {
		return []copilot.ModelInfo{}, nil
	}
	return m.models, nil
}

func (m *mockClient) CreateSession(config *copilot.SessionConfig) (SessionInterface, error) {
	if m.createError != nil {
		return nil, m.createError
	}
	return &mockSession{model: config.Model}, nil
}

func (m *mockClient) Start() error {
	return nil
}

func (m *mockClient) Stop() []error {
	return nil
}

type mockSession struct {
	model string
}

func (m *mockSession) On(handler copilot.SessionEventHandler) func() {
	// Mock event handler registration
	return func() {} // Return unsubscribe function
}

func (m *mockSession) SendAndWait(options copilot.MessageOptions, timeout time.Duration) (*copilot.SessionEvent, error) {
	// Mock send functionality
	return nil, nil
}

// Test helper to capture stdout
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// TestGetCurrentModel tests the GetCurrentModel method
func TestGetCurrentModel(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected string
	}{
		{
			name:     "default model",
			model:    "Claude Haiku 4.5",
			expected: "Claude Haiku 4.5",
		},
		{
			name:     "gpt model",
			model:    "gpt-4.1",
			expected: "gpt-4.1",
		},
		{
			name:     "custom model",
			model:    "custom-model",
			expected: "custom-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &Manager{
				currentModel: tt.model,
			}
			result := mgr.GetCurrentModel()
			if result != tt.expected {
				t.Errorf("GetCurrentModel() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestGetCurrentMultiplier tests the GetCurrentMultiplier method
func TestGetCurrentMultiplier(t *testing.T) {
	tests := []struct {
		name       string
		multiplier float64
		expected   float64
	}{
		{
			name:       "zero multiplier",
			multiplier: 0,
			expected:   0,
		},
		{
			name:       "standard multiplier",
			multiplier: 1.0,
			expected:   1.0,
		},
		{
			name:       "low cost multiplier",
			multiplier: 0.33,
			expected:   0.33,
		},
		{
			name:       "high cost multiplier",
			multiplier: 3.0,
			expected:   3.0,
		},
		{
			name:       "fractional multiplier",
			multiplier: 0.5,
			expected:   0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &Manager{
				currentMultiplier: tt.multiplier,
			}
			result := mgr.GetCurrentMultiplier()
			if result != tt.expected {
				t.Errorf("GetCurrentMultiplier() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestGetTokenLimit tests the GetTokenLimit method
func TestGetTokenLimit(t *testing.T) {
	tests := []struct {
		name     string
		limit    int64
		expected int64
	}{
		{
			name:     "no limit",
			limit:    0,
			expected: 0,
		},
		{
			name:     "standard limit",
			limit:    4000,
			expected: 4000,
		},
		{
			name:     "high limit",
			limit:    128000,
			expected: 128000,
		},
		{
			name:     "low limit",
			limit:    1000,
			expected: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &Manager{
				tokenLimit: tt.limit,
			}
			result := mgr.GetTokenLimit()
			if result != tt.expected {
				t.Errorf("GetTokenLimit() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestGetTokensLeft tests the GetTokensLeft method
func TestGetTokensLeft(t *testing.T) {
	tests := []struct {
		name          string
		tokenLimit    int64
		currentTokens int64
		expected      int64
	}{
		{
			name:          "no tokens used",
			tokenLimit:    4000,
			currentTokens: 0,
			expected:      4000,
		},
		{
			name:          "half tokens used",
			tokenLimit:    4000,
			currentTokens: 2000,
			expected:      2000,
		},
		{
			name:          "all tokens used",
			tokenLimit:    4000,
			currentTokens: 4000,
			expected:      0,
		},
		{
			name:          "no limit set",
			tokenLimit:    0,
			currentTokens: 0,
			expected:      0,
		},
		{
			name:          "large limit",
			tokenLimit:    128000,
			currentTokens: 50000,
			expected:      78000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &Manager{
				tokenLimit:    tt.tokenLimit,
				currentTokens: tt.currentTokens,
			}
			result := mgr.GetTokensLeft()
			if result != tt.expected {
				t.Errorf("GetTokensLeft() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestHasTokenLimit tests the HasTokenLimit method
func TestHasTokenLimit(t *testing.T) {
	tests := []struct {
		name     string
		limit    int64
		expected bool
	}{
		{
			name:     "no limit",
			limit:    0,
			expected: false,
		},
		{
			name:     "limit set",
			limit:    4000,
			expected: true,
		},
		{
			name:     "large limit",
			limit:    128000,
			expected: true,
		},
		{
			name:     "small limit",
			limit:    1,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &Manager{
				tokenLimit: tt.limit,
			}
			result := mgr.HasTokenLimit()
			if result != tt.expected {
				t.Errorf("HasTokenLimit() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestTokenCalculations tests combined token-related operations
func TestTokenCalculations(t *testing.T) {
	tests := []struct {
		name           string
		tokenLimit     int64
		currentTokens  int64
		wantHasLimit   bool
		wantTokensLeft int64
	}{
		{
			name:           "new session",
			tokenLimit:     4000,
			currentTokens:  0,
			wantHasLimit:   true,
			wantTokensLeft: 4000,
		},
		{
			name:           "in progress session",
			tokenLimit:     4000,
			currentTokens:  500,
			wantHasLimit:   true,
			wantTokensLeft: 3500,
		},
		{
			name:           "almost exhausted",
			tokenLimit:     4000,
			currentTokens:  3999,
			wantHasLimit:   true,
			wantTokensLeft: 1,
		},
		{
			name:           "exhausted session",
			tokenLimit:     4000,
			currentTokens:  4000,
			wantHasLimit:   true,
			wantTokensLeft: 0,
		},
		{
			name:           "no limit",
			tokenLimit:     0,
			currentTokens:  0,
			wantHasLimit:   false,
			wantTokensLeft: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &Manager{
				tokenLimit:    tt.tokenLimit,
				currentTokens: tt.currentTokens,
			}

			if got := mgr.HasTokenLimit(); got != tt.wantHasLimit {
				t.Errorf("HasTokenLimit() = %v, want %v", got, tt.wantHasLimit)
			}

			if got := mgr.GetTokensLeft(); got != tt.wantTokensLeft {
				t.Errorf("GetTokensLeft() = %v, want %v", got, tt.wantTokensLeft)
			}
		})
	}
}

// TestModelState tests model-related state
func TestModelState(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		multiplier float64
		wantModel  string
		wantMult   float64
	}{
		{
			name:       "haiku model",
			model:      "claude-haiku-4.5",
			multiplier: 0.33,
			wantModel:  "claude-haiku-4.5",
			wantMult:   0.33,
		},
		{
			name:       "sonnet model",
			model:      "claude-sonnet-4.5",
			multiplier: 1.0,
			wantModel:  "claude-sonnet-4.5",
			wantMult:   1.0,
		},
		{
			name:       "opus model",
			model:      "claude-opus-4.5",
			multiplier: 3.0,
			wantModel:  "claude-opus-4.5",
			wantMult:   3.0,
		},
		{
			name:       "gpt model",
			model:      "gpt-4.1",
			multiplier: 1.0,
			wantModel:  "gpt-4.1",
			wantMult:   1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &Manager{
				currentModel:      tt.model,
				currentMultiplier: tt.multiplier,
			}

			if got := mgr.GetCurrentModel(); got != tt.wantModel {
				t.Errorf("GetCurrentModel() = %v, want %v", got, tt.wantModel)
			}

			if got := mgr.GetCurrentMultiplier(); got != tt.wantMult {
				t.Errorf("GetCurrentMultiplier() = %v, want %v", got, tt.wantMult)
			}
		})
	}
}

// TestCreate tests the Create method with mocked client
func TestCreate(t *testing.T) {
	tests := []struct {
		name        string
		model       string
		createError error
		wantError   bool
		errorMsg    string
	}{
		{
			name:      "successful create",
			model:     "claude-haiku-4.5",
			wantError: false,
		},
		{
			name:        "create fails",
			model:       "invalid-model",
			createError: fmt.Errorf("model not found"),
			wantError:   true,
			errorMsg:    "failed to create session",
		},
		{
			name:      "different model",
			model:     "gpt-4.1",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockClient{
				createError: tt.createError,
			}
			mgr := NewManagerWithClient(mockClient)

			err := mgr.Create(tt.model)

			if tt.wantError {
				if err == nil {
					t.Errorf("Create() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Create() error = %v, expected to contain %v", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Create() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestSetModel tests the SetModel method
func TestSetModel(t *testing.T) {
	tests := []struct {
		name        string
		modelID     string
		multiplier  float64
		createError error
		wantError   bool
		wantModel   string
		wantMult    float64
	}{
		{
			name:       "successful model switch",
			modelID:    "claude-sonnet-4.5",
			multiplier: 1.0,
			wantError:  false,
			wantModel:  "claude-sonnet-4.5",
			wantMult:   1.0,
		},
		{
			name:        "create session fails",
			modelID:     "invalid-model",
			multiplier:  1.0,
			createError: fmt.Errorf("session creation failed"),
			wantError:   true,
		},
		{
			name:       "zero multiplier",
			modelID:    "gpt-5-mini",
			multiplier: 0.0,
			wantError:  false,
			wantModel:  "gpt-5-mini",
			wantMult:   0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockClient{
				createError: tt.createError,
			}
			mgr := NewManagerWithClient(mockClient)

			err := mgr.SetModel(tt.modelID, tt.multiplier)

			if tt.wantError {
				if err == nil {
					t.Errorf("SetModel() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("SetModel() unexpected error = %v", err)
				}
				if mgr.GetCurrentModel() != tt.wantModel {
					t.Errorf("SetModel() model = %v, want %v", mgr.GetCurrentModel(), tt.wantModel)
				}
				if mgr.GetCurrentMultiplier() != tt.wantMult {
					t.Errorf("SetModel() multiplier = %v, want %v", mgr.GetCurrentMultiplier(), tt.wantMult)
				}
			}
		})
	}
}

// TestGetModels tests the GetModels method (caching and fetching)
func TestGetModels(t *testing.T) {
	testModels := []copilot.ModelInfo{
		{ID: "claude-haiku-4.5", Name: "Claude Haiku 4.5", Billing: &copilot.ModelBilling{Multiplier: 0.33}},
		{ID: "claude-sonnet-4.5", Name: "Claude Sonnet 4.5", Billing: &copilot.ModelBilling{Multiplier: 1.0}},
	}

	tests := []struct {
		name         string
		cachedModels []copilot.ModelInfo
		serverModels []copilot.ModelInfo
		listError    error
		wantError    bool
		wantModels   int
		shouldFetch  bool
	}{
		{
			name:         "fetch from server when cache empty",
			cachedModels: []copilot.ModelInfo{},
			serverModels: testModels,
			wantError:    false,
			wantModels:   2,
			shouldFetch:  true,
		},
		{
			name:         "return cached models",
			cachedModels: testModels,
			serverModels: []copilot.ModelInfo{},
			wantError:    false,
			wantModels:   2,
			shouldFetch:  false,
		},
		{
			name:         "server error",
			cachedModels: []copilot.ModelInfo{},
			serverModels: []copilot.ModelInfo{},
			listError:    fmt.Errorf("server unavailable"),
			wantError:    true,
		},
		{
			name:         "empty server response",
			cachedModels: []copilot.ModelInfo{},
			serverModels: []copilot.ModelInfo{},
			wantError:    false,
			wantModels:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockClient{
				models:    tt.serverModels,
				listError: tt.listError,
			}
			mgr := NewManagerWithClient(mockClient)
			mgr.models = tt.cachedModels

			// Capture output to verify "Fetching..." message
			output := captureOutput(func() {
				models, err := mgr.GetModels()

				if tt.wantError {
					if err == nil {
						t.Errorf("GetModels() expected error, got nil")
					}
				} else {
					if err != nil {
						t.Errorf("GetModels() unexpected error = %v", err)
					}
					if len(models) != tt.wantModels {
						t.Errorf("GetModels() got %d models, want %d", len(models), tt.wantModels)
					}
				}
			})

			if tt.shouldFetch && !strings.Contains(output, "Fetching available models") {
				t.Errorf("GetModels() should have printed fetching message")
			}
		})
	}
}

// TestListModels tests the direct ListModels method
func TestListModels(t *testing.T) {
	testModels := []copilot.ModelInfo{
		{ID: "model1", Name: "Model 1"},
		{ID: "model2", Name: "Model 2"},
	}

	tests := []struct {
		name      string
		models    []copilot.ModelInfo
		listError error
		wantError bool
		wantCount int
	}{
		{
			name:      "successful list",
			models:    testModels,
			wantError: false,
			wantCount: 2,
		},
		{
			name:      "empty list",
			models:    []copilot.ModelInfo{},
			wantError: false,
			wantCount: 0,
		},
		{
			name:      "list error",
			models:    []copilot.ModelInfo{},
			listError: fmt.Errorf("list failed"),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockClient{
				models:    tt.models,
				listError: tt.listError,
			}
			mgr := NewManagerWithClient(mockClient)

			models, err := mgr.ListModels()

			if tt.wantError {
				if err == nil {
					t.Errorf("ListModels() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ListModels() unexpected error = %v", err)
				}
				if len(models) != tt.wantCount {
					t.Errorf("ListModels() got %d models, want %d", len(models), tt.wantCount)
				}
			}
		})
	}
}

// TestDisplayModels tests the DisplayModels method
func TestDisplayModels(t *testing.T) {
	testModels := []copilot.ModelInfo{
		{ID: "claude-haiku-4.5", Name: "Claude Haiku 4.5", Billing: &copilot.ModelBilling{Multiplier: 0.33}},
		{ID: "claude-sonnet-4.5", Name: "Claude Sonnet 4.5", Billing: &copilot.ModelBilling{Multiplier: 1.0}},
		{ID: "claude-opus-4.5", Name: "Claude Opus 4.5", Billing: &copilot.ModelBilling{Multiplier: 3.0}},
	}

	tests := []struct {
		name           string
		models         []copilot.ModelInfo
		currentModel   string
		wantError      bool
		expectInOutput []string
	}{
		{
			name:         "display models with current selection",
			models:       testModels,
			currentModel: "claude-sonnet-4.5",
			wantError:    false,
			expectInOutput: []string{
				"Available models:",
				"1. Claude Haiku 4.5 (ID: claude-haiku-4.5) (0.33x)",
				"* 2. Claude Sonnet 4.5 (ID: claude-sonnet-4.5) (1.00x)",
				"3. Claude Opus 4.5 (ID: claude-opus-4.5) (3.00x)",
			},
		},
		{
			name:      "no models error",
			models:    []copilot.ModelInfo{},
			wantError: true,
		},
		{
			name: "model without billing info",
			models: []copilot.ModelInfo{
				{ID: "free-model", Name: "Free Model"},
			},
			currentModel: "free-model",
			wantError:    false,
			expectInOutput: []string{
				"* 1. Free Model (ID: free-model)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewManagerWithClient(&mockClient{})
			mgr.models = tt.models
			mgr.currentModel = tt.currentModel

			output := captureOutput(func() {
				err := mgr.DisplayModels()

				if tt.wantError {
					if err == nil {
						t.Errorf("DisplayModels() expected error, got nil")
					}
				} else {
					if err != nil {
						t.Errorf("DisplayModels() unexpected error = %v", err)
					}
				}
			})

			if !tt.wantError {
				for _, expected := range tt.expectInOutput {
					if !strings.Contains(output, expected) {
						t.Errorf("DisplayModels() output should contain %q, got:\n%s", expected, output)
					}
				}
			}
		})
	}
}

// TestNewManagerWithClient tests the NewManagerWithClient function
func TestNewManagerWithClient(t *testing.T) {
	tests := []struct {
		name          string
		client        ClientInterface
		expectedModel string
		expectedMult  float64
	}{
		{
			name:          "successful creation with mock client",
			client:        &mockClient{},
			expectedModel: "Claude Haiku 4.5",
			expectedMult:  0,
		},
		{
			name:          "creation with different client",
			client:        &mockClient{models: []copilot.ModelInfo{}},
			expectedModel: "Claude Haiku 4.5",
			expectedMult:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewManagerWithClient(tt.client)

			if mgr == nil {
				t.Error("NewManagerWithClient() returned nil")
				return
			}

			if mgr.GetCurrentModel() != tt.expectedModel {
				t.Errorf("Expected model %s, got %s", tt.expectedModel, mgr.GetCurrentModel())
			}

			if mgr.GetCurrentMultiplier() != tt.expectedMult {
				t.Errorf("Expected multiplier %f, got %f", tt.expectedMult, mgr.GetCurrentMultiplier())
			}

			// Verify models slice is initialized
			models, err := mgr.GetModels()
			if err != nil {
				t.Errorf("GetModels() returned error: %v", err)
			}

			// Should be empty initially or contain mocked models
			if models == nil {
				t.Error("GetModels() returned nil slice")
			}
		})
	}
}

// TestCreateSession tests session creation with different scenarios
func TestCreateSession(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		createError   error
		wantError     bool
		errorContains string
	}{
		{
			name:      "successful session creation",
			model:     "claude-haiku-4.5",
			wantError: false,
		},
		{
			name:          "session creation failure",
			model:         "invalid-model",
			createError:   fmt.Errorf("model not supported"),
			wantError:     true,
			errorContains: "failed to create session",
		},
		{
			name:      "different valid model",
			model:     "gpt-4.1",
			wantError: false,
		},
		{
			name:          "client connection error",
			model:         "claude-sonnet-4.5",
			createError:   fmt.Errorf("connection refused"),
			wantError:     true,
			errorContains: "failed to create session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockClient{
				createError: tt.createError,
			}
			mgr := NewManagerWithClient(mockClient)

			err := mgr.Create(tt.model)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				// Verify session was created successfully
				// Token limits should be reset
				if mgr.GetTokenLimit() != 0 {
					t.Errorf("Expected token limit to be reset to 0, got %d", mgr.GetTokenLimit())
				}
				if mgr.GetTokensLeft() != 0 {
					t.Errorf("Expected tokens left to be 0, got %d", mgr.GetTokensLeft())
				}
			}
		})
	}
}

// TestSessionLifecycle tests the complete session lifecycle
func TestSessionLifecycle(t *testing.T) {
	mockClient := &mockClient{}
	mgr := NewManagerWithClient(mockClient)

	// Test initial state
	initialModel := mgr.GetCurrentModel()
	if initialModel != "Claude Haiku 4.5" {
		t.Errorf("Expected initial model 'Claude Haiku 4.5', got %s", initialModel)
	}

	// Test creating a session
	err := mgr.Create("test-model")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Test setting a new model (which creates a new session)
	err = mgr.SetModel("new-model", 1.5)
	if err != nil {
		t.Fatalf("Failed to set model: %v", err)
	}

	if mgr.GetCurrentModel() != "new-model" {
		t.Errorf("Expected model 'new-model', got %s", mgr.GetCurrentModel())
	}

	if mgr.GetCurrentMultiplier() != 1.5 {
		t.Errorf("Expected multiplier 1.5, got %f", mgr.GetCurrentMultiplier())
	}

	// Test token handling
	if mgr.HasTokenLimit() {
		t.Error("Expected no token limit initially")
	}

	// Simulate setting token limits (normally done by event handlers)
	mgr.tokenLimit = 4000
	mgr.currentTokens = 1000

	if !mgr.HasTokenLimit() {
		t.Error("Expected token limit to be set")
	}

	if mgr.GetTokensLeft() != 3000 {
		t.Errorf("Expected 3000 tokens left, got %d", mgr.GetTokensLeft())
	}
}

// Test helper functions and edge cases
func TestManagerEdgeCases(t *testing.T) {
	t.Run("nil client handling", func(t *testing.T) {
		// This tests what happens with a nil client - should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("NewManagerWithClient panicked with nil client: %v", r)
			}
		}()

		mgr := NewManagerWithClient(nil)
		if mgr == nil {
			t.Error("NewManagerWithClient returned nil with nil client")
		}
	})

	t.Run("multiple create calls", func(t *testing.T) {
		mgr := NewManagerWithClient(&mockClient{})

		// First create
		err1 := mgr.Create("model1")
		if err1 != nil {
			t.Fatalf("First create failed: %v", err1)
		}

		// Second create should replace the session
		err2 := mgr.Create("model2")
		if err2 != nil {
			t.Fatalf("Second create failed: %v", err2)
		}

		// Should not have any residual state issues
		if mgr.GetTokenLimit() != 0 {
			t.Error("Token limit should be reset after new session creation")
		}
	})

	t.Run("close with errors", func(t *testing.T) {
		// Test client that returns errors on close
		errorClient := &mockClientWithCloseError{
			mockClient:  &mockClient{},
			closeErrors: []error{fmt.Errorf("cleanup error 1"), fmt.Errorf("cleanup error 2")},
		}

		mgr := NewManagerWithClient(errorClient)
		errors := mgr.Close()

		if len(errors) != 2 {
			t.Errorf("Expected 2 close errors, got %d", len(errors))
		}
	})
}

// Additional mock for testing close errors
type mockClientWithCloseError struct {
	*mockClient
	closeErrors []error
}

func (m *mockClientWithCloseError) Stop() []error {
	return m.closeErrors
}
