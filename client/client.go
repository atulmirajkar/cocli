package client

import (
	"fmt"

	"atulm/cocli/server"

	copilot "github.com/github/copilot-sdk/go"
)

// ClientInterface defines the interface for copilot client operations
type ClientInterface interface {
	CreateSession(*copilot.SessionConfig) (*copilot.Session, error)
	ListModels() ([]copilot.ModelInfo, error)
	Start() error
	Stop() []error
}

// DaemonChecker interface for checking daemon status (for testability)
type DaemonChecker interface {
	IsRunning() bool
	GetPort() (int, error)
}

// sdkClient wraps the actual copilot.Client to implement ClientInterface
type sdkClient struct {
	*copilot.Client
}

func (s *sdkClient) CreateSession(config *copilot.SessionConfig) (*copilot.Session, error) {
	return s.Client.CreateSession(config)
}

// Client manages the copilot SDK client and model caching
type Client struct {
	sdk         ClientInterface
	models      []copilot.ModelInfo
	usingDaemon bool
}

// NewClient creates a new client, automatically connecting to a running daemon
// if available, otherwise starting an embedded server.
func NewClient() (*Client, error) {
	return NewClientWithDaemonChecker(nil)
}

// NewClientWithDaemonChecker creates a new client with a custom daemon checker (for testing)
func NewClientWithDaemonChecker(daemonChecker DaemonChecker) (*Client, error) {
	var sdkCli *copilot.Client
	var usingDaemon bool

	// Use default daemon checker if not provided
	if daemonChecker == nil {
		dm, err := server.DefaultDaemonManager()
		if err == nil {
			daemonChecker = dm
		}
	}

	// Try to connect to daemon first
	if daemonChecker != nil && daemonChecker.IsRunning() {
		port, err := daemonChecker.GetPort()
		if err == nil {
			sdkCli = copilot.NewClient(&copilot.ClientOptions{
				CLIUrl: fmt.Sprintf("localhost:%d", port),
			})
			usingDaemon = true
		}
	}

	// Fallback to embedded server
	if sdkCli == nil {
		sdkCli = copilot.NewClient(nil)
	}

	if err := sdkCli.Start(); err != nil {
		if usingDaemon {
			// Daemon connection failed, fall back to embedded server
			fmt.Println("Warning: daemon connection failed, starting embedded server...")
			sdkCli = copilot.NewClient(nil)
			usingDaemon = false
			if err := sdkCli.Start(); err != nil {
				return nil, fmt.Errorf("failed to start client: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to start client: %w", err)
		}
	}

	return &Client{
		sdk:         &sdkClient{sdkCli},
		models:      []copilot.ModelInfo{},
		usingDaemon: usingDaemon,
	}, nil
}

// NewClientWithSDK creates a client with a custom SDK client (for testing)
func NewClientWithSDK(sdk ClientInterface) *Client {
	return &Client{
		sdk:         sdk,
		models:      []copilot.ModelInfo{},
		usingDaemon: false,
	}
}

// CreateSession creates a new copilot session with the given configuration
func (c *Client) CreateSession(config *copilot.SessionConfig) (*copilot.Session, error) {
	return c.sdk.CreateSession(config)
}

// ListModels returns available models from the server (no caching)
func (c *Client) ListModels() ([]copilot.ModelInfo, error) {
	return c.sdk.ListModels()
}

// GetModels returns cached models, fetching from server if needed
func (c *Client) GetModels() ([]copilot.ModelInfo, error) {
	if len(c.models) == 0 {
		fmt.Println("Fetching available models from server...")
		var err error
		c.models, err = c.sdk.ListModels()
		if err != nil {
			return nil, err
		}
	}
	return c.models, nil
}

// IsUsingDaemon returns true if the client is connected to a daemon
func (c *Client) IsUsingDaemon() bool {
	return c.usingDaemon
}

// Stop stops the client and cleans up resources
func (c *Client) Stop() []error {
	return c.sdk.Stop()
}
