package gitlab

import (
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	gl "gitlab.com/gitlab-org/api/client-go"
)

// HandleAPIError provides centralized error handling for GitLab API calls
// It checks the response status code and returns appropriate MCP error results
// Returns:
//   - (*mcp.CallToolResult, nil) if the error should be returned to the user
//   - (nil, error) if the error should be propagated as an internal error
//   - (nil, nil) if err is nil
func HandleAPIError(err error, resp *gl.Response, resourceDescription string) (*mcp.CallToolResult, error) {
	if err == nil {
		return nil, nil
	}

	code := http.StatusInternalServerError
	if resp != nil {
		code = resp.StatusCode
	}

	// Handle 401 Unauthorized - token expired or invalid
	if code == http.StatusUnauthorized {
		msg := fmt.Sprintf("Authentication failed (401). Your GitLab token may be expired. Please update it using the updateToken tool.")
		return mcp.NewToolResultError(msg), nil
	}

	// Handle 404 Not Found
	if code == http.StatusNotFound {
		msg := fmt.Sprintf("%s not found or access denied (%d)", resourceDescription, code)
		return mcp.NewToolResultError(msg), nil
	}

	// Handle 400 Bad Request / 422 Unprocessable Entity
	if code == http.StatusBadRequest || code == http.StatusUnprocessableEntity {
		msg := fmt.Sprintf("failed to process %s: %v (status: %d)", resourceDescription, err, code)
		return mcp.NewToolResultError(msg), nil
	}

	// For all other errors, return as internal error
	return nil, fmt.Errorf("failed to process %s: %w (status: %d)", resourceDescription, err, code)
}

// HandleListAPIError provides centralized error handling for GitLab list API calls
// Unlike HandleAPIError, this doesn't treat 404 as an error (empty list is valid)
// Returns:
//   - (*mcp.CallToolResult, nil) if the error should be returned to the user
//   - (nil, error) if the error should be propagated as an internal error
//   - (nil, nil) if err is nil
func HandleListAPIError(err error, resp *gl.Response, resourceDescription string) (*mcp.CallToolResult, error) {
	if err == nil {
		return nil, nil
	}

	code := http.StatusInternalServerError
	if resp != nil {
		code = resp.StatusCode
	}

	// Handle 401 Unauthorized - token expired or invalid
	if code == http.StatusUnauthorized {
		msg := fmt.Sprintf("Authentication failed (401). Your GitLab token may be expired. Please update it using the updateToken tool.")
		return mcp.NewToolResultError(msg), nil
	}

	// For list operations, don't treat 404 as an error - empty list is valid
	// For all other errors, return as internal error
	return nil, fmt.Errorf("failed to list %s: %w (status: %d)", resourceDescription, err, code)
}

// HandleCreateUpdateAPIError provides centralized error handling for GitLab create/update API calls
// This handles both 404 (resource not found) and 400/422 (validation errors)
// Returns:
//   - (*mcp.CallToolResult, nil) if the error should be returned to the user
//   - (nil, error) if the error should be propagated as an internal error
//   - (nil, nil) if err is nil
func HandleCreateUpdateAPIError(err error, resp *gl.Response, resourceDescription, operation string) (*mcp.CallToolResult, error) {
	if err == nil {
		return nil, nil
	}

	code := http.StatusInternalServerError
	if resp != nil {
		code = resp.StatusCode
	}

	// Handle 401 Unauthorized - token expired or invalid
	if code == http.StatusUnauthorized {
		msg := fmt.Sprintf("Authentication failed (401). Your GitLab token may be expired. Please update it using the updateToken tool.")
		return mcp.NewToolResultError(msg), nil
	}

	// Handle 404 Not Found
	if code == http.StatusNotFound {
		msg := fmt.Sprintf("%s not found or access denied (%d)", resourceDescription, code)
		return mcp.NewToolResultError(msg), nil
	}

	// Handle 400 Bad Request / 422 Unprocessable Entity
	if code == http.StatusBadRequest || code == http.StatusUnprocessableEntity {
		msg := fmt.Sprintf("failed to %s: %v (status: %d)", operation, err, code)
		return mcp.NewToolResultError(msg), nil
	}

	// For all other errors, return as internal error
	return nil, fmt.Errorf("failed to %s %s: %w (status: %d)", operation, resourceDescription, err, code)
}

// HandleGraphQLError provides error handling for GraphQL queries
func HandleGraphQLError(err error, resp *gl.Response, resourceDescription string) (*mcp.CallToolResult, error) {
	if err == nil {
		return nil, nil
	}

	// Check if it's a GraphQL response error
	// GraphQL errors may be returned even when HTTP status is 200
	if resp != nil && resp.Response != nil {
		code := resp.Response.StatusCode

		// Handle 401 Unauthorized - token expired or invalid
		if code == http.StatusUnauthorized {
			msg := fmt.Sprintf("Authentication failed (401). Your GitLab token may be expired. Please update it using the updateToken tool.")
			return mcp.NewToolResultError(msg), nil
		}

		// Handle 404 Not Found
		if code == http.StatusNotFound {
			msg := fmt.Sprintf("%s not found or access denied (404)", resourceDescription)
			return mcp.NewToolResultError(msg), nil
		}

		// Handle 400 Bad Request / 422 Unprocessable Entity
		if code == http.StatusBadRequest || code == http.StatusUnprocessableEntity {
			msg := fmt.Sprintf("failed to process %s: %v (status: %d)", resourceDescription, err, code)
			return mcp.NewToolResultError(msg), nil
		}
	}

	// For all other errors, return as internal error
	return nil, fmt.Errorf("failed to process %s: %w", resourceDescription, err)
}
