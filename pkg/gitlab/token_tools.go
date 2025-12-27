package gitlab

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gl "gitlab.com/gitlab-org/api/client-go"
	log "github.com/sirupsen/logrus"
)

// ClientFactory creates a GitLab client from a token
type ClientFactory func(token string, opts ...gl.ClientOptionFunc) (*gl.Client, error)

// DefaultClientFactory creates a real GitLab client
func DefaultClientFactory(token string, opts ...gl.ClientOptionFunc) (*gl.Client, error) {
	return gl.NewClient(token, opts...)
}

// AddToken adds a new GitLab token configuration
// Note: This tool does NOT persist tokens to MCP config - that requires installer update
// It only stores tokens in runtime memory for the current session
func AddToken(clientFactory ClientFactory, logger *log.Logger, tokenStore *TokenStore) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	// Use default factory if none provided
	if clientFactory == nil {
		clientFactory = DefaultClientFactory
	}
	return mcp.NewTool(
			"addToken",
			mcp.WithDescription("Adds a new GitLab token configuration to the runtime token store. Note: This only affects the current session. For permanent configuration, use the installer."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Token/server name (e.g., 'work', 'personal'). Must be unique."),
			),
			mcp.WithString("token",
				mcp.Required(),
				mcp.Description("GitLab Personal Access Token."),
			),
			mcp.WithString("gitlabHost",
				mcp.Description("GitLab host URL (e.g., 'https://gitlab.com'). Defaults to https://gitlab.com."),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Parse parameters
			name, err := requiredParam[string](&request, "name")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			token, err := requiredParam[string](&request, "token")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			gitlabHost, _ := request.Params.Arguments["gitlabHost"].(string)
			if gitlabHost == "" {
				gitlabHost = "https://gitlab.com"
			}

			// Create client to validate token
			clientOpts := []gl.ClientOptionFunc{}
			if gitlabHost != "" && gitlabHost != "https://gitlab.com" {
				clientOpts = append(clientOpts, gl.WithBaseURL(gitlabHost))
			}

			glClient, err := clientFactory(token, clientOpts...)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to create GitLab client: %v", err)), nil
			}

			// Validate token and get user info
			metadata, err := tokenStore.ValidateToken(ctx, name, glClient)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Token validation failed: %v", err)), nil
			}

			// Update metadata with additional info
			metadata.Token = token
			metadata.GitLabHost = gitlabHost

			// Re-add to store with all metadata
			if err := tokenStore.AddToken(name, metadata); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to store token: %v", err)), nil
			}

			// Send notification
			notifyTokenValidated(logger, name, metadata.UserID, metadata.Username)

			// Return success
			result := map[string]interface{}{
				"success":    true,
				"message":    fmt.Sprintf("Token '%s' added and validated successfully", name),
				"tokenName":  name,
				"userId":     metadata.UserID,
				"username":   metadata.Username,
				"gitlabHost": gitlabHost,
			}

			data, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		}
}

// ListTokens lists all configured tokens with their status
func ListTokens(tokenStore *TokenStore) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"listTokens",
			mcp.WithDescription("Lists all configured GitLab tokens with their validation status and metadata."),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			tokens := tokenStore.ListTokens()

			if len(tokens) == 0 {
				return mcp.NewToolResultText(`{
  "tokens": [],
  "message": "No tokens configured. Tokens are loaded from environment variables (GITLAB_TOKEN, GITLAB_SERVER_NAME) or added via addToken tool."
}`), nil
			}

			// Convert to JSON-serializable format
			tokensList := make([]map[string]interface{}, 0, len(tokens))
			for name, metadata := range tokens {
				tokenInfo := map[string]interface{}{
					"name":            name,
					"gitlabHost":      metadata.GitLabHost,
					"userId":          metadata.UserID,
					"username":        metadata.Username,
					"createdAt":       metadata.CreatedAt,
					"lastValidated":   metadata.LastValidated,
					"isExpired":       metadata.IsExpired(),
					"daysUntilExpiry": metadata.DaysUntilExpiry(),
				}
				if metadata.ExpiresAt != nil {
					tokenInfo["expiresAt"] = metadata.ExpiresAt
				}
				tokensList = append(tokensList, tokenInfo)
			}

			result := map[string]interface{}{
				"tokens":  tokensList,
				"count":   len(tokensList),
				"message": fmt.Sprintf("Found %d configured token(s)", len(tokensList)),
			}

			data, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		}
}

// UpdateToken updates an existing token configuration
func UpdateToken(clientFactory ClientFactory, logger *log.Logger, tokenStore *TokenStore) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	// Use default factory if none provided
	if clientFactory == nil {
		clientFactory = DefaultClientFactory
	}
	return mcp.NewTool(
			"updateToken",
			mcp.WithDescription("Updates an existing GitLab token. Validates the new token before updating."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Token name to update."),
			),
			mcp.WithString("token",
				mcp.Description("New GitLab Personal Access Token. If not provided, only revalidates existing token."),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Parse parameters
			name, err := requiredParam[string](&request, "name")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// Get existing token
			existing, err := tokenStore.GetToken(name)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Token '%s' not found: %v", name, err)), nil
			}

			// Check if new token value is provided
			newToken, hasNewToken := request.Params.Arguments["token"].(string)

			var tokenToValidate string
			if hasNewToken {
				tokenToValidate = newToken
			} else {
				tokenToValidate = existing.Token
			}

			// Create client to validate token
			clientOpts := []gl.ClientOptionFunc{}
			if existing.GitLabHost != "" && existing.GitLabHost != "https://gitlab.com" {
				clientOpts = append(clientOpts, gl.WithBaseURL(existing.GitLabHost))
			}

			glClient, err := clientFactory(tokenToValidate, clientOpts...)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to create GitLab client: %v", err)), nil
			}

			// Validate token
			metadata, err := tokenStore.ValidateToken(ctx, name, glClient)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Token validation failed: %v", err)), nil
			}

			// Update metadata
			if hasNewToken {
				metadata.Token = newToken
			} else {
				metadata.Token = existing.Token
			}
			metadata.GitLabHost = existing.GitLabHost
			metadata.CreatedAt = existing.CreatedAt

			// Re-add to store
			if err := tokenStore.AddToken(name, metadata); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to update token: %v", err)), nil
			}

			// Send notification
			notifyTokenValidated(logger, name, metadata.UserID, metadata.Username)

			// Return success
			result := map[string]interface{}{
				"success":   true,
				"message":   fmt.Sprintf("Token '%s' updated successfully", name),
				"tokenName": name,
				"userId":    metadata.UserID,
				"username":  metadata.Username,
				"updated":   hasNewToken,
			}

			data, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		}
}

// ValidateToken manually validates a token (or all tokens)
func ValidateToken(clientFactory ClientFactory, logger *log.Logger, tokenStore *TokenStore) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	// Use default factory if none provided
	if clientFactory == nil {
		clientFactory = DefaultClientFactory
	}
	return mcp.NewTool(
			"validateToken",
			mcp.WithDescription("Manually validates a GitLab token by calling the GitLab API. If no token name is provided, validates all configured tokens."),
			mcp.WithString("name",
				mcp.Description("Token name to validate. If not provided, validates all tokens."),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Parse parameters
			tokenName, _ := request.Params.Arguments["name"].(string)

			// Get client function for validation
			getClientForName := func(name string) (*gl.Client, error) {
				// Get token metadata
				metadata, err := tokenStore.GetToken(name)
				if err != nil {
					return nil, err
				}

				// Create client using factory
				clientOpts := []gl.ClientOptionFunc{}
				if metadata.GitLabHost != "" && metadata.GitLabHost != "https://gitlab.com" {
					clientOpts = append(clientOpts, gl.WithBaseURL(metadata.GitLabHost))
				}
				return clientFactory(metadata.Token, clientOpts...)
			}

			if tokenName != "" {
				// Validate specific token
				client, err := getClientForName(tokenName)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to create client: %v", err)), nil
				}

				validated, err := tokenStore.ValidateToken(ctx, tokenName, client)
				if err != nil {
					notifyTokenIssue(logger, tokenName, err)
					return mcp.NewToolResultError(fmt.Sprintf("Token validation failed: %v", err)), nil
				}

				notifyTokenValidated(logger, tokenName, validated.UserID, validated.Username)

				result := map[string]interface{}{
					"success":  true,
					"tokenName": tokenName,
					"userId":    validated.UserID,
					"username":  validated.Username,
					"message":   fmt.Sprintf("Token '%s' is valid", tokenName),
				}

				data, _ := json.MarshalIndent(result, "", "  ")
				return mcp.NewToolResultText(string(data)), nil
			} else {
				// Validate all tokens
				results := tokenStore.CheckAllTokens(ctx, getClientForName)

				// Count successes and failures
				successCount := 0
				failureCount := 0
				for _, result := range results {
					if result.Success {
						successCount++
					} else {
						failureCount++
					}
				}

				result := map[string]interface{}{
					"results":      results,
					"total":         len(results),
					"successCount":  successCount,
					"failureCount":  failureCount,
					"message":       fmt.Sprintf("Validated %d token(s): %d succeeded, %d failed", len(results), successCount, failureCount),
				}

				data, _ := json.MarshalIndent(result, "", "  ")
				return mcp.NewToolResultText(string(data)), nil
			}
		}
}

// GetNotificationsTool returns recent notifications
func GetNotificationsTool(logger *log.Logger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"getNotifications",
			mcp.WithDescription("Returns recent notifications about token issues, validation results, and other important messages."),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			notifications := GetNotifications()

			if len(notifications) == 0 {
				return mcp.NewToolResultText(`{
  "notifications": [],
  "message": "No notifications"
}`), nil
			}

			result := map[string]interface{}{
				"notifications": notifications,
				"count":          len(notifications),
			}

			data, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		}
}

// ClearNotificationsTool clears all stored notifications
func ClearNotificationsTool(logger *log.Logger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"clearNotifications",
			mcp.WithDescription("Clears all stored notifications."),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ClearNotifications()

			result := map[string]interface{}{
				"success": true,
				"message": "All notifications cleared",
			}

			data, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		}
}

// RemoveToken removes a token from the runtime store
func RemoveToken(tokenStore *TokenStore) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"removeToken",
			mcp.WithDescription("Removes a token from the runtime token store. Note: This does not update MCP configuration files."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Token name to remove."),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name, err := requiredParam[string](&request, "name")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			if err := tokenStore.RemoveToken(name); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to remove token: %v", err)), nil
			}

			result := map[string]interface{}{
				"success": true,
				"message": fmt.Sprintf("Token '%s' removed from runtime store", name),
			}

			data, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		}
}
