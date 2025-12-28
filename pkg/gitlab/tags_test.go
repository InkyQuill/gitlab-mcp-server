package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/mark3labs/mcp-go/mcp"
	gl "gitlab.com/gitlab-org/api/client-go"

	"github.com/InkyQuill/gitlab-mcp-server/internal/toolsnaps"
)

// TestListRepositoryTagsHandler tests the listRepositoryTags tool
func TestListRepositoryTagsHandler(t *testing.T) {
	// Tool schema snapshot test
	listRepositoryTagsTool, _ := ListRepositoryTags(nil, nil)
	require.NoError(t, toolsnaps.Test(listRepositoryTagsTool.Name, listRepositoryTagsTool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockTags, ctrl := setupMockClientForTags(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	listRepositoryTagsTool, listRepositoryTagsHandler := ListRepositoryTags(mockGetClient, nil)

	projectID := "group/project"
	searchQuery := "v1.0"

	createTag := func(name string, message string) *gl.Tag {
		return &gl.Tag{
			Name: name,
			Target: fmt.Sprintf("commit-sha-for-%s", name),
			Message: message,
			Commit: &gl.Commit{
				ID: fmt.Sprintf("commit-sha-for-%s", name),
			},
		}
	}

	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectedResult     []*gl.Tag
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - List Tags - No Filters",
			inputArgs: map[string]any{
				"projectId": projectID,
			},
			mockSetup: func() {
				mockTags.EXPECT().
					ListTags(projectID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ interface{}, opts *gl.ListTagsOptions, _ ...gl.RequestOptionFunc) ([]*gl.Tag, *gl.Response, error) {
						assert.Nil(t, opts.Search)
						assert.Equal(t, 1, opts.Page)
						return []*gl.Tag{
							createTag("v1.0.0", "Release v1.0.0"),
							createTag("v1.1.0", "Release v1.1.0"),
						}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Tag{
				createTag("v1.0.0", "Release v1.0.0"),
				createTag("v1.1.0", "Release v1.1.0"),
			},
		},
		{
			name: "Success - List Tags - With Search",
			inputArgs: map[string]any{
				"projectId": projectID,
				"search":    searchQuery,
			},
			mockSetup: func() {
				mockTags.EXPECT().
					ListTags(projectID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ interface{}, opts *gl.ListTagsOptions, _ ...gl.RequestOptionFunc) ([]*gl.Tag, *gl.Response, error) {
						require.NotNil(t, opts.Search)
						assert.Equal(t, searchQuery, *opts.Search)
						return []*gl.Tag{createTag("v1.0.0", "Release v1.0.0")}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Tag{createTag("v1.0.0", "Release v1.0.0")},
		},
		{
			name: "Success - List Tags - With Pagination",
			inputArgs: map[string]any{
				"projectId": projectID,
				"page":      2,
				"per_page":  1,
			},
			mockSetup: func() {
				mockTags.EXPECT().
					ListTags(projectID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ interface{}, opts *gl.ListTagsOptions, _ ...gl.RequestOptionFunc) ([]*gl.Tag, *gl.Response, error) {
						assert.Equal(t, 2, opts.Page)
						assert.Equal(t, 1, opts.PerPage)
						return []*gl.Tag{createTag("v2.0.0", "Release v2.0.0")}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Tag{createTag("v2.0.0", "Release v2.0.0")},
		},
		{
			name: "Success - Empty List",
			inputArgs: map[string]any{
				"projectId": projectID,
				"search":    "no-match-tag",
			},
			mockSetup: func() {
				mockTags.EXPECT().
					ListTags(projectID, gomock.Any(), gomock.Any()).
					Return([]*gl.Tag{}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: []*gl.Tag{},
		},
		{
			name: "Error - Project Not Found (404)",
			inputArgs: map[string]any{
				"projectId": "nonexistent/project",
			},
			mockSetup: func() {
				mockTags.EXPECT().
					ListTags("nonexistent/project", gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("404 Not Found"))
			},
			expectHandlerError: true,
			errorContains:      "failed to list tags",
		},
		{
			name: "Error - GitLab API Error (500)",
			inputArgs: map[string]any{
				"projectId": projectID,
			},
			mockSetup: func() {
				mockTags.EXPECT().
					ListTags(projectID, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("500 Internal Server Error"))
			},
			expectHandlerError: true,
			errorContains:      "failed to list tags",
		},
		{
			name:               "Error - Missing projectId",
			inputArgs:          map[string]any{"search": searchQuery},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: projectId",
		},
		{
			name: "Error - Unauthorized (401)",
			inputArgs: map[string]any{
				"projectId": projectID,
			},
			mockSetup: func() {
				mockTags.EXPECT().
					ListTags(projectID, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("401 Unauthorized"))
			},
			expectResultError: true,
			errorContains:      "Authentication failed (401)",
		},
		{
			name: "Error - Forbidden (403)",
			inputArgs: map[string]any{
				"projectId": projectID,
			},
			mockSetup: func() {
				mockTags.EXPECT().
					ListTags(projectID, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 403}}, errors.New("403 Forbidden"))
			},
			expectHandlerError: true,
			errorContains:      "failed to list tags",
		},
		{
			name: "Error - Network Error",
			inputArgs: map[string]any{
				"projectId": projectID,
			},
			mockSetup: func() {
				mockTags.EXPECT().
					ListTags(projectID, gomock.Any(), gomock.Any()).
					Return(nil, nil, errors.New("network error"))
			},
			expectHandlerError: true,
			errorContains:      "failed to list tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      listRepositoryTagsTool.Name,
					Arguments: tt.inputArgs,
				},
			}

			result, err := listRepositoryTagsHandler(ctx, request)

			if tt.expectHandlerError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				return
			}

			require.NoError(t, err)

			if tt.expectResultError {
				require.NotNil(t, result)
				textContent := getTextResult(t, result)
				assert.Contains(t, textContent.Text, tt.errorContains)
				return
			}

			require.NotNil(t, result)
			textContent := getTextResult(t, result)
			assert.NotEmpty(t, textContent.Text)

			if tt.expectedResult != nil {
				var tags []*gl.Tag
				err := json.Unmarshal([]byte(textContent.Text), &tags)
				require.NoError(t, err)
				assert.Equal(t, len(tt.expectedResult), len(tags))
			}
		})
	}
}

// TestGetRepositoryTagHandler tests the getRepositoryTag tool
func TestGetRepositoryTagHandler(t *testing.T) {
	getRepositoryTagTool, _ := GetRepositoryTag(nil, nil)
	require.NoError(t, toolsnaps.Test(getRepositoryTagTool.Name, getRepositoryTagTool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockTags, ctrl := setupMockClientForTags(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	getRepositoryTagTool, getRepositoryTagHandler := GetRepositoryTag(mockGetClient, nil)

	projectID := "group/project"
	tagName := "v1.0.0"

	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - Get Tag",
			inputArgs: map[string]any{
				"projectId": projectID,
				"tagName":   tagName,
			},
			mockSetup: func() {
				mockTags.EXPECT().
					GetTag(projectID, tagName, gomock.Any()).
					Return(&gl.Tag{
						Name:    tagName,
						Message: "Release v1.0.0",
						Target:  "abc123",
					}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name: "Error - Tag Not Found (404)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"tagName":   "nonexistent",
			},
			mockSetup: func() {
				mockTags.EXPECT().
					GetTag(projectID, "nonexistent", gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("404 Not Found"))
			},
			expectResultError: true,
			errorContains:      "not found or access denied (404)",
		},
		{
			name:               "Error - Missing tagName",
			inputArgs:          map[string]any{"projectId": projectID},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: tagName",
		},
		{
			name:               "Error - Missing projectId",
			inputArgs:          map[string]any{"tagName": tagName},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: projectId",
		},
		{
			name: "Error - Unauthorized (401)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"tagName":   tagName,
			},
			mockSetup: func() {
				mockTags.EXPECT().
					GetTag(projectID, tagName, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("401 Unauthorized"))
			},
			expectResultError: true,
			errorContains:      "Authentication failed (401)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      getRepositoryTagTool.Name,
					Arguments: tt.inputArgs,
				},
			}

			result, err := getRepositoryTagHandler(ctx, request)

			if tt.expectHandlerError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				return
			}

			require.NoError(t, err)

			if tt.expectResultError {
				require.NotNil(t, result)
				textContent := getTextResult(t, result)
				assert.Contains(t, textContent.Text, tt.errorContains)
				return
			}

			require.NotNil(t, result)
			textContent := getTextResult(t, result)
			assert.NotEmpty(t, textContent.Text)
		})
	}
}

// TestCreateRepositoryTagHandler tests the createRepositoryTag tool
func TestCreateRepositoryTagHandler(t *testing.T) {
	createRepositoryTagTool, _ := CreateRepositoryTag(nil, nil)
	require.NoError(t, toolsnaps.Test(createRepositoryTagTool.Name, createRepositoryTagTool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockTags, ctrl := setupMockClientForTags(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	createRepositoryTagTool, createRepositoryTagHandler := CreateRepositoryTag(mockGetClient, nil)

	projectID := "group/project"
	tagName := "v1.2.0"
	ref := "main"
	message := "Release v1.2.0"

	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - Create Annotated Tag",
			inputArgs: map[string]any{
				"projectId": projectID,
				"tagName":   tagName,
				"ref":       ref,
				"message":   message,
			},
			mockSetup: func() {
				mockTags.EXPECT().
					CreateTag(projectID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ interface{}, opts *gl.CreateTagOptions, _ ...gl.RequestOptionFunc) (*gl.Tag, *gl.Response, error) {
						assert.Equal(t, tagName, *opts.TagName)
						assert.Equal(t, ref, *opts.Ref)
						assert.Equal(t, message, *opts.Message)
						return &gl.Tag{Name: tagName, Message: message}, &gl.Response{Response: &http.Response{StatusCode: 201}}, nil
					})
			},
		},
		{
			name: "Success - Create Lightweight Tag",
			inputArgs: map[string]any{
				"projectId": projectID,
				"tagName":   tagName,
				"ref":       ref,
			},
			mockSetup: func() {
				mockTags.EXPECT().
					CreateTag(projectID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ interface{}, opts *gl.CreateTagOptions, _ ...gl.RequestOptionFunc) (*gl.Tag, *gl.Response, error) {
						assert.Equal(t, tagName, *opts.TagName)
						assert.Equal(t, ref, *opts.Ref)
						assert.Nil(t, opts.Message)
						return &gl.Tag{Name: tagName}, &gl.Response{Response: &http.Response{StatusCode: 201}}, nil
					})
			},
		},
		{
			name: "Error - Tag Already Exists (400)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"tagName":   tagName,
				"ref":       ref,
			},
			mockSetup: func() {
				mockTags.EXPECT().
					CreateTag(projectID, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 400}}, errors.New("tag already exists"))
			},
			expectResultError: true,
			errorContains:      "failed to create",
		},
		{
			name:               "Error - Missing tagName",
			inputArgs:          map[string]any{"projectId": projectID, "ref": ref},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: tagName",
		},
		{
			name:               "Error - Missing ref",
			inputArgs:          map[string]any{"projectId": projectID, "tagName": tagName},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: ref",
		},
		{
			name: "Error - Invalid Ref (404)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"tagName":   tagName,
				"ref":       "invalid-branch",
			},
			mockSetup: func() {
				mockTags.EXPECT().
					CreateTag(projectID, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("branch not found"))
			},
			expectResultError: true,
			errorContains:      "not found or access denied (404)",
		},
		{
			name: "Error - Unauthorized (401)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"tagName":   tagName,
				"ref":       ref,
			},
			mockSetup: func() {
				mockTags.EXPECT().
					CreateTag(projectID, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("401 Unauthorized"))
			},
			expectResultError: true,
			errorContains:      "Authentication failed (401)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      createRepositoryTagTool.Name,
					Arguments: tt.inputArgs,
				},
			}

			result, err := createRepositoryTagHandler(ctx, request)

			if tt.expectHandlerError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				return
			}

			require.NoError(t, err)

			if tt.expectResultError {
				require.NotNil(t, result)
				textContent := getTextResult(t, result)
				assert.Contains(t, textContent.Text, tt.errorContains)
				return
			}

			require.NotNil(t, result)
			textContent := getTextResult(t, result)
			assert.NotEmpty(t, textContent.Text)
		})
	}
}

// TestDeleteRepositoryTagHandler tests the deleteRepositoryTag tool
func TestDeleteRepositoryTagHandler(t *testing.T) {
	deleteRepositoryTagTool, _ := DeleteRepositoryTag(nil, nil)
	require.NoError(t, toolsnaps.Test(deleteRepositoryTagTool.Name, deleteRepositoryTagTool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockTags, ctrl := setupMockClientForTags(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	deleteRepositoryTagTool, deleteRepositoryTagHandler := DeleteRepositoryTag(mockGetClient, nil)

	projectID := "group/project"
	tagName := "v1.0.0"

	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
		successMessage     bool
	}{
		{
			name: "Success - Delete Tag",
			inputArgs: map[string]any{
				"projectId": projectID,
				"tagName":   tagName,
			},
			mockSetup: func() {
				mockTags.EXPECT().
					DeleteTag(projectID, tagName, gomock.Any()).
					Return(&gl.Response{Response: &http.Response{StatusCode: 204}}, nil)
			},
			successMessage: true,
		},
		{
			name: "Error - Tag Not Found (404)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"tagName":   "nonexistent",
			},
			mockSetup: func() {
				mockTags.EXPECT().
					DeleteTag(projectID, "nonexistent", gomock.Any()).
					Return(&gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("404 Not Found"))
			},
			expectResultError: true,
			errorContains:      "not found or access denied (404)",
		},
		{
			name:               "Error - Missing tagName",
			inputArgs:          map[string]any{"projectId": projectID},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: tagName",
		},
		{
			name: "Error - Forbidden (403)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"tagName":   tagName,
			},
			mockSetup: func() {
				mockTags.EXPECT().
					DeleteTag(projectID, tagName, gomock.Any()).
					Return(&gl.Response{Response: &http.Response{StatusCode: 403}}, errors.New("403 Forbidden"))
			},
			expectHandlerError: true,
			errorContains:      "failed to process tag",
		},
		{
			name: "Error - Unauthorized (401)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"tagName":   tagName,
			},
			mockSetup: func() {
				mockTags.EXPECT().
					DeleteTag(projectID, tagName, gomock.Any()).
					Return(&gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("401 Unauthorized"))
			},
			expectResultError: true,
			errorContains:      "Authentication failed (401)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      deleteRepositoryTagTool.Name,
					Arguments: tt.inputArgs,
				},
			}

			result, err := deleteRepositoryTagHandler(ctx, request)

			if tt.expectHandlerError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				return
			}

			require.NoError(t, err)

			if tt.expectResultError {
				require.NotNil(t, result)
				textContent := getTextResult(t, result)
				assert.Contains(t, textContent.Text, tt.errorContains)
				return
			}

			require.NotNil(t, result)
			textContent := getTextResult(t, result)
			assert.NotEmpty(t, textContent.Text)

			if tt.successMessage {
				assert.Contains(t, textContent.Text, "successfully deleted")
			}
		})
	}
}

// TestGetTagCommitHandler tests the getTagCommit tool
func TestGetTagCommitHandler(t *testing.T) {
	getTagCommitTool, _ := GetTagCommit(nil, nil)
	require.NoError(t, toolsnaps.Test(getTagCommitTool.Name, getTagCommitTool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockReleases, ctrl := setupMockClientForReleases(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	getTagCommitTool, getTagCommitHandler := GetTagCommit(mockGetClient, nil)

	projectID := "group/project"
	tagName := "v1.0.0"

	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - Get Tag Release",
			inputArgs: map[string]any{
				"projectId": projectID,
				"tagName":   tagName,
			},
			mockSetup: func() {
				mockReleases.EXPECT().
					GetRelease(projectID, tagName, gomock.Any()).
					Return(&gl.Release{
						TagName: tagName,
						Name:    "Release v1.0.0",
						Description: "First release",
					}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectHandlerError: false,
		},
		{
			name:               "Error - Missing tagName",
			inputArgs:          map[string]any{"projectId": projectID},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: tagName",
		},
		{
			name:               "Error - Missing projectId",
			inputArgs:          map[string]any{"tagName": tagName},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: projectId",
		},
		{
			name: "Error - Release Not Found (404)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"tagName":   "nonexistent",
			},
			mockSetup: func() {
				mockReleases.EXPECT().
					GetRelease(projectID, "nonexistent", gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("404 Not Found"))
			},
			expectResultError: true,
			errorContains:     "not found or access denied (404)",
		},
		{
			name: "Error - Unauthorized (401)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"tagName":   tagName,
			},
			mockSetup: func() {
				mockReleases.EXPECT().
					GetRelease(projectID, tagName, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("401 Unauthorized"))
			},
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name: "Error - Server Error (500)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"tagName":   tagName,
			},
			mockSetup: func() {
				mockReleases.EXPECT().
					GetRelease(projectID, tagName, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("500 Internal Server Error"))
			},
			expectHandlerError: true,
			errorContains:      "failed to process release",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      getTagCommitTool.Name,
					Arguments: tt.inputArgs,
				},
			}

			result, err := getTagCommitHandler(ctx, request)

			if tt.expectHandlerError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				return
			}

			require.NoError(t, err)

			if tt.expectResultError {
				require.NotNil(t, result)
				textContent := getTextResult(t, result)
				assert.Contains(t, textContent.Text, tt.errorContains)
				return
			}

			require.NotNil(t, result)
			textContent := getTextResult(t, result)
			assert.NotEmpty(t, textContent.Text)
		})
	}
}
