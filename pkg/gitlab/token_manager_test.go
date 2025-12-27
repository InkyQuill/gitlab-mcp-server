package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gl "gitlab.com/gitlab-org/api/client-go"
	gltesting "gitlab.com/gitlab-org/api/client-go/testing"
	"go.uber.org/mock/gomock"
)

func TestTokenMetadata_IsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	tests := []struct {
		name     string
		metadata *TokenMetadata
		expected bool
	}{
		{
			name:     "Expired via flag",
			metadata: &TokenMetadata{IsExpiredFlag: true},
			expected: true,
		},
		{
			name:     "Expired via expiration time",
			metadata: &TokenMetadata{ExpiresAt: &past},
			expected: true,
		},
		{
			name:     "Not expired - future expiration",
			metadata: &TokenMetadata{ExpiresAt: &future},
			expected: false,
		},
		{
			name:     "Not expired - no expiration set",
			metadata: &TokenMetadata{ExpiresAt: nil},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.metadata.IsExpired()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTokenMetadata_DaysUntilExpiry(t *testing.T) {
	now := time.Now()
	past := now.Add(-48 * time.Hour)  // 2 days ago
	future := now.Add(72 * time.Hour) // 3 days from now

	tests := []struct {
		name     string
		metadata *TokenMetadata
		expected int
	}{
		{
			name:     "No expiration set",
			metadata: &TokenMetadata{ExpiresAt: nil},
			expected: 0,
		},
		{
			name:     "Already expired (negative days)",
			metadata: &TokenMetadata{ExpiresAt: &past},
			expected: -2,
		},
		{
			name:     "Future expiration",
			metadata: &TokenMetadata{ExpiresAt: &future},
			expected: 2, // May be 2 or 3 depending on timing
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.metadata.DaysUntilExpiry()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNewTokenStore(t *testing.T) {
	ts := NewTokenStore()

	assert.NotNil(t, ts)
	assert.NotNil(t, ts.tokens)
	assert.Equal(t, 0, len(ts.tokens))
}

func TestTokenStore_AddToken(t *testing.T) {
	ts := NewTokenStore()

	metadata := &TokenMetadata{
		Token:      "test-token",
		GitLabHost: "https://gitlab.com",
		CreatedAt:  time.Now(),
	}

	tests := []struct {
		name        string
		tokenName   string
		metadata    *TokenMetadata
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Success - Add valid token",
			tokenName:   "test-token",
			metadata:    metadata,
			expectError: false,
		},
		{
			name:        "Error - Empty name",
			tokenName:   "",
			metadata:    metadata,
			expectError: true,
			errorMsg:    "token name cannot be empty",
		},
		{
			name:        "Error - Nil metadata",
			tokenName:   "test-token",
			metadata:    nil,
			expectError: true,
			errorMsg:    "token metadata cannot be nil",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ts.AddToken(tc.tokenName, tc.metadata)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				require.NoError(t, err)
				// Verify token was added
				token, err := ts.GetToken(tc.tokenName)
				require.NoError(t, err)
				assert.Equal(t, tc.tokenName, token.Name)
			}
		})
	}

	t.Run("Success - Update existing token", func(t *testing.T) {
		// Add initial token
		metadata1 := &TokenMetadata{
			Token:      "token1",
			GitLabHost: "https://gitlab.com",
			CreatedAt:  time.Now(),
		}
		err := ts.AddToken("work", metadata1)
		require.NoError(t, err)

		// Update with new metadata
		metadata2 := &TokenMetadata{
			Token:      "token2",
			GitLabHost: "https://gitlab.example.com",
			CreatedAt:  time.Now(),
		}
		err = ts.AddToken("work", metadata2)
		require.NoError(t, err)

		// Verify it was updated
		token, err := ts.GetToken("work")
		require.NoError(t, err)
		assert.Equal(t, "token2", token.Token)
		assert.Equal(t, "https://gitlab.example.com", token.GitLabHost)
	})
}

func TestTokenStore_GetToken(t *testing.T) {
	ts := NewTokenStore()

	metadata := &TokenMetadata{
		Token:      "test-token",
		GitLabHost: "https://gitlab.com",
		CreatedAt:  time.Now(),
	}
	err := ts.AddToken("test-token", metadata)
	require.NoError(t, err)

	tests := []struct {
		name        string
		tokenName   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Success - Get existing token",
			tokenName:   "test-token",
			expectError: false,
		},
		{
			name:        "Error - Token not found",
			tokenName:   "non-existent",
			expectError: true,
			errorMsg:    "token 'non-existent' not found",
		},
		{
			name:        "Error - Empty name",
			tokenName:   "",
			expectError: true,
			errorMsg:    "token '' not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, err := ts.GetToken(tc.tokenName)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
				assert.Nil(t, token)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, token)
				assert.Equal(t, tc.tokenName, token.Name)
			}
		})
	}
}

func TestTokenStore_ListTokens(t *testing.T) {
	ts := NewTokenStore()

	t.Run("Success - Empty store", func(t *testing.T) {
		tokens := ts.ListTokens()
		assert.NotNil(t, tokens)
		assert.Equal(t, 0, len(tokens))
	})

	t.Run("Success - List all tokens", func(t *testing.T) {
		// Add multiple tokens
		metadata1 := &TokenMetadata{Token: "token1", CreatedAt: time.Now()}
		metadata2 := &TokenMetadata{Token: "token2", CreatedAt: time.Now()}

		err := ts.AddToken("work", metadata1)
		require.NoError(t, err)
		err = ts.AddToken("personal", metadata2)
		require.NoError(t, err)

		tokens := ts.ListTokens()
		assert.Equal(t, 2, len(tokens))
		assert.Contains(t, tokens, "work")
		assert.Contains(t, tokens, "personal")

		// Verify it's a copy (modifying returned map shouldn't affect store)
		delete(tokens, "work")
		tokens2 := ts.ListTokens()
		assert.Contains(t, tokens2, "work")
	})
}

func TestTokenStore_RemoveToken(t *testing.T) {
	ts := NewTokenStore()

	metadata := &TokenMetadata{
		Token:      "test-token",
		GitLabHost: "https://gitlab.com",
		CreatedAt:  time.Now(),
	}
	err := ts.AddToken("test-token", metadata)
	require.NoError(t, err)

	tests := []struct {
		name        string
		tokenName   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Success - Remove existing token",
			tokenName:   "test-token",
			expectError: false,
		},
		{
			name:        "Error - Token not found",
			tokenName:   "non-existent",
			expectError: true,
			errorMsg:    "token 'non-existent' not found",
		},
		{
			name:        "Error - Empty name",
			tokenName:   "",
			expectError: true,
			errorMsg:    "token '' not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ts.RemoveToken(tc.tokenName)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				require.NoError(t, err)
				// Verify token was removed
				_, err := ts.GetToken(tc.tokenName)
				require.Error(t, err)
			}
		})
	}
}

func TestTokenStore_ConcurrentAccess(t *testing.T) {
	ts := NewTokenStore()

	metadata := &TokenMetadata{
		Token:      "test-token",
		GitLabHost: "https://gitlab.com",
		CreatedAt:  time.Now(),
	}

	// Test concurrent writes
	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			name := fmt.Sprintf("token-%d", index)
			ts.AddToken(name, metadata)
			ts.GetToken(name)
			ts.ListTokens()
		}(i)
	}

	wg.Wait()

	// Verify all tokens were added
	tokens := ts.ListTokens()
	assert.Equal(t, numGoroutines, len(tokens))

	// Test concurrent reads and removals
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			name := fmt.Sprintf("token-%d", index)
			ts.RemoveToken(name)
			ts.ListTokens()
		}(i)
	}

	wg.Wait()

	// Verify some tokens were removed
	tokens = ts.ListTokens()
	assert.Less(t, len(tokens), numGoroutines)
}

func TestTokenStore_ValidateToken(t *testing.T) {
	ts := NewTokenStore()

	t.Run("Success - Validate token and update metadata", func(t *testing.T) {
		// Setup: Add a token without user info
		metadata := &TokenMetadata{
			Token:     "glpat-test123",
			CreatedAt: time.Now(),
		}
		err := ts.AddToken("test-token", metadata)
		require.NoError(t, err)

		// Create a test client with mocked services
		testClient := gltesting.NewTestClient(t)

		// Setup expectation for CurrentUser call
		testClient.MockUsers.EXPECT().
			CurrentUser(gomock.Any()).
			Return(&gl.User{
				ID:       12345,
				Username: "testuser",
				Name:     "Test User",
				Email:    "test@example.com",
			}, &gl.Response{
				Response: &http.Response{
					StatusCode: 200,
				},
			}, nil)

		// Validate token
		ctx := context.Background()
		validatedMetadata, err := ts.ValidateToken(ctx, "test-token", testClient.Client)
		require.NoError(t, err)
		assert.NotNil(t, validatedMetadata)

		// Verify metadata was updated
		assert.Equal(t, 12345, validatedMetadata.UserID)
		assert.Equal(t, "testuser", validatedMetadata.Username)
		assert.False(t, validatedMetadata.IsExpiredFlag)
		assert.True(t, validatedMetadata.LastValidated.After(validatedMetadata.CreatedAt))

		// Verify token was updated in store
		storedToken, err := ts.GetToken("test-token")
		require.NoError(t, err)
		assert.Equal(t, 12345, storedToken.UserID)
		assert.Equal(t, "testuser", storedToken.Username)
	})

	t.Run("Success - Create new token metadata if not exists", func(t *testing.T) {
		// Create a test client with mocked services
		testClient := gltesting.NewTestClient(t)

		// Setup expectation for CurrentUser call
		testClient.MockUsers.EXPECT().
			CurrentUser(gomock.Any()).
			Return(&gl.User{
				ID:       67890,
				Username: "newuser",
				Name:     "New User",
				Email:    "new@example.com",
			}, &gl.Response{
				Response: &http.Response{
					StatusCode: 200,
				},
			}, nil)

		// Validate token that doesn't exist yet
		ctx := context.Background()
		validatedMetadata, err := ts.ValidateToken(ctx, "new-token", testClient.Client)
		require.NoError(t, err)
		assert.NotNil(t, validatedMetadata)

		// Verify new metadata was created
		assert.Equal(t, 67890, validatedMetadata.UserID)
		assert.Equal(t, "newuser", validatedMetadata.Username)
		assert.Equal(t, "new-token", validatedMetadata.Name)

		// Verify token was added to store
		storedToken, err := ts.GetToken("new-token")
		require.NoError(t, err)
		assert.Equal(t, 67890, storedToken.UserID)
	})

	t.Run("Error - Invalid token (401)", func(t *testing.T) {
		// Setup: Add a token
		metadata := &TokenMetadata{
			Token:     "glpat-invalid",
			CreatedAt: time.Now(),
		}
		err := ts.AddToken("invalid-token", metadata)
		require.NoError(t, err)

		// Create a test client with mocked services
		testClient := gltesting.NewTestClient(t)

		// Setup expectation for CurrentUser call returning error
		testClient.MockUsers.EXPECT().
			CurrentUser(gomock.Any()).
			Return(nil, &gl.Response{
				Response: &http.Response{
					StatusCode: 401,
				},
			}, fmt.Errorf("401 Unauthorized"))

		// Validate token
		ctx := context.Background()
		_, err = ts.ValidateToken(ctx, "invalid-token", testClient.Client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})

	t.Run("Error - Network error", func(t *testing.T) {
		// Setup: Add a token
		metadata := &TokenMetadata{
			Token:     "glpat-network-error",
			CreatedAt: time.Now(),
		}
		err := ts.AddToken("network-error-token", metadata)
		require.NoError(t, err)

		// Create a test client with mocked services
		testClient := gltesting.NewTestClient(t)

		// Setup expectation for CurrentUser call returning network error
		testClient.MockUsers.EXPECT().
			CurrentUser(gomock.Any()).
			Return(nil, (*gl.Response)(nil), fmt.Errorf("network timeout"))

		// Validate token
		ctx := context.Background()
		_, err = ts.ValidateToken(ctx, "network-error-token", testClient.Client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "network timeout")
	})
}

func TestTokenStore_CheckAllTokens(t *testing.T) {
	ts := NewTokenStore()

	// Add some tokens
	metadata1 := &TokenMetadata{Token: "token1", CreatedAt: time.Now()}
	metadata2 := &TokenMetadata{Token: "token2", CreatedAt: time.Now()}

	err := ts.AddToken("token1", metadata1)
	require.NoError(t, err)
	err = ts.AddToken("token2", metadata2)
	require.NoError(t, err)

	t.Run("Success - Check all tokens", func(t *testing.T) {
		ctx := context.Background()

		// Mock getClientFunc that returns error
		getClientFunc := func(name string) (*gl.Client, error) {
			return nil, fmt.Errorf("client not found")
		}

		results := ts.CheckAllTokens(ctx, getClientFunc)
		assert.NotNil(t, results)
		assert.Equal(t, 2, len(results))

		// All should fail due to client error
		for _, result := range results {
			assert.False(t, result.Success)
			assert.Contains(t, result.Error, "Failed to get client")
		}
	})

	t.Run("Success - Empty store", func(t *testing.T) {
		ts2 := NewTokenStore()
		ctx := context.Background()

		getClientFunc := func(name string) (*gl.Client, error) {
			return nil, fmt.Errorf("client not found")
		}

		results := ts2.CheckAllTokens(ctx, getClientFunc)
		assert.NotNil(t, results)
		assert.Equal(t, 0, len(results))
	})
}
