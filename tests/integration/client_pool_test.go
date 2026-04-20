package integration

import (
	"context"
	"testing"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	"github.com/InkyQuill/gitlab-mcp-server/pkg/gitlab"
	gl "gitlab.com/gitlab-org/api/client-go"
	log "github.com/sirupsen/logrus"
)

func TestClientPoolFlow(t *testing.T) {
	t.Run("add and retrieve clients", func(t *testing.T) {
		store := gitlab.NewTokenStore()
		pool := gitlab.NewClientPool(store, log.New())

		// Create test clients
		client1, err := gl.NewClient("token1")
		if err != nil {
			t.Fatalf("failed to create client1: %v", err)
		}

		client2, err := gl.NewClient("token2")
		if err != nil {
			t.Fatalf("failed to create client2: %v", err)
		}

		// Add clients
		err = pool.AddClient("server1", client1)
		if err != nil {
			t.Fatalf("AddClient(server1) error = %v", err)
		}

		err = pool.AddClient("server2", client2)
		if err != nil {
			t.Fatalf("AddClient(server2) error = %v", err)
		}

		// Retrieve clients
		retrieved1, err := pool.GetClient("server1")
		if err != nil {
			t.Fatalf("GetClient(server1) error = %v", err)
		}

		if retrieved1 == nil {
			t.Error("retrieved client1 is nil")
		}

		retrieved2, err := pool.GetClient("server2")
		if err != nil {
			t.Fatalf("GetClient(server2) error = %v", err)
		}

		if retrieved2 == nil {
			t.Error("retrieved client2 is nil")
		}

		// Verify they are different instances
		if retrieved1 == retrieved2 {
			t.Error("clients should be different instances")
		}

		// List clients
		clients := pool.ListClients()
		if len(clients) != 2 {
			t.Errorf("expected 2 clients, got %d", len(clients))
		}

		// Verify client names
		nameMap := make(map[string]bool)
		for _, name := range clients {
			nameMap[name] = true
		}

		if !nameMap["server1"] || !nameMap["server2"] {
			t.Error("expected server1 and server2 in client list")
		}
	})

	t.Run("get default client", func(t *testing.T) {
		store := gitlab.NewTokenStore()
		pool := gitlab.NewClientPool(store, log.New())

		// Initially empty pool
		_, name, err := pool.GetDefaultClient()
		if err == nil {
			t.Error("expected error when pool is empty")
		}

		// Add a client named "default"
		client, err := gl.NewClient("token")
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		err = pool.AddClient("default", client)
		if err != nil {
			t.Fatalf("AddClient() error = %v", err)
		}

		// Get default should return the "default" client
		defaultClient, name, err := pool.GetDefaultClient()
		if err != nil {
			t.Fatalf("GetDefaultClient() error = %v", err)
		}

		if defaultClient == nil {
			t.Error("default client is nil")
		}

		if name != "default" {
			t.Errorf("expected name 'default', got '%s'", name)
		}

		// Add another client
		client2, _ := gl.NewClient("token2")
		pool.AddClient("other", client2)

		// Default should still be "default"
		defaultClient, name, _ = pool.GetDefaultClient()
		if name != "default" {
			t.Errorf("expected name 'default', got '%s'", name)
		}
	})

	t.Run("default falls back to first available", func(t *testing.T) {
		store := gitlab.NewTokenStore()
		pool := gitlab.NewClientPool(store, log.New())

		// Add clients without explicit "default" name
		client1, _ := gl.NewClient("token1")
		client2, _ := gl.NewClient("token2")

		pool.AddClient("server1", client1)
		pool.AddClient("server2", client2)

		// GetDefaultClient should return first available
		defaultClient, name, err := pool.GetDefaultClient()
		if err != nil {
			t.Fatalf("GetDefaultClient() error = %v", err)
		}

		if defaultClient == nil {
			t.Error("default client is nil")
		}

		if name != "server1" && name != "server2" {
			t.Errorf("expected name to be 'server1' or 'server2', got '%s'", name)
		}
	})

	t.Run("remove client", func(t *testing.T) {
		store := gitlab.NewTokenStore()
		pool := gitlab.NewClientPool(store, log.New())

		client1, _ := gl.NewClient("token1")
		client2, _ := gl.NewClient("token2")

		pool.AddClient("server1", client1)
		pool.AddClient("server2", client2)

		// Remove server1
		err := pool.RemoveClient("server1")
		if err != nil {
			t.Fatalf("RemoveClient() error = %v", err)
		}

		// Verify it's gone
		_, err = pool.GetClient("server1")
		if err == nil {
			t.Error("expected error when getting removed client")
		}

		// Verify server2 still exists
		_, err = pool.GetClient("server2")
		if err != nil {
			t.Errorf("server2 should still exist: %v", err)
		}

		// List should only have server2
		clients := pool.ListClients()
		if len(clients) != 1 {
			t.Errorf("expected 1 client, got %d", len(clients))
		}
	})

	t.Run("add client validation", func(t *testing.T) {
		store := gitlab.NewTokenStore()
		pool := gitlab.NewClientPool(store, log.New())

		client, _ := gl.NewClient("token")

		// Empty name should fail
		err := pool.AddClient("", client)
		if err == nil {
			t.Error("expected error for empty client name")
		}

		// Nil client should fail
		err = pool.AddClient("test", nil)
		if err == nil {
			t.Error("expected error for nil client")
		}
	})

	t.Run("get non-existent client", func(t *testing.T) {
		store := gitlab.NewTokenStore()
		pool := gitlab.NewClientPool(store, log.New())

		_, err := pool.GetClient("nonexistent")
		if err == nil {
			t.Error("expected error when getting non-existent client")
		}
	})

	t.Run("remove non-existent client", func(t *testing.T) {
		store := gitlab.NewTokenStore()
		pool := gitlab.NewClientPool(store, log.New())

		err := pool.RemoveClient("nonexistent")
		if err == nil {
			t.Error("expected error when removing non-existent client")
		}
	})

	t.Run("initialize from config", func(t *testing.T) {
		store := gitlab.NewTokenStore()
		pool := gitlab.NewClientPool(store, log.New())

		cfg := &config.Config{
			Version: "1.0",
			Servers: map[string]*config.ServerConfig{
				"gitlab": {
					Name:  "gitlab",
					Host:  "https://gitlab.com",
					Token: "test-token",
				},
				"custom": {
					Name:  "custom",
					Host:  "https://gitlab.example.com",
					Token: "custom-token",
				},
			},
		}

		ctx := context.Background()
		err := pool.InitializeFromConfig(ctx, cfg)
		if err != nil {
			t.Fatalf("InitializeFromConfig() error = %v", err)
		}

		// Verify clients were created
		clients := pool.ListClients()
		if len(clients) != 2 {
			t.Errorf("expected 2 clients, got %d", len(clients))
		}

		// Verify we can retrieve them
		_, err = pool.GetClient("gitlab")
		if err != nil {
			t.Errorf("gitlab client not found: %v", err)
		}

		_, err = pool.GetClient("custom")
		if err != nil {
			t.Errorf("custom client not found: %v", err)
		}
	})

	t.Run("token store integration", func(t *testing.T) {
		store := gitlab.NewTokenStore()
		pool := gitlab.NewClientPool(store, log.New())

		// Add client with server config
		serverCfg := &config.ServerConfig{
			Name:  "test-server",
			Host:  "https://gitlab.example.com",
			Token: "test-token",
		}

		ctx := context.Background()
		resolve := func(_ context.Context, _ string) (string, error) {
			return "test-token", nil
		}
		err := pool.AddServerFromConfig(ctx, serverCfg, resolve)
		if err != nil {
			t.Fatalf("AddServerFromConfig() error = %v", err)
		}

		// Verify client is in pool
		client, err := pool.GetClient("test-server")
		if err != nil {
			t.Fatalf("client not found in pool: %v", err)
		}

		if client == nil {
			t.Fatal("client is nil")
		}

		// Verify token metadata was stored
		metadata, err := store.GetToken("test-server")
		if err != nil {
			t.Fatalf("token metadata not found in store: %v", err)
		}

		if metadata.Name != "test-server" {
			t.Errorf("metadata name = %s, want 'test-server'", metadata.Name)
		}

		if metadata.GitLabHost != "https://gitlab.example.com" {
			t.Errorf("metadata host = %s, want 'https://gitlab.example.com'", metadata.GitLabHost)
		}
	})
}
