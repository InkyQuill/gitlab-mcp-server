package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/gitlab"
	gl "gitlab.com/gitlab-org/api/client-go"
	log "github.com/sirupsen/logrus"
)

func TestClientResolverFlow(t *testing.T) {
	t.Run("resolve from .gmcprc with tokenName", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		os.Chdir(tmpDir)

		// Create .gmcprc with tokenName
		pc := &gitlab.ProjectConfig{
			ProjectID:  "owner/repo",
			GitLabHost: "https://gitlab.com",
			TokenName:   "server1",
		}
		_, err := gitlab.WriteProjectConfig(tmpDir, pc)
		if err != nil {
			t.Fatalf("WriteProjectConfig() error = %v", err)
		}

		// Create pool and resolver
		store := gitlab.NewTokenStore()
		pool := gitlab.NewClientPool(store, log.New())

		client1, _ := gl.NewClient("token1")
		err = pool.AddClient("server1", client1)
		if err != nil {
			t.Fatalf("AddClient() error = %v", err)
		}

		resolver := gitlab.NewClientResolver(pool, "default", log.New())

		// Resolve should use tokenName from .gmcprc
		ctx := context.Background()
		client, name, err := resolver.Resolve(ctx)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}

		if name != "server1" {
			t.Errorf("expected name 'server1', got '%s'", name)
		}

		if client == nil {
			t.Error("resolved client is nil")
		}
	})

	t.Run("resolve falls back to default when tokenName not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		os.Chdir(tmpDir)

		// Create .gmcprc with non-existent tokenName
		pc := &gitlab.ProjectConfig{
			ProjectID:  "owner/repo",
			GitLabHost: "https://gitlab.com",
			TokenName:   "nonexistent",
		}
		gitlab.WriteProjectConfig(tmpDir, pc)

		// Create pool and resolver
		store := gitlab.NewTokenStore()
		pool := gitlab.NewClientPool(store, log.New())

		client1, _ := gl.NewClient("token1")
		pool.AddClient("default", client1)

		resolver := gitlab.NewClientResolver(pool, "default", log.New())

		// Resolve should fall back to default
		ctx := context.Background()
		client, name, err := resolver.Resolve(ctx)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}

		if name != "default" {
			t.Errorf("expected name 'default', got '%s'", name)
		}

		if client == nil {
			t.Error("resolved client is nil")
		}
	})

	t.Run("resolve without .gmcprc uses default", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		os.Chdir(tmpDir)

		// Don't create .gmcprc

		// Create pool and resolver
		store := gitlab.NewTokenStore()
		pool := gitlab.NewClientPool(store, log.New())

		client1, _ := gl.NewClient("token1")
		pool.AddClient("default", client1)

		resolver := gitlab.NewClientResolver(pool, "default", log.New())

		// Resolve should use default
		ctx := context.Background()
		client, name, err := resolver.Resolve(ctx)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}

		if name != "default" {
			t.Errorf("expected name 'default', got '%s'", name)
		}

		if client == nil {
			t.Error("resolved client is nil")
		}
	})

	t.Run("resolve with host matching", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		os.Chdir(tmpDir)

		// Create .gmcprc with custom host (no tokenName)
		pc := &gitlab.ProjectConfig{
			ProjectID:  "owner/repo",
			GitLabHost: "https://gitlab.example.com",
		}
		gitlab.WriteProjectConfig(tmpDir, pc)

		// Create pool and resolver
		store := gitlab.NewTokenStore()
		pool := gitlab.NewClientPool(store, log.New())

		// Add client with matching hostname
		client1, _ := gl.NewClient("token1")
		pool.AddClient("https://gitlab.example.com", client1)

		// Add default client
		client2, _ := gl.NewClient("token2")
		pool.AddClient("default", client2)

		resolver := gitlab.NewClientResolver(pool, "default", log.New())

		// Resolve should match by host
		ctx := context.Background()
		client, name, err := resolver.Resolve(ctx)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}

		if name != "https://gitlab.example.com" {
			t.Errorf("expected name 'https://gitlab.example.com', got '%s'", name)
		}

		if client == nil {
			t.Error("resolved client is nil")
		}
	})

	t.Run("resolve priority: tokenName > host > default", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		os.Chdir(tmpDir)

		// Create .gmcprc with both tokenName and host
		pc := &gitlab.ProjectConfig{
			ProjectID:  "owner/repo",
			GitLabHost: "https://gitlab.example.com",
			TokenName:   "preferred",
		}
		gitlab.WriteProjectConfig(tmpDir, pc)

		// Create pool with all three
		store := gitlab.NewTokenStore()
		pool := gitlab.NewClientPool(store, log.New())

		preferredClient, _ := gl.NewClient("preferred-token")
		pool.AddClient("preferred", preferredClient)

		hostClient, _ := gl.NewClient("host-token")
		pool.AddClient("https://gitlab.example.com", hostClient)

		defaultClient, _ := gl.NewClient("default-token")
		pool.AddClient("default", defaultClient)

		resolver := gitlab.NewClientResolver(pool, "default", log.New())

		// Should prefer tokenName
		ctx := context.Background()
		_, name, err := resolver.Resolve(ctx)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}

		if name != "preferred" {
			t.Errorf("expected name 'preferred', got '%s'", name)
		}
	})

	t.Run("resolve for project", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		os.Chdir(tmpDir)

		// Create .gmcprc
		pc := &gitlab.ProjectConfig{
			ProjectID: "mygroup/myproject",
			TokenName:  "work",
		}
		gitlab.WriteProjectConfig(tmpDir, pc)

		// Create pool and resolver
		store := gitlab.NewTokenStore()
		pool := gitlab.NewClientPool(store, log.New())

		workClient, _ := gl.NewClient("work-token")
		pool.AddClient("work", workClient)

		resolver := gitlab.NewClientResolver(pool, "default", log.New())

		// Resolve for specific project
		ctx := context.Background()
		client, name, err := resolver.ResolveForProject(ctx, "mygroup/myproject")
		if err != nil {
			t.Fatalf("ResolveForProject() error = %v", err)
		}

		if name != "work" {
			t.Errorf("expected name 'work', got '%s'", name)
		}

		if client == nil {
			t.Error("resolved client is nil")
		}
	})

	t.Run("GetClientFn returns usable function", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		os.Chdir(tmpDir)

		// Create .gmcprc
		pc := &gitlab.ProjectConfig{
			ProjectID: "owner/repo",
			TokenName:  "test",
		}
		gitlab.WriteProjectConfig(tmpDir, pc)

		// Create pool and resolver
		store := gitlab.NewTokenStore()
		pool := gitlab.NewClientPool(store, log.New())

		testClient, _ := gl.NewClient("test-token")
		pool.AddClient("test", testClient)

		resolver := gitlab.NewClientResolver(pool, "default", log.New())

		// Get the function
		getClientFn := resolver.GetClientFn()
		if getClientFn == nil {
			t.Fatal("GetClientFn() returned nil")
		}

		// Use the function
		ctx := context.Background()
		client, err := getClientFn(ctx)
		if err != nil {
			t.Fatalf("getClientFn() error = %v", err)
		}

		if client == nil {
			t.Error("client from getClientFn is nil")
		}
	})

	t.Run("resolve with nested .gmcprc", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		// Create nested structure
		subDir := filepath.Join(tmpDir, "subdir")
		os.Mkdir(subDir, 0755)

		// Create .gmcprc in parent
		parentPC := &gitlab.ProjectConfig{
			ProjectID: "parent/repo",
			TokenName:  "parent-token",
		}
		gitlab.WriteProjectConfig(tmpDir, parentPC)

		// Create .gmcprc in subdirectory
		subPC := &gitlab.ProjectConfig{
			ProjectID: "child/repo",
			TokenName:  "child-token",
		}
		gitlab.WriteProjectConfig(subDir, subPC)

		// Create pool
		store := gitlab.NewTokenStore()
		pool := gitlab.NewClientPool(store, log.New())

		childClient, _ := gl.NewClient("child-token")
		pool.AddClient("child-token", childClient)

		resolver := gitlab.NewClientResolver(pool, "default", log.New())

		// Change to subdirectory
		os.Chdir(subDir)

		// Should find child .gmcprc
		ctx := context.Background()
		_, name, err := resolver.Resolve(ctx)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}

		if name != "child-token" {
			t.Errorf("expected name 'child-token', got '%s'", name)
		}
	})
}
