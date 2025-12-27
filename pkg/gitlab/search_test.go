package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
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

	searchProjectsTool, handler := SearchProjects(mockGetClient, nil)

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
				"search": "test",
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
				"search": "nonexistent",
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
			args:                map[string]any{},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: search",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - API Error (500)",
			args: map[string]any{
				"search": "test",
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

	searchIssuesTool, handler := SearchIssues(mockGetClient, nil)

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
				"search": "bug",
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
				"search": "nonexistent",
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
			args:                map[string]any{},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: search",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - API Error (500)",
			args: map[string]any{
				"search": "bug",
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

	searchMRsTool, handler := SearchMergeRequests(mockGetClient, nil)

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
				"search": "feature",
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
			args:                map[string]any{},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: search",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - API Error (500)",
			args: map[string]any{
				"search": "feature",
			},
			mockSetup: func() {
				mockSearch.EXPECT().
					MergeRequests("feature", gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("internal error"))
			},
			expectedResult:      nil,
			expectResultError:   true,
			expectInternalError: true,
			errorContains:       "failed to list merge requests",
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

	searchBlobsTool, handler := SearchBlobs(mockGetClient, nil)

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
				"search": "function",
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
			args:                map[string]any{},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: search",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - API Error (500)",
			args: map[string]any{
				"search": "function",
			},
			mockSetup: func() {
				mockSearch.EXPECT().
					Blobs("function", gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("internal error"))
			},
			expectedResult:      nil,
			expectResultError:   true,
			expectInternalError: true,
			errorContains:       "failed to list code blobs",
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

	searchCommitsTool, handler := SearchCommits(mockGetClient, nil)

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
				"search": "fix",
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
			args:                map[string]any{},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: search",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - API Error (500)",
			args: map[string]any{
				"search": "fix",
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
		_, handler := SearchProjects(errorGetClientFn, nil)
		tool, _ := SearchProjects(nil, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      tool.Name,
				Arguments: map[string]any{"search": "test"},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to initialize GitLab client: mock init error")
		assert.Nil(t, result)
	})

	t.Run("searchIssues", func(t *testing.T) {
		_, handler := SearchIssues(errorGetClientFn, nil)
		tool, _ := SearchIssues(nil, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      tool.Name,
				Arguments: map[string]any{"search": "test"},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to initialize GitLab client: mock init error")
		assert.Nil(t, result)
	})

	t.Run("searchMergeRequests", func(t *testing.T) {
		_, handler := SearchMergeRequests(errorGetClientFn, nil)
		tool, _ := SearchMergeRequests(nil, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      tool.Name,
				Arguments: map[string]any{"search": "test"},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to initialize GitLab client: mock init error")
		assert.Nil(t, result)
	})

	t.Run("searchBlobs", func(t *testing.T) {
		_, handler := SearchBlobs(errorGetClientFn, nil)
		tool, _ := SearchBlobs(nil, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      tool.Name,
				Arguments: map[string]any{"search": "test"},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to initialize GitLab client: mock init error")
		assert.Nil(t, result)
	})

	t.Run("searchCommits", func(t *testing.T) {
		_, handler := SearchCommits(errorGetClientFn, nil)
		tool, _ := SearchCommits(nil, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      tool.Name,
				Arguments: map[string]any{"search": "test"},
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

	tool, handler := SearchProjectsByGroup(mockGetClient, nil)

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
				"gid":    "mygroup",
				"search": "test",
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
			args:                map[string]any{"search": "test"},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: gid",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name:                "Error - Missing search parameter",
			args:                map[string]any{"gid": "mygroup"},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: search",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - 401 Unauthorized",
			args: map[string]any{
				"gid":    "mygroup",
				"search": "test",
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
				"gid":    "mygroup",
				"search": "test",
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

	tool, handler := SearchIssuesByGroup(mockGetClient, nil)

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
			args: map[string]any{"gid": "mygroup", "search": "bug"},
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
			args:                map[string]any{"search": "bug"},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: gid",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name:                "Error - Missing search",
			args:                map[string]any{"gid": "mygroup"},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: search",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - 401",
			args: map[string]any{"gid": "mygroup", "search": "bug"},
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
			args: map[string]any{"gid": "mygroup", "search": "bug"},
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
	tool, handler := SearchMergeRequestsByGroup(mockGetClient, nil)

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
			args: map[string]any{"gid": "mygroup", "search": "feature"},
			mockSetup: func() {
				mockSearch.EXPECT().MergeRequestsByGroup("mygroup", "feature", gomock.Any(), gomock.Any()).
					Return([]*gl.MergeRequest{{BasicMergeRequest: gl.BasicMergeRequest{ID: 1, IID: 1, Title: "Feature"}}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:                "Error - Missing gid",
			args:                map[string]any{"search": "feature"},
			mockSetup:           func() {},
			expectResultError:   true,
			errorContains:       "missing required parameter: gid",
		},
		{
			name:                "Error - Missing search",
			args:                map[string]any{"gid": "mygroup"},
			mockSetup:           func() {},
			expectResultError:   true,
			errorContains:       "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"gid": "mygroup", "search": "feature"},
			mockSetup:         func() { mockSearch.EXPECT().MergeRequestsByGroup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"gid": "mygroup", "search": "feature"},
			mockSetup:           func() { mockSearch.EXPECT().MergeRequestsByGroup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list merge requests",
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
	tool, handler := SearchMilestonesByGroup(mockGetClient, nil)

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
			args: map[string]any{"gid": "mygroup", "search": "v1.0"},
			mockSetup: func() {
				mockSearch.EXPECT().MilestonesByGroup("mygroup", "v1.0", gomock.Any(), gomock.Any()).
					Return([]*gl.Milestone{{ID: 1, Title: "v1.0"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing gid",
			args:              map[string]any{"search": "v1.0"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: gid",
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"gid": "mygroup"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"gid": "mygroup", "search": "v1.0"},
			mockSetup:         func() { mockSearch.EXPECT().MilestonesByGroup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"gid": "mygroup", "search": "v1.0"},
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
	tool, handler := SearchBlobsByGroup(mockGetClient, nil)

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
			args: map[string]any{"gid": "mygroup", "search": "function"},
			mockSetup: func() {
				mockSearch.EXPECT().BlobsByGroup("mygroup", "function", gomock.Any(), gomock.Any()).
					Return([]*gl.Blob{{Filename: "test.go"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing gid",
			args:              map[string]any{"search": "function"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: gid",
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"gid": "mygroup"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"gid": "mygroup", "search": "function"},
			mockSetup:         func() { mockSearch.EXPECT().BlobsByGroup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"gid": "mygroup", "search": "function"},
			mockSetup:           func() { mockSearch.EXPECT().BlobsByGroup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list code blobs",
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
	tool, handler := SearchIssuesByProject(mockGetClient, nil)

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
			args: map[string]any{"pid": "myproject", "search": "bug"},
			mockSetup: func() {
				mockSearch.EXPECT().IssuesByProject("myproject", "bug", gomock.Any(), gomock.Any()).
					Return([]*gl.Issue{{ID: 1, IID: 1, Title: "Bug"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing pid",
			args:              map[string]any{"search": "bug"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: pid",
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"pid": "myproject"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"pid": "myproject", "search": "bug"},
			mockSetup:         func() { mockSearch.EXPECT().IssuesByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"pid": "myproject", "search": "bug"},
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
	tool, handler := SearchMergeRequestsByProject(mockGetClient, nil)

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
			args: map[string]any{"pid": "myproject", "search": "feature"},
			mockSetup: func() {
				mockSearch.EXPECT().MergeRequestsByProject("myproject", "feature", gomock.Any(), gomock.Any()).
					Return([]*gl.MergeRequest{{BasicMergeRequest: gl.BasicMergeRequest{ID: 1, IID: 1, Title: "Feature"}}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing pid",
			args:              map[string]any{"search": "feature"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: pid",
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"pid": "myproject"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"pid": "myproject", "search": "feature"},
			mockSetup:         func() { mockSearch.EXPECT().MergeRequestsByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"pid": "myproject", "search": "feature"},
			mockSetup:           func() { mockSearch.EXPECT().MergeRequestsByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list merge requests",
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
	tool, handler := SearchMilestonesByProject(mockGetClient, nil)

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
			args: map[string]any{"pid": "myproject", "search": "v1.0"},
			mockSetup: func() {
				mockSearch.EXPECT().MilestonesByProject("myproject", "v1.0", gomock.Any(), gomock.Any()).
					Return([]*gl.Milestone{{ID: 1, Title: "v1.0"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing pid",
			args:              map[string]any{"search": "v1.0"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: pid",
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"pid": "myproject"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"pid": "myproject", "search": "v1.0"},
			mockSetup:         func() { mockSearch.EXPECT().MilestonesByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"pid": "myproject", "search": "v1.0"},
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
	tool, handler := SearchBlobsByProject(mockGetClient, nil)

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
			args: map[string]any{"pid": "myproject", "search": "function"},
			mockSetup: func() {
				mockSearch.EXPECT().BlobsByProject("myproject", "function", gomock.Any(), gomock.Any()).
					Return([]*gl.Blob{{Filename: "test.go"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing pid",
			args:              map[string]any{"search": "function"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: pid",
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"pid": "myproject"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"pid": "myproject", "search": "function"},
			mockSetup:         func() { mockSearch.EXPECT().BlobsByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"pid": "myproject", "search": "function"},
			mockSetup:           func() { mockSearch.EXPECT().BlobsByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list code blobs",
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
	tool, handler := SearchCommitsByProject(mockGetClient, nil)

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
			args: map[string]any{"pid": "myproject", "search": "fix"},
			mockSetup: func() {
				mockSearch.EXPECT().CommitsByProject("myproject", "fix", gomock.Any(), gomock.Any()).
					Return([]*gl.Commit{{ID: "abc123", Message: "Fix bug"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing pid",
			args:              map[string]any{"search": "fix"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: pid",
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"pid": "myproject"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"pid": "myproject", "search": "fix"},
			mockSetup:         func() { mockSearch.EXPECT().CommitsByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"pid": "myproject", "search": "fix"},
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
	tool, handler := SearchMilestones(mockGetClient, nil)

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
			args: map[string]any{"search": "v1.0"},
			mockSetup: func() {
				mockSearch.EXPECT().Milestones("v1.0", gomock.Any(), gomock.Any()).
					Return([]*gl.Milestone{{ID: 1, Title: "v1.0"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"search": "v1.0"},
			mockSetup:         func() { mockSearch.EXPECT().Milestones(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"search": "v1.0"},
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
	tool, handler := SearchSnippetTitles(mockGetClient, nil)

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
			args: map[string]any{"search": "example"},
			mockSetup: func() {
				mockSearch.EXPECT().SnippetTitles("example", gomock.Any(), gomock.Any()).
					Return([]*gl.Snippet{{ID: 1, Title: "Example"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"search": "example"},
			mockSetup:         func() { mockSearch.EXPECT().SnippetTitles(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"search": "example"},
			mockSetup:           func() { mockSearch.EXPECT().SnippetTitles(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list snippet titles",
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
	tool, handler := SearchSnippetBlobs(mockGetClient, nil)

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
			args: map[string]any{"search": "function"},
			mockSetup: func() {
				mockSearch.EXPECT().SnippetBlobs("function", gomock.Any(), gomock.Any()).
					Return([]*gl.Snippet{{ID: 1, Title: "Code"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"search": "function"},
			mockSetup:         func() { mockSearch.EXPECT().SnippetBlobs(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"search": "function"},
			mockSetup:           func() { mockSearch.EXPECT().SnippetBlobs(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list snippet blobs",
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
	tool, handler := SearchWikiBlobs(mockGetClient, nil)

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
			args: map[string]any{"search": "documentation"},
			mockSetup: func() {
				mockSearch.EXPECT().WikiBlobs("documentation", gomock.Any(), gomock.Any()).
					Return([]*gl.Wiki{{Title: "Documentation", Slug: "docs"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"search": "documentation"},
			mockSetup:         func() { mockSearch.EXPECT().WikiBlobs(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"search": "documentation"},
			mockSetup:           func() { mockSearch.EXPECT().WikiBlobs(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("server error")) },
			expectInternalError: true,
			errorContains:       "failed to list wiki blobs",
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
	tool, handler := SearchNotesByProject(mockGetClient, nil)

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
			args: map[string]any{"pid": "myproject", "search": "comment"},
			mockSetup: func() {
				mockSearch.EXPECT().NotesByProject("myproject", "comment", gomock.Any(), gomock.Any()).
					Return([]*gl.Note{{ID: 1, Body: "Great comment"}}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name:              "Error - Missing pid",
			args:              map[string]any{"search": "comment"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: pid",
		},
		{
			name:              "Error - Missing search",
			args:              map[string]any{"pid": "myproject"},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "missing required parameter: search",
		},
		{
			name:              "Error - 401",
			args:              map[string]any{"pid": "myproject", "search": "comment"},
			mockSetup:         func() { mockSearch.EXPECT().NotesByProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("unauthorized")) },
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name:                "Error - 500",
			args:                map[string]any{"pid": "myproject", "search": "comment"},
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
