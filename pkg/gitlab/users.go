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
			return mcp.NewToolResultText("[]"), nil
		}

		jsonData, err := json.Marshal(users)
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
			return mcp.NewToolResultText("[]"), nil
		}

		jsonData, err := json.Marshal(members)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal project members data: %w", err)
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// Admin tools with permission checking

// BlockUser defines the MCP tool for blocking a user (admin only).
func BlockUser(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"blockUser",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_BLOCK_USER_DESCRIPTION)),
		mcp.WithNumber("userId",
			mcp.Description("The ID of the user to block"),
			mcp.Required(),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Block GitLab User",
			ReadOnlyHint: false,
		}),
	), adminUserHandler(getClient, "block", "blocked", func(client *gl.Client, userId int, ctx context.Context) error {
		return client.Users.BlockUser(userId, gl.WithContext(ctx))
	})
}

// UnblockUser defines the MCP tool for unblocking a user (admin only).
func UnblockUser(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"unblockUser",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_UNBLOCK_USER_DESCRIPTION)),
		mcp.WithNumber("userId",
			mcp.Description("The ID of the user to unblock"),
			mcp.Required(),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Unblock GitLab User",
			ReadOnlyHint: false,
		}),
	), adminUserHandler(getClient, "unblock", "unblocked", func(client *gl.Client, userId int, ctx context.Context) error {
		return client.Users.UnblockUser(userId, gl.WithContext(ctx))
	})
}

// BanUser defines the MCP tool for banning a user (admin only).
func BanUser(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"banUser",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_BAN_USER_DESCRIPTION)),
		mcp.WithNumber("userId",
			mcp.Description("The ID of the user to ban"),
			mcp.Required(),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Ban GitLab User",
			ReadOnlyHint: false,
		}),
	), adminUserHandler(getClient, "ban", "banned", func(client *gl.Client, userId int, ctx context.Context) error {
		return client.Users.BanUser(userId, gl.WithContext(ctx))
	})
}

// UnbanUser defines the MCP tool for unbanning a user (admin only).
func UnbanUser(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"unbanUser",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_UNBAN_USER_DESCRIPTION)),
		mcp.WithNumber("userId",
			mcp.Description("The ID of the user to unban"),
			mcp.Required(),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Unban GitLab User",
			ReadOnlyHint: false,
		}),
	), adminUserHandler(getClient, "unban", "unbanned", func(client *gl.Client, userId int, ctx context.Context) error {
		return client.Users.UnbanUser(userId, gl.WithContext(ctx))
	})
}

// ActivateUser defines the MCP tool for activating a user (admin only).
func ActivateUser(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"activateUser",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_ACTIVATE_USER_DESCRIPTION)),
		mcp.WithNumber("userId",
			mcp.Description("The ID of the user to activate"),
			mcp.Required(),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Activate GitLab User",
			ReadOnlyHint: false,
		}),
	), adminUserHandler(getClient, "activate", "activated", func(client *gl.Client, userId int, ctx context.Context) error {
		return client.Users.ActivateUser(userId, gl.WithContext(ctx))
	})
}

// DeactivateUser defines the MCP tool for deactivating a user (admin only).
func DeactivateUser(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"deactivateUser",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_DEACTIVATE_USER_DESCRIPTION)),
		mcp.WithNumber("userId",
			mcp.Description("The ID of the user to deactivate"),
			mcp.Required(),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Deactivate GitLab User",
			ReadOnlyHint: false,
		}),
	), adminUserHandler(getClient, "deactivate", "deactivated", func(client *gl.Client, userId int, ctx context.Context) error {
		return client.Users.DeactivateUser(userId, gl.WithContext(ctx))
	})
}

// ApproveUser defines the MCP tool for approving a user (admin only).
func ApproveUser(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"approveUser",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_APPROVE_USER_DESCRIPTION)),
		mcp.WithNumber("userId",
			mcp.Description("The ID of the user to approve"),
			mcp.Required(),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Approve GitLab User",
			ReadOnlyHint: false,
		}),
	), adminUserHandler(getClient, "approve", "approved", func(client *gl.Client, userId int, ctx context.Context) error {
		return client.Users.ApproveUser(userId, gl.WithContext(ctx))
	})
}

// Helper function for admin user operations with permission checking
type adminUserAction func(client *gl.Client, userId int, ctx context.Context) error

func adminUserHandler(getClient GetClientFn, action string, pastTense string, exec adminUserAction) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		// Execute the admin action
		err = exec(client, userId, ctx)
		if err != nil {
			// Most admin actions return Response, but the API returns error
			// Check if it's a known error type
			if resp != nil && resp.Response != nil {
				switch resp.Response.StatusCode {
				case 401, 403:
					return mcp.NewToolResultError(fmt.Sprintf("Failed to %s user: %s (status: %d)", action, err.Error(), resp.Response.StatusCode)), nil
				case 404:
					return mcp.NewToolResultError(fmt.Sprintf("User not found or access denied (404)")), nil
				}
			}
			return nil, fmt.Errorf("failed to %s user: %w", action, err)
		}

		return mcp.NewToolResultText(fmt.Sprintf(`{"message":"User %d successfully %s"}`, userId, pastTense)), nil
	}
}
