package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	gl "gitlab.com/gitlab-org/api/client-go"
)

const (
	// ConfigFileName is the name of the global config file
	ConfigFileName = "gitlab-mcp-server-config.json"
	// ConfigDir is the directory name for the config
	ConfigDir = ".gitlab-mcp-server"
)

// ServerConfig holds configuration for a single GitLab server
type ServerConfig struct {
	Name      string `json:"name"`
	Host      string `json:"host"`
	Token     string `json:"token,omitempty"`    // legacy; present until v3.0 clean break
	TokenRef  string `json:"tokenRef,omitempty"` // new in v2.1; takes precedence when set
	ReadOnly  bool   `json:"readOnly,omitempty"`
	IsDefault bool   `json:"isDefault,omitempty"`
	// Metadata fields populated from API
	UserID        int    `json:"userId,omitempty"`
	Username      string `json:"username,omitempty"`
	LastValidated string `json:"lastValidated,omitempty"` // ISO 8601 format
}

// BackendsConfig holds configuration for pluggable secret backends.
type BackendsConfig struct {
	External map[string]string `json:"external,omitempty"`
}

// Config holds the global configuration
type Config struct {
	Version  string                   `json:"version"`
	Servers  map[string]*ServerConfig `json:"servers,omitempty"`
	Backends *BackendsConfig          `json:"backends,omitempty"`
}

// Manager manages the global configuration
type Manager struct {
	mu       sync.RWMutex
	config   *Config
	FilePath string
	crypto   *CryptoManager
	registry *BackendRegistry // nil-safe; optional
}

// NewManager creates a new config manager with encryption disabled by default.
// For encryption, use NewManagerWithEncryption.
func NewManager(configPath string) (*Manager, error) {
	return NewManagerWithEncryption(configPath, false)
}

// NewManagerWithEncryption creates a new config manager with optional encryption.
func NewManagerWithEncryption(configPath string, useEncryption bool) (*Manager, error) {
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir := filepath.Join(homeDir, ConfigDir)
		configPath = filepath.Join(configDir, ConfigFileName)
	}

	crypto, err := NewCryptoManager(useEncryption)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize crypto manager: %w", err)
	}

	m := &Manager{
		config: &Config{
			Version: "1.0",
			Servers: make(map[string]*ServerConfig),
		},
		FilePath: configPath,
		crypto:   crypto,
	}

	// Load existing config if it exists
	if err := m.Load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		// Config doesn't exist, will be created on first save
	}

	return m, nil
}

// Load loads the configuration from disk
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.FilePath)
	if err != nil {
		return err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.Servers == nil {
		cfg.Servers = make(map[string]*ServerConfig)
	}

	m.config = &cfg
	return nil
}

// Save saves the configuration to disk. When any server carries a TokenRef,
// the on-disk schema is bumped to "2.0" and a one-time backup of the prior
// file is written next to it as config.json.bak.
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	hasTokenRef := false
	for _, s := range m.config.Servers {
		if s.TokenRef != "" {
			hasTokenRef = true
			break
		}
	}

	if hasTokenRef && m.config.Version != "2.0" {
		// Backup the prior file (best-effort; missing prior file is fine).
		if prior, err := os.ReadFile(m.FilePath); err == nil {
			_ = os.WriteFile(m.FilePath+".bak", prior, 0600)
		}
		m.config.Version = "2.0"
	}
	if !hasTokenRef && m.config.Version == "" {
		m.config.Version = "1.0"
	}

	configDir := filepath.Dir(m.FilePath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(m.FilePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

// GetConfigPath returns the path to the config file
func (m *Manager) GetConfigPath() string {
	return m.FilePath
}

// HasServers returns true if the config has any servers configured
func (m *Manager) HasServers() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.config.Servers) > 0
}

// Config returns the internal Config struct (for compatibility with client_pool)
func (m *Manager) Config() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// ServerCount returns the number of configured servers
func (m *Manager) ServerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.config.Servers)
}

// GetDefaultServerName returns the default server name
func (m *Manager) GetDefaultServerName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, cfg := range m.config.Servers {
		if cfg.IsDefault {
			return name
		}
	}

	// Return first available server
	for name := range m.config.Servers {
		return name
	}
	return ""
}

// GetServer returns a server configuration by name with decrypted token.
func (m *Manager) GetServer(name string) (*ServerConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	server, ok := m.config.Servers[name]
	if !ok {
		return nil, fmt.Errorf("server '%s' not found", name)
	}

	// Decrypt token for use
	decryptedToken, err := m.crypto.Decrypt(server.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token: %w", err)
	}

	// Return a copy with decrypted token
	result := *server
	result.Token = decryptedToken
	return &result, nil
}

// AddServer adds a server configuration with token encryption.
func (m *Manager) AddServer(server *ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if server.Name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	if _, exists := m.config.Servers[server.Name]; exists {
		return fmt.Errorf("server '%s' already exists", server.Name)
	}

	// Encrypt token before storing
	encryptedToken, err := m.crypto.Encrypt(server.Token)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Create a copy with encrypted token for storage
	serverCopy := *server
	serverCopy.Token = encryptedToken

	// If this is the first server or explicitly marked as default, make it default
	if len(m.config.Servers) == 0 || server.IsDefault {
		// Remove default flag from all other servers
		for _, s := range m.config.Servers {
			s.IsDefault = false
		}
		serverCopy.IsDefault = true
		// Update input struct for backward compatibility
		server.IsDefault = true
	}

	m.config.Servers[server.Name] = &serverCopy
	return nil
}

// RemoveServer removes a server configuration
func (m *Manager) RemoveServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, exists := m.config.Servers[name]
	if !exists {
		return fmt.Errorf("server '%s' not found", name)
	}

	if cfg.IsDefault && len(m.config.Servers) > 1 {
		return fmt.Errorf("cannot remove default server '%s'. Set another server as default first", name)
	}

	delete(m.config.Servers, name)
	return nil
}

// UpdateServer updates an existing server configuration
func (m *Manager) UpdateServer(name string, cfg *ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.config.Servers[name]; !exists {
		return fmt.Errorf("server '%s' not found", name)
	}

	// Preserve the name
	cfg.Name = name

	// Handle default flag changes
	if cfg.IsDefault {
		for _, s := range m.config.Servers {
			s.IsDefault = false
		}
	}

	m.config.Servers[name] = cfg
	return nil
}

// ListServers returns all server configurations with decrypted tokens.
func (m *Manager) ListServers() []*ServerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	servers := make([]*ServerConfig, 0, len(m.config.Servers))
	for _, cfg := range m.config.Servers {
		// Decrypt token for each server
		decryptedToken, err := m.crypto.Decrypt(cfg.Token)
		if err != nil {
			// If decryption fails, keep the original token
			decryptedToken = cfg.Token
		}
		// Return a copy with decrypted token
		serverCopy := *cfg
		serverCopy.Token = decryptedToken
		servers = append(servers, &serverCopy)
	}

	return servers
}

// GetDefaultServer returns the default server configuration
func (m *Manager) GetDefaultServer() (*ServerConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, cfg := range m.config.Servers {
		if cfg.IsDefault {
			return cfg, nil
		}
	}

	return nil, fmt.Errorf("no default server configured")
}

// SetDefaultServer sets a server as the default
func (m *Manager) SetDefaultServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.config.Servers[name]; !exists {
		return fmt.Errorf("server '%s' not found", name)
	}

	// Remove default flag from all servers
	for _, s := range m.config.Servers {
		s.IsDefault = false
	}

	// Set default flag on specified server
	m.config.Servers[name].IsDefault = true
	return nil
}

// HasServer checks if a server with the given name exists
func (m *Manager) HasServer(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.config.Servers[name]
	return exists
}

// ValidateServer validates a server configuration by creating a GitLab client
// and calling the API. Returns user information if successful.
func (m *Manager) ValidateServer(ctx context.Context, name string) (*ServerConfig, error) {
	m.mu.Lock()
	cfg, exists := m.config.Servers[name]
	if !exists {
		m.mu.Unlock()
		return nil, fmt.Errorf("server '%s' not found", name)
	}

	// Decrypt token for validation
	decryptedToken, err := m.crypto.Decrypt(cfg.Token)
	if err != nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to decrypt token: %w", err)
	}
	host := cfg.Host
	userID := cfg.UserID
	username := cfg.Username
	m.mu.Unlock()

	// Create GitLab client
	clientOpts := []gl.ClientOptionFunc{}
	if host != "" && host != "https://gitlab.com" {
		clientOpts = append(clientOpts, gl.WithBaseURL(host))
	}

	client, err := gl.NewClient(decryptedToken, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	// Validate token by calling /user endpoint
	user, resp, err := client.Users.CurrentUser(gl.WithContext(ctx))
	if err != nil {
		if resp != nil && resp.StatusCode == 401 {
			return nil, fmt.Errorf("token is invalid or expired (401 Unauthorized)")
		}
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Update server config with user info
	m.mu.Lock()
	cfg.UserID = user.ID
	cfg.Username = user.Username
	m.mu.Unlock()

	// Save updated config
	if err := m.Save(); err != nil {
		return nil, fmt.Errorf("failed to save updated config: %w", err)
	}

	// Return copy with decrypted token
	return &ServerConfig{
		Name:          cfg.Name,
		Host:          cfg.Host,
		Token:         decryptedToken,
		ReadOnly:      cfg.ReadOnly,
		IsDefault:     cfg.IsDefault,
		UserID:        userID,
		Username:      username,
		LastValidated: cfg.LastValidated,
	}, nil
}

// CreateClient creates a GitLab client for the specified server.
// GetServer already returns a decrypted token, so we can use it directly.
func (m *Manager) CreateClient(name string) (*gl.Client, error) {
	cfg, err := m.GetServer(name)
	if err != nil {
		return nil, err
	}

	clientOpts := []gl.ClientOptionFunc{}
	if cfg.Host != "" && cfg.Host != "https://gitlab.com" {
		clientOpts = append(clientOpts, gl.WithBaseURL(cfg.Host))
	}

	return gl.NewClient(cfg.Token, clientOpts...)
}

// ValidateAllServers validates all server configurations
// Returns a map of server name to error (nil if validation succeeded)
func (m *Manager) ValidateAllServers(ctx context.Context) map[string]error {
	m.mu.RLock()
	serverNames := make([]string, 0, len(m.config.Servers))
	for name := range m.config.Servers {
		serverNames = append(serverNames, name)
	}
	m.mu.RUnlock()

	results := make(map[string]error)
	for _, name := range serverNames {
		if _, err := m.ValidateServer(ctx, name); err != nil {
			results[name] = err
		} else {
			results[name] = nil
		}
	}

	return results
}

// NewManagerWithRegistry creates a Manager wired to a BackendRegistry.
// If registry is nil, ResolveServerToken falls back to the legacy Token field.
func NewManagerWithRegistry(configPath string, registry *BackendRegistry) (*Manager, error) {
	m, err := NewManager(configPath)
	if err != nil {
		return nil, err
	}
	m.registry = registry
	return m, nil
}

// ResolveServerToken returns the plaintext token for the given server, preferring
// TokenRef (via the registry) over the legacy Token field.
func (m *Manager) ResolveServerToken(ctx context.Context, name string) (string, error) {
	m.mu.RLock()
	srv, ok := m.config.Servers[name]
	registry := m.registry
	m.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("server %q not found", name)
	}
	if srv.TokenRef != "" && registry != nil {
		return registry.Resolve(ctx, srv.TokenRef)
	}
	return m.crypto.Decrypt(srv.Token)
}
