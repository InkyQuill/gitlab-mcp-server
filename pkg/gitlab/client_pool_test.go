package gitlab

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gl "gitlab.com/gitlab-org/api/client-go"
	gltesting "gitlab.com/gitlab-org/api/client-go/testing"
	"go.uber.org/mock/gomock"
)

func TestNewClientPool(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel) // Reduce noise in tests

	store := NewTokenStore()
	cp := NewClientPool(store, logger)

	assert.NotNil(t, cp)
	assert.NotNil(t, cp.clients)
	assert.NotNil(t, cp.store)
	assert.NotNil(t, cp.logger)
	assert.Equal(t, 0, len(cp.clients))
}

func TestClientPool_AddClient(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	cp := NewClientPool(store, logger)

	// Create mock client
	mockClient := &gl.Client{}

	tests := []struct {
		name        string
		clientName  string
		client      *gl.Client
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Success - Add valid client",
			clientName:  "test-server",
			client:      mockClient,
			expectError: false,
		},
		{
			name:        "Error - Empty name",
			clientName:  "",
			client:      mockClient,
			expectError: true,
			errorMsg:    "client name cannot be empty",
		},
		{
			name:        "Error - Nil client",
			clientName:  "test-server",
			client:      nil,
			expectError: true,
			errorMsg:    "client cannot be nil",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := cp.AddClient(tc.clientName, tc.client)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				require.NoError(t, err)
				// Verify client was added
				client, err := cp.GetClient(tc.clientName)
				require.NoError(t, err)
				assert.Same(t, tc.client, client)
			}
		})
	}

	t.Run("Success - Overwrite existing client", func(t *testing.T) {
		// Add a client
		mockClient1 := &gl.Client{}
		err := cp.AddClient("server1", mockClient1)
		require.NoError(t, err)

		// Overwrite with new client
		mockClient2 := &gl.Client{}
		err = cp.AddClient("server1", mockClient2)
		require.NoError(t, err)

		// Verify it was overwritten
		client, err := cp.GetClient("server1")
		require.NoError(t, err)
		assert.Same(t, mockClient2, client)
	})
}

func TestClientPool_GetClient(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	cp := NewClientPool(store, logger)

	mockClient := &gl.Client{}
	err := cp.AddClient("test-server", mockClient)
	require.NoError(t, err)

	tests := []struct {
		name        string
		clientName  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Success - Get existing client",
			clientName:  "test-server",
			expectError: false,
		},
		{
			name:        "Error - Client not found",
			clientName:  "non-existent",
			expectError: true,
			errorMsg:    "client 'non-existent' not found in pool",
		},
		{
			name:        "Error - Empty name",
			clientName:  "",
			expectError: true,
			errorMsg:    "client '' not found in pool",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, err := cp.GetClient(tc.clientName)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.Same(t, mockClient, client)
			}
		})
	}
}

func TestClientPool_GetDefaultClient(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	cp := NewClientPool(store, logger)

	mockClient1 := &gl.Client{}
	mockClient2 := &gl.Client{}

	t.Run("Success - Return default client", func(t *testing.T) {
		err := cp.AddClient("default", mockClient1)
		require.NoError(t, err)

		client, name, err := cp.GetDefaultClient()
		require.NoError(t, err)
		assert.Same(t, mockClient1, client)
		assert.Equal(t, "default", name)
	})

	t.Run("Success - Return first available when no default", func(t *testing.T) {
		// Create new pool without default
		cp2 := NewClientPool(store, logger)
		err := cp2.AddClient("server1", mockClient1)
		require.NoError(t, err)

		client, name, err := cp2.GetDefaultClient()
		require.NoError(t, err)
		assert.Same(t, mockClient1, client)
		assert.Equal(t, "server1", name)
	})

	t.Run("Success - Default takes precedence over others", func(t *testing.T) {
		cp3 := NewClientPool(store, logger)
		err := cp3.AddClient("server1", mockClient1)
		require.NoError(t, err)
		err = cp3.AddClient("default", mockClient2)
		require.NoError(t, err)

		client, name, err := cp3.GetDefaultClient()
		require.NoError(t, err)
		assert.Same(t, mockClient2, client)
		assert.Equal(t, "default", name)
	})

	t.Run("Error - No clients available", func(t *testing.T) {
		cp4 := NewClientPool(store, logger)

		client, name, err := cp4.GetDefaultClient()
		require.Error(t, err)
		assert.Nil(t, client)
		assert.Empty(t, name)
		assert.Contains(t, err.Error(), "no clients available in pool")
	})
}

func TestClientPool_ListClients(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	cp := NewClientPool(store, logger)

	t.Run("Success - Empty pool", func(t *testing.T) {
		names := cp.ListClients()
		assert.NotNil(t, names)
		assert.Equal(t, 0, len(names))
	})

	t.Run("Success - List all clients", func(t *testing.T) {
		mockClient1 := &gl.Client{}
		mockClient2 := &gl.Client{}

		err := cp.AddClient("server1", mockClient1)
		require.NoError(t, err)
		err = cp.AddClient("server2", mockClient2)
		require.NoError(t, err)

		names := cp.ListClients()
		assert.Equal(t, 2, len(names))
		assert.Contains(t, names, "server1")
		assert.Contains(t, names, "server2")
	})

	t.Run("Success - List clients after removal", func(t *testing.T) {
		cp2 := NewClientPool(store, logger)
		mockClient := &gl.Client{}

		cp2.AddClient("server1", mockClient)
		cp2.AddClient("server2", mockClient)

		err := cp2.RemoveClient("server1")
		require.NoError(t, err)

		names := cp2.ListClients()
		assert.Equal(t, 1, len(names))
		assert.Contains(t, names, "server2")
		assert.NotContains(t, names, "server1")
	})
}

func TestClientPool_RemoveClient(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	cp := NewClientPool(store, logger)

	mockClient := &gl.Client{}
	err := cp.AddClient("test-server", mockClient)
	require.NoError(t, err)

	tests := []struct {
		name        string
		clientName  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Success - Remove existing client",
			clientName:  "test-server",
			expectError: false,
		},
		{
			name:        "Error - Client not found",
			clientName:  "non-existent",
			expectError: true,
			errorMsg:    "client 'non-existent' not found in pool",
		},
		{
			name:        "Error - Empty name",
			clientName:  "",
			expectError: true,
			errorMsg:    "client '' not found in pool",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := cp.RemoveClient(tc.clientName)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				require.NoError(t, err)
				// Verify client was removed
				_, err := cp.GetClient(tc.clientName)
				require.Error(t, err)
			}
		})
	}
}

func TestClientPool_ConcurrentAccess(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	cp := NewClientPool(store, logger)

	mockClient := &gl.Client{}

	// Test concurrent writes
	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			name := "server-" + string(rune('A'+index%26)) + "-" + string(rune('0'+index%10))
			cp.AddClient(name, mockClient)
			cp.GetClient(name)
			cp.ListClients()
		}(i)
	}

	wg.Wait()

	// Verify all clients were added
	names := cp.ListClients()
	assert.Equal(t, numGoroutines, len(names))

	// Test concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cp.ListClients()
			cp.GetDefaultClient()
		}()
	}

	wg.Wait()
}

func TestClientPool_InitializeFromEnv(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	cp := NewClientPool(store, logger)

	t.Run("Success - Initialize with default host", func(t *testing.T) {
		ctx := context.Background()
		token := "test-token"

		err := cp.InitializeFromEnv(ctx, token, "")
		require.NoError(t, err)

		// Verify default client was added
		client, name, err := cp.GetDefaultClient()
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "default", name)
	})

	t.Run("Success - Initialize with custom host", func(t *testing.T) {
		cp2 := NewClientPool(store, logger)
		ctx := context.Background()
		token := "test-token"
		host := "https://gitlab.example.com"

		err := cp2.InitializeFromEnv(ctx, token, host)
		require.NoError(t, err)

		// Verify client with hostname was added
		names := cp2.ListClients()
		assert.Equal(t, 1, len(names))
		assert.Contains(t, names, host)
	})

	t.Run("Success - Initialize with gitlab.com (uses default client options)", func(t *testing.T) {
		cp3 := NewClientPool(store, logger)
		ctx := context.Background()
		token := "test-token"
		host := "https://gitlab.com"

		err := cp3.InitializeFromEnv(ctx, token, host)
		require.NoError(t, err)

		// gitlab.com uses the full URL as server name (not "default")
		client, name, err := cp3.GetDefaultClient()
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "https://gitlab.com", name)
	})
}

func TestClientPool_ValidateAllClients(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	cp := NewClientPool(store, logger)

	// Add some tokens to store
	metadata1 := &TokenMetadata{Token: "token1", CreatedAt: time.Now()}
	metadata2 := &TokenMetadata{Token: "token2", CreatedAt: time.Now()}
	_ = store.AddToken("client1", metadata1)
	_ = store.AddToken("client2", metadata2)

	t.Run("Success - Empty pool returns validation results for all tokens", func(t *testing.T) {
		ctx := context.Background()

		// Store has tokens but pool is empty
		results := cp.ValidateAllClients(ctx)
		require.NotNil(t, results)
		assert.Equal(t, 2, len(results)) // Still returns results for all tokens

		// All should fail with "client not found" error
		for _, result := range results {
			assert.False(t, result.Success)
			assert.Contains(t, result.Error, "Failed to get client")
			assert.Contains(t, result.Error, "not found")
		}
	})

	t.Run("Success - Returns empty results when store is empty", func(t *testing.T) {
		cp2 := NewClientPool(NewTokenStore(), logger)
		ctx := context.Background()

		results := cp2.ValidateAllClients(ctx)
		require.NotNil(t, results)
		assert.Equal(t, 0, len(results))
	})

	t.Run("Success - Validate all clients with mocked GitLab API", func(t *testing.T) {
		// Setup: Create a test client pool with tokens
		logger := log.New()
		logger.SetLevel(log.ErrorLevel)
		store := NewTokenStore()
		cp := NewClientPool(store, logger)

		// Create separate test clients for each token to avoid race conditions
		testClient1 := gltesting.NewTestClient(t)
		testClient2 := gltesting.NewTestClient(t)

		// Setup expectations for each token separately
		testClient1.MockUsers.EXPECT().
			CurrentUser(gomock.Any()).
			Return(&gl.User{
				ID:       111,
				Username: "user1",
				Name:     "User One",
				Email:    "user1@example.com",
			}, &gl.Response{
				Response: &http.Response{
					StatusCode: 200,
				},
			}, nil)

		testClient2.MockUsers.EXPECT().
			CurrentUser(gomock.Any()).
			Return(&gl.User{
				ID:       222,
				Username: "user2",
				Name:     "User Two",
				Email:    "user2@example.com",
			}, &gl.Response{
				Response: &http.Response{
					StatusCode: 200,
				},
			}, nil)

		// Add tokens to store
		metadata1 := &TokenMetadata{
			Token:      "glpat-token1",
			GitLabHost: "https://gitlab.example.com",
			CreatedAt:  time.Now(),
		}
		metadata2 := &TokenMetadata{
			Token:      "glpat-token2",
			GitLabHost: "https://gitlab.com",
			CreatedAt:  time.Now(),
		}
		_ = store.AddToken("client1", metadata1)
		_ = store.AddToken("client2", metadata2)

		// Add clients to pool (each with its own test client)
		_ = cp.AddClient("client1", testClient1.Client)
		_ = cp.AddClient("client2", testClient2.Client)

		// Validate all clients
		ctx := context.Background()
		results := cp.ValidateAllClients(ctx)

		// Verify results
		require.NotNil(t, results)
		assert.Equal(t, 2, len(results))

		// Check results (order may vary)
		client1Result := findResult(results, "client1")
		require.NotNil(t, client1Result)
		assert.True(t, client1Result.Success)
		assert.Equal(t, 111, client1Result.UserID)
		assert.Equal(t, "user1", client1Result.Username)

		client2Result := findResult(results, "client2")
		require.NotNil(t, client2Result)
		assert.True(t, client2Result.Success)
		assert.Equal(t, 222, client2Result.UserID)
		assert.Equal(t, "user2", client2Result.Username)
	})

	t.Run("Success - Mixed validation results", func(t *testing.T) {
		// Setup
		logger := log.New()
		logger.SetLevel(log.ErrorLevel)
		store := NewTokenStore()
		cp := NewClientPool(store, logger)

		// Create separate test clients for each token
		validTestClient := gltesting.NewTestClient(t)
		invalidTestClient := gltesting.NewTestClient(t)

		// First token succeeds, second fails
		validTestClient.MockUsers.EXPECT().
			CurrentUser(gomock.Any()).
			Return(&gl.User{
				ID:       333,
				Username: "valid-user",
				Name:     "Valid User",
				Email:    "valid@example.com",
			}, &gl.Response{
				Response: &http.Response{
					StatusCode: 200,
				},
			}, nil)

		invalidTestClient.MockUsers.EXPECT().
			CurrentUser(gomock.Any()).
			Return(nil, (*gl.Response)(nil), assert.AnError)

		// Add tokens
		metadata1 := &TokenMetadata{
			Token:      "glpat-valid",
			GitLabHost: "https://gitlab.example.com",
			CreatedAt:  time.Now(),
		}
		metadata2 := &TokenMetadata{
			Token:      "glpat-invalid",
			GitLabHost: "https://gitlab.com",
			CreatedAt:  time.Now(),
		}
		_ = store.AddToken("valid-client", metadata1)
		_ = store.AddToken("invalid-client", metadata2)

		// Add clients
		_ = cp.AddClient("valid-client", validTestClient.Client)
		_ = cp.AddClient("invalid-client", invalidTestClient.Client)

		// Validate all clients
		ctx := context.Background()
		results := cp.ValidateAllClients(ctx)

		// Verify results
		require.NotNil(t, results)
		assert.Equal(t, 2, len(results))

		// First should succeed
		validResult := findResult(results, "valid-client")
		require.NotNil(t, validResult)
		assert.True(t, validResult.Success)
		assert.Equal(t, 333, validResult.UserID)

		// Second should fail
		invalidResult := findResult(results, "invalid-client")
		require.NotNil(t, invalidResult)
		assert.False(t, invalidResult.Success)
		assert.Contains(t, invalidResult.Error, "error")
	})
}

// Helper function to find a result by token name
func findResult(results []TokenValidationResult, tokenName string) *TokenValidationResult {
	for i := range results {
		if results[i].TokenName == tokenName {
			return &results[i]
		}
	}
	return nil
}
