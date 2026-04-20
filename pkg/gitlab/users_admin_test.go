package gitlab

import (
	"context"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gl "gitlab.com/gitlab-org/api/client-go"
	"go.uber.org/mock/gomock"
)

// Admin tools tests with permission checking

func TestBlockUserHandler_AdminCheck(t *testing.T) {
	ctx := context.Background()
	mockClient, mockUsers, ctrl := setupMockClientForUsers(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := ManageUserState(mockGetClient, nil)

	t.Run("Success - Admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: true}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
		mockUsers.EXPECT().BlockUser(int64(1), gomock.Any()).Return(nil)

		request := mcp.CallToolRequest{Params: mcp.CallToolParams{Name: tool.Name, Arguments: map[string]any{"userId": float64(1), "action": "block"}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "successfully blocked")
	})

	t.Run("Error - Non-admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: false}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)

		request := mcp.CallToolRequest{Params: mcp.CallToolParams{Name: tool.Name, Arguments: map[string]any{"userId": float64(1), "action": "block"}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "Admin permission required")
	})
}

func TestUnblockUserHandler_AdminCheck(t *testing.T) {
	ctx := context.Background()
	mockClient, mockUsers, ctrl := setupMockClientForUsers(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := ManageUserState(mockGetClient, nil)

	t.Run("Success - Admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: true}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
		mockUsers.EXPECT().UnblockUser(int64(1), gomock.Any()).Return(nil)

		request := mcp.CallToolRequest{Params: mcp.CallToolParams{Name: tool.Name, Arguments: map[string]any{"userId": float64(1), "action": "unblock"}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "successfully unblocked")
	})

	t.Run("Error - Non-admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: false}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)

		request := mcp.CallToolRequest{Params: mcp.CallToolParams{Name: tool.Name, Arguments: map[string]any{"userId": float64(1), "action": "unblock"}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "Admin permission required")
	})
}

func TestBanUserHandler_AdminCheck(t *testing.T) {
	ctx := context.Background()
	mockClient, mockUsers, ctrl := setupMockClientForUsers(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := ManageUserState(mockGetClient, nil)

	t.Run("Success - Admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: true}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
		mockUsers.EXPECT().BanUser(int64(1), gomock.Any()).Return(nil)

		request := mcp.CallToolRequest{Params: mcp.CallToolParams{Name: tool.Name, Arguments: map[string]any{"userId": float64(1), "action": "ban"}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "successfully banned")
	})

	t.Run("Error - Non-admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: false}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)

		request := mcp.CallToolRequest{Params: mcp.CallToolParams{Name: tool.Name, Arguments: map[string]any{"userId": float64(1), "action": "ban"}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "Admin permission required")
	})
}

func TestUnbanUserHandler_AdminCheck(t *testing.T) {
	ctx := context.Background()
	mockClient, mockUsers, ctrl := setupMockClientForUsers(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := ManageUserState(mockGetClient, nil)

	t.Run("Success - Admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: true}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
		mockUsers.EXPECT().UnbanUser(int64(1), gomock.Any()).Return(nil)

		request := mcp.CallToolRequest{Params: mcp.CallToolParams{Name: tool.Name, Arguments: map[string]any{"userId": float64(1), "action": "unban"}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "successfully unbanned")
	})

	t.Run("Error - Non-admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: false}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)

		request := mcp.CallToolRequest{Params: mcp.CallToolParams{Name: tool.Name, Arguments: map[string]any{"userId": float64(1), "action": "unban"}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "Admin permission required")
	})
}

func TestActivateUserHandler_AdminCheck(t *testing.T) {
	ctx := context.Background()
	mockClient, mockUsers, ctrl := setupMockClientForUsers(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := ManageUserState(mockGetClient, nil)

	t.Run("Success - Admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: true}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
		mockUsers.EXPECT().ActivateUser(int64(1), gomock.Any()).Return(nil)

		request := mcp.CallToolRequest{Params: mcp.CallToolParams{Name: tool.Name, Arguments: map[string]any{"userId": float64(1), "action": "activate"}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "successfully activated")
	})

	t.Run("Error - Non-admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: false}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)

		request := mcp.CallToolRequest{Params: mcp.CallToolParams{Name: tool.Name, Arguments: map[string]any{"userId": float64(1), "action": "activate"}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "Admin permission required")
	})
}

func TestDeactivateUserHandler_AdminCheck(t *testing.T) {
	ctx := context.Background()
	mockClient, mockUsers, ctrl := setupMockClientForUsers(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := ManageUserState(mockGetClient, nil)

	t.Run("Success - Admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: true}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
		mockUsers.EXPECT().DeactivateUser(int64(1), gomock.Any()).Return(nil)

		request := mcp.CallToolRequest{Params: mcp.CallToolParams{Name: tool.Name, Arguments: map[string]any{"userId": float64(1), "action": "deactivate"}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "successfully deactivated")
	})

	t.Run("Error - Non-admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: false}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)

		request := mcp.CallToolRequest{Params: mcp.CallToolParams{Name: tool.Name, Arguments: map[string]any{"userId": float64(1), "action": "deactivate"}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "Admin permission required")
	})
}

func TestApproveUserHandler_AdminCheck(t *testing.T) {
	ctx := context.Background()
	mockClient, mockUsers, ctrl := setupMockClientForUsers(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := ManageUserState(mockGetClient, nil)

	t.Run("Success - Admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: true}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
		mockUsers.EXPECT().ApproveUser(int64(1), gomock.Any()).Return(nil)

		request := mcp.CallToolRequest{Params: mcp.CallToolParams{Name: tool.Name, Arguments: map[string]any{"userId": float64(1), "action": "approve"}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "successfully approved")
	})

	t.Run("Error - Non-admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: false}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)

		request := mcp.CallToolRequest{Params: mcp.CallToolParams{Name: tool.Name, Arguments: map[string]any{"userId": float64(1), "action": "approve"}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "Admin permission required")
	})
}
