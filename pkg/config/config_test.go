package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Test creating a new manager with non-existent file
	m, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if m == nil {
		t.Fatal("NewManager() returned nil manager")
	}

	if m.FilePath != configPath {
		t.Errorf("FilePath = %s, want %s", m.FilePath, configPath)
	}

	if m.config == nil {
		t.Fatal("config is nil")
	}

	if m.config.Version != "1.0" {
		t.Errorf("Version = %s, want 1.0", m.config.Version)
	}

	if m.config.Servers == nil {
		t.Fatal("Servers map is nil")
	}
}

func TestManagerAddServer(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	m, _ := NewManager(configPath)

	// Test adding a server
	cfg := &ServerConfig{
		Name:     "test",
		Host:     "https://gitlab.example.com",
		Token:    "test-token",
		ReadOnly: true,
	}

	err := m.AddServer(cfg)
	if err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	// Verify server was added
	retrieved, err := m.GetServer("test")
	if err != nil {
		t.Fatalf("GetServer() error = %v", err)
	}

	if retrieved.Name != "test" {
		t.Errorf("Name = %s, want test", retrieved.Name)
	}

	if retrieved.Host != "https://gitlab.example.com" {
		t.Errorf("Host = %s, want https://gitlab.example.com", retrieved.Host)
	}

	if retrieved.Token != "test-token" {
		t.Errorf("Token = %s, want test-token", retrieved.Token)
	}

	if !cfg.IsDefault {
		t.Error("First server should be marked as default")
	}
}

func TestManagerAddDuplicateServer(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	m, _ := NewManager(configPath)

	cfg := &ServerConfig{
		Name:  "test",
		Host:  "https://gitlab.example.com",
		Token: "test-token",
	}

	// Add server twice
	m.AddServer(cfg)
	err := m.AddServer(cfg)

	if err == nil {
		t.Error("Expected error when adding duplicate server")
	}
}

func TestManagerAddServerEmptyName(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	m, _ := NewManager(configPath)

	cfg := &ServerConfig{
		Name:  "",
		Host:  "https://gitlab.example.com",
		Token: "test-token",
	}

	err := m.AddServer(cfg)
	if err == nil {
		t.Error("Expected error when adding server with empty name")
	}
}

func TestManagerRemoveServer(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	m, _ := NewManager(configPath)

	// Add a server
	cfg := &ServerConfig{
		Name:  "test",
		Host:  "https://gitlab.example.com",
		Token: "test-token",
	}
	m.AddServer(cfg)

	// Remove it
	err := m.RemoveServer("test")
	if err != nil {
		t.Fatalf("RemoveServer() error = %v", err)
	}

	// Verify it's gone
	if m.HasServer("test") {
		t.Error("Server still exists after removal")
	}
}

func TestManagerRemoveNonExistentServer(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	m, _ := NewManager(configPath)

	err := m.RemoveServer("nonexistent")
	if err == nil {
		t.Error("Expected error when removing non-existent server")
	}
}

func TestManagerRemoveDefaultServer(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	m, _ := NewManager(configPath)

	// Add two servers
	cfg1 := &ServerConfig{
		Name:  "default",
		Host:  "https://gitlab.com",
		Token: "token1",
	}
	cfg2 := &ServerConfig{
		Name:  "other",
		Host:  "https://gitlab.example.com",
		Token: "token2",
	}
	m.AddServer(cfg1)
	m.AddServer(cfg2)

	// Try to remove default server
	err := m.RemoveServer("default")
	if err == nil {
		t.Error("Expected error when removing default server with other servers present")
	}
}

func TestManagerSetDefaultServer(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	m, _ := NewManager(configPath)

	// Add two servers
	cfg1 := &ServerConfig{
		Name:  "server1",
		Host:  "https://gitlab.com",
		Token: "token1",
	}
	cfg2 := &ServerConfig{
		Name:  "server2",
		Host:  "https://gitlab.example.com",
		Token: "token2",
	}
	m.AddServer(cfg1)
	m.AddServer(cfg2)

	// Set server2 as default
	err := m.SetDefaultServer("server2")
	if err != nil {
		t.Fatalf("SetDefaultServer() error = %v", err)
	}

	// Verify
	s1, _ := m.GetServer("server1")
	s2, _ := m.GetServer("server2")

	if s1.IsDefault {
		t.Error("server1 should not be default")
	}

	if !s2.IsDefault {
		t.Error("server2 should be default")
	}
}

func TestManagerSetDefaultNonExistentServer(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	m, _ := NewManager(configPath)

	err := m.SetDefaultServer("nonexistent")
	if err == nil {
		t.Error("Expected error when setting non-existent server as default")
	}
}

func TestManagerSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create manager and add server
	m1, _ := NewManager(configPath)
	cfg := &ServerConfig{
		Name:     "test",
		Host:     "https://gitlab.example.com",
		Token:    "test-token",
		ReadOnly: true,
	}
	m1.AddServer(cfg)

	// Save
	err := m1.Save()
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Create new manager and load
	m2, _ := NewManager(configPath)
	err = m2.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify loaded data
	retrieved, err := m2.GetServer("test")
	if err != nil {
		t.Fatalf("GetServer() error = %v", err)
	}

	if retrieved.Name != "test" {
		t.Errorf("Name = %s, want test", retrieved.Name)
	}

	if retrieved.Host != "https://gitlab.example.com" {
		t.Errorf("Host = %s, want https://gitlab.example.com", retrieved.Host)
	}

	if retrieved.Token != "test-token" {
		t.Errorf("Token = %s, want test-token", retrieved.Token)
	}

	if !retrieved.ReadOnly {
		t.Error("ReadOnly = false, want true")
	}
}

func TestManagerGetDefaultServer(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	m, _ := NewManager(configPath)

	// No servers yet
	_, err := m.GetDefaultServer()
	if err == nil {
		t.Error("Expected error when no default server")
	}

	// Add server
	cfg := &ServerConfig{
		Name:  "test",
		Host:  "https://gitlab.com",
		Token: "token",
	}
	m.AddServer(cfg)

	// Get default
	def, err := m.GetDefaultServer()
	if err != nil {
		t.Fatalf("GetDefaultServer() error = %v", err)
	}

	if def.Name != "test" {
		t.Errorf("Default server name = %s, want test", def.Name)
	}
}

func TestManagerListServers(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	m, _ := NewManager(configPath)

	// Initially empty
	servers := m.ListServers()
	if len(servers) != 0 {
		t.Errorf("Server count = %d, want 0", len(servers))
	}

	// Add servers
	m.AddServer(&ServerConfig{Name: "s1", Host: "https://gitlab.com", Token: "t1"})
	m.AddServer(&ServerConfig{Name: "s2", Host: "https://example.com", Token: "t2"})

	servers = m.ListServers()
	if len(servers) != 2 {
		t.Errorf("Server count = %d, want 2", len(servers))
	}
}

func TestManagerUpdateServer(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	m, _ := NewManager(configPath)

	// Add server
	cfg := &ServerConfig{
		Name:     "test",
		Host:     "https://gitlab.com",
		Token:    "token1",
		ReadOnly: false,
	}
	m.AddServer(cfg)

	// Update server
	updated := &ServerConfig{
		Name:     "test", // Name preserved even if different
		Host:     "https://gitlab.example.com",
		Token:    "token2",
		ReadOnly: true,
	}
	err := m.UpdateServer("test", updated)
	if err != nil {
		t.Fatalf("UpdateServer() error = %v", err)
	}

	// Verify
	retrieved, _ := m.GetServer("test")
	if retrieved.Host != "https://gitlab.example.com" {
		t.Errorf("Host = %s, want https://gitlab.example.com", retrieved.Host)
	}

	if retrieved.Token != "token2" {
		t.Errorf("Token = %s, want token2", retrieved.Token)
	}

	if !retrieved.ReadOnly {
		t.Error("ReadOnly = false, want true")
	}
}

func TestManagerUpdateNonExistentServer(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	m, _ := NewManager(configPath)

	cfg := &ServerConfig{
		Name:  "nonexistent",
		Host:  "https://gitlab.com",
		Token: "token",
	}
	err := m.UpdateServer("nonexistent", cfg)
	if err == nil {
		t.Error("Expected error when updating non-existent server")
	}
}

func TestManagerServerCount(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	m, _ := NewManager(configPath)

	if m.ServerCount() != 0 {
		t.Errorf("Server count = %d, want 0", m.ServerCount())
	}

	m.AddServer(&ServerConfig{Name: "s1", Host: "https://gitlab.com", Token: "t1"})
	m.AddServer(&ServerConfig{Name: "s2", Host: "https://example.com", Token: "t2"})

	if m.ServerCount() != 2 {
		t.Errorf("Server count = %d, want 2", m.ServerCount())
	}
}

func TestManagerValidateServer(t *testing.T) {
	// This test requires a mock GitLab server or uses a real token
	// For now, we'll test the error path when server doesn't exist
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	m, _ := NewManager(configPath)

	ctx := context.Background()
	_, err := m.ValidateServer(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error when validating non-existent server")
	}
}

func TestConfigFileCreation(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create manager and save
	m, _ := NewManager(configPath)
	m.AddServer(&ServerConfig{
		Name:  "test",
		Host:  "https://gitlab.com",
		Token: "token",
	})
	m.Save()

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Verify file permissions
	info, _ := os.Stat(configPath)
	if info.Mode().Perm() != 0600 {
		t.Errorf("Config file permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestFirstServerIsDefault(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	m, _ := NewManager(configPath)

	// Add first server without explicitly setting IsDefault
	cfg1 := &ServerConfig{
		Name:  "server1",
		Host:  "https://gitlab.com",
		Token: "token1",
	}
	m.AddServer(cfg1)

	if !cfg1.IsDefault {
		t.Error("First server should automatically be marked as default")
	}

	// Add second server
	cfg2 := &ServerConfig{
		Name:  "server2",
		Host:  "https://example.com",
		Token: "token2",
	}
	m.AddServer(cfg2)

	// First server should still be default
	s1, _ := m.GetServer("server1")
	if !s1.IsDefault {
		t.Error("First server should still be default")
	}

	s2, _ := m.GetServer("server2")
	if s2.IsDefault {
		t.Error("Second server should not be default")
	}
}

func TestAddServerWithDefaultFlag(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	m, _ := NewManager(configPath)

	// Add first server
	m.AddServer(&ServerConfig{
		Name:  "server1",
		Host:  "https://gitlab.com",
		Token: "token1",
	})

	// Add second server with IsDefault=true
	cfg2 := &ServerConfig{
		Name:      "server2",
		Host:      "https://example.com",
		Token:     "token2",
		IsDefault: true,
	}
	m.AddServer(cfg2)

	// Verify only server2 is default
	s1, _ := m.GetServer("server1")
	s2, _ := m.GetServer("server2")

	if s1.IsDefault {
		t.Error("server1 should not be default after server2 added with IsDefault=true")
	}

	if !s2.IsDefault {
		t.Error("server2 should be default")
	}
}

func TestUpdateServerWithDefaultFlag(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	m, _ := NewManager(configPath)

	// Add two servers
	m.AddServer(&ServerConfig{Name: "s1", Host: "https://gitlab.com", Token: "t1"})
	m.AddServer(&ServerConfig{Name: "s2", Host: "https://example.com", Token: "t2"})

	// Update s2 to be default
	m.UpdateServer("s2", &ServerConfig{
		Name:      "s2",
		Host:      "https://example.com",
		Token:     "t2",
		IsDefault: true,
	})

	s1, _ := m.GetServer("s1")
	s2, _ := m.GetServer("s2")

	if s1.IsDefault {
		t.Error("s1 should not be default")
	}

	if !s2.IsDefault {
		t.Error("s2 should be default")
	}
}

func TestManager_WithRegistry_ResolvesTokenRef(t *testing.T) {
	reg := NewBackendRegistry()
	fake := NewFakeSecretBackend("keyring")
	fake.SetEntry("work", "glpat-resolved")
	require.NoError(t, reg.Register(fake))

	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.json")
	m, err := NewManagerWithRegistry(path, reg)
	require.NoError(t, err)

	require.NoError(t, m.AddServer(&ServerConfig{
		Name:     "work",
		Host:     "https://gitlab.example.com",
		TokenRef: "keyring://work",
	}))

	got, err := m.ResolveServerToken(context.Background(), "work")
	require.NoError(t, err)
	assert.Equal(t, "glpat-resolved", got)
}

func TestManager_WithRegistry_PrefersTokenRefOverToken(t *testing.T) {
	reg := NewBackendRegistry()
	fake := NewFakeSecretBackend("keyring")
	fake.SetEntry("work", "FROM-REF")
	require.NoError(t, reg.Register(fake))

	tmp := t.TempDir()
	m, err := NewManagerWithRegistry(filepath.Join(tmp, "config.json"), reg)
	require.NoError(t, err)

	require.NoError(t, m.AddServer(&ServerConfig{
		Name:     "work",
		Host:     "https://gitlab.example.com",
		Token:    "FROM-TOKEN",
		TokenRef: "keyring://work",
	}))

	got, err := m.ResolveServerToken(context.Background(), "work")
	require.NoError(t, err)
	assert.Equal(t, "FROM-REF", got)
}

func TestManager_WithRegistry_FallsBackToLegacyToken(t *testing.T) {
	reg := NewBackendRegistry()
	tmp := t.TempDir()
	m, err := NewManagerWithRegistry(filepath.Join(tmp, "config.json"), reg)
	require.NoError(t, err)

	require.NoError(t, m.AddServer(&ServerConfig{
		Name:  "work",
		Host:  "https://gitlab.example.com",
		Token: "FROM-TOKEN",
	}))

	got, err := m.ResolveServerToken(context.Background(), "work")
	require.NoError(t, err)
	assert.Equal(t, "FROM-TOKEN", got)
}

func TestConfig_BackendsSection_Roundtrips(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.json")
	m, err := NewManager(path)
	require.NoError(t, err)
	m.Config().Backends = &BackendsConfig{
		External: map[string]string{"op": "op read %s"},
	}
	require.NoError(t, m.Save())

	m2, err := NewManager(path)
	require.NoError(t, err)
	require.NotNil(t, m2.Config().Backends)
	assert.Equal(t, "op read %s", m2.Config().Backends.External["op"])
}
