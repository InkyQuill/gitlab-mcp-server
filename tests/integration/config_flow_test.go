package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/config"
)

func TestConfigFlow(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	t.Run("create and load config", func(t *testing.T) {
		// Create manager with test directory
		cfgPath := filepath.Join(tmpDir, "config.json")

		m, err := config.NewManager(cfgPath)
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}

		// Add server
		server := &config.ServerConfig{
			Name:  "test",
			Host:  "https://gitlab.com",
			Token: "glpat-test",
		}

		err = m.AddServer(server)
		if err != nil {
			t.Fatalf("AddServer() error = %v", err)
		}

		// Verify server was added and marked as default (retrieve to check internal state)
		retrieved, err := m.GetServer("test")
		if err != nil {
			t.Fatalf("GetServer() error = %v", err)
		}

		if !retrieved.IsDefault {
			t.Error("First server should be marked as default")
		}

		// Save
		err = m.Save()
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
			t.Fatal("config file not created")
		}

		// Load in new manager
		m2, err := config.NewManager(cfgPath)
		if err != nil {
			t.Fatalf("NewManager() second time error = %v", err)
		}

		err = m2.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		// Verify loaded data
		retrieved, err = m2.GetServer("test")
		if err != nil {
			t.Fatalf("GetServer() error = %v", err)
		}

		if retrieved.Name != "test" {
			t.Errorf("expected Name 'test', got '%s'", retrieved.Name)
		}

		if retrieved.Host != "https://gitlab.com" {
			t.Errorf("expected Host 'https://gitlab.com', got '%s'", retrieved.Host)
		}

		if retrieved.Token != "glpat-test" {
			t.Errorf("expected Token 'glpat-test', got '%s'", retrieved.Token)
		}

		if !retrieved.IsDefault {
			t.Error("expected IsDefault to be true")
		}
	})

	t.Run("multiple servers with default management", func(t *testing.T) {
		cfgPath := filepath.Join(tmpDir, "multi-server.json")

		m, _ := config.NewManager(cfgPath)

		// Add first server
		s1 := &config.ServerConfig{
			Name:  "server1",
			Host:  "https://gitlab.com",
			Token: "token1",
		}
		m.AddServer(s1)

		// Verify first is default (need to retrieve it)
		s1Check, _ := m.GetServer("server1")
		if !s1Check.IsDefault {
			t.Error("First server should be default")
		}

		// Add second server without default flag
		s2 := &config.ServerConfig{
			Name:  "server2",
			Host:  "https://example.com",
			Token: "token2",
		}
		m.AddServer(s2)

		// First should still be default
		s1Check, _ = m.GetServer("server1")
		if !s1Check.IsDefault {
			t.Error("server1 should still be default")
		}

		s2Check, _ := m.GetServer("server2")
		if s2Check.IsDefault {
			t.Error("server2 should not be default")
		}

		// Add third server with default flag
		s3 := &config.ServerConfig{
			Name:      "server3",
			Host:      "https://custom.com",
			Token:     "token3",
			IsDefault: true,
		}
		m.AddServer(s3)

		// Save and reload
		m.Save()

		m2, _ := config.NewManager(cfgPath)
		m2.Load()

		// Verify only server3 is default after reload
		s1Check2, _ := m2.GetServer("server1")
		s2Check2, _ := m2.GetServer("server2")
		s3Check2, _ := m2.GetServer("server3")

		if s1Check2.IsDefault {
			t.Error("server1 should not be default after server3 added")
		}

		if s2Check2.IsDefault {
			t.Error("server2 should not be default after server3 added")
		}

		if !s3Check2.IsDefault {
			t.Error("server3 should be default")
		}
	})

	t.Run("server update and removal", func(t *testing.T) {
		cfgPath := filepath.Join(tmpDir, "update-test.json")

		m, _ := config.NewManager(cfgPath)

		// Add two servers
		s1 := &config.ServerConfig{
			Name:     "test",
			Host:     "https://gitlab.com",
			Token:    "token1",
			ReadOnly: false,
		}
		m.AddServer(s1)

		s2 := &config.ServerConfig{
			Name:  "other",
			Host:  "https://other.com",
			Token: "token2",
		}
		m.AddServer(s2)

		// Verify "test" is default (first added)
		testServer, _ := m.GetServer("test")
		if !testServer.IsDefault {
			t.Error("'test' should be default as first server")
		}

		// Try to remove default server (should fail)
		err := m.RemoveServer("test")
		if err == nil {
			t.Error("expected error when removing default server with other servers present")
		}

		// Set other as default
		err = m.SetDefaultServer("other")
		if err != nil {
			t.Fatalf("SetDefaultServer() error = %v", err)
		}

		// Now remove "test" should succeed
		err = m.RemoveServer("test")
		if err != nil {
			t.Fatalf("RemoveServer() after changing default error = %v", err)
		}

		// Verify removal
		if m.HasServer("test") {
			t.Error("'test' server should be removed")
		}

		// Verify 'other' is still there and is default
		otherServer, _ := m.GetServer("other")
		if !otherServer.IsDefault {
			t.Error("'other' should still be default")
		}
	})

	t.Run("list servers", func(t *testing.T) {
		cfgPath := filepath.Join(tmpDir, "list-test.json")

		m, _ := config.NewManager(cfgPath)

		// Initially empty
		servers := m.ListServers()
		if len(servers) != 0 {
			t.Errorf("expected 0 servers, got %d", len(servers))
		}

		// Add servers
		m.AddServer(&config.ServerConfig{Name: "s1", Host: "https://gitlab.com", Token: "t1"})
		m.AddServer(&config.ServerConfig{Name: "s2", Host: "https://example.com", Token: "t2"})
		m.AddServer(&config.ServerConfig{Name: "s3", Host: "https://custom.com", Token: "t3"})

		servers = m.ListServers()
		if len(servers) != 3 {
			t.Errorf("expected 3 servers, got %d", len(servers))
		}

		// Save and reload
		m.Save()

		m2, _ := config.NewManager(cfgPath)
		m2.Load()

		servers2 := m2.ListServers()
		if len(servers2) != 3 {
			t.Errorf("expected 3 servers after reload, got %d", len(servers2))
		}
	})
}

func TestConfigFilePathCreation(t *testing.T) {
	t.Run("config directory is created", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfgPath := filepath.Join(tmpDir, ".gitlab-mcp-server", "config.json")
		m, _ := config.NewManager(cfgPath)

		m.AddServer(&config.ServerConfig{
			Name:  "test",
			Host:  "https://gitlab.com",
			Token: "token",
		})

		err := m.Save()
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		// Verify directory was created
		configDir := filepath.Dir(cfgPath)
		info, err := os.Stat(configDir)
		if err != nil {
			t.Fatalf("config directory not created: %v", err)
		}

		if !info.IsDir() {
			t.Error("config path is not a directory")
		}
	})

	t.Run("config file permissions are secure", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.json")

		m, _ := config.NewManager(cfgPath)
		m.AddServer(&config.ServerConfig{
			Name:  "test",
			Host:  "https://gitlab.com",
			Token: "token",
		})

		m.Save()

		// Check file permissions
		info, err := os.Stat(cfgPath)
		if err != nil {
			t.Fatalf("failed to stat config file: %v", err)
		}

		// On Unix systems, 0600 means read/write for owner only
		perm := info.Mode().Perm()
		if perm != 0600 {
			t.Errorf("config file permissions = %o, want 0600", perm)
		}
	})
}
