package gitlab

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	"github.com/InkyQuill/gitlab-mcp-server/pkg/toolsets"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gl "gitlab.com/gitlab-org/api/client-go"
)

// TestRaceCondition_ClientPool_ConcurrentAccess tests for race conditions
// in ClientPool when multiple goroutines access it simultaneously
func TestRaceCondition_ClientPool_ConcurrentAccess(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	store := NewTokenStore()
	cp := NewClientPool(store, logger)

	// Create multiple mock clients
	clients := make([]*gl.Client, 10)
	for i := range clients {
		clients[i] = &gl.Client{}
	}

	// Test concurrent writes and reads
	var wg sync.WaitGroup
	numGoroutines := 100
	iterations := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				name := "client-" + string(rune('A'+index%26)) + "-" + string(rune('0'+index%10))
				_ = cp.AddClient(name, clients[index%len(clients)])
				_, _ = cp.GetClient(name)
				cp.ListClients()
				_, _, _ = cp.GetDefaultClient()
			}
		}(i)
	}

	wg.Wait()

	// Verify state is consistent
	names := cp.ListClients()
	assert.Greater(t, len(names), 0, "Should have clients")
}

// TestRaceCondition_TokenStore_ConcurrentAccess tests for race conditions
// in TokenStore when multiple goroutines access it simultaneously
func TestRaceCondition_TokenStore_ConcurrentAccess(t *testing.T) {
	ts := NewTokenStore()

	metadata := &TokenMetadata{
		Token:      "test-token",
		GitLabHost: "https://gitlab.com",
		CreatedAt:  time.Now(),
	}

	var wg sync.WaitGroup
	numGoroutines := 100
	iterations := 50

	// Test concurrent writes - each goroutine uses unique name
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Use unique name per goroutine to avoid overwrites
				name := fmt.Sprintf("token-%d-%d", index, j)
				ts.AddToken(name, metadata)
				ts.GetToken(name)
				ts.ListTokens()
			}
		}(i)
	}

	wg.Wait()

	tokens := ts.ListTokens()
	// We expect numGoroutines * iterations tokens
	expectedCount := numGoroutines * iterations
	assert.Equal(t, expectedCount, len(tokens), "Should have all tokens")

	// Test concurrent reads and removals - remove some
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			// Remove half of the tokens this goroutine added
			for j := 0; j < iterations/2; j++ {
				name := fmt.Sprintf("token-%d-%d", index, j)
				ts.RemoveToken(name)
				ts.ListTokens()
			}
		}(i)
	}

	wg.Wait()

	tokens = ts.ListTokens()
	assert.Less(t, len(tokens), expectedCount, "Some tokens should be removed")
}

// TestRaceCondition_Notifications_ConcurrentAccess tests for race conditions
// in notification system when multiple goroutines send notifications
func TestRaceCondition_Notifications_ConcurrentAccess(t *testing.T) {
	ClearNotifications()

	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	var wg sync.WaitGroup
	numGoroutines := 50
	notificationsPerGoroutine := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for j := 0; j < notificationsPerGoroutine; j++ {
				SendNotification(logger, Notification{
					Level:   NotificationInfo,
					Title:   "Concurrent Test",
					Message: "Notification",
				})
			}
		}(i)
	}

	wg.Wait()

	notifications := GetNotifications()
	// The notification store caps at 100, so we expect max 100
	// The test verifies no race occurred and we got close to the cap
	expectedMin := 100
	if numGoroutines*notificationsPerGoroutine < 100 {
		expectedMin = numGoroutines * notificationsPerGoroutine
	}
	assert.GreaterOrEqual(t, len(notifications), expectedMin, "Should have at least min notifications")
	assert.LessOrEqual(t, len(notifications), 100, "Should not exceed cap of 100")
}

// TestRaceCondition_ConfigManager_ConcurrentAccess tests for race conditions
// in ConfigManager when multiple goroutines access it simultaneously
func TestRaceCondition_ConfigManager_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	cfg, err := config.NewManager(configPath)
	require.NoError(t, err)

	var wg sync.WaitGroup
	numGoroutines := 50

	// Add initial servers
	for i := 0; i < 10; i++ {
		server := &config.ServerConfig{
			Name:  "server-" + string(rune('A'+i)),
			Host:  "https://gitlab.example.com",
			Token: "token",
		}
		err := cfg.AddServer(server)
		require.NoError(t, err)
	}

	// Test concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			cfg.ListServers()
			cfg.GetDefaultServerName()
			cfg.HasServers()
			cfg.ServerCount()
		}(i)
	}

	wg.Wait()

	assert.Equal(t, 10, cfg.ServerCount(), "Should have all servers")
}

// TestRaceCondition_ToolsetGroup_ConcurrentAccess tests for race conditions
// in ToolsetGroup when multiple goroutines access it simultaneously
func TestRaceCondition_ToolsetGroup_ConcurrentAccess(t *testing.T) {
	tg := toolsets.NewToolsetGroup(false)

	// Add some toolsets
	for i := 0; i < 10; i++ {
		name := string(rune('A' + i))
		ts := toolsets.NewToolset(name, "Test toolset "+name)
		tg.AddToolset(ts)
	}

	var wg sync.WaitGroup
	numGoroutines := 50
	iterations := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				tg.ListToolsets()
			}
		}(i)
	}

	wg.Wait()

	infos := tg.ListToolsets()
	assert.Equal(t, 10, len(infos), "Should have all toolsets")
}

// TestRaceCondition_ClientPool_AddRemoveRace tests for specific race
// condition when adding and removing clients concurrently
func TestRaceCondition_ClientPool_AddRemoveRace(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	store := NewTokenStore()
	cp := NewClientPool(store, logger)

	mockClient := &gl.Client{}

	var wg sync.WaitGroup
	numGoroutines := 50

	// Half add, half remove
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			name := "client-" + string(rune('A'+index%26))
			if index%2 == 0 {
				cp.AddClient(name, mockClient)
			} else {
				cp.RemoveClient(name) // May or may not exist
			}
		}(i)
	}

	wg.Wait()

	// Should not panic and state should be consistent
	names := cp.ListClients()
	assert.GreaterOrEqual(t, len(names), 0, "Should have valid state")
}

// TestRaceCondition_ClientResolver_ConcurrentResolve tests for race
// conditions when resolving clients concurrently
func TestRaceCondition_ClientResolver_ConcurrentResolve(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	store := NewTokenStore()
	cp := NewClientPool(store, logger)

	// Add some clients
	for i := 0; i < 10; i++ {
		name := string(rune('A' + i))
		cp.AddClient(name, &gl.Client{})
	}

	cr := NewClientResolver(cp, "A", logger)

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			cr.Resolve(ctx)
		}()
	}

	wg.Wait()

	// Should not panic
}
