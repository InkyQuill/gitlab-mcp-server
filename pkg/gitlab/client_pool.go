package gitlab

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	log "github.com/sirupsen/logrus"
	gl "gitlab.com/gitlab-org/api/client-go"
)

// ClientInfo describes a GitLab client registered in the pool.
type ClientInfo struct {
	Name     string
	Host     string
	APIHost  string
	ReadOnly bool
	UserID   int64
	Username string
}

// ClientPool manages multiple GitLab clients for different servers
type ClientPool struct {
	clients map[string]*gl.Client // key: server name
	info    map[string]ClientInfo
	store   *TokenStore
	logger  *log.Logger
	mu      sync.RWMutex
}

// NewClientPool creates a new client pool
func NewClientPool(store *TokenStore, logger *log.Logger) *ClientPool {
	return &ClientPool{
		clients: make(map[string]*gl.Client),
		info:    make(map[string]ClientInfo),
		store:   store,
		logger:  logger,
	}
}

// AddClient adds a new client to the pool
func (cp *ClientPool) AddClient(name string, client *gl.Client) error {
	return cp.AddClientWithInfo(ClientInfo{Name: name}, client)
}

// AddClientWithInfo adds a new client and its metadata to the pool.
func (cp *ClientPool) AddClientWithInfo(info ClientInfo, client *gl.Client) error {
	if info.Name == "" {
		return fmt.Errorf("client name cannot be empty")
	}
	if client == nil {
		return fmt.Errorf("client cannot be nil")
	}
	if info.APIHost == "" {
		info.APIHost = info.Host
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.clients[info.Name] = client
	cp.info[info.Name] = info
	cp.logger.Infof("Added client '%s' to pool", info.Name)
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

// GetClientInfo retrieves metadata for a client by name.
func (cp *ClientPool) GetClientInfo(name string) (ClientInfo, error) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	info, ok := cp.info[name]
	if !ok {
		return ClientInfo{}, fmt.Errorf("client '%s' not found in pool", name)
	}
	return info, nil
}

// ListClientInfo returns metadata for all clients in the pool.
func (cp *ClientPool) ListClientInfo() []ClientInfo {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	infos := make([]ClientInfo, 0, len(cp.info))
	for _, info := range cp.info {
		infos = append(infos, info)
	}
	return infos
}

// FindClientByHost finds a client whose Host or APIHost matches host.
// Duplicate host matches return the lexicographically first server name.
func (cp *ClientPool) FindClientByHost(host string) (string, ClientInfo, bool) {
	normalized := normalizePoolHost(host)
	if normalized == "" {
		return "", ClientInfo{}, false
	}

	cp.mu.RLock()
	defer cp.mu.RUnlock()

	matches := make([]string, 0)
	for name, info := range cp.info {
		if normalizePoolHost(info.Host) == normalized || normalizePoolHost(info.APIHost) == normalized {
			matches = append(matches, name)
		}
	}
	if len(matches) == 0 {
		return "", ClientInfo{}, false
	}
	sort.Strings(matches)
	name := matches[0]
	return name, cp.info[name], true
}

func normalizePoolHost(host string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "/"))
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
	delete(cp.info, name)
	cp.logger.Infof("Removed client '%s' from pool", name)
	return nil
}

// InitializeFromEnv initializes clients from environment variables and token store
// This is called during server startup to set up the initial client(s)
func (cp *ClientPool) InitializeFromEnv(ctx context.Context, token string, host string) error {
	actualHost := host
	if actualHost == "" {
		actualHost = "https://gitlab.com"
	}

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
	if err := cp.AddClientWithInfo(ClientInfo{
		Name:    serverName,
		Host:    actualHost,
		APIHost: actualHost,
	}, glClient); err != nil {
		return err
	}

	cp.logger.Infof("Initialized client '%s' from environment", serverName)
	return nil
}

// TokenResolver resolves a server's token at runtime. Returns the plaintext
// token for a given server name.
type TokenResolver func(ctx context.Context, serverName string) (string, error)

// AddServerFromConfig adds a client using a TokenResolver for secret fetching.
// The resolver may hit a keyring / external tool / encrypted file.
func (cp *ClientPool) AddServerFromConfig(ctx context.Context, server *config.ServerConfig, resolve TokenResolver) error {
	if server == nil {
		return fmt.Errorf("server config cannot be nil")
	}
	if server.Name == "" {
		return fmt.Errorf("server name cannot be empty")
	}
	token, err := resolve(ctx, server.Name)
	if err != nil {
		return fmt.Errorf("resolve token for %q: %w", server.Name, err)
	}
	clientOpts := []gl.ClientOptionFunc{}
	if server.Host != "" && server.Host != "https://gitlab.com" {
		clientOpts = append(clientOpts, gl.WithBaseURL(server.Host))
	}
	glClient, err := gl.NewClient(token, clientOpts...)
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}
	if err := cp.AddClientWithInfo(ClientInfo{
		Name:     server.Name,
		Host:     server.Host,
		APIHost:  server.Host,
		ReadOnly: server.ReadOnly,
		UserID:   server.UserID,
		Username: server.Username,
	}, glClient); err != nil {
		return err
	}
	metadata := &TokenMetadata{
		Token:         token,
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
	if err := cp.AddClientWithInfo(ClientInfo{
		Name:     name,
		Host:     server.Host,
		APIHost:  server.Host,
		ReadOnly: server.ReadOnly,
		UserID:   server.UserID,
		Username: server.Username,
	}, glClient); err != nil {
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
