package server

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const (
	configDirName  = ".cocli"
	configFileName = "server.json"
)

// ErrConfigNotFound is returned when the config file doesn't exist
var ErrConfigNotFound = errors.New("config file not found")

// DaemonConfig represents the persisted daemon state
type DaemonConfig struct {
	PID       int       `json:"pid"`
	Port      int       `json:"port"`
	StartedAt time.Time `json:"started_at"`
}

// ConfigStore interface for config file operations
type ConfigStore interface {
	// Load reads the daemon config from storage
	Load() (*DaemonConfig, error)
	// Save persists the daemon config to storage
	Save(config *DaemonConfig) error
	// Delete removes the config from storage
	Delete() error
	// GetPath returns the path to the config file
	GetPath() string
}

// FileConfigStore implements ConfigStore using the filesystem
type FileConfigStore struct {
	configDir string
}

// NewFileConfigStore creates a ConfigStore with a custom config directory
func NewFileConfigStore(configDir string) *FileConfigStore {
	return &FileConfigStore{configDir: configDir}
}

// DefaultConfigStore returns a ConfigStore using ~/.cocli
func DefaultConfigStore() (*FileConfigStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	configDir := filepath.Join(home, configDirName)
	return &FileConfigStore{configDir: configDir}, nil
}

// GetPath returns the full path to the config file
func (s *FileConfigStore) GetPath() string {
	return filepath.Join(s.configDir, configFileName)
}

// Load reads the daemon config from the file
func (s *FileConfigStore) Load() (*DaemonConfig, error) {
	path := s.GetPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrConfigNotFound
		}
		return nil, err
	}

	var config DaemonConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// Save persists the daemon config to the file
func (s *FileConfigStore) Save(config *DaemonConfig) error {
	// Ensure config directory exists
	if err := os.MkdirAll(s.configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	path := s.GetPath()
	return os.WriteFile(path, data, 0644)
}

// Delete removes the config file
func (s *FileConfigStore) Delete() error {
	path := s.GetPath()
	err := os.Remove(path)
	if err != nil && os.IsNotExist(err) {
		// File doesn't exist, that's fine
		return nil
	}
	return err
}
