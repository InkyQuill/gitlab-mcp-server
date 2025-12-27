package gitlab

import (
	"context"
	"fmt"
	"sync"

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
