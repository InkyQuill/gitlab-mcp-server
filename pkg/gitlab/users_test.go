package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/InkyQuill/gitlab-mcp-server/internal/toolsnaps"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gl "gitlab.com/gitlab-org/api/client-go"
	mock_gitlab "gitlab.com/gitlab-org/api/client-go/testing"
	"go.uber.org/mock/gomock"
)

func TestGetCurrentUserHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockUsers, ctrl := setupMockClientForUsers(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := GetCurrentUser(mockGetClient, nil)

	tests := []struct {
		name                string
		mockSetup           func()
		expectResultError   bool
		expectInternalError bool
		errorContains       string
	}{
		{
			name: "Success",
			mockSetup: func() {
				mockUsers.EXPECT().CurrentUser(gomock.Any()).
					Return(&gl.User{ID: 1, Username: "testuser"}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - 401",
			mockSetup:         func() { mockUsers.EXPECT().CurrentUser(gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			mockSetup:           func() { mockUsers.EXPECT().CurrentUser(gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to process current user",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()
			request := mcp.CallToolRequest{Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct { ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"` } `json:"_meta,omitempty"`
			}{Name: tool.Name}}
			result, err := handler(ctx, request)
			if tc.expectInternalError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else if tc.expectResultError {
				require.NoError(t, err)
				assert.Contains(t, getTextResult(t, result).Text, tc.errorContains)
			} else {
				require.NoError(t, err)
				var user gl.User
				json.Unmarshal([]byte(getTextResult(t, result).Text), &user)
				assert.Equal(t, "testuser", user.Username)
			}
		})
	}
}

func TestGetUserHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockUsers, ctrl := setupMockClientForUsers(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := GetUser(mockGetClient, nil)

	tests := []struct {
		name                string
		args                map[string]any
		mockSetup           func()
		expectResultError   bool
		expectInternalError bool
		errorContains       string
	}{
		{
			name: "Success",
			args: map[string]any{"userId": float64(1)},
			mockSetup: func() {
				mockUsers.EXPECT().GetUser(1, gomock.Any(), gomock.Any()).
					Return(&gl.User{ID: 1, Username: "testuser"}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing userId",
			args:              map[string]any{},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: userId",
		},
		{
			name:              "Error - 404",
			args:              map[string]any{"userId": float64(999)},
			mockSetup:         func() { mockUsers.EXPECT().GetUser(999, gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("not found")) },
			expectResultError: true,
			errorContains:     "user not found or access denied (404)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"userId": float64(1)},
			mockSetup:           func() { mockUsers.EXPECT().GetUser(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to process user",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()
			request := mcp.CallToolRequest{Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct { ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"` } `json:"_meta,omitempty"`
			}{Name: tool.Name, Arguments: tc.args}}
			result, err := handler(ctx, request)
			if tc.expectInternalError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else if tc.expectResultError {
				require.NoError(t, err)
				assert.Contains(t, getTextResult(t, result).Text, tc.errorContains)
			} else {
				require.NoError(t, err)
				var user gl.User
				json.Unmarshal([]byte(getTextResult(t, result).Text), &user)
				assert.Equal(t, 1, user.ID)
			}
		})
	}
}

func TestGetUserStatusHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockUsers, ctrl := setupMockClientForUsers(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := GetUserStatus(mockGetClient, nil)

	tests := []struct {
		name                string
		args                map[string]any
		mockSetup           func()
		expectResultError   bool
		expectInternalError bool
		errorContains       string
	}{
		{
			name: "Success",
			args: map[string]any{"userId": float64(1)},
			mockSetup: func() {
				mockUsers.EXPECT().GetUserStatus(1, gomock.Any()).
					Return(&gl.UserStatus{Message: "Working on it"}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing userId",
			args:              map[string]any{},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: userId",
		},
		{
			name:              "Error - 404",
			args:              map[string]any{"userId": float64(999)},
			mockSetup:         func() { mockUsers.EXPECT().GetUserStatus(999, gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("not found")) },
			expectResultError: true,
			errorContains:     "user status not found or access denied (404)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"userId": float64(1)},
			mockSetup:           func() { mockUsers.EXPECT().GetUserStatus(gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to process user status",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()
			request := mcp.CallToolRequest{Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct { ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"` } `json:"_meta,omitempty"`
			}{Name: tool.Name, Arguments: tc.args}}
			result, err := handler(ctx, request)
			if tc.expectInternalError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else if tc.expectResultError {
				require.NoError(t, err)
				assert.Contains(t, getTextResult(t, result).Text, tc.errorContains)
			} else {
				require.NoError(t, err)
				var status gl.UserStatus
				json.Unmarshal([]byte(getTextResult(t, result).Text), &status)
				assert.Equal(t, "Working on it", status.Message)
			}
		})
	}
}

func TestListUsersHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockUsers, ctrl := setupMockClientForUsers(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := ListUsers(mockGetClient, nil)

	tests := []struct {
		name                string
		args                map[string]any
		mockSetup           func()
		expectResultError   bool
		expectInternalError bool
		errorContains       string
	}{
		{
			name: "Success",
			args: map[string]any{},
			mockSetup: func() {
				mockUsers.EXPECT().ListUsers(gomock.Any(), gomock.Any()).
					Return([]*gl.User{{ID: 1, Username: "user1"}, {ID: 2, Username: "user2"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name: "Success - With search",
			args: map[string]any{"search": "test"},
			mockSetup: func() {
				mockUsers.EXPECT().ListUsers(gomock.Any(), gomock.Any()).
					Return([]*gl.User{{ID: 1, Username: "testuser"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - 401",
			args:              map[string]any{},
			mockSetup:         func() { mockUsers.EXPECT().ListUsers(gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{},
			mockSetup:           func() { mockUsers.EXPECT().ListUsers(gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list users",
		},
		{
			name: "Success - Long Bio Truncation",
			args: map[string]any{},
			mockSetup: func() {
				longBio := strings.Repeat("f", 500)
				user := &gl.User{
					ID:       1,
					Name:     "User with long bio",
					Username: "longbio",
					Bio:      longBio,
				}
				mockUsers.EXPECT().ListUsers(gomock.Any(), gomock.Any()).
					Return([]*gl.User{user}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()
			request := mcp.CallToolRequest{Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct { ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"` } `json:"_meta,omitempty"`
			}{Name: tool.Name, Arguments: tc.args}}
			result, err := handler(ctx, request)
			if tc.expectInternalError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else if tc.expectResultError {
				require.NoError(t, err)
				assert.Contains(t, getTextResult(t, result).Text, tc.errorContains)
			} else {
				require.NoError(t, err)
				var resp PaginatedResponse
				json.Unmarshal([]byte(getTextResult(t, result).Text), &resp)
				itemsSlice, ok := resp.Items.([]interface{})
				require.True(t, ok, "Items should be []interface{}")
				assert.NotEmpty(t, itemsSlice)
			}
		})
	}
}

func TestListProjectUsersHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, _, ctrl := setupMockClientForUsers(t)
	defer ctrl.Finish()

	// Need ProjectMembers mock too
	mockPM := mock_gitlab.NewMockProjectMembersServiceInterface(ctrl)
	mockClient.ProjectMembers = mockPM

	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := ListProjectUsers(mockGetClient, nil)

	tests := []struct {
		name                string
		args                map[string]any
		mockSetup           func()
		expectResultError   bool
		expectInternalError bool
		errorContains       string
	}{
		{
			name: "Success",
			args: map[string]any{"pid": "myproject"},
			mockSetup: func() {
				mockPM.EXPECT().ListProjectMembers("myproject", gomock.Any(), gomock.Any()).
					Return([]*gl.ProjectMember{{ID: 1, Username: "user1"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing pid",
			args:              map[string]any{},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: pid",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"pid": "myproject"},
			mockSetup:         func() { mockPM.EXPECT().ListProjectMembers(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"pid": "myproject"},
			mockSetup:           func() { mockPM.EXPECT().ListProjectMembers(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list project members",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()
			request := mcp.CallToolRequest{Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct { ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"` } `json:"_meta,omitempty"`
			}{Name: tool.Name, Arguments: tc.args}}
			result, err := handler(ctx, request)
			if tc.expectInternalError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else if tc.expectResultError {
				require.NoError(t, err)
				assert.Contains(t, getTextResult(t, result).Text, tc.errorContains)
			} else {
				require.NoError(t, err)
				var resp PaginatedResponse
				json.Unmarshal([]byte(getTextResult(t, result).Text), &resp)
				itemsSlice, ok := resp.Items.([]interface{})
				require.True(t, ok, "Items should be []interface{}")
				assert.NotEmpty(t, itemsSlice)
			}
		})
	}
}


// TestUserTools_SchemaSnapshots verifies that user tool schemas match their snapshots
func TestUserTools_SchemaSnapshots(t *testing.T) {
	tools := []struct {
		name string
		tool mcp.Tool
	}{
		{"getCurrentUser", mustMakeUserTool(GetCurrentUser)},
		{"getUser", mustMakeUserTool(GetUser)},
		{"getUserStatus", mustMakeUserTool(GetUserStatus)},
		{"listUsers", mustMakeUserTool(ListUsers)},
		{"listProjectUsers", mustMakeUserTool(ListProjectUsers)},
		{"manageUserState", mustMakeUserTool(ManageUserState)},
	}

	for _, tc := range tools {
		t.Run(tc.name, func(t *testing.T) {
			err := toolsnaps.Test(tc.name, tc.tool)
			require.NoError(t, err, "tool schema should match snapshot")
		})
	}
}

// Helper function to create tool from user tool definition function
func mustMakeUserTool(fn func(GetClientFn, map[string]string) (mcp.Tool, server.ToolHandlerFunc)) mcp.Tool {
	tool, _ := fn(nil, nil)
	return tool
}
