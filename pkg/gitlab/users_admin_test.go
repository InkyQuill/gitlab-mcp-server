package gitlab

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gl "gitlab.com/gitlab-org/api/client-go"
	"go.uber.org/mock/gomock"
)

// Common error variables for testing
var (
	errUnauthorized      = errors.New("401 Unauthorized")
	errForbidden         = errors.New("403 Forbidden")
	errNotFound          = errors.New("404 Not Found")
	errInternalServer    = errors.New("500 Internal Server Error")
	errBadGateway        = errors.New("502 Bad Gateway")
	errServiceUnavailable = errors.New("503 Service Unavailable")
	errNetwork           = errors.New("network error")
)

// Admin tools tests with permission checking

func TestBlockUserHandler_AdminCheck(t *testing.T) {
	ctx := context.Background()
	mockClient, mockUsers, ctrl := setupMockClientForUsers(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := BlockUser(mockGetClient, nil)

	t.Run("Success - Admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: true}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
		mockUsers.EXPECT().BlockUser(1, gomock.Any()).Return(nil)

		request := mcp.CallToolRequest{Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct { ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"` } `json:"_meta,omitempty"`
		}{Name: tool.Name, Arguments: map[string]any{"userId": float64(1)}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "successfully blocked")
	})

	t.Run("Error - Non-admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: false}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)

		request := mcp.CallToolRequest{Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct { ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"` } `json:"_meta,omitempty"`
		}{Name: tool.Name, Arguments: map[string]any{"userId": float64(1)}}}
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
	tool, handler := UnblockUser(mockGetClient, nil)

	t.Run("Success - Admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: true}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
		mockUsers.EXPECT().UnblockUser(1, gomock.Any()).Return(nil)

		request := mcp.CallToolRequest{Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct { ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"` } `json:"_meta,omitempty"`
		}{Name: tool.Name, Arguments: map[string]any{"userId": float64(1)}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "successfully unblocked")
	})

	t.Run("Error - Non-admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: false}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)

		request := mcp.CallToolRequest{Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct { ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"` } `json:"_meta,omitempty"`
		}{Name: tool.Name, Arguments: map[string]any{"userId": float64(1)}}}
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
	tool, handler := BanUser(mockGetClient, nil)

	t.Run("Success - Admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: true}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
		mockUsers.EXPECT().BanUser(1, gomock.Any()).Return(nil)

		request := mcp.CallToolRequest{Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct { ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"` } `json:"_meta,omitempty"`
		}{Name: tool.Name, Arguments: map[string]any{"userId": float64(1)}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "successfully banned")
	})

	t.Run("Error - Non-admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: false}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)

		request := mcp.CallToolRequest{Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct { ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"` } `json:"_meta,omitempty"`
		}{Name: tool.Name, Arguments: map[string]any{"userId": float64(1)}}}
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
	tool, handler := UnbanUser(mockGetClient, nil)

	t.Run("Success - Admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: true}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
		mockUsers.EXPECT().UnbanUser(1, gomock.Any()).Return(nil)

		request := mcp.CallToolRequest{Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct { ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"` } `json:"_meta,omitempty"`
		}{Name: tool.Name, Arguments: map[string]any{"userId": float64(1)}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "successfully unbanned")
	})

	t.Run("Error - Non-admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: false}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)

		request := mcp.CallToolRequest{Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct { ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"` } `json:"_meta,omitempty"`
		}{Name: tool.Name, Arguments: map[string]any{"userId": float64(1)}}}
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
	tool, handler := ActivateUser(mockGetClient, nil)

	t.Run("Success - Admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: true}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
		mockUsers.EXPECT().ActivateUser(1, gomock.Any()).Return(nil)

		request := mcp.CallToolRequest{Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct { ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"` } `json:"_meta,omitempty"`
		}{Name: tool.Name, Arguments: map[string]any{"userId": float64(1)}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "successfully activated")
	})

	t.Run("Error - Non-admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: false}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)

		request := mcp.CallToolRequest{Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct { ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"` } `json:"_meta,omitempty"`
		}{Name: tool.Name, Arguments: map[string]any{"userId": float64(1)}}}
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
	tool, handler := DeactivateUser(mockGetClient, nil)

	t.Run("Success - Admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: true}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
		mockUsers.EXPECT().DeactivateUser(1, gomock.Any()).Return(nil)

		request := mcp.CallToolRequest{Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct { ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"` } `json:"_meta,omitempty"`
		}{Name: tool.Name, Arguments: map[string]any{"userId": float64(1)}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "successfully deactivated")
	})

	t.Run("Error - Non-admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: false}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)

		request := mcp.CallToolRequest{Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct { ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"` } `json:"_meta,omitempty"`
		}{Name: tool.Name, Arguments: map[string]any{"userId": float64(1)}}}
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
	tool, handler := ApproveUser(mockGetClient, nil)

	t.Run("Success - Admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: true}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
		mockUsers.EXPECT().ApproveUser(1, gomock.Any()).Return(nil)

		request := mcp.CallToolRequest{Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct { ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"` } `json:"_meta,omitempty"`
		}{Name: tool.Name, Arguments: map[string]any{"userId": float64(1)}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "successfully approved")
	})

	t.Run("Error - Non-admin", func(t *testing.T) {
		mockUsers.EXPECT().CurrentUser(gomock.Any()).
			Return(&gl.User{ID: 1, IsAdmin: false}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)

		request := mcp.CallToolRequest{Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct { ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"` } `json:"_meta,omitempty"`
		}{Name: tool.Name, Arguments: map[string]any{"userId": float64(1)}}}
		result, err := handler(ctx, request)
		require.NoError(t, err)
		assert.Contains(t, getTextResult(t, result).Text, "Admin permission required")
	})
}
