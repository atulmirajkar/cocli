package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileConfigStore_GetPath(t *testing.T) {
	store := NewFileConfigStore("/home/user/.cocli")
	expected := "/home/user/.cocli/server.json"
	if got := store.GetPath(); got != expected {
		t.Errorf("GetPath() = %q, want %q", got, expected)
	}
}

func TestFileConfigStore_Save_Load(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileConfigStore(tmpDir)

	startTime := time.Now().Truncate(time.Second) // Truncate for comparison
	original := &DaemonConfig{
		PID:       12345,
		Port:      4321,
		StartedAt: startTime,
	}

	// Save config
	if err := store.Save(original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(store.GetPath()); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load config back
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify fields
	if loaded.PID != original.PID {
		t.Errorf("PID = %d, want %d", loaded.PID, original.PID)
	}
	if loaded.Port != original.Port {
		t.Errorf("Port = %d, want %d", loaded.Port, original.Port)
	}
	if !loaded.StartedAt.Equal(original.StartedAt) {
		t.Errorf("StartedAt = %v, want %v", loaded.StartedAt, original.StartedAt)
	}
}

func TestFileConfigStore_Load_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileConfigStore(tmpDir)

	_, err := store.Load()
	if err != ErrConfigNotFound {
		t.Errorf("Load() error = %v, want ErrConfigNotFound", err)
	}
}

func TestFileConfigStore_Load_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileConfigStore(tmpDir)

	// Write invalid JSON
	path := store.GetPath()
	if err := os.WriteFile(path, []byte("not valid json{"), 0644); err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}

	_, err := store.Load()
	if err == nil {
		t.Error("Load() expected error for invalid JSON, got nil")
	}

	// Verify it's a JSON syntax error
	var syntaxErr *json.SyntaxError
	if !isJSONError(err) {
		t.Errorf("Load() error = %T, want JSON unmarshal error", err)
	}
	_ = syntaxErr // suppress unused warning
}

func isJSONError(err error) bool {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	if err == nil {
		return false
	}
	// Check if it's any kind of JSON error
	return json.Unmarshal([]byte("invalid"), nil) != nil &&
		(err.Error() != "" && (contains(err.Error(), "invalid") ||
			contains(err.Error(), "unexpected") ||
			contains(err.Error(), "cannot unmarshal")))
	_ = syntaxErr
	_ = typeErr
	return true
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestFileConfigStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileConfigStore(tmpDir)

	// Save first
	config := &DaemonConfig{PID: 123, Port: 4321, StartedAt: time.Now()}
	if err := store.Save(config); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(store.GetPath()); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Delete
	if err := store.Delete(); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(store.GetPath()); !os.IsNotExist(err) {
		t.Error("Config file still exists after Delete()")
	}
}

func TestFileConfigStore_Delete_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileConfigStore(tmpDir)

	// Delete non-existent file should not error
	if err := store.Delete(); err != nil {
		t.Errorf("Delete() error = %v, want nil for non-existent file", err)
	}
}

func TestFileConfigStore_Save_CreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "nested", "config", "dir")
	store := NewFileConfigStore(nestedDir)

	config := &DaemonConfig{PID: 123, Port: 4321, StartedAt: time.Now()}
	if err := store.Save(config); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Error("Nested directory was not created")
	}

	// Verify file exists
	if _, err := os.Stat(store.GetPath()); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

func TestFileConfigStore_Save_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileConfigStore(tmpDir)

	// Save first config
	config1 := &DaemonConfig{PID: 111, Port: 1111, StartedAt: time.Now()}
	if err := store.Save(config1); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Save second config (overwrite)
	config2 := &DaemonConfig{PID: 222, Port: 2222, StartedAt: time.Now()}
	if err := store.Save(config2); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load and verify it's the second config
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.PID != 222 {
		t.Errorf("PID = %d, want 222 (overwritten value)", loaded.PID)
	}
	if loaded.Port != 2222 {
		t.Errorf("Port = %d, want 2222 (overwritten value)", loaded.Port)
	}
}

func TestDefaultConfigStore(t *testing.T) {
	store, err := DefaultConfigStore()
	if err != nil {
		t.Fatalf("DefaultConfigStore() error = %v", err)
	}

	// Verify path contains .cocli
	path := store.GetPath()
	if !contains(path, ".cocli") {
		t.Errorf("GetPath() = %q, want path containing .cocli", path)
	}
	if !contains(path, "server.json") {
		t.Errorf("GetPath() = %q, want path containing server.json", path)
	}
}
