package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/InkyQuill/gitlab-mcp-server/internal/toolsnaps"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gl "gitlab.com/gitlab-org/api/client-go"
	"go.uber.org/mock/gomock"
)

func TestSearchProjectsHandler(t *testing.T) {
	ctx := context.Background()

	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	searchProjectsTool, handler := Search(mockGetClient, nil)

	tests := []struct {
		name                string
		args                map[string]any
		mockSetup           func()
		expectedResult      interface{}
		expectResultError   bool
		expectInternalError bool
		errorContains       string
	}{
		{
			name: "Success - Search Projects",
			args: map[string]any{
				"resourceType": "projects",
				"search":       "test",
			},
			mockSetup: func() {
				expectedProjects := []*gl.Project{
					{ID: 1, Name: "Test Project 1"},
					{ID: 2, Name: "Test Project 2"},
				}
				mockSearch.EXPECT().
					Projects("test", gomock.Any(), gomock.Any()).
					Return(expectedProjects, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: []*gl.Project{
				{ID: 1, Name: "Test Project 1"},
				{ID: 2, Name: "Test Project 2"},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Search Projects - Empty Results",
			args: map[string]any{
				"resourceType": "projects",
				"search":       "nonexistent",
			},
			mockSetup: func() {
				mockSearch.EXPECT().
					Projects("nonexistent", gomock.Any(), gomock.Any()).
					Return([]*gl.Project{}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult:      "[]",
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name:                "Error - Missing search parameter",
			args:                map[string]any{"resourceType": "projects"},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: search",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - API Error (500)",
			args: map[string]any{
				"resourceType": "projects",
				"search":       "test",
			},
			mockSetup: func() {
				mockSearch.EXPECT().
					Projects("test", gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("internal error"))
			},
			expectedResult:      nil,
			expectResultError:   true,
			expectInternalError: true,
			errorContains:       "failed to list projects",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      searchProjectsTool.Name,
					Arguments: tc.args,
				},
			}

			result, err := handler(ctx, request)

			if tc.expectInternalError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.ErrorContains(t, err, tc.errorContains)
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				textContent := getTextResult(t, result)

				if tc.expectResultError {
					expectedErrString, ok := tc.expectedResult.(string)
					require.True(t, ok)
					assert.Contains(t, textContent.Text, expectedErrString)
				} else {
					if expectedStr, ok := tc.expectedResult.(string); ok {
						assert.Equal(t, expectedStr, textContent.Text)
					} else {
						expectedProjects, ok := tc.expectedResult.([]*gl.Project)
						require.True(t, ok)
						expectedJSON, err := json.Marshal(expectedProjects)
						require.NoError(t, err)
						assert.JSONEq(t, string(expectedJSON), textContent.Text)
					}
				}
			}
		})
	}
}

func TestSearchIssuesHandler(t *testing.T) {
	ctx := context.Background()

	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	searchIssuesTool, handler := Search(mockGetClient, nil)

	tests := []struct {
		name                string
		args                map[string]any
		mockSetup           func()
		expectedResult      interface{}
		expectResultError   bool
		expectInternalError bool
		errorContains       string
	}{
		{
			name: "Success - Search Issues",
			args: map[string]any{
				"resourceType": "issues",
				"search":       "bug",
			},
			mockSetup: func() {
				expectedIssues := []*gl.Issue{
					{ID: 1, IID: 1, Title: "Bug fix"},
					{ID: 2, IID: 2, Title: "Another bug"},
				}
				mockSearch.EXPECT().
					Issues("bug", gomock.Any(), gomock.Any()).
					Return(expectedIssues, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: []*gl.Issue{
				{ID: 1, IID: 1, Title: "Bug fix"},
				{ID: 2, IID: 2, Title: "Another bug"},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Search Issues - Empty Results",
			args: map[string]any{
				"resourceType": "issues",
				"search":       "nonexistent",
			},
			mockSetup: func() {
				mockSearch.EXPECT().
					Issues("nonexistent", gomock.Any(), gomock.Any()).
					Return([]*gl.Issue{}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult:      "[]",
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name:                "Error - Missing search parameter",
			args:                map[string]any{"resourceType": "issues"},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: search",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - API Error (500)",
			args: map[string]any{
				"resourceType": "issues",
				"search":       "bug",
			},
			mockSetup: func() {
				mockSearch.EXPECT().
					Issues("bug", gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("internal error"))
			},
			expectedResult:      nil,
			expectResultError:   true,
			expectInternalError: true,
			errorContains:       "failed to list issues",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      searchIssuesTool.Name,
					Arguments: tc.args,
				},
			}

			result, err := handler(ctx, request)

			if tc.expectInternalError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.ErrorContains(t, err, tc.errorContains)
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				textContent := getTextResult(t, result)

				if tc.expectResultError {
					expectedErrString, ok := tc.expectedResult.(string)
					require.True(t, ok)
					assert.Contains(t, textContent.Text, expectedErrString)
				} else {
					if expectedStr, ok := tc.expectedResult.(string); ok {
						assert.Equal(t, expectedStr, textContent.Text)
					} else {
						expectedIssues, ok := tc.expectedResult.([]*gl.Issue)
						require.True(t, ok)
						expectedJSON, err := json.Marshal(expectedIssues)
						require.NoError(t, err)
						assert.JSONEq(t, string(expectedJSON), textContent.Text)
					}
				}
			}
		})
	}
}

func TestSearchMergeRequestsHandler(t *testing.T) {
	ctx := context.Background()

	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	searchMRsTool, handler := Search(mockGetClient, nil)

	tests := []struct {
		name                string
		args                map[string]any
		mockSetup           func()
		expectedResult      interface{}
		expectResultError   bool
		expectInternalError bool
		errorContains       string
	}{
		{
			name: "Success - Search Merge Requests",
			args: map[string]any{
				"resourceType": "merge_requests",
				"search":       "feature",
			},
			mockSetup: func() {
				expectedMRs := []*gl.MergeRequest{
					{BasicMergeRequest: gl.BasicMergeRequest{ID: 1, IID: 1, Title: "Feature branch"}},
					{BasicMergeRequest: gl.BasicMergeRequest{ID: 2, IID: 2, Title: "Another feature"}},
				}
				mockSearch.EXPECT().
					MergeRequests("feature", gomock.Any(), gomock.Any()).
					Return(expectedMRs, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: []*gl.MergeRequest{
				{BasicMergeRequest: gl.BasicMergeRequest{ID: 1, IID: 1, Title: "Feature branch"}},
				{BasicMergeRequest: gl.BasicMergeRequest{ID: 2, IID: 2, Title: "Another feature"}},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name:                "Error - Missing search parameter",
			args:                map[string]any{"resourceType": "merge_requests"},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: search",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - API Error (500)",
			args: map[string]any{
				"resourceType": "merge_requests",
				"search":       "feature",
			},
			mockSetup: func() {
				mockSearch.EXPECT().
					MergeRequests("feature", gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("internal error"))
			},
			expectedResult:      nil,
			expectResultError:   true,
			expectInternalError: true,
			errorContains:       "failed to list merge_requests (scope=global)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      searchMRsTool.Name,
					Arguments: tc.args,
				},
			}

			result, err := handler(ctx, request)

			if tc.expectInternalError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.ErrorContains(t, err, tc.errorContains)
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				textContent := getTextResult(t, result)

				if tc.expectResultError {
					expectedErrString, ok := tc.expectedResult.(string)
					require.True(t, ok)
					assert.Contains(t, textContent.Text, expectedErrString)
				} else {
					expectedMRs, ok := tc.expectedResult.([]*gl.MergeRequest)
					require.True(t, ok)
					expectedJSON, err := json.Marshal(expectedMRs)
					require.NoError(t, err)
					assert.JSONEq(t, string(expectedJSON), textContent.Text)
				}
			}
		})
	}
}

func TestSearchBlobsHandler(t *testing.T) {
	ctx := context.Background()

	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	searchBlobsTool, handler := Search(mockGetClient, nil)

	tests := []struct {
		name                string
		args                map[string]any
		mockSetup           func()
		expectedResult      interface{}
		expectResultError   bool
		expectInternalError bool
		errorContains       string
	}{
		{
			name: "Success - Search Code Blobs",
			args: map[string]any{
				"resourceType": "blobs",
				"search":       "function",
			},
			mockSetup: func() {
				expectedBlobs := []*gl.Blob{
					{Filename: "main.go", Basename: "main.go"},
					{ID: "abc123", Data: "function test() {}"},
				}
				mockSearch.EXPECT().
					Blobs("function", gomock.Any(), gomock.Any()).
					Return(expectedBlobs, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: []*gl.Blob{
				{Filename: "main.go", Basename: "main.go"},
				{ID: "abc123", Data: "function test() {}"},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name:                "Error - Missing search parameter",
			args:                map[string]any{"resourceType": "blobs"},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: search",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - API Error (500)",
			args: map[string]any{
				"resourceType": "blobs",
				"search":       "function",
			},
			mockSetup: func() {
				mockSearch.EXPECT().
					Blobs("function", gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("internal error"))
			},
			expectedResult:      nil,
			expectResultError:   true,
			expectInternalError: true,
			errorContains:       "failed to list blobs (scope=global)",
		},
		{
			name: "Success - Long Data Truncation",
			args: map[string]any{
				"resourceType": "blobs",
				"search":       "long content",
			},
			mockSetup: func() {
				longData := strings.Repeat("e", 500)
				blob := &gl.Blob{
					Basename: "large_file.go",
					Ref:      "main",
					Path:     "pkg/large_file.go",
					Data:     longData,
				}
				mockSearch.EXPECT().
					Blobs("long content", gomock.Any(), gomock.Any()).
					Return([]*gl.Blob{blob}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: []*gl.Blob{
				{
					Basename: "large_file.go",
					Ref:      "main",
					Path:     "pkg/large_file.go",
					Data:     strings.Repeat("e", 300) + "...",
				},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      searchBlobsTool.Name,
					Arguments: tc.args,
				},
			}

			result, err := handler(ctx, request)

			if tc.expectInternalError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.ErrorContains(t, err, tc.errorContains)
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				textContent := getTextResult(t, result)

				if tc.expectResultError {
					expectedErrString, ok := tc.expectedResult.(string)
					require.True(t, ok)
					assert.Contains(t, textContent.Text, expectedErrString)
				} else {
					expectedBlobs, ok := tc.expectedResult.([]*gl.Blob)
					require.True(t, ok)
					expectedJSON, err := json.Marshal(expectedBlobs)
					require.NoError(t, err)
					assert.JSONEq(t, string(expectedJSON), textContent.Text)
				}
			}
		})
	}
}

func TestSearchCommitsHandler(t *testing.T) {
	ctx := context.Background()

	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	searchCommitsTool, handler := Search(mockGetClient, nil)

	tests := []struct {
		name                string
		args                map[string]any
		mockSetup           func()
		expectedResult      interface{}
		expectResultError   bool
		expectInternalError bool
		errorContains       string
	}{
		{
			name: "Success - Search Commits",
			args: map[string]any{
				"resourceType": "commits",
				"search":       "fix",
			},
			mockSetup: func() {
				expectedCommits := []*gl.Commit{
					{ID: "abc123", Message: "Fix bug"},
					{ID: "def456", Message: "Fix another bug"},
				}
				mockSearch.EXPECT().
					Commits("fix", gomock.Any(), gomock.Any()).
					Return(expectedCommits, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: []*gl.Commit{
				{ID: "abc123", Message: "Fix bug"},
				{ID: "def456", Message: "Fix another bug"},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name:                "Error - Missing search parameter",
			args:                map[string]any{"resourceType": "commits"},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: search",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - API Error (500)",
			args: map[string]any{
				"resourceType": "commits",
				"search":       "fix",
			},
			mockSetup: func() {
				mockSearch.EXPECT().
					Commits("fix", gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("internal error"))
			},
			expectedResult:      nil,
			expectResultError:   true,
			expectInternalError: true,
			errorContains:       "failed to list commits",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      searchCommitsTool.Name,
					Arguments: tc.args,
				},
			}

			result, err := handler(ctx, request)

			if tc.expectInternalError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.ErrorContains(t, err, tc.errorContains)
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				textContent := getTextResult(t, result)

				if tc.expectResultError {
					expectedErrString, ok := tc.expectedResult.(string)
					require.True(t, ok)
					assert.Contains(t, textContent.Text, expectedErrString)
				} else {
					expectedCommits, ok := tc.expectedResult.([]*gl.Commit)
					require.True(t, ok)
					expectedJSON, err := json.Marshal(expectedCommits)
					require.NoError(t, err)
					assert.JSONEq(t, string(expectedJSON), textContent.Text)
				}
			}
		})
	}
}

// TestClientInitError tests client initialization error for all search tools
func TestSearchTools_ClientInitError(t *testing.T) {
	ctx := context.Background()

	errorGetClientFn := func(_ context.Context) (*gl.Client, error) {
		return nil, fmt.Errorf("mock init error")
	}

	t.Run("searchProjects", func(t *testing.T) {
		_, handler := Search(errorGetClientFn, nil)
		tool, _ := Search(nil, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      tool.Name,
				Arguments: map[string]any{"resourceType": "projects", "search": "test"},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to initialize GitLab client: mock init error")
		assert.Nil(t, result)
	})

	t.Run("searchIssues", func(t *testing.T) {
		_, handler := Search(errorGetClientFn, nil)
		tool, _ := Search(nil, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      tool.Name,
				Arguments: map[string]any{"resourceType": "issues", "search": "test"},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to initialize GitLab client: mock init error")
		assert.Nil(t, result)
	})

	t.Run("searchMergeRequests", func(t *testing.T) {
		_, handler := Search(errorGetClientFn, nil)
		tool, _ := Search(nil, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      tool.Name,
				Arguments: map[string]any{"resourceType": "merge_requests", "search": "test"},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to initialize GitLab client: mock init error")
		assert.Nil(t, result)
	})

	t.Run("searchBlobs", func(t *testing.T) {
		_, handler := Search(errorGetClientFn, nil)
		tool, _ := Search(nil, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      tool.Name,
				Arguments: map[string]any{"resourceType": "blobs", "search": "test"},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to initialize GitLab client: mock init error")
		assert.Nil(t, result)
	})

	t.Run("searchCommits", func(t *testing.T) {
		_, handler := Search(errorGetClientFn, nil)
		tool, _ := Search(nil, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      tool.Name,
				Arguments: map[string]any{"resourceType": "commits", "search": "test"},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to initialize GitLab client: mock init error")
		assert.Nil(t, result)
	})
}

// Tests for scoped search tools

func TestSearchProjectsByGroupHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	tool, handler := Search(mockGetClient, nil)

	tests := []struct {
		name                string
		args                map[string]any
		mockSetup           func()
		expectedResult      interface{}
		expectResultError   bool
		expectInternalError bool
		errorContains       string
	}{
		{
			name: "Success - Search Projects by Group",
			args: map[string]any{
				"resourceType": "projects",
				"scope":        "group",
				"gid":          "mygroup",
				"search":       "test",
			},
			mockSetup: func() {
				mockSearch.EXPECT().
					ProjectsByGroup("mygroup", "test", gomock.Any(), gomock.Any()).
					Return([]*gl.Project{{ID: 1, Name: "Test"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult:      []*gl.Project{{ID: 1, Name: "Test"}},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name:                "Error - Missing gid parameter",
			args:                map[string]any{"resourceType": "projects", "scope": "group", "search": "test"},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: gid is required when scope='group'",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name:                "Error - Missing search parameter",
			args:                map[string]any{"resourceType": "projects", "scope": "group", "gid": "mygroup"},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: search",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - 401 Unauthorized",
			args: map[string]any{
				"resourceType": "projects",
				"scope":        "group",
				"gid":          "mygroup",
				"search":       "test",
			},
			mockSetup: func() {
				mockSearch.EXPECT().
					ProjectsByGroup("mygroup", "test", gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized"))
			},
			expectedResult:      "Authentication failed (401)",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - 500 Server Error",
			args: map[string]any{
				"resourceType": "projects",
				"scope":        "group",
				"gid":          "mygroup",
				"search":       "test",
			},
			mockSetup: func() {
				mockSearch.EXPECT().
					ProjectsByGroup("mygroup", "test", gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error"))
			},
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to list projects",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      tool.Name,
					Arguments: tc.args,
				},
			}

			result, err := handler(ctx, request)

			if tc.expectInternalError {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else if tc.expectResultError {
				require.NoError(t, err)
				require.NotNil(t, result)
				textContent := getTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedResult.(string))
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				textContent := getTextResult(t, result)

				if strResult, ok := tc.expectedResult.(string); ok {
					assert.Equal(t, strResult, textContent.Text)
				} else {
					var projects []*gl.Project
					err := json.Unmarshal([]byte(textContent.Text), &projects)
					require.NoError(t, err)
					assert.Len(t, projects, 1)
					assert.Equal(t, "Test", projects[0].Name)
				}
			}
		})
	}
}

func TestSearchIssuesByGroupHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	tool, handler := Search(mockGetClient, nil)

	tests := []struct {
		name                string
		args                map[string]any
		mockSetup           func()
		expectedResult      interface{}
		expectResultError   bool
		expectInternalError bool
		errorContains       string
	}{
		{
			name: "Success - Search Issues by Group",
			args: map[string]any{"resourceType": "issues", "scope": "group", "gid": "mygroup", "search": "bug"},
			mockSetup: func() {
				mockSearch.EXPECT().
					IssuesByGroup("mygroup", "bug", gomock.Any(), gomock.Any()).
					Return([]*gl.Issue{{ID: 1, IID: 1, Title: "Bug"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult:      []*gl.Issue{{ID: 1, IID: 1, Title: "Bug"}},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name:                "Error - Missing gid",
			args:                map[string]any{"resourceType": "issues", "scope": "group", "search": "bug"},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: gid is required when scope='group'",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name:                "Error - Missing search",
			args:                map[string]any{"resourceType": "issues", "scope": "group", "gid": "mygroup"},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: search",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - 401",
			args: map[string]any{"resourceType": "issues", "scope": "group", "gid": "mygroup", "search": "bug"},
			mockSetup: func() {
				mockSearch.EXPECT().
					IssuesByGroup("mygroup", "bug", gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized"))
			},
			expectedResult:      "Authentication failed (401)",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - 500",
			args: map[string]any{"resourceType": "issues", "scope": "group", "gid": "mygroup", "search": "bug"},
			mockSetup: func() {
				mockSearch.EXPECT().
					IssuesByGroup("mygroup", "bug", gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error"))
			},
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to list issues",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      tool.Name,
					Arguments: tc.args,
				},
			}
			result, err := handler(ctx, request)
			if tc.expectInternalError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else if tc.expectResultError {
				require.NoError(t, err)
				textContent := getTextResult(t, result)
				assert.Contains(t, textContent.Text, tc.expectedResult.(string))
			} else {
				require.NoError(t, err)
				textContent := getTextResult(t, result)
				var issues []*gl.Issue
				json.Unmarshal([]byte(textContent.Text), &issues)
				assert.Len(t, issues, 1)
				assert.Equal(t, "Bug", issues[0].Title)
			}
		})
	}
}

func TestSearchMergeRequestsByGroupHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()
	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := Search(mockGetClient, nil)

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
			args: map[string]any{"resourceType": "merge_requests", "scope": "group", "gid": "mygroup", "search": "feature"},
			mockSetup: func() {
				mockSearch.EXPECT().MergeRequestsByGroup("mygroup", "feature", gomock.Any(), gomock.Any()).
					Return([]*gl.MergeRequest{{BasicMergeRequest: gl.BasicMergeRequest{ID: 1, IID: 1, Title: "Feature"}}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:                "Error - Missing gid",
			args:                map[string]any{"resourceType": "merge_requests", "scope": "group", "search": "feature"},
			mockSetup:           func() {},
			expectResultError:   true,
			errorContains:       "gid is required when scope='group'",
		},
		{
			name:                "Error - Missing search",
			args:                map[string]any{"resourceType": "merge_requests", "scope": "group", "gid": "mygroup"},
			mockSetup:           func() {},
			expectResultError:   true,
			errorContains:       "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"resourceType": "merge_requests", "scope": "group", "gid": "mygroup", "search": "feature"},
			mockSetup:         func() { mockSearch.EXPECT().MergeRequestsByGroup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"resourceType": "merge_requests", "scope": "group", "gid": "mygroup", "search": "feature"},
			mockSetup:           func() { mockSearch.EXPECT().MergeRequestsByGroup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list merge_requests (scope=group)",
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
				var mrs []*gl.MergeRequest
				json.Unmarshal([]byte(getTextResult(t, result).Text), &mrs)
				assert.Len(t, mrs, 1)
			}
		})
	}
}

func TestSearchMilestonesByGroupHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()
	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := Search(mockGetClient, nil)

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
			args: map[string]any{"resourceType": "milestones", "scope": "group", "gid": "mygroup", "search": "v1.0"},
			mockSetup: func() {
				mockSearch.EXPECT().MilestonesByGroup("mygroup", "v1.0", gomock.Any(), gomock.Any()).
					Return([]*gl.Milestone{{ID: 1, Title: "v1.0"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing gid",
			args:              map[string]any{"resourceType": "milestones", "scope": "group", "search": "v1.0"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "gid is required when scope='group'",
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"resourceType": "milestones", "scope": "group", "gid": "mygroup"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"resourceType": "milestones", "scope": "group", "gid": "mygroup", "search": "v1.0"},
			mockSetup:         func() { mockSearch.EXPECT().MilestonesByGroup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"resourceType": "milestones", "scope": "group", "gid": "mygroup", "search": "v1.0"},
			mockSetup:           func() { mockSearch.EXPECT().MilestonesByGroup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list milestones",
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
				var ms []*gl.Milestone
				json.Unmarshal([]byte(getTextResult(t, result).Text), &ms)
				assert.Len(t, ms, 1)
			}
		})
	}
}

func TestSearchBlobsByGroupHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()
	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := Search(mockGetClient, nil)

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
			args: map[string]any{"resourceType": "blobs", "scope": "group", "gid": "mygroup", "search": "function"},
			mockSetup: func() {
				mockSearch.EXPECT().BlobsByGroup("mygroup", "function", gomock.Any(), gomock.Any()).
					Return([]*gl.Blob{{Filename: "test.go"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing gid",
			args:              map[string]any{"resourceType": "blobs", "scope": "group", "search": "function"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "gid is required when scope='group'",
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"resourceType": "blobs", "scope": "group", "gid": "mygroup"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"resourceType": "blobs", "scope": "group", "gid": "mygroup", "search": "function"},
			mockSetup:         func() { mockSearch.EXPECT().BlobsByGroup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"resourceType": "blobs", "scope": "group", "gid": "mygroup", "search": "function"},
			mockSetup:           func() { mockSearch.EXPECT().BlobsByGroup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list blobs (scope=group)",
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
				var blobs []*gl.Blob
				json.Unmarshal([]byte(getTextResult(t, result).Text), &blobs)
				assert.Len(t, blobs, 1)
			}
		})
	}
}

func TestSearchIssuesByProjectHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()
	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := Search(mockGetClient, nil)

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
			args: map[string]any{"resourceType": "issues", "scope": "project", "pid": "myproject", "search": "bug"},
			mockSetup: func() {
				mockSearch.EXPECT().IssuesByProject("myproject", "bug", gomock.Any(), gomock.Any()).
					Return([]*gl.Issue{{ID: 1, IID: 1, Title: "Bug"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing pid",
			args:              map[string]any{"resourceType": "issues", "scope": "project", "search": "bug"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "pid is required when scope='project'",
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"resourceType": "issues", "scope": "project", "pid": "myproject"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"resourceType": "issues", "scope": "project", "pid": "myproject", "search": "bug"},
			mockSetup:         func() { mockSearch.EXPECT().IssuesByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"resourceType": "issues", "scope": "project", "pid": "myproject", "search": "bug"},
			mockSetup:           func() { mockSearch.EXPECT().IssuesByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list issues",
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
				var issues []*gl.Issue
				json.Unmarshal([]byte(getTextResult(t, result).Text), &issues)
				assert.Len(t, issues, 1)
			}
		})
	}
}

func TestSearchMergeRequestsByProjectHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()
	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := Search(mockGetClient, nil)

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
			args: map[string]any{"resourceType": "merge_requests", "scope": "project", "pid": "myproject", "search": "feature"},
			mockSetup: func() {
				mockSearch.EXPECT().MergeRequestsByProject("myproject", "feature", gomock.Any(), gomock.Any()).
					Return([]*gl.MergeRequest{{BasicMergeRequest: gl.BasicMergeRequest{ID: 1, IID: 1, Title: "Feature"}}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing pid",
			args:              map[string]any{"resourceType": "merge_requests", "scope": "project", "search": "feature"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "pid is required when scope='project'",
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"resourceType": "merge_requests", "scope": "project", "pid": "myproject"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"resourceType": "merge_requests", "scope": "project", "pid": "myproject", "search": "feature"},
			mockSetup:         func() { mockSearch.EXPECT().MergeRequestsByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"resourceType": "merge_requests", "scope": "project", "pid": "myproject", "search": "feature"},
			mockSetup:           func() { mockSearch.EXPECT().MergeRequestsByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list merge_requests (scope=project)",
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
				var mrs []*gl.MergeRequest
				json.Unmarshal([]byte(getTextResult(t, result).Text), &mrs)
				assert.Len(t, mrs, 1)
			}
		})
	}
}

func TestSearchMilestonesByProjectHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()
	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := Search(mockGetClient, nil)

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
			args: map[string]any{"resourceType": "milestones", "scope": "project", "pid": "myproject", "search": "v1.0"},
			mockSetup: func() {
				mockSearch.EXPECT().MilestonesByProject("myproject", "v1.0", gomock.Any(), gomock.Any()).
					Return([]*gl.Milestone{{ID: 1, Title: "v1.0"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing pid",
			args:              map[string]any{"resourceType": "milestones", "scope": "project", "search": "v1.0"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "pid is required when scope='project'",
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"resourceType": "milestones", "scope": "project", "pid": "myproject"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"resourceType": "milestones", "scope": "project", "pid": "myproject", "search": "v1.0"},
			mockSetup:         func() { mockSearch.EXPECT().MilestonesByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"resourceType": "milestones", "scope": "project", "pid": "myproject", "search": "v1.0"},
			mockSetup:           func() { mockSearch.EXPECT().MilestonesByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list milestones",
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
				var ms []*gl.Milestone
				json.Unmarshal([]byte(getTextResult(t, result).Text), &ms)
				assert.Len(t, ms, 1)
			}
		})
	}
}

func TestSearchBlobsByProjectHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()
	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := Search(mockGetClient, nil)

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
			args: map[string]any{"resourceType": "blobs", "scope": "project", "pid": "myproject", "search": "function"},
			mockSetup: func() {
				mockSearch.EXPECT().BlobsByProject("myproject", "function", gomock.Any(), gomock.Any()).
					Return([]*gl.Blob{{Filename: "test.go"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing pid",
			args:              map[string]any{"resourceType": "blobs", "scope": "project", "search": "function"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "pid is required when scope='project'",
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"resourceType": "blobs", "scope": "project", "pid": "myproject"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"resourceType": "blobs", "scope": "project", "pid": "myproject", "search": "function"},
			mockSetup:         func() { mockSearch.EXPECT().BlobsByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"resourceType": "blobs", "scope": "project", "pid": "myproject", "search": "function"},
			mockSetup:           func() { mockSearch.EXPECT().BlobsByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list blobs (scope=project)",
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
				var blobs []*gl.Blob
				json.Unmarshal([]byte(getTextResult(t, result).Text), &blobs)
				assert.Len(t, blobs, 1)
			}
		})
	}
}

func TestSearchCommitsByProjectHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()
	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := Search(mockGetClient, nil)

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
			args: map[string]any{"resourceType": "commits", "scope": "project", "pid": "myproject", "search": "fix"},
			mockSetup: func() {
				mockSearch.EXPECT().CommitsByProject("myproject", "fix", gomock.Any(), gomock.Any()).
					Return([]*gl.Commit{{ID: "abc123", Message: "Fix bug"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing pid",
			args:              map[string]any{"resourceType": "commits", "scope": "project", "search": "fix"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "pid is required when scope='project'",
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"resourceType": "commits", "scope": "project", "pid": "myproject"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"resourceType": "commits", "scope": "project", "pid": "myproject", "search": "fix"},
			mockSetup:         func() { mockSearch.EXPECT().CommitsByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"resourceType": "commits", "scope": "project", "pid": "myproject", "search": "fix"},
			mockSetup:           func() { mockSearch.EXPECT().CommitsByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list commits",
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
				var commits []*gl.Commit
				json.Unmarshal([]byte(getTextResult(t, result).Text), &commits)
				assert.Len(t, commits, 1)
			}
		})
	}
}

// Tests for additional search tools (Day 3)

func TestSearchMilestonesHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()
	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := Search(mockGetClient, nil)

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
			args: map[string]any{"resourceType": "milestones", "search": "v1.0"},
			mockSetup: func() {
				mockSearch.EXPECT().Milestones("v1.0", gomock.Any(), gomock.Any()).
					Return([]*gl.Milestone{{ID: 1, Title: "v1.0"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"resourceType": "milestones"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"resourceType": "milestones", "search": "v1.0"},
			mockSetup:         func() { mockSearch.EXPECT().Milestones(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"resourceType": "milestones", "search": "v1.0"},
			mockSetup:           func() { mockSearch.EXPECT().Milestones(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list milestones",
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
				var ms []*gl.Milestone
				json.Unmarshal([]byte(getTextResult(t, result).Text), &ms)
				assert.Len(t, ms, 1)
			}
		})
	}
}

func TestSearchSnippetTitlesHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()
	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := Search(mockGetClient, nil)

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
			args: map[string]any{"resourceType": "snippet_titles", "search": "example"},
			mockSetup: func() {
				mockSearch.EXPECT().SnippetTitles("example", gomock.Any(), gomock.Any()).
					Return([]*gl.Snippet{{ID: 1, Title: "Example"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"resourceType": "snippet_titles"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"resourceType": "snippet_titles", "search": "example"},
			mockSetup:         func() { mockSearch.EXPECT().SnippetTitles(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"resourceType": "snippet_titles", "search": "example"},
			mockSetup:           func() { mockSearch.EXPECT().SnippetTitles(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list snippet_titles (scope=global)",
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
				var snippets []*gl.Snippet
				json.Unmarshal([]byte(getTextResult(t, result).Text), &snippets)
				assert.Len(t, snippets, 1)
			}
		})
	}
}

func TestSearchSnippetBlobsHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()
	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := Search(mockGetClient, nil)

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
			args: map[string]any{"resourceType": "snippet_blobs", "search": "function"},
			mockSetup: func() {
				mockSearch.EXPECT().SnippetBlobs("function", gomock.Any(), gomock.Any()).
					Return([]*gl.Snippet{{ID: 1, Title: "Code"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"resourceType": "snippet_blobs"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"resourceType": "snippet_blobs", "search": "function"},
			mockSetup:         func() { mockSearch.EXPECT().SnippetBlobs(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"resourceType": "snippet_blobs", "search": "function"},
			mockSetup:           func() { mockSearch.EXPECT().SnippetBlobs(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list snippet_blobs (scope=global)",
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
				var snippets []*gl.Snippet
				json.Unmarshal([]byte(getTextResult(t, result).Text), &snippets)
				assert.Len(t, snippets, 1)
			}
		})
	}
}

func TestSearchWikiBlobsHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()
	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := Search(mockGetClient, nil)

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
			args: map[string]any{"resourceType": "wiki_blobs", "search": "documentation"},
			mockSetup: func() {
				mockSearch.EXPECT().WikiBlobs("documentation", gomock.Any(), gomock.Any()).
					Return([]*gl.Wiki{{Title: "Documentation", Slug: "docs"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"resourceType": "wiki_blobs"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"resourceType": "wiki_blobs", "search": "documentation"},
			mockSetup:         func() { mockSearch.EXPECT().WikiBlobs(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"resourceType": "wiki_blobs", "search": "documentation"},
			mockSetup:           func() { mockSearch.EXPECT().WikiBlobs(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list wiki_blobs (scope=global)",
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
				var wikis []*gl.Wiki
				json.Unmarshal([]byte(getTextResult(t, result).Text), &wikis)
				assert.Len(t, wikis, 1)
			}
		})
	}
}

func TestSearchNotesByProjectHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockSearch, ctrl := setupMockClientForSearch(t)
	defer ctrl.Finish()
	mockGetClient := func(_ context.Context) (*gl.Client, error) { return mockClient, nil }
	tool, handler := Search(mockGetClient, nil)

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
			args: map[string]any{"resourceType": "notes", "scope": "project", "pid": "myproject", "search": "comment"},
			mockSetup: func() {
				mockSearch.EXPECT().NotesByProject("myproject", "comment", gomock.Any(), gomock.Any()).
					Return([]*gl.Note{{ID: 1, Body: "Great comment"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing pid",
			args:              map[string]any{"resourceType": "notes", "scope": "project", "search": "comment"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "pid is required when scope='project'",
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"resourceType": "notes", "scope": "project", "pid": "myproject"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"resourceType": "notes", "scope": "project", "pid": "myproject", "search": "comment"},
			mockSetup:         func() { mockSearch.EXPECT().NotesByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"resourceType": "notes", "scope": "project", "pid": "myproject", "search": "comment"},
			mockSetup:           func() { mockSearch.EXPECT().NotesByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list notes",
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
				var notes []*gl.Note
				json.Unmarshal([]byte(getTextResult(t, result).Text), &notes)
				assert.Len(t, notes, 1)
			}
		})
	}
}


// TestSearchTools_SchemaSnapshots verifies that search tool schemas match their snapshots
func TestSearchTools_SchemaSnapshots(t *testing.T) {
	tools := []struct {
		name string
		tool mcp.Tool
	}{
		{"search", mustMakeSearchTool(Search)},
	}

	for _, tc := range tools {
		t.Run(tc.name, func(t *testing.T) {
			err := toolsnaps.Test(tc.name, tc.tool)
			require.NoError(t, err, "tool schema should match snapshot")
		})
	}
}

// Helper function to create tool from search tool definition function
func mustMakeSearchTool(fn func(GetClientFn, map[string]string) (mcp.Tool, server.ToolHandlerFunc)) mcp.Tool {
	tool, _ := fn(nil, nil)
	return tool
}
