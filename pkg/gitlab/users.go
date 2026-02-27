package gitlab

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gl "gitlab.com/gitlab-org/api/client-go"
	"github.com/InkyQuill/gitlab-mcp-server/pkg/translations"
)

// GetCurrentUser defines the MCP tool for retrieving the currently authenticated user.
func GetCurrentUser(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"getCurrentUser",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_GET_CURRENT_USER_DESCRIPTION)),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Current GitLab User",
			ReadOnlyHint: true,
		}),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
		}

		user, resp, err := client.Users.CurrentUser(gl.WithContext(ctx))
		if err != nil {
			result, apiErr := HandleAPIError(err, resp, "current user")
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		jsonData, err := json.Marshal(user)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal user data: %w", err)
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// GetUser defines the MCP tool for retrieving a specific user by ID.
func GetUser(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"getUser",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_GET_USER_DESCRIPTION)),
		mcp.WithNumber("userId",
			mcp.Description("The ID of the user"),
			mcp.Required(),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get GitLab User by ID",
			ReadOnlyHint: true,
		}),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
		}

		userIdFloat, err := requiredParam[float64](&req, "userId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		userId := int(userIdFloat)
		if float64(userId) != userIdFloat {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: userId %v is not a valid integer", userIdFloat)), nil
		}

		user, resp, err := client.Users.GetUser(userId, gl.GetUsersOptions{}, gl.WithContext(ctx))
		if err != nil {
			result, apiErr := HandleAPIError(err, resp, "user")
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		jsonData, err := json.Marshal(user)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal user data: %w", err)
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// GetUserStatus defines the MCP tool for retrieving a user's status.
func GetUserStatus(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"getUserStatus",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_GET_USER_STATUS_DESCRIPTION)),
		mcp.WithNumber("userId",
			mcp.Description("The ID of the user"),
			mcp.Required(),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get GitLab User Status",
			ReadOnlyHint: true,
		}),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
		}

		userIdFloat, err := requiredParam[float64](&req, "userId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		userId := int(userIdFloat)
		if float64(userId) != userIdFloat {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: userId %v is not a valid integer", userIdFloat)), nil
		}

		status, resp, err := client.Users.GetUserStatus(userId, gl.WithContext(ctx))
		if err != nil {
			result, apiErr := HandleAPIError(err, resp, "user status")
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		jsonData, err := json.Marshal(status)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal user status data: %w", err)
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// ListUsers defines the MCP tool for listing users.
func ListUsers(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"listUsers",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_LIST_USERS_DESCRIPTION)),
		mcp.WithString("search",
			mcp.Description("Search users by username, name or email"),
		),
		mcp.WithNumber("per_page",
			mcp.Description("The number of results per page (default: 20, max: 100)"),
		),
		mcp.WithNumber("page",
			mcp.Description("The page number to retrieve (default: 1)"),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List GitLab Users",
			ReadOnlyHint: true,
		}),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
		}

		page, perPage, err := OptionalPaginationParams(&req)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		opts := &gl.ListUsersOptions{
			ListOptions: gl.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		}

		// Optional search parameter
		search, err := OptionalParam[string](&req, "search")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		if search != "" {
			opts.Search = &search
		}

		users, resp, err := client.Users.ListUsers(opts, gl.WithContext(ctx))
		if err != nil {
			result, apiErr := HandleListAPIError(err, resp, "users")
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		if len(users) == 0 {
			emptyResponse := &PaginatedResponse{
				Items:      []interface{}{},
				Pagination: ExtractPagination(resp),
			}
			jsonData, err := json.Marshal(emptyResponse)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal empty users response: %w", err)
			}
			return mcp.NewToolResultText(string(jsonData)), nil
		}

		// --- Optimize response (truncate + filter fields + add pagination)
		optimizer := NewResponseOptimizer("user")
		optimized, err := optimizer.OptimizeListResponse(users, resp)
		if err != nil {
			return nil, fmt.Errorf("failed to optimize users response: %w", err)
		}

		jsonData, err := json.Marshal(optimized)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal users data: %w", err)
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// ListProjectUsers defines the MCP tool for listing users in a specific project.
func ListProjectUsers(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"listProjectUsers",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_LIST_PROJECT_USERS_DESCRIPTION)),
		mcp.WithString("pid",
			mcp.Description("The ID or URL-encoded path of the project"),
			mcp.Required(),
		),
		mcp.WithNumber("page",
			mcp.Description("The page number to retrieve (default: 1)"),
		),
		mcp.WithNumber("per_page",
			mcp.Description("The number of results per page (default: 20, max: 100)"),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List GitLab Project Users",
			ReadOnlyHint: true,
		}),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
		}

		pid, err := requiredParam[string](&req, "pid")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		page, perPage, err := OptionalPaginationParams(&req)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		// Get project members as a proxy for project users
		members, resp, err := client.ProjectMembers.ListProjectMembers(pid, &gl.ListProjectMembersOptions{
			ListOptions: gl.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		}, gl.WithContext(ctx))
		if err != nil {
			result, apiErr := HandleListAPIError(err, resp, "project members")
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		if len(members) == 0 {
			emptyResponse := &PaginatedResponse{
				Items:      []interface{}{},
				Pagination: ExtractPagination(resp),
			}
			jsonData, err := json.Marshal(emptyResponse)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal empty project members response: %w", err)
			}
			return mcp.NewToolResultText(string(jsonData)), nil
		}

		// --- Optimize response (truncate + filter fields + add pagination)
		// Note: Using "user" entity type since project members are users
		optimizer := NewResponseOptimizer("user")
		optimized, err := optimizer.OptimizeListResponse(members, resp)
		if err != nil {
			return nil, fmt.Errorf("failed to optimize project members response: %w", err)
		}

		jsonData, err := json.Marshal(optimized)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal project members data: %w", err)
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// Admin tools with permission checking

// ManageUserState defines the unified MCP tool for managing user state (admin only).
// This consolidates blockUser, unblockUser, banUser, unbanUser, activateUser, deactivateUser, and approveUser.
func ManageUserState(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"manageUserState",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_MANAGE_USER_STATE_DESCRIPTION)),
		mcp.WithNumber("userId",
			mcp.Description("The ID of the user"),
			mcp.Required(),
		),
		mcp.WithString("action",
			mcp.Description("The administrative action to perform on the user"),
			mcp.Required(),
			mcp.Enum("block", "unblock", "ban", "unban", "activate", "deactivate", "approve"),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Manage GitLab User State",
			ReadOnlyHint: false,
		}),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
		}

		// Check admin permission FIRST
		currentUser, resp, err := client.Users.CurrentUser(gl.WithContext(ctx))
		if err != nil {
			result, apiErr := HandleAPIError(err, resp, "current user")
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		if !currentUser.IsAdmin {
			return mcp.NewToolResultError("Admin permission required. Your GitLab token must have admin access to perform this action."), nil
		}

		// Get userId from request
		userIdFloat, err := requiredParam[float64](&req, "userId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		userId := int(userIdFloat)
		if float64(userId) != userIdFloat {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: userId %v is not a valid integer", userIdFloat)), nil
		}

		// Get action from request
		action, err := requiredParam[string](&req, "action")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		// Execute the appropriate admin action
		var pastTense string
		var execErr error

		switch action {
		case "block":
			pastTense = "blocked"
			execErr = client.Users.BlockUser(userId, gl.WithContext(ctx))
		case "unblock":
			pastTense = "unblocked"
			execErr = client.Users.UnblockUser(userId, gl.WithContext(ctx))
		case "ban":
			pastTense = "banned"
			execErr = client.Users.BanUser(userId, gl.WithContext(ctx))
		case "unban":
			pastTense = "unbanned"
			execErr = client.Users.UnbanUser(userId, gl.WithContext(ctx))
		case "activate":
			pastTense = "activated"
			execErr = client.Users.ActivateUser(userId, gl.WithContext(ctx))
		case "deactivate":
			pastTense = "deactivated"
			execErr = client.Users.DeactivateUser(userId, gl.WithContext(ctx))
		case "approve":
			pastTense = "approved"
			execErr = client.Users.ApproveUser(userId, gl.WithContext(ctx))
		default:
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: unsupported action '%s'", action)), nil
		}

		// Handle errors
		if execErr != nil {
			if resp != nil && resp.Response != nil {
				switch resp.Response.StatusCode {
				case 401, 403:
					return mcp.NewToolResultError(fmt.Sprintf("Failed to %s user: %s (status: %d)", action, execErr.Error(), resp.Response.StatusCode)), nil
				case 404:
					return mcp.NewToolResultError(fmt.Sprintf("User not found or access denied (404)")), nil
				}
			}
			return nil, fmt.Errorf("failed to %s user: %w", action, execErr)
		}

		return mcp.NewToolResultText(fmt.Sprintf(`{"message":"User %d successfully %s"}`, userId, pastTense)), nil
	}
}
