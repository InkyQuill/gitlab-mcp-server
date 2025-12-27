package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	gl "gitlab.com/gitlab-org/api/client-go"
	gltesting "gitlab.com/gitlab-org/api/client-go/testing"
	"go.uber.org/mock/gomock"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/InkyQuill/gitlab-mcp-server/internal/toolsnaps"
)

// TestAddTokenHandler tests the AddToken tool
func TestAddTokenHandler(t *testing.T) {
	// Tool schema snapshot test
	tool, _ := AddToken(nil, nil, nil)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	tokenStore := NewTokenStore()

	tool, handler := AddToken(nil, logger, tokenStore)

	// Test parameter validation (happens before GitLab API calls)
	validationTests := []struct {
		name              string
		inputArgs         map[string]any
		expectResultError bool
		errorContains     string
	}{
		{
			name:              "Error - Missing name",
			inputArgs:         map[string]any{"token": "glpat-test"},
			expectResultError: true,
			errorContains:     "Validation Error: missing required parameter: name",
		},
		{
			name:              "Error - Missing token",
			inputArgs:         map[string]any{"name": "test"},
			expectResultError: true,
			errorContains:     "Validation Error: missing required parameter: token",
		},
	}

	for _, tc := range validationTests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      tool.Name,
					Arguments: tc.inputArgs,
				},
			}

			result, err := handler(context.Background(), req)

			if tc.expectResultError {
				require.NoError(t, err)
				require.NotNil(t, result)
				textContent := getTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.errorContains)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}
		})
	}

	// GitLab API tests with proper mocking
	t.Run("Success - Add token with all params", func(t *testing.T) {
		// Create a test client with mocked services
		testClient := gltesting.NewTestClient(t)

		// Setup expectation for CurrentUser call (made by ValidateToken)
		testClient.MockUsers.EXPECT().
			CurrentUser(gomock.Any()).
			Return(&gl.User{
				ID:       123,
				Username: "testuser",
				Name:     "Test User",
				Email:    "test@example.com",
			}, &gl.Response{
				Response: &http.Response{
					StatusCode: 200,
				},
			}, nil)

		// Create a mock client factory that returns our test client
		mockClientFactory := func(token string, opts ...gl.ClientOptionFunc) (*gl.Client, error) {
			return testClient.Client, nil
		}

		tool, handler := AddToken(mockClientFactory, logger, tokenStore)

		req := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name: tool.Name,
				Arguments: map[string]any{
					"name":       "test-token",
					"token":      "glpat-testtoken123",
					"gitlabHost": "https://gitlab.example.com",
				},
			},
		}

		result, err := handler(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, result)

		textContent := getTextResult(t, result)

		// Verify response
		var resultMap map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &resultMap)
		require.NoError(t, err)
		assert.True(t, resultMap["success"].(bool))
		assert.Equal(t, "test-token", resultMap["tokenName"])
		assert.Equal(t, 123, int(resultMap["userId"].(float64)))
		assert.Equal(t, "testuser", resultMap["username"])
		assert.Equal(t, "https://gitlab.example.com", resultMap["gitlabHost"])

		// Verify token was actually stored
		storedToken, err := tokenStore.GetToken("test-token")
		require.NoError(t, err)
		assert.Equal(t, 123, storedToken.UserID)
		assert.Equal(t, "testuser", storedToken.Username)
		assert.Equal(t, "https://gitlab.example.com", storedToken.GitLabHost)
		assert.Equal(t, "glpat-testtoken123", storedToken.Token)
	})

	t.Run("Success - Add token with default host", func(t *testing.T) {
		// Create a test client with mocked services
		testClient := gltesting.NewTestClient(t)

		// Setup expectation for CurrentUser call
		testClient.MockUsers.EXPECT().
			CurrentUser(gomock.Any()).
			Return(&gl.User{
				ID:       456,
				Username: "defaultuser",
				Name:     "Default User",
				Email:    "default@example.com",
			}, &gl.Response{
				Response: &http.Response{
					StatusCode: 200,
				},
			}, nil)

		// Create a mock client factory
		mockClientFactory := func(token string, opts ...gl.ClientOptionFunc) (*gl.Client, error) {
			return testClient.Client, nil
		}

		tool, handler := AddToken(mockClientFactory, logger, tokenStore)

		req := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name: tool.Name,
				Arguments: map[string]any{
					"name":  "default-token",
					"token": "glpat-default456",
					// gitlabHost not provided - should default to https://gitlab.com
				},
			},
		}

		result, err := handler(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, result)

		textContent := getTextResult(t, result)

		// Verify response
		var resultMap map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &resultMap)
		require.NoError(t, err)
		assert.True(t, resultMap["success"].(bool))
		assert.Equal(t, "https://gitlab.com", resultMap["gitlabHost"])

		// Verify token was stored with default host
		storedToken, err := tokenStore.GetToken("default-token")
		require.NoError(t, err)
		assert.Equal(t, "https://gitlab.com", storedToken.GitLabHost)
	})
}

// TestListTokensHandler tests the ListTokens tool
func TestListTokensHandler(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	tokenStore := NewTokenStore()

	// Tool schema snapshot test
	tool, handler := ListTokens(tokenStore)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	tests := []struct {
		name     string
		setup    func()
		expected func(*testing.T, string)
	}{
		{
			name:  "Empty token store",
			setup: func() {},
			expected: func(t *testing.T, resultText string) {
				assert.Contains(t, resultText, `"tokens": []`)
				assert.Contains(t, resultText, "No tokens configured")
			},
		},
		{
			name: "List tokens with entries",
			setup: func() {
				metadata1 := &TokenMetadata{
					Token:      "token1",
					GitLabHost: "https://gitlab.com",
					CreatedAt:  time.Now(),
					UserID:     123,
					Username:   "user1",
				}
				metadata2 := &TokenMetadata{
					Token:      "token2",
					GitLabHost: "https://gitlab.example.com",
					CreatedAt:  time.Now(),
					UserID:     456,
					Username:   "user2",
				}
				_ = tokenStore.AddToken("work", metadata1)
				_ = tokenStore.AddToken("personal", metadata2)
			},
			expected: func(t *testing.T, resultText string) {
				assert.Contains(t, resultText, `"count": 2`)
				assert.Contains(t, resultText, `"name": "work"`)
				assert.Contains(t, resultText, `"name": "personal"`)
				assert.Contains(t, resultText, `"userId": 123`)
				assert.Contains(t, resultText, `"userId": 456`)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()

			req := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name: tool.Name,
				},
			}

			result, err := handler(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, result)

			textContent := getTextResult(t, result)
			tc.expected(t, textContent.Text)
		})
	}
}

// TestUpdateTokenHandler tests the UpdateToken tool
func TestUpdateTokenHandler(t *testing.T) {
	// Tool schema snapshot test
	tool, _ := UpdateToken(nil, nil, nil)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	tokenStore := NewTokenStore()

	tool, handler := UpdateToken(nil, logger, tokenStore)

	// Setup: Add an existing token
	existingMetadata := &TokenMetadata{
		Token:      "old-token",
		GitLabHost: "https://gitlab.com",
		CreatedAt:  time.Now(),
	}
	_ = tokenStore.AddToken("work", existingMetadata)

	// Test parameter validation (happens before GitLab API calls)
	validationTests := []struct {
		name              string
		inputArgs         map[string]any
		expectResultError bool
		errorContains     string
	}{
		{
			name:              "Error - Missing name",
			inputArgs:         map[string]any{},
			expectResultError: true,
			errorContains:     "Validation Error: missing required parameter: name",
		},
	}

	for _, tc := range validationTests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      tool.Name,
					Arguments: tc.inputArgs,
				},
			}

			result, err := handler(context.Background(), req)

			if tc.expectResultError {
				require.NoError(t, err)
				require.NotNil(t, result)
				textContent := getTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.errorContains)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}
		})
	}

	// GitLab API tests with proper mocking
	t.Run("Error - Token not found", func(t *testing.T) {
		// This tests the error path when token doesn't exist in store
		req := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      tool.Name,
				Arguments: map[string]any{"name": "nonexistent"},
			},
		}

		result, err := handler(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, result)

		textContent := getTextResult(t, result)
		assert.Contains(t, textContent.Text, "Token 'nonexistent' not found")
	})

	t.Run("Success - Revalidate existing token", func(t *testing.T) {
		// Create a test client with mocked services
		testClient := gltesting.NewTestClient(t)

		// Setup expectation for CurrentUser call
		testClient.MockUsers.EXPECT().
			CurrentUser(gomock.Any()).
			Return(&gl.User{
				ID:       789,
				Username: "workuser",
				Name:     "Work User",
				Email:    "work@example.com",
			}, &gl.Response{
				Response: &http.Response{
					StatusCode: 200,
				},
			}, nil)

		// Create a mock client factory
		mockClientFactory := func(token string, opts ...gl.ClientOptionFunc) (*gl.Client, error) {
			return testClient.Client, nil
		}

		tool, handler := UpdateToken(mockClientFactory, logger, tokenStore)

		req := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      tool.Name,
				Arguments: map[string]any{"name": "work"}, // No new token provided, just revalidate
			},
		}

		result, err := handler(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, result)

		textContent := getTextResult(t, result)

		// Verify response
		var resultMap map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &resultMap)
		require.NoError(t, err)
		assert.True(t, resultMap["success"].(bool))
		assert.Equal(t, false, resultMap["updated"]) // No new token provided
		assert.Equal(t, 789, int(resultMap["userId"].(float64)))

		// Verify token metadata was updated
		storedToken, err := tokenStore.GetToken("work")
		require.NoError(t, err)
		assert.Equal(t, 789, storedToken.UserID)
		assert.Equal(t, "workuser", storedToken.Username)
	})

	t.Run("Success - Update with new token value", func(t *testing.T) {
		// Setup: Create another existing token
		existingMetadata := &TokenMetadata{
			Token:      "old-token-value",
			GitLabHost: "https://gitlab.com",
			CreatedAt:  time.Now(),
		}
		_ = tokenStore.AddToken("personal", existingMetadata)

		// Create a test client with mocked services
		testClient := gltesting.NewTestClient(t)

		// Setup expectation for CurrentUser call with new token
		testClient.MockUsers.EXPECT().
			CurrentUser(gomock.Any()).
			Return(&gl.User{
				ID:       999,
				Username: "personaluser",
				Name:     "Personal User",
				Email:    "personal@example.com",
			}, &gl.Response{
				Response: &http.Response{
					StatusCode: 200,
				},
			}, nil)

		// Create a mock client factory
		mockClientFactory := func(token string, opts ...gl.ClientOptionFunc) (*gl.Client, error) {
			return testClient.Client, nil
		}

		tool, handler := UpdateToken(mockClientFactory, logger, tokenStore)

		req := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name: tool.Name,
				Arguments: map[string]any{
					"name":  "personal",
					"token": "new-personal-token", // Provide new token value
				},
			},
		}

		result, err := handler(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, result)

		textContent := getTextResult(t, result)

		// Verify response
		var resultMap map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &resultMap)
		require.NoError(t, err)
		assert.True(t, resultMap["success"].(bool))
		assert.Equal(t, true, resultMap["updated"]) // New token provided
		assert.Equal(t, 999, int(resultMap["userId"].(float64)))

		// Verify token metadata was updated with new token
		storedToken, err := tokenStore.GetToken("personal")
		require.NoError(t, err)
		assert.Equal(t, "new-personal-token", storedToken.Token)
		assert.Equal(t, 999, storedToken.UserID)
	})
}

// TestValidateTokenHandler tests the ValidateToken tool
func TestValidateTokenHandler(t *testing.T) {
	// Tool schema snapshot test
	tool, _ := ValidateToken(nil, nil, nil)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	// Test empty token store (no GitLab API calls needed)
	t.Run("Validate all tokens - empty store", func(t *testing.T) {
		tokenStore := NewTokenStore()
		tool, handler := ValidateToken(nil, logger, tokenStore)

		req := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name: tool.Name,
			},
		}

		result, err := handler(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, result)

		textContent := getTextResult(t, result)
		assert.Contains(t, textContent.Text, `"total": 0`)
	})

	// GitLab API tests with proper mocking
	t.Run("Validate specific token - success", func(t *testing.T) {
		tokenStore := NewTokenStore()
		// Setup: Add a token to the store
		existingMetadata := &TokenMetadata{
			Token:      "glpat-test123",
			GitLabHost: "https://gitlab.com",
			CreatedAt:  time.Now(),
		}
		_ = tokenStore.AddToken("test-token", existingMetadata)

		// Create a test client with mocked services
		testClient := gltesting.NewTestClient(t)

		// Setup expectation for CurrentUser call
		testClient.MockUsers.EXPECT().
			CurrentUser(gomock.Any()).
			Return(&gl.User{
				ID:       111,
				Username: "testtokenuser",
				Name:     "Test Token User",
				Email:    "testtoken@example.com",
			}, &gl.Response{
				Response: &http.Response{
					StatusCode: 200,
				},
			}, nil)

		// Create a mock client factory
		mockClientFactory := func(token string, opts ...gl.ClientOptionFunc) (*gl.Client, error) {
			return testClient.Client, nil
		}

		tool, handler := ValidateToken(mockClientFactory, logger, tokenStore)

		req := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      tool.Name,
				Arguments: map[string]any{"name": "test-token"},
			},
		}

		result, err := handler(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, result)

		textContent := getTextResult(t, result)

		// Verify response
		var resultMap map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &resultMap)
		require.NoError(t, err)
		assert.True(t, resultMap["success"].(bool))
		assert.Equal(t, "test-token", resultMap["tokenName"])
		assert.Equal(t, 111, int(resultMap["userId"].(float64)))
		assert.Equal(t, "testtokenuser", resultMap["username"])
		assert.Contains(t, textContent.Text, "Token 'test-token' is valid")
	})

	// DEADLOCK FIXED: CheckAllTokens no longer holds lock while calling ValidateToken
	t.Run("Validate all tokens - with tokens", func(t *testing.T) {
		tokenStore := NewTokenStore()
		// Setup: Add multiple tokens to the store
		metadata1 := &TokenMetadata{
			Token:      "glpat-token1",
			GitLabHost: "https://gitlab.com",
			CreatedAt:  time.Now(),
		}
		metadata2 := &TokenMetadata{
			Token:      "glpat-token2",
			GitLabHost: "https://gitlab.example.com",
			CreatedAt:  time.Now(),
		}
		_ = tokenStore.AddToken("token1", metadata1)
		_ = tokenStore.AddToken("token2", metadata2)

		// Create a test client with mocked services
		testClient := gltesting.NewTestClient(t)

		// Setup expectations for both tokens
		testClient.MockUsers.EXPECT().
			CurrentUser(gomock.Any()).
			Return(&gl.User{
				ID:       222,
				Username: "user1",
				Name:     "User One",
				Email:    "user1@example.com",
			}, &gl.Response{
				Response: &http.Response{
					StatusCode: 200,
				},
			}, nil)

		testClient.MockUsers.EXPECT().
			CurrentUser(gomock.Any()).
			Return(&gl.User{
				ID:       333,
				Username: "user2",
				Name:     "User Two",
				Email:    "user2@example.com",
			}, &gl.Response{
				Response: &http.Response{
					StatusCode: 200,
				},
			}, nil)

		// Create a mock client factory
		mockClientFactory := func(token string, opts ...gl.ClientOptionFunc) (*gl.Client, error) {
			return testClient.Client, nil
		}

		tool, handler := ValidateToken(mockClientFactory, logger, tokenStore)

		req := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name: tool.Name, // No name provided = validate all
			},
		}

		result, err := handler(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, result)

		textContent := getTextResult(t, result)

		// Verify response
		var resultMap map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &resultMap)
		require.NoError(t, err)
		assert.Equal(t, 2, int(resultMap["total"].(float64)))
		assert.Equal(t, 2, int(resultMap["successCount"].(float64)))
		assert.Equal(t, 0, int(resultMap["failureCount"].(float64)))

		// Verify results array
		results := resultMap["results"].([]interface{})
		require.Len(t, results, 2)
	})

	t.Run("Validate specific token - invalid token", func(t *testing.T) {
		tokenStore := NewTokenStore()
		// Setup: Add a token that will fail validation
		existingMetadata := &TokenMetadata{
			Token:      "glpat-invalid",
			GitLabHost: "https://gitlab.com",
			CreatedAt:  time.Now(),
		}
		_ = tokenStore.AddToken("invalid-token", existingMetadata)

		// Create a test client with mocked services
		testClient := gltesting.NewTestClient(t)

		// Setup: Return error for invalid token
		testClient.MockUsers.EXPECT().
			CurrentUser(gomock.Any()).
			Return(nil, (*gl.Response)(nil), assert.AnError)

		// Create a mock client factory
		mockClientFactory := func(token string, opts ...gl.ClientOptionFunc) (*gl.Client, error) {
			return testClient.Client, nil
		}

		tool, handler := ValidateToken(mockClientFactory, logger, tokenStore)

		req := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      tool.Name,
				Arguments: map[string]any{"name": "invalid-token"},
			},
		}

		result, err := handler(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should return error result
		textContent := getTextResult(t, result)
		assert.Contains(t, textContent.Text, "Token validation failed")
	})
}

// TestGetNotificationsToolHandler tests the GetNotifications tool
func TestGetNotificationsToolHandler(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	// Tool schema snapshot test
	tool, handler := GetNotificationsTool(logger)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	tests := []struct {
		name     string
		setup    func()
		expected func(*testing.T, string)
	}{
		{
			name:  "No notifications",
			setup: func() {},
			expected: func(t *testing.T, resultText string) {
				assert.Contains(t, resultText, `"notifications": []`)
				assert.Contains(t, resultText, "No notifications")
			},
		},
		{
			name: "With notifications",
			setup: func() {
				SendNotification(logger, Notification{
					Level:   NotificationInfo,
					Title:   "Test",
					Message: "Test notification 1",
				})
				SendNotification(logger, Notification{
					Level:   NotificationInfo,
					Title:   "Test",
					Message: "Test notification 2",
				})
			},
			expected: func(t *testing.T, resultText string) {
				assert.Contains(t, resultText, `"count": 2`)
				assert.Contains(t, resultText, "Test notification 1")
				assert.Contains(t, resultText, "Test notification 2")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Clear notifications before test
			ClearNotifications()
			tc.setup()

			req := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name: tool.Name,
				},
			}

			result, err := handler(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, result)

			textContent := getTextResult(t, result)
			tc.expected(t, textContent.Text)
		})
	}
}

// TestClearNotificationsToolHandler tests the ClearNotifications tool
func TestClearNotificationsToolHandler(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	// Tool schema snapshot test
	tool, handler := ClearNotificationsTool(logger)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	t.Run("Clear notifications", func(t *testing.T) {
		// Add some notifications
		SendNotification(logger, Notification{
			Level:   NotificationInfo,
			Title:   "Test",
			Message: "Test notification",
		})
		SendNotification(logger, Notification{
			Level:   NotificationInfo,
			Title:   "Test",
			Message: "Another notification",
		})

		req := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name: tool.Name,
			},
		}

		result, err := handler(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, result)

		textContent := getTextResult(t, result)

		// Verify success response
		var resultMap map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &resultMap)
		require.NoError(t, err)
		assert.True(t, resultMap["success"].(bool))
		assert.Contains(t, resultMap["message"], "All notifications cleared")

		// Verify notifications were actually cleared
		notifications := GetNotifications()
		assert.Equal(t, 0, len(notifications))
	})
}

// TestRemoveTokenHandler tests the RemoveToken tool
func TestRemoveTokenHandler(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	tokenStore := NewTokenStore()

	// Tool schema snapshot test
	tool, handler := RemoveToken(tokenStore)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	// Setup: Add a token
	metadata := &TokenMetadata{
		Token:      "test-token",
		GitLabHost: "https://gitlab.com",
		CreatedAt:  time.Now(),
	}
	_ = tokenStore.AddToken("test-token", metadata)

	tests := []struct {
		name              string
		inputArgs         map[string]any
		expectResultError bool
		errorContains     string
		verifyTokenExists bool
	}{
		{
			name: "Success - Remove existing token",
			inputArgs: map[string]any{
				"name": "test-token",
			},
			expectResultError: false,
			verifyTokenExists: false,
		},
		{
			name: "Error - Token not found",
			inputArgs: map[string]any{
				"name": "non-existent",
			},
			expectResultError: true,
			errorContains:     "Failed to remove token",
			verifyTokenExists: false,
		},
		{
			name:              "Error - Missing name",
			inputArgs:         map[string]any{},
			expectResultError: true,
			errorContains:     "Validation Error: missing required parameter: name",
			verifyTokenExists: true, // Original token should still exist
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Re-add token before each test if needed
			if tc.name == "Error - Missing name" {
				// Make sure token exists for this test
				metadata := &TokenMetadata{
					Token:      "test-token",
					GitLabHost: "https://gitlab.com",
					CreatedAt:  time.Now(),
				}
				_ = tokenStore.AddToken("test-token", metadata)
			}
			// For "Error - Token not found", we want the token to already be removed

			req := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      tool.Name,
					Arguments: tc.inputArgs,
				},
			}

			result, err := handler(context.Background(), req)

			if tc.expectResultError {
				require.NoError(t, err)
				require.NotNil(t, result)
				textContent := getTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.errorContains)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				textContent := getTextResult(t, result)
				var resultMap map[string]interface{}
				err = json.Unmarshal([]byte(textContent.Text), &resultMap)
				require.NoError(t, err)
				assert.True(t, resultMap["success"].(bool))
			}

			// Verify token existence
			_, err = tokenStore.GetToken("test-token")
			if tc.verifyTokenExists {
				assert.NoError(t, err, "Token should still exist")
			} else {
				assert.Error(t, err, "Token should not exist")
			}
		})
	}
}

// TestTokenManagementIntegration tests the full token management workflow
func TestTokenManagementIntegration(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	t.Run("Add -> List -> Remove workflow", func(t *testing.T) {
		tokenStore := NewTokenStore()

		// Create a test client with mocked services
		testClient := gltesting.NewTestClient(t)

		// Setup expectation for AddToken (calls CurrentUser)
		testClient.MockUsers.EXPECT().
			CurrentUser(gomock.Any()).
			Return(&gl.User{
				ID:       999,
				Username: "integrationuser",
				Name:     "Integration User",
				Email:    "integration@example.com",
			}, &gl.Response{
				Response: &http.Response{
					StatusCode: 200,
				},
			}, nil)

		// Create a mock client factory
		mockClientFactory := func(token string, opts ...gl.ClientOptionFunc) (*gl.Client, error) {
			return testClient.Client, nil
		}

		// Step 1: Add token
		addTool, addHandler := AddToken(mockClientFactory, logger, tokenStore)
		addReq := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name: addTool.Name,
				Arguments: map[string]any{
					"name":  "integration-test",
					"token": "glpat-integration",
				},
			},
		}

		addResult, err := addHandler(context.Background(), addReq)
		require.NoError(t, err)
		require.NotNil(t, addResult)

		addTextContent := getTextResult(t, addResult)
		var addResultMap map[string]interface{}
		err = json.Unmarshal([]byte(addTextContent.Text), &addResultMap)
		require.NoError(t, err)
		assert.True(t, addResultMap["success"].(bool))

		// Step 2: List tokens
		listTool, listHandler := ListTokens(tokenStore)
		listReq := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name: listTool.Name,
			},
		}

		listResult, err := listHandler(context.Background(), listReq)
		require.NoError(t, err)
		require.NotNil(t, listResult)

		listTextContent := getTextResult(t, listResult)
		assert.Contains(t, listTextContent.Text, "integration-test")

		// Verify token metadata
		storedToken, err := tokenStore.GetToken("integration-test")
		require.NoError(t, err)
		assert.Equal(t, 999, storedToken.UserID)
		assert.Equal(t, "integrationuser", storedToken.Username)

		// Step 3: Remove token
		removeTool, removeHandler := RemoveToken(tokenStore)
		removeReq := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name: removeTool.Name,
				Arguments: map[string]any{
					"name": "integration-test",
				},
			},
		}

		removeResult, err := removeHandler(context.Background(), removeReq)
		require.NoError(t, err)
		require.NotNil(t, removeResult)

		removeTextContent := getTextResult(t, removeResult)
		var removeResultMap map[string]interface{}
		err = json.Unmarshal([]byte(removeTextContent.Text), &removeResultMap)
		require.NoError(t, err)
		assert.True(t, removeResultMap["success"].(bool))

		// Verify token was removed
		_, err = tokenStore.GetToken("integration-test")
		assert.Error(t, err, "Token should be removed")
	})
}

// TestDefaultClientFactory tests the DefaultClientFactory function
func TestDefaultClientFactory(t *testing.T) {
	t.Run("Success - Create client with token", func(t *testing.T) {
		token := "glpat-test123"
		
		client, err := DefaultClientFactory(token)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})
	
	t.Run("Success - Create client with options", func(t *testing.T) {
		token := "glpat-test456"
		
		// Create client with custom base URL
		client, err := DefaultClientFactory(token, gl.WithBaseURL("https://gitlab.example.com"))
		require.NoError(t, err)
		assert.NotNil(t, client)
	})
}
