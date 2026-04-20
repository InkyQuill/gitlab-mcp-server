package gitlab

import (
	"context"
	"testing"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	log "github.com/sirupsen/logrus"
	gl "gitlab.com/gitlab-org/api/client-go"
)

func newTestLogger() *log.Logger {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel) // Disable debug output during benchmarks
	return logger
}

func BenchmarkClientPool_GetClient(b *testing.B) {
	pool := NewClientPool(NewTokenStore(), newTestLogger())
	client, _ := gl.NewClient("test-token", nil)
	_ = pool.AddClient("test", client)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = pool.GetClient("test")
	}
}

func BenchmarkClientPool_AddClient(b *testing.B) {
	pool := NewClientPool(NewTokenStore(), newTestLogger())
	client, _ := gl.NewClient("test-token", nil)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Use unique names to avoid "already exists" errors
		name := "test"
		pool.clients = make(map[string]*gl.Client) // Reset map
		_ = pool.AddClient(name, client)
	}
}

func BenchmarkClientPool_GetDefaultClient(b *testing.B) {
	pool := NewClientPool(NewTokenStore(), newTestLogger())
	client, _ := gl.NewClient("test-token", nil)
	_ = pool.AddClient("default", client)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = pool.GetDefaultClient()
	}
}

func BenchmarkClientPool_ListClients(b *testing.B) {
	pool := NewClientPool(NewTokenStore(), newTestLogger())

	// Add multiple clients
	for i := 0; i < 10; i++ {
		client, _ := gl.NewClient("test-token", nil)
		name := "client" + string(rune('0'+i))
		_ = pool.AddClient(name, client)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = pool.ListClients()
	}
}

func BenchmarkClientPool_RemoveClient(b *testing.B) {
	pool := NewClientPool(NewTokenStore(), newTestLogger())
	client, _ := gl.NewClient("test-token", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Re-add client for each iteration
		_ = pool.AddClient("test", client)
		_ = pool.RemoveClient("test")
	}
}

func BenchmarkTokenStore_AddToken(b *testing.B) {
	store := NewTokenStore()
	metadata := &TokenMetadata{
		Token:      "test-token",
		GitLabHost: "https://gitlab.com",
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		store.tokens = make(map[string]*TokenMetadata) // Reset map
		_ = store.AddToken("test", metadata)
	}
}

func BenchmarkTokenStore_GetToken(b *testing.B) {
	store := NewTokenStore()
	metadata := &TokenMetadata{
		Token:      "test-token",
		GitLabHost: "https://gitlab.com",
	}
	_ = store.AddToken("test", metadata)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = store.GetToken("test")
	}
}

func BenchmarkTokenStore_ListTokens(b *testing.B) {
	store := NewTokenStore()

	// Add multiple tokens
	for i := 0; i < 10; i++ {
		metadata := &TokenMetadata{
			Token:      "test-token",
			GitLabHost: "https://gitlab.com",
		}
		_ = store.AddToken("token"+string(rune('0'+i)), metadata)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = store.ListTokens()
	}
}

func BenchmarkTokenStore_RemoveToken(b *testing.B) {
	store := NewTokenStore()
	metadata := &TokenMetadata{
		Token:      "test-token",
		GitLabHost: "https://gitlab.com",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Re-add token for each iteration
		_ = store.AddToken("test", metadata)
		_ = store.RemoveToken("test")
	}
}

func BenchmarkClientPool_InitializeFromEnv(b *testing.B) {
	ctx := context.Background()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		pool := NewClientPool(NewTokenStore(), newTestLogger())
		_ = pool.InitializeFromEnv(ctx, "test-token", "https://gitlab.example.com")
	}
}

func BenchmarkClientPool_AddServerFromConfig(b *testing.B) {
	ctx := context.Background()
	pool := NewClientPool(NewTokenStore(), newTestLogger())
	server := &config.ServerConfig{
		Name:  "test",
		Host:  "https://gitlab.example.com",
		Token: "test-token",
	}
	resolve := func(_ context.Context, _ string) (string, error) {
		return "test-token", nil
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		pool.clients = make(map[string]*gl.Client) // Reset map
		_ = pool.AddServerFromConfig(ctx, server, resolve)
	}
}
