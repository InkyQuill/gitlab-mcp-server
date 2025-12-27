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
