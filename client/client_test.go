package client

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	copilot "github.com/github/copilot-sdk/go"
)

// Mock implementations
type mockSDKClient struct {
	models       []copilot.ModelInfo
	createError  error
	listError    error
	startError   error
	stopErrors   []error
	startCalled  bool
	stopCalled   bool
	listCalled   int
	createCalled int
}

func (m *mockSDKClient) ListModels() ([]copilot.ModelInfo, error) {
	m.listCalled++
	if m.listError != nil {
		return nil, m.listError
	}
	if m.models == nil {
		return []copilot.ModelInfo{}, nil
	}
	return m.models, nil
}

func (m *mockSDKClient) CreateSession(config *copilot.SessionConfig) (*copilot.Session, error) {
	m.createCalled++
	if m.createError != nil {
		return nil, m.createError
	}
	return nil, nil // Return nil session for testing
}

func (m *mockSDKClient) Start() error {
	m.startCalled = true
	return m.startError
}

func (m *mockSDKClient) Stop() []error {
	m.stopCalled = true
	return m.stopErrors
}

// Mock daemon checker
type mockDaemonChecker struct {
	running bool
	port    int
	portErr error
}

func (m *mockDaemonChecker) IsRunning() bool {
	return m.running
}

func (m *mockDaemonChecker) GetPort() (int, error) {
	if m.portErr != nil {
		return 0, m.portErr
	}
	return m.port, nil
}

// captureOutput helper
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

// TestNewClientWithSDK tests creation with mock SDK
func TestNewClientWithSDK(t *testing.T) {
	tests := []struct {
		name       string
		sdk        ClientInterface
		wantNil    bool
		wantDaemon bool
	}{
		{
			name:       "successful creation",
			sdk:        &mockSDKClient{},
			wantNil:    false,
			wantDaemon: false,
		},
		{
			name:       "with models",
			sdk:        &mockSDKClient{models: []copilot.ModelInfo{{ID: "test"}}},
			wantNil:    false,
			wantDaemon: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClientWithSDK(tt.sdk)

			if (client == nil) != tt.wantNil {
				t.Errorf("NewClientWithSDK() nil = %v, want %v", client == nil, tt.wantNil)
			}

			if client != nil && client.IsUsingDaemon() != tt.wantDaemon {
				t.Errorf("IsUsingDaemon() = %v, want %v", client.IsUsingDaemon(), tt.wantDaemon)
			}
		})
	}
}

// TestGetModels_Caching tests that models are cached after first fetch
func TestGetModels_Caching(t *testing.T) {
	testModels := []copilot.ModelInfo{
		{ID: "model1", Name: "Model 1"},
		{ID: "model2", Name: "Model 2"},
	}

	mock := &mockSDKClient{models: testModels}
	client := NewClientWithSDK(mock)

	// First call should fetch and print message
	output := captureOutput(func() {
		models, err := client.GetModels()
		if err != nil {
			t.Fatalf("GetModels() first call error: %v", err)
		}
		if len(models) != 2 {
			t.Errorf("Expected 2 models, got %d", len(models))
		}
	})

	if !strings.Contains(output, "Fetching available models") {
		t.Error("Expected fetching message on first call")
	}

	if mock.listCalled != 1 {
		t.Errorf("Expected ListModels to be called once, called %d times", mock.listCalled)
	}

	// Second call should use cache (no output, no additional ListModels call)
	output = captureOutput(func() {
		models, err := client.GetModels()
		if err != nil {
			t.Fatalf("GetModels() second call error: %v", err)
		}
		if len(models) != 2 {
			t.Errorf("Expected 2 models, got %d", len(models))
		}
	})

	if strings.Contains(output, "Fetching available models") {
		t.Error("Should not fetch on second call (cached)")
	}

	if mock.listCalled != 1 {
		t.Errorf("Expected ListModels to still be called only once, called %d times", mock.listCalled)
	}
}

// TestGetModels_Error tests error handling
func TestGetModels_Error(t *testing.T) {
	mock := &mockSDKClient{listError: fmt.Errorf("network error")}
	client := NewClientWithSDK(mock)

	output := captureOutput(func() {
		_, err := client.GetModels()
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !strings.Contains(err.Error(), "network error") {
			t.Errorf("Expected network error, got: %v", err)
		}
	})

	// Should still print fetching message before error
	if !strings.Contains(output, "Fetching available models") {
		t.Error("Expected fetching message before error")
	}
}

// TestListModels tests direct ListModels passthrough
func TestListModels(t *testing.T) {
	tests := []struct {
		name      string
		models    []copilot.ModelInfo
		listError error
		wantError bool
		wantCount int
	}{
		{
			name:      "successful list",
			models:    []copilot.ModelInfo{{ID: "model1"}, {ID: "model2"}},
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
			name:      "nil models",
			models:    nil,
			wantError: false,
			wantCount: 0,
		},
		{
			name:      "list error",
			listError: fmt.Errorf("server error"),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSDKClient{models: tt.models, listError: tt.listError}
			client := NewClientWithSDK(mock)

			models, err := client.ListModels()

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(models) != tt.wantCount {
					t.Errorf("Expected %d models, got %d", tt.wantCount, len(models))
				}
			}
		})
	}
}

// TestCreateSession tests session creation passthrough
func TestCreateSession(t *testing.T) {
	tests := []struct {
		name        string
		createError error
		wantError   bool
	}{
		{
			name:      "successful create",
			wantError: false,
		},
		{
			name:        "create error",
			createError: fmt.Errorf("session creation failed"),
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSDKClient{createError: tt.createError}
			client := NewClientWithSDK(mock)

			_, err := client.CreateSession(&copilot.SessionConfig{Model: "test-model"})

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if mock.createCalled != 1 {
				t.Errorf("Expected CreateSession to be called once, called %d times", mock.createCalled)
			}
		})
	}
}

// TestStop tests client cleanup
func TestStop(t *testing.T) {
	tests := []struct {
		name       string
		stopErrors []error
		wantCount  int
	}{
		{
			name:       "no errors",
			stopErrors: nil,
			wantCount:  0,
		},
		{
			name:       "single error",
			stopErrors: []error{fmt.Errorf("cleanup error")},
			wantCount:  1,
		},
		{
			name:       "multiple errors",
			stopErrors: []error{fmt.Errorf("error1"), fmt.Errorf("error2")},
			wantCount:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSDKClient{stopErrors: tt.stopErrors}
			client := NewClientWithSDK(mock)

			errs := client.Stop()

			if len(errs) != tt.wantCount {
				t.Errorf("Expected %d errors, got %d", tt.wantCount, len(errs))
			}

			if !mock.stopCalled {
				t.Error("Expected Stop to be called on SDK")
			}
		})
	}
}

// TestIsUsingDaemon tests the daemon flag
func TestIsUsingDaemon(t *testing.T) {
	mock := &mockSDKClient{}
	client := NewClientWithSDK(mock)

	if client.IsUsingDaemon() {
		t.Error("NewClientWithSDK should set usingDaemon to false")
	}
}

// TestClientInterface ensures Client implements expected methods
func TestClientInterface(t *testing.T) {
	// This test verifies that our mock implements the interface correctly
	var _ ClientInterface = (*mockSDKClient)(nil)
}

// TestListModels_DoesNotAffectCache verifies ListModels doesn't update cache
func TestListModels_DoesNotAffectCache(t *testing.T) {
	testModels := []copilot.ModelInfo{
		{ID: "model1", Name: "Model 1"},
	}

	mock := &mockSDKClient{models: testModels}
	client := NewClientWithSDK(mock)

	// Call ListModels (should not cache)
	_, _ = client.ListModels()

	// GetModels should still fetch (cache should be empty)
	output := captureOutput(func() {
		_, _ = client.GetModels()
	})

	if !strings.Contains(output, "Fetching available models") {
		t.Error("GetModels should fetch even after ListModels was called")
	}

	// ListModels should have been called twice now
	if mock.listCalled != 2 {
		t.Errorf("Expected ListModels to be called twice, called %d times", mock.listCalled)
	}
}
