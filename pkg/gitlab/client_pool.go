package gitlab

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	log "github.com/sirupsen/logrus"
	gl "gitlab.com/gitlab-org/api/client-go"
)

// ClientPool manages multiple GitLab clients for different servers
type ClientPool struct {
	clients map[string]*gl.Client // key: server name
	store   *TokenStore
	logger  *log.Logger
	mu      sync.RWMutex
}

// NewClientPool creates a new client pool
func NewClientPool(store *TokenStore, logger *log.Logger) *ClientPool {
	return &ClientPool{
		clients: make(map[string]*gl.Client),
		store:   store,
		logger:  logger,
	}
}

// AddClient adds a new client to the pool
func (cp *ClientPool) AddClient(name string, client *gl.Client) error {
	if name == "" {
		return fmt.Errorf("client name cannot be empty")
	}
	if client == nil {
		return fmt.Errorf("client cannot be nil")
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.clients[name] = client
	cp.logger.Infof("Added client '%s' to pool", name)
	return nil
}

// GetClient retrieves a client by name
func (cp *ClientPool) GetClient(name string) (*gl.Client, error) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	client, ok := cp.clients[name]
	if !ok {
		return nil, fmt.Errorf("client '%s' not found in pool", name)
	}

	return client, nil
}

// GetDefaultClient returns the default client ("default" or first available)
func (cp *ClientPool) GetDefaultClient() (*gl.Client, string, error) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	// Try "default" first
	if client, ok := cp.clients["default"]; ok {
		return client, "default", nil
	}

	// Fall back to first available client
	for name, client := range cp.clients {
		return client, name, nil
	}

	return nil, "", fmt.Errorf("no clients available in pool")
}

// ListClients returns all client names in the pool
func (cp *ClientPool) ListClients() []string {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	names := make([]string, 0, len(cp.clients))
	for name := range cp.clients {
		names = append(names, name)
	}
	return names
}

// RemoveClient removes a client from the pool
func (cp *ClientPool) RemoveClient(name string) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if _, ok := cp.clients[name]; !ok {
		return fmt.Errorf("client '%s' not found in pool", name)
	}

	delete(cp.clients, name)
	cp.logger.Infof("Removed client '%s' from pool", name)
	return nil
}

// InitializeFromEnv initializes clients from environment variables and token store
// This is called during server startup to set up the initial client(s)
func (cp *ClientPool) InitializeFromEnv(ctx context.Context, token string, host string) error {
	// Create client options
	clientOpts := []gl.ClientOptionFunc{}
	if host != "" && host != "https://gitlab.com" {
		clientOpts = append(clientOpts, gl.WithBaseURL(host))
	}

	// Create GitLab client
	glClient, err := gl.NewClient(token, clientOpts...)
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}

	// Determine server name from env or use "default"
	serverName := "default"
	if host != "" {
		// Use hostname as server name if custom host is provided
		// e.g., "gitlab.example.com" -> "gitlab.example.com"
		serverName = host
	}

	// Add to pool
	if err := cp.AddClient(serverName, glClient); err != nil {
		return err
	}

	cp.logger.Infof("Initialized client '%s' from environment", serverName)
	return nil
}

// AddServerFromConfig adds a client from a ServerConfig pointer
func (cp *ClientPool) AddServerFromConfig(ctx context.Context, server *config.ServerConfig) error {
	if server == nil {
		return fmt.Errorf("server config cannot be nil")
	}
	if server.Name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	// Create client options
	clientOpts := []gl.ClientOptionFunc{}
	if server.Host != "" && server.Host != "https://gitlab.com" {
		clientOpts = append(clientOpts, gl.WithBaseURL(server.Host))
	}

	// Create GitLab client
	glClient, err := gl.NewClient(server.Token, clientOpts...)
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}

	// Add to pool
	if err := cp.AddClient(server.Name, glClient); err != nil {
		return err
	}

	// Store token metadata in token store (minimal info, will be validated later)
	metadata := &TokenMetadata{
		Token:         server.Token,
		GitLabHost:    server.Host,
		CreatedAt:     time.Now(),
		LastValidated: time.Now(),
	}
	if server.UserID > 0 {
		metadata.UserID = server.UserID
		metadata.Username = server.Username
	}
	if err := cp.store.AddToken(server.Name, metadata); err != nil {
		cp.logger.Warnf("Failed to store token metadata for '%s': %v", server.Name, err)
	}

	return nil
}

// ValidateAllClients validates all clients in the pool
func (cp *ClientPool) ValidateAllClients(ctx context.Context) []TokenValidationResult {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	// Create getClientFunc for CheckAllTokens
	getClientFunc := func(name string) (*gl.Client, error) {
		client, ok := cp.clients[name]
		if !ok {
			return nil, fmt.Errorf("client '%s' not found", name)
		}
		return client, nil
	}

	return cp.store.CheckAllTokens(ctx, getClientFunc)
}

// InitializeFromConfig initializes clients from a global config
// This is called during server startup to set up clients from config file
func (cp *ClientPool) InitializeFromConfig(ctx context.Context, cfg *config.Config) error {
	if len(cfg.Servers) == 0 {
		return fmt.Errorf("no servers configured in config")
	}

	// Create a client for each server in the config
	for name, server := range cfg.Servers {
		if err := cp.initializeServer(ctx, name, server); err != nil {
			cp.logger.Warnf("Failed to initialize client '%s': %v", name, err)
			// Continue with other servers instead of failing completely
			continue
		}
	}

	cp.logger.Infof("Initialized %d clients from config", len(cfg.Servers))
	return nil
}

// initializeServer initializes a single server client
func (cp *ClientPool) initializeServer(ctx context.Context, name string, server *config.ServerConfig) error {
	// Create client options
	clientOpts := []gl.ClientOptionFunc{}
	if server.Host != "" && server.Host != "https://gitlab.com" {
		clientOpts = append(clientOpts, gl.WithBaseURL(server.Host))
	}

	// Create GitLab client
	glClient, err := gl.NewClient(server.Token, clientOpts...)
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}

	// Add to pool
	if err := cp.AddClient(name, glClient); err != nil {
		return err
	}

	// Store token metadata in token store (minimal info, will be validated later)
	metadata := &TokenMetadata{
		Token:         server.Token,
		GitLabHost:    server.Host,
		CreatedAt:     time.Now(),
		LastValidated: time.Now(),
	}
	if server.UserID > 0 {
		metadata.UserID = server.UserID
		metadata.Username = server.Username
	}
	if err := cp.store.AddToken(name, metadata); err != nil {
		cp.logger.Warnf("Failed to store token metadata for '%s': %v", name, err)
	}

	return nil
}
