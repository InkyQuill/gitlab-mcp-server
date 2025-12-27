package gitlab

import (
	"context"       // Needed for assertions
	"encoding/json" // Needed for assertions
	"errors"        // Added for creating mock errors
	"fmt"
	"net/http"
	"testing"
	"time" // Add time import

	// "net/http/httptest"
	// "net/url" // No longer needed for http server
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gl "gitlab.com/gitlab-org/api/client-go" // GitLab client library

	// Import for mocks
	// Gomock mocks
	"go.uber.org/mock/gomock" // Added for gomock
)

// mockGetClientFn is NOT defined locally, assumed provided by other tests or helpers (like setupMockClient*)
// getTextResult is assumed defined elsewhere in package gitlab_test

func TestGetIssueHandler(t *testing.T) {
	ctx := context.Background()

	// --- Setup Mock Client and GetClientFn once ---
	mockClient, mockIssues, ctrl := setupMockClientForIssues(t)
	defer ctrl.Finish()

	// Create the mock getClient function once, capturing the mockClient
	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	// --- Define the Tool and Handler once ---
	getIssueTool, handler := GetIssue(mockGetClient, nil)

	// --- Test Cases ---
	tests := []struct {
		name                string
		projectID           string      // Can be string or int representation for API
		issueIid            float64     // MCP number type maps to float64
		mockSetup           func()      // Gomock setup (no args, uses mockIssues from outer scope)
		expectedResult      interface{} // Expect *gl.Issue for success, or string for user error message
		expectResultError   bool        // True if the returned mcp.CallToolResult should represent an error
		expectInternalError bool        // True if the handler itself should return a non-nil internal error
		errorContains       string      // Substring for internal error check ONLY
	}{
		// --- Success Case ---
		{
			name:      "Success - Get Issue by ID",
			projectID: "group/project",
			issueIid:  1.0,
			mockSetup: func() {
				expectedIssue := &gl.Issue{
					ID:          123,
					IID:         1,
					ProjectID:   456,
					Title:       "Test Issue",
					Description: "This is a test issue.",
				}
				mockIssues.EXPECT().
					GetIssue("group/project", 1, gomock.Any(), gomock.Any()).
					Return(expectedIssue, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: &gl.Issue{ // Store expected struct for success
				ID:          123,
				IID:         1,
				ProjectID:   456,
				Title:       "Test Issue",
				Description: "This is a test issue.",
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		// --- User Error Cases ---
		{
			name:      "Error - Issue Not Found (404)",
			projectID: "group/project",
			issueIid:  999.0,
			mockSetup: func() {
				mockIssues.EXPECT().
					GetIssue("group/project", 999, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Issue Not Found"))
			},
			expectedResult:      "issue 999 in project \"group/project\" not found or access denied (404)", // Expect error message string
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name:                "Error - Missing projectId parameter",
			projectID:           "", // Will be omitted from args
			issueIid:            1.0,
			mockSetup:           func() { /* No API call expected */ },
			expectedResult:      "Validation Error: missing required parameter: projectId", // Expect error message string
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name:                "Error - Invalid issueIid (not integer)",
			projectID:           "group/project",
			issueIid:            1.5, // Non-integer float
			mockSetup:           func() { /* No API call expected */ },
			expectedResult:      "Validation Error: issueIid 1.5 is not a valid integer", // Expect error message string
			expectResultError:   true,
			expectInternalError: false,
		},
		// --- Internal Error Cases ---
		{
			name:      "Error - GitLab API Error (500)",
			projectID: "group/project",
			issueIid:  2.0,
			mockSetup: func() {
				mockIssues.EXPECT().
					GetIssue("group/project", 2, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:      nil,                                                                                       // No result content expected when handler errors
			expectResultError:   true,                                                                                      // Result is nil due to handler error
			expectInternalError: true,                                                                                      // Handler returns an actual error
			errorContains:       "failed to process issue 2 in project \"group/project\": gitlab: 500 Internal Server Error", // Check internal error here
		},
	}

	// Test case for Client Initialization Error (outside the loop)
	t.Run("Error - Client Initialization Error", func(t *testing.T) {
		// Define a GetClientFn that returns an error
		errorGetClientFn := func(_ context.Context) (*gl.Client, error) {
			return nil, fmt.Errorf("mock init error")
		}
		_, handler := GetIssue(errorGetClientFn, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      getIssueTool.Name,
				Arguments: map[string]any{"projectId": "any", "issueIid": 1.0},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to initialize GitLab client: mock init error")
		assert.Nil(t, result)
	})

	// --- Run Loop for other tests ---
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock expectations for this specific test case
			tc.mockSetup()

			// Prepare request arguments
			args := map[string]any{}
			if tc.projectID != "" { // Only add if not empty for missing param test
				args["projectId"] = tc.projectID
			}
			// issueIid is always added as it's required, invalid type tested separately
			args["issueIid"] = tc.issueIid

			// Prepare request using correct structure
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      getIssueTool.Name, // Use the tool name from the definition
					Arguments: args,
				},
			}

			// Execute handler (defined outside the loop)
			result, err := handler(ctx, request)

			// Assertions
			if tc.expectInternalError {
				require.Error(t, err) // Expect the handler itself to return an error
				if tc.errorContains != "" {
					assert.ErrorContains(t, err, tc.errorContains)
				}
				assert.Nil(t, result) // Result should be nil when handler errors
			} else {
				require.NoError(t, err)                 // Handler should not return an error
				require.NotNil(t, result)               // Result should not be nil
				textContent := getTextResult(t, result) // Use helper

				if tc.expectResultError {
					// User-facing errors (validation, not found) check textContent.Text
					expectedErrString, ok := tc.expectedResult.(string)
					require.True(t, ok, "Expected user error result to be a string")
					assert.Contains(t, textContent.Text, expectedErrString, "User error message mismatch")
				} else {
					// Successful result - compare JSON content
					expectedIssue, ok := tc.expectedResult.(*gl.Issue)
					require.True(t, ok, "Expected success result should be *gl.Issue")
					expectedJSON, err := json.Marshal(expectedIssue)
					require.NoError(t, err, "Failed to marshal expected success result")
					assert.JSONEq(t, string(expectedJSON), textContent.Text, "Result JSON mismatch")
				}
			}
		})
	}
}

// TestListIssuesHandler tests the ListIssues tool handler
func TestListIssuesHandler(t *testing.T) {
	ctx := context.Background()

	// --- Setup Mock Client and GetClientFn once ---
	mockClient, mockIssues, ctrl := setupMockClientForIssues(t)
	defer ctrl.Finish()

	// Create the mock getClient function once, capturing the mockClient
	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	// --- Define the Tool and Handler once ---
	listIssuesTool, handler := ListIssues(mockGetClient, nil)

	// --- Test Cases ---
	tests := []struct {
		name                string
		args                map[string]any // All request arguments in one map
		mockSetup           func()         // Gomock setup (no args, uses mockIssues from outer scope)
		expectedResult      interface{}    // Expect slice of issues for success, or string for user error message
		expectResultError   bool           // True if the returned mcp.CallToolResult should represent an error
		expectInternalError bool           // True if the handler itself should return a non-nil internal error
		errorContains       string         // Substring for internal error check ONLY
	}{
		// --- Success Cases ---
		{
			name: "Success - List Issues - No Filters",
			args: map[string]any{
				"projectId": "group/project",
			},
			mockSetup: func() {
				// For simplicity, we'll just return a small list of issues
				expectedIssues := []*gl.Issue{
					{
						ID:          123,
						IID:         1,
						ProjectID:   456,
						Title:       "First Issue",
						Description: "This is the first test issue.",
					},
					{
						ID:          124,
						IID:         2,
						ProjectID:   456,
						Title:       "Second Issue",
						Description: "This is the second test issue.",
					},
				}

				// Match the default pagination values from server.go
				listOpts := &gl.ListProjectIssuesOptions{
					ListOptions: gl.ListOptions{
						Page:    1,
						PerPage: 20,
					},
				}

				mockIssues.EXPECT().
					ListProjectIssues("group/project", listOpts, gomock.Any()).
					Return(expectedIssues, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: []*gl.Issue{
				{
					ID:          123,
					IID:         1,
					ProjectID:   456,
					Title:       "First Issue",
					Description: "This is the first test issue.",
				},
				{
					ID:          124,
					IID:         2,
					ProjectID:   456,
					Title:       "Second Issue",
					Description: "This is the second test issue.",
				},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - List Issues - With Filters",
			args: map[string]any{
				"projectId": "group/project",
				"state":     "opened",
				"labels":    "bug,critical",
				"page":      2.0, // JSON numbers come in as float64
				"per_page":  5.0,
			},
			mockSetup: func() {
				// We'll set up a mock that expects the specific filter values
				expectedIssues := []*gl.Issue{
					{
						ID:        125,
						IID:       3,
						ProjectID: 456,
						State:     "opened",
						Title:     "Critical Bug",
						Labels:    []string{"bug", "critical"},
					},
				}

				// Create a matcher that checks for expected filter values
				// The ListProjectIssues function should convert the "bug,critical" string to a slice
				labelOpts := gl.LabelOptions([]string{"bug", "critical"})
				state := "opened"

				// Expected options object - not used directly, but referenced for verification
				// in the DoAndReturn function below
				_ = &gl.ListProjectIssuesOptions{
					ListOptions: gl.ListOptions{
						Page:    2,
						PerPage: 5,
					},
					State:  &state,
					Labels: &labelOpts,
				}

				mockIssues.EXPECT().
					ListProjectIssues("group/project", gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ interface{}, opts *gl.ListProjectIssuesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Issue, *gl.Response, error) {
						// Verify the expected filters are set
						assert.Equal(t, 2, opts.Page)
						assert.Equal(t, 5, opts.PerPage)
						assert.Equal(t, "opened", *opts.State)
						assert.Equal(t, 2, len(*opts.Labels))
						assert.Contains(t, *opts.Labels, "bug")
						assert.Contains(t, *opts.Labels, "critical")

						return expectedIssues, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Issue{
				{
					ID:        125,
					IID:       3,
					ProjectID: 456,
					State:     "opened",
					Title:     "Critical Bug",
					Labels:    []string{"bug", "critical"},
				},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Empty Result",
			args: map[string]any{
				"projectId": "group/project",
				"state":     "closed",
			},
			mockSetup: func() {
				// Return an empty list for this filter
				state := "closed"
				listOpts := &gl.ListProjectIssuesOptions{
					ListOptions: gl.ListOptions{
						Page:    1,
						PerPage: 20,
					},
					State: &state,
				}

				mockIssues.EXPECT().
					ListProjectIssues("group/project", listOpts, gomock.Any()).
					Return([]*gl.Issue{}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult:      "[]", // Empty JSON array string
			expectResultError:   false,
			expectInternalError: false,
		},
		// --- Error Cases ---
		{
			name: "Error - Project Not Found (404)",
			args: map[string]any{
				"projectId": "nonexistent",
			},
			mockSetup: func() {
				mockIssues.EXPECT().
					ListProjectIssues("nonexistent", gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Project Not Found"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to list issues from project \"nonexistent\"",
		},
		{
			name: "Error - GitLab API Error (500)",
			args: map[string]any{
				"projectId": "group/project",
			},
			mockSetup: func() {
				mockIssues.EXPECT().
					ListProjectIssues("group/project", gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to list issues from project \"group/project\": gitlab: 500 Internal Server Error",
		},
		{
			name:                "Error - Missing projectId parameter",
			args:                map[string]any{}, // Deliberately empty
			mockSetup:           func() { /* No API call expected */ },
			expectedResult:      "Validation Error: missing required parameter: projectId",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Invalid Date Format",
			args: map[string]any{
				"projectId":    "group/project",
				"createdAfter": "not-a-date",
			},
			mockSetup:           func() { /* No API call expected */ },
			expectedResult:      "Validation Error: parameter 'createdAfter' must be a valid ISO 8601 timestamp string (e.g., '2006-01-02T15:04:05Z'), got \"not-a-date\":",
			expectResultError:   true,
			expectInternalError: false,
		},
		// --- Additional Filter Tests ---
		{
			name: "Success - Filter by milestone",
			args: map[string]any{
				"projectId": "group/project",
				"milestone": "v1.0",
			},
			mockSetup: func() {
				expectedIssues := []*gl.Issue{
					{ID: 123, IID: 1, ProjectID: 456, Title: "Issue in v1.0"},
				}
				mockIssues.EXPECT().
					ListProjectIssues("group/project", gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ interface{}, opts *gl.ListProjectIssuesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Issue, *gl.Response, error) {
						require.NotNil(t, opts.Milestone)
						assert.Equal(t, "v1.0", *opts.Milestone)
						return expectedIssues, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Issue{
				{ID: 123, IID: 1, ProjectID: 456, Title: "Issue in v1.0"},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Filter by scope (created_by_me)",
			args: map[string]any{
				"projectId": "group/project",
				"scope":     "created_by_me",
			},
			mockSetup: func() {
				expectedIssues := []*gl.Issue{
					{ID: 124, IID: 2, ProjectID: 456, Title: "My Issue"},
				}
				mockIssues.EXPECT().
					ListProjectIssues("group/project", gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ interface{}, opts *gl.ListProjectIssuesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Issue, *gl.Response, error) {
						require.NotNil(t, opts.Scope)
						assert.Equal(t, "created_by_me", *opts.Scope)
						return expectedIssues, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Issue{
				{ID: 124, IID: 2, ProjectID: 456, Title: "My Issue"},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Filter by authorId",
			args: map[string]any{
				"projectId": "group/project",
				"authorId":  123.0,
			},
			mockSetup: func() {
				expectedIssues := []*gl.Issue{
					{ID: 125, IID: 3, ProjectID: 456, Title: "Authored Issue"},
				}
				mockIssues.EXPECT().
					ListProjectIssues("group/project", gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ interface{}, opts *gl.ListProjectIssuesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Issue, *gl.Response, error) {
						require.NotNil(t, opts.AuthorID)
						assert.Equal(t, 123, *opts.AuthorID)
						return expectedIssues, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Issue{
				{ID: 125, IID: 3, ProjectID: 456, Title: "Authored Issue"},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Filter by assigneeId",
			args: map[string]any{
				"projectId":  "group/project",
				"assigneeId": 456.0,
			},
			mockSetup: func() {
				expectedIssues := []*gl.Issue{
					{ID: 126, IID: 4, ProjectID: 456, Title: "Assigned Issue"},
				}
				mockIssues.EXPECT().
					ListProjectIssues("group/project", gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ interface{}, opts *gl.ListProjectIssuesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Issue, *gl.Response, error) {
						require.NotNil(t, opts.AssigneeID)
						assert.Equal(t, 456, *opts.AssigneeID)
						return expectedIssues, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Issue{
				{ID: 126, IID: 4, ProjectID: 456, Title: "Assigned Issue"},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Filter by search",
			args: map[string]any{
				"projectId": "group/project",
				"search":    "bug",
			},
			mockSetup: func() {
				expectedIssues := []*gl.Issue{
					{ID: 127, IID: 5, ProjectID: 456, Title: "Bug Issue"},
				}
				mockIssues.EXPECT().
					ListProjectIssues("group/project", gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ interface{}, opts *gl.ListProjectIssuesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Issue, *gl.Response, error) {
						require.NotNil(t, opts.Search)
						assert.Equal(t, "bug", *opts.Search)
						return expectedIssues, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Issue{
				{ID: 127, IID: 5, ProjectID: 456, Title: "Bug Issue"},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Filter with orderBy and sort",
			args: map[string]any{
				"projectId": "group/project",
				"orderBy":   "created_at",
				"sort":      "asc",
			},
			mockSetup: func() {
				expectedIssues := []*gl.Issue{
					{ID: 123, IID: 1, ProjectID: 456, Title: "Old Issue"},
					{ID: 124, IID: 2, ProjectID: 456, Title: "New Issue"},
				}
				mockIssues.EXPECT().
					ListProjectIssues("group/project", gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ interface{}, opts *gl.ListProjectIssuesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Issue, *gl.Response, error) {
						require.NotNil(t, opts.OrderBy)
						assert.Equal(t, "created_at", *opts.OrderBy)
						require.NotNil(t, opts.Sort)
						assert.Equal(t, "asc", *opts.Sort)
						return expectedIssues, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Issue{
				{ID: 123, IID: 1, ProjectID: 456, Title: "Old Issue"},
				{ID: 124, IID: 2, ProjectID: 456, Title: "New Issue"},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Combined multiple filters",
			args: map[string]any{
				"projectId": "group/project",
				"state":     "opened",
				"labels":    "bug",
				"authorId":  123.0,
				"search":    "critical",
				"orderBy":   "priority",
				"sort":      "desc",
				"milestone": "v1.0",
			},
			mockSetup: func() {
				expectedIssues := []*gl.Issue{
					{ID: 128, IID: 6, ProjectID: 456, State: "opened", Title: "Critical Bug", Labels: []string{"bug"}},
				}
				labelOpts := gl.LabelOptions([]string{"bug"})
				state := "opened"
				mockIssues.EXPECT().
					ListProjectIssues("group/project", gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ interface{}, opts *gl.ListProjectIssuesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Issue, *gl.Response, error) {
						require.NotNil(t, opts.State)
						assert.Equal(t, state, *opts.State)
						require.NotNil(t, opts.Labels)
						assert.Equal(t, labelOpts, *opts.Labels)
						require.NotNil(t, opts.AuthorID)
						assert.Equal(t, 123, *opts.AuthorID)
						require.NotNil(t, opts.Search)
						assert.Equal(t, "critical", *opts.Search)
						require.NotNil(t, opts.OrderBy)
						assert.Equal(t, "priority", *opts.OrderBy)
						require.NotNil(t, opts.Sort)
						assert.Equal(t, "desc", *opts.Sort)
						require.NotNil(t, opts.Milestone)
						assert.Equal(t, "v1.0", *opts.Milestone)
						return expectedIssues, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Issue{
				{ID: 128, IID: 6, ProjectID: 456, State: "opened", Title: "Critical Bug", Labels: []string{"bug"}},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
	}

	// Test case for Client Initialization Error (outside the loop)
	t.Run("Error - Client Initialization Error", func(t *testing.T) {
		// Define a GetClientFn that returns an error
		errorGetClientFn := func(_ context.Context) (*gl.Client, error) {
			return nil, fmt.Errorf("mock init error")
		}
		_, handler := ListIssues(errorGetClientFn, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      listIssuesTool.Name,
				Arguments: map[string]any{"projectId": "any"},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to initialize GitLab client: mock init error")
		assert.Nil(t, result)
	})

	// --- Run Loop for other tests ---
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock expectations for this specific test case
			tc.mockSetup()

			// Prepare request using correct structure
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      listIssuesTool.Name, // Use the tool name from the definition
					Arguments: tc.args,
				},
			}

			// Execute handler (defined outside the loop)
			result, err := handler(ctx, request)

			// Assertions
			if tc.expectInternalError {
				require.Error(t, err) // Expect the handler itself to return an error
				if tc.errorContains != "" {
					assert.ErrorContains(t, err, tc.errorContains)
				}
				assert.Nil(t, result) // Result should be nil when handler errors
			} else {
				require.NoError(t, err)                 // Handler should not return an error
				require.NotNil(t, result)               // Result should not be nil
				textContent := getTextResult(t, result) // Use helper

				if tc.expectResultError {
					// User-facing errors (validation, not found) check textContent.Text
					expectedErrString, ok := tc.expectedResult.(string)
					require.True(t, ok, "Expected user error result to be a string")
					assert.Contains(t, textContent.Text, expectedErrString, "User error message mismatch")
				} else {
					if expectedStr, ok := tc.expectedResult.(string); ok {
						// Special case for empty array
						assert.Equal(t, expectedStr, textContent.Text, "Result JSON mismatch")
					} else {
						// Successful result - compare JSON content
						expectedIssues, ok := tc.expectedResult.([]*gl.Issue)
						require.True(t, ok, "Expected success result should be []*gl.Issue")
						expectedJSON, err := json.Marshal(expectedIssues)
						require.NoError(t, err, "Failed to marshal expected success result")
						assert.JSONEq(t, string(expectedJSON), textContent.Text, "Result JSON mismatch")
					}
				}
			}
		})
	}
}

// TestGetIssueCommentsHandler tests the GetIssueComments tool handler
func TestGetIssueCommentsHandler(t *testing.T) {
	ctx := context.Background()

	// --- Setup Mock Client and GetClientFn once ---
	mockClient, mockNotes, ctrl := setupMockClientForNotes(t)
	defer ctrl.Finish()

	// Create the mock getClient function once, capturing the mockClient
	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	// --- Define the Tool and Handler once ---
	getIssueCommentsTool, handler := GetIssueComments(mockGetClient, nil)

	// Define common test data
	projectID := "group/project"
	issueIid := 1.0 // MCP number type maps to float64

	// Create time values to use in tests
	timeNow := time.Now()
	time24HoursAgo := timeNow.Add(-24 * time.Hour)
	time12HoursAgo := timeNow.Add(-12 * time.Hour)
	time6HoursAgo := timeNow.Add(-6 * time.Hour)

	// --- Test Cases ---
	tests := []struct {
		name                string
		args                map[string]any // All request arguments in one map
		mockSetup           func()         // Gomock setup (uses mockNotes from outer scope)
		expectedResult      interface{}    // Expect slice of notes for success, or string for user error message
		expectResultError   bool           // True if the returned mcp.CallToolResult should represent an error
		expectInternalError bool           // True if the handler itself should return a non-nil internal error
		errorContains       string         // Substring for internal error check ONLY
	}{
		// --- Success Cases ---
		{
			name: "Success - Get Issue Comments",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  issueIid,
			},
			mockSetup: func() {
				// Create expected notes for the response with correct fields
				expectedNotes := []*gl.Note{
					{
						ID:   123,
						Body: "This is a comment",
						Author: gl.NoteAuthor{
							Name: "Test User",
						},
						CreatedAt: &time24HoursAgo,
					},
					{
						ID:   124,
						Body: "This is another comment",
						Author: gl.NoteAuthor{
							Name: "Another User",
						},
						CreatedAt: &time12HoursAgo,
					},
				}

				mockNotes.EXPECT().
					ListIssueNotes(projectID, int(issueIid), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gl.ListIssueNotesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Note, *gl.Response, error) {
						// Verify pagination settings
						assert.Equal(t, 1, opts.Page)
						assert.Equal(t, DefaultPerPage, opts.PerPage)

						return expectedNotes, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Note{
				{
					ID:   123,
					Body: "This is a comment",
					Author: gl.NoteAuthor{
						Name: "Test User",
					},
					CreatedAt: &time24HoursAgo,
				},
				{
					ID:   124,
					Body: "This is another comment",
					Author: gl.NoteAuthor{
						Name: "Another User",
					},
					CreatedAt: &time12HoursAgo,
				},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Issue Comments With Pagination",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  issueIid,
				"page":      2,
				"per_page":  5,
			},
			mockSetup: func() {
				// Create expected notes for the response with correct fields
				expectedNotes := []*gl.Note{
					{
						ID:   125,
						Body: "Paginated comment",
						Author: gl.NoteAuthor{
							Name: "Test User",
						},
						CreatedAt: &time6HoursAgo,
					},
				}

				mockNotes.EXPECT().
					ListIssueNotes(projectID, int(issueIid), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gl.ListIssueNotesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Note, *gl.Response, error) {
						// Verify pagination settings
						assert.Equal(t, 2, opts.Page)
						assert.Equal(t, 5, opts.PerPage)

						return expectedNotes, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Note{
				{
					ID:   125,
					Body: "Paginated comment",
					Author: gl.NoteAuthor{
						Name: "Test User",
					},
					CreatedAt: &time6HoursAgo,
				},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Empty Comments",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  2.0, // Different issue with no comments
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					ListIssueNotes(projectID, 2, gomock.Any(), gomock.Any()).
					Return([]*gl.Note{}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult:      "[]", // Empty JSON array string
			expectResultError:   false,
			expectInternalError: false,
		},
		// --- Error Cases ---
		{
			name: "Error - Issue Not Found (404)",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  999.0,
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					ListIssueNotes(projectID, 999, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Issue Not Found"))
			},
			expectedResult:      "comments for issue 999 in project \"group/project\" not found or access denied (404)",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - GitLab API Error (500)",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  issueIid,
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					ListIssueNotes(projectID, int(issueIid), gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to process comments for issue 1 in project \"group/project\"",
		},
		{
			name:                "Error - Missing projectId parameter",
			args:                map[string]any{"issueIid": issueIid}, // Missing projectId
			mockSetup:           func() { /* No API call expected */ },
			expectedResult:      "Validation Error: missing required parameter: projectId",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name:                "Error - Missing issueIid parameter",
			args:                map[string]any{"projectId": projectID}, // Missing issueIid
			mockSetup:           func() { /* No API call expected */ },
			expectedResult:      "Validation Error: missing required parameter: issueIid",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name:                "Error - Invalid issueIid (not integer)",
			args:                map[string]any{"projectId": projectID, "issueIid": 1.5}, // Non-integer float
			mockSetup:           func() { /* No API call expected */ },
			expectedResult:      "Validation Error: issueIid 1.5 is not a valid integer",
			expectResultError:   true,
			expectInternalError: false,
		},
	}

	// Test case for Client Initialization Error (outside the loop)
	t.Run("Error - Client Initialization Error", func(t *testing.T) {
		// Define a GetClientFn that returns an error
		errorGetClientFn := func(_ context.Context) (*gl.Client, error) {
			return nil, fmt.Errorf("mock init error")
		}
		_, handler := GetIssueComments(errorGetClientFn, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      getIssueCommentsTool.Name,
				Arguments: map[string]any{"projectId": "any", "issueIid": 1.0},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to initialize GitLab client: mock init error")
		assert.Nil(t, result)
	})

	// --- Run Loop for other tests ---
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock expectations for this specific test case
			tc.mockSetup()

			// Prepare request using correct structure
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      getIssueCommentsTool.Name,
					Arguments: tc.args,
				},
			}

			// Execute handler (defined outside the loop)
			result, err := handler(ctx, request)

			// Assertions
			if tc.expectInternalError {
				require.Error(t, err) // Expect the handler itself to return an error
				if tc.errorContains != "" {
					assert.ErrorContains(t, err, tc.errorContains)
				}
				assert.Nil(t, result) // Result should be nil when handler errors
			} else {
				require.NoError(t, err)                 // Handler should not return an error
				require.NotNil(t, result)               // Result should not be nil
				textContent := getTextResult(t, result) // Use helper

				if tc.expectResultError {
					// User-facing errors (validation, not found) check textContent.Text
					expectedErrString, ok := tc.expectedResult.(string)
					require.True(t, ok, "Expected user error result to be a string")
					assert.Contains(t, textContent.Text, expectedErrString, "User error message mismatch")
				} else {
					if expectedStr, ok := tc.expectedResult.(string); ok {
						// Special case for empty array
						assert.Equal(t, expectedStr, textContent.Text, "Result JSON mismatch")
					} else {
						// Successful result - compare JSON content
						expectedNotes, ok := tc.expectedResult.([]*gl.Note)
						require.True(t, ok, "Expected success result should be []*gl.Note")

						// For time-related fields, we need to be more careful with comparisons
						// Instead of direct JSON comparison, we could extract and compare the relevant fields
						// or use a custom comparison function

						// Unmarshal the actual result
						var actualNotes []*gl.Note
						err := json.Unmarshal([]byte(textContent.Text), &actualNotes)
						require.NoError(t, err, "Failed to unmarshal actual result JSON")

						// Compare lengths
						assert.Equal(t, len(expectedNotes), len(actualNotes), "Number of notes doesn't match")

						// Compare important fields for each note
						for i, expectedNote := range expectedNotes {
							if i < len(actualNotes) {
								assert.Equal(t, expectedNote.ID, actualNotes[i].ID, "Note ID mismatch")
								assert.Equal(t, expectedNote.Body, actualNotes[i].Body, "Note body mismatch")
								assert.Equal(t, expectedNote.Author.Name, actualNotes[i].Author.Name, "Note author mismatch")
							}
						}
					}
				}
			}
		})
	}
}

// TestGetIssueLabelsHandler tests the GetIssueLabels tool handler
func TestGetIssueLabelsHandler(t *testing.T) {
	ctx := context.Background()

	// --- Setup Mock Client and GetClientFn once ---
	mockClient, mockIssues, ctrl := setupMockClientForIssues(t)
	defer ctrl.Finish()

	// Create the mock getClient function once, capturing the mockClient
	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	// --- Define the Tool and Handler once ---
	getIssueLabelssTool, handler := GetIssueLabels(mockGetClient, nil)

	// Define common test data
	projectID := "group/project"
	issueIid := 1.0 // MCP number type maps to float64

	// --- Test Cases ---
	tests := []struct {
		name                string
		args                map[string]any // All request arguments in one map
		mockSetup           func()         // Gomock setup (uses mockIssues from outer scope)
		expectedResult      interface{}    // Expect slice of labels for success, or string for user error message
		expectResultError   bool           // True if the returned mcp.CallToolResult should represent an error
		expectInternalError bool           // True if the handler itself should return a non-nil internal error
		errorContains       string         // Substring for internal error check ONLY
	}{
		// --- Success Cases ---
		{
			name: "Success - Get Issue Labels",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  issueIid,
			},
			mockSetup: func() {
				// Create expected issue with labels for the response
				expectedIssue := &gl.Issue{
					ID:          123,
					IID:         1,
					ProjectID:   456,
					Title:       "Test Issue",
					Description: "This is a test issue.",
					Labels:      []string{"bug", "critical", "needs-review"},
				}

				mockIssues.EXPECT().
					GetIssue(projectID, int(issueIid), gomock.Any(), gomock.Any()).
					Return(expectedIssue, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult:      []string{"bug", "critical", "needs-review"},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Empty Labels",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  2.0, // Different issue with no labels
			},
			mockSetup: func() {
				// Create expected issue with no labels
				expectedIssue := &gl.Issue{
					ID:          124,
					IID:         2,
					ProjectID:   456,
					Title:       "Another Test Issue",
					Description: "This is another test issue.",
					Labels:      []string{}, // Empty array of labels
				}

				mockIssues.EXPECT().
					GetIssue(projectID, 2, gomock.Any(), gomock.Any()).
					Return(expectedIssue, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult:      "[]", // Empty JSON array string
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Nil Labels",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  3.0, // Different issue with nil labels
			},
			mockSetup: func() {
				// Create expected issue with nil labels
				expectedIssue := &gl.Issue{
					ID:          125,
					IID:         3,
					ProjectID:   456,
					Title:       "Yet Another Test Issue",
					Description: "This is yet another test issue.",
					Labels:      nil, // Nil array of labels
				}

				mockIssues.EXPECT().
					GetIssue(projectID, 3, gomock.Any(), gomock.Any()).
					Return(expectedIssue, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult:      "[]", // Empty JSON array string
			expectResultError:   false,
			expectInternalError: false,
		},
		// --- Error Cases ---
		{
			name: "Error - Issue Not Found (404)",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  999.0,
			},
			mockSetup: func() {
				mockIssues.EXPECT().
					GetIssue(projectID, 999, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Issue Not Found"))
			},
			expectedResult:      "labels for issue 999 in project \"group/project\" not found or access denied (404)",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - GitLab API Error (500)",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  issueIid,
			},
			mockSetup: func() {
				mockIssues.EXPECT().
					GetIssue(projectID, int(issueIid), gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to process labels for issue 1 in project \"group/project\"",
		},
		{
			name:                "Error - Missing projectId parameter",
			args:                map[string]any{"issueIid": issueIid}, // Missing projectId
			mockSetup:           func() { /* No API call expected */ },
			expectedResult:      "Validation Error: missing required parameter: projectId",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name:                "Error - Missing issueIid parameter",
			args:                map[string]any{"projectId": projectID}, // Missing issueIid
			mockSetup:           func() { /* No API call expected */ },
			expectedResult:      "Validation Error: missing required parameter: issueIid",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name:                "Error - Invalid issueIid (not integer)",
			args:                map[string]any{"projectId": projectID, "issueIid": 1.5}, // Non-integer float
			mockSetup:           func() { /* No API call expected */ },
			expectedResult:      "Validation Error: issueIid 1.5 is not a valid integer",
			expectResultError:   true,
			expectInternalError: false,
		},
	}

	// Test case for Client Initialization Error (outside the loop)
	t.Run("Error - Client Initialization Error", func(t *testing.T) {
		// Define a GetClientFn that returns an error
		errorGetClientFn := func(_ context.Context) (*gl.Client, error) {
			return nil, fmt.Errorf("mock init error")
		}
		_, handler := GetIssueLabels(errorGetClientFn, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      getIssueLabelssTool.Name,
				Arguments: map[string]any{"projectId": "any", "issueIid": 1.0},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to initialize GitLab client: mock init error")
		assert.Nil(t, result)
	})

	// --- Run Loop for other tests ---
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock expectations for this specific test case
			tc.mockSetup()

			// Prepare request using correct structure
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      getIssueLabelssTool.Name,
					Arguments: tc.args,
				},
			}

			// Execute handler (defined outside the loop)
			result, err := handler(ctx, request)

			// Assertions
			if tc.expectInternalError {
				require.Error(t, err) // Expect the handler itself to return an error
				if tc.errorContains != "" {
					assert.ErrorContains(t, err, tc.errorContains)
				}
				assert.Nil(t, result) // Result should be nil when handler errors
			} else {
				require.NoError(t, err)                 // Handler should not return an error
				require.NotNil(t, result)               // Result should not be nil
				textContent := getTextResult(t, result) // Use helper

				if tc.expectResultError {
					// User-facing errors (validation, not found) check textContent.Text
					expectedErrString, ok := tc.expectedResult.(string)
					require.True(t, ok, "Expected user error result to be a string")
					assert.Contains(t, textContent.Text, expectedErrString, "User error message mismatch")
				} else {
					if expectedStr, ok := tc.expectedResult.(string); ok {
						// Special case for empty array
						assert.Equal(t, expectedStr, textContent.Text, "Result JSON mismatch")
					} else {
						// Successful result - compare JSON content
						expectedLabels, ok := tc.expectedResult.([]string)
						require.True(t, ok, "Expected success result should be []string")

						// Marshal expected labels for comparison
						expectedJSON, err := json.Marshal(expectedLabels)
						require.NoError(t, err, "Failed to marshal expected labels")

						// Compare JSON directly
						assert.JSONEq(t, string(expectedJSON), textContent.Text, "Labels JSON mismatch")
					}
				}
			}
		})
	}
}

// TestCreateIssueHandler tests the CreateIssue tool handler
func TestCreateIssueHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockIssues, ctrl := setupMockClientForIssues(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	createIssueTool, handler := CreateIssue(mockGetClient, nil)

	projectID := "group/project"
	timeNow := time.Now()

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
			name: "Success - Create Issue with minimal fields",
			args: map[string]any{
				"projectId": projectID,
				"title":     "New Issue",
			},
			mockSetup: func() {
				expectedIssue := &gl.Issue{
					ID:          123,
					IID:         1,
					ProjectID:   456,
					Title:       "New Issue",
					Description: "",
					State:       "opened",
					CreatedAt:   &timeNow,
				}
				mockIssues.EXPECT().
					CreateIssue(projectID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gl.CreateIssueOptions, _ ...gl.RequestOptionFunc) (*gl.Issue, *gl.Response, error) {
						assert.Equal(t, "New Issue", *opts.Title)
						return expectedIssue, &gl.Response{Response: &http.Response{StatusCode: 201}}, nil
					})
			},
			expectedResult: &gl.Issue{
				ID:          123,
				IID:         1,
				ProjectID:   456,
				Title:       "New Issue",
				Description: "",
				State:       "opened",
				CreatedAt:   &timeNow,
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Create Issue with all fields",
			args: map[string]any{
				"projectId":   projectID,
				"title":       "Complete Issue",
				"description": "This is a complete issue",
				"labels":      "bug,critical",
				"assigneeIds": "1,2",
				"milestoneId": 5.0,
				"dueDate":     "2024-12-31",
			},
			mockSetup: func() {
				expectedIssue := &gl.Issue{
					ID:          124,
					IID:         2,
					ProjectID:   456,
					Title:       "Complete Issue",
					Description: "This is a complete issue",
					State:       "opened",
					Labels:      []string{"bug", "critical"},
					CreatedAt:   &timeNow,
				}
				mockIssues.EXPECT().
					CreateIssue(projectID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gl.CreateIssueOptions, _ ...gl.RequestOptionFunc) (*gl.Issue, *gl.Response, error) {
						assert.Equal(t, "Complete Issue", *opts.Title)
						assert.Equal(t, "This is a complete issue", *opts.Description)
						assert.NotNil(t, opts.Labels)
						assert.NotNil(t, opts.AssigneeIDs)
						assert.Equal(t, 2, len(*opts.AssigneeIDs))
						assert.Equal(t, 5, *opts.MilestoneID)
						assert.NotNil(t, opts.DueDate)
						return expectedIssue, &gl.Response{Response: &http.Response{StatusCode: 201}}, nil
					})
			},
			expectedResult: &gl.Issue{
				ID:          124,
				IID:         2,
				ProjectID:   456,
				Title:       "Complete Issue",
				Description: "This is a complete issue",
				State:       "opened",
				Labels:      []string{"bug", "critical"},
				CreatedAt:   &timeNow,
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Error - Missing projectId",
			args: map[string]any{
				"title": "New Issue",
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: projectId",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Missing title",
			args: map[string]any{
				"projectId": projectID,
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: title",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Invalid dueDate format",
			args: map[string]any{
				"projectId": projectID,
				"title":     "New Issue",
				"dueDate":   "invalid-date",
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: dueDate must be in YYYY-MM-DD format",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Invalid assigneeIds",
			args: map[string]any{
				"projectId":   projectID,
				"title":       "New Issue",
				"assigneeIds": "not-a-number",
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: invalid assignee ID",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Project Not Found (404)",
			args: map[string]any{
				"projectId": "nonexistent",
				"title":     "New Issue",
			},
			mockSetup: func() {
				mockIssues.EXPECT().
					CreateIssue("nonexistent", gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Project Not Found"))
			},
			expectedResult:      "project \"nonexistent\" not found or access denied (404)",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - GitLab API Error (500)",
			args: map[string]any{
				"projectId": projectID,
				"title":     "New Issue",
			},
			mockSetup: func() {
				mockIssues.EXPECT().
					CreateIssue(projectID, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to create issue project",
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
					Name:      createIssueTool.Name,
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
					expectedIssue, ok := tc.expectedResult.(*gl.Issue)
					require.True(t, ok)
					expectedJSON, err := json.Marshal(expectedIssue)
					require.NoError(t, err)
					assert.JSONEq(t, string(expectedJSON), textContent.Text)
				}
			}
		})
	}

	t.Run("Error - Client Initialization Error", func(t *testing.T) {
		errorGetClientFn := func(_ context.Context) (*gl.Client, error) {
			return nil, fmt.Errorf("mock init error")
		}
		_, handler := CreateIssue(errorGetClientFn, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      createIssueTool.Name,
				Arguments: map[string]any{"projectId": projectID, "title": "Test"},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to initialize GitLab client")
		assert.Nil(t, result)
	})
}

// TestUpdateIssueHandler tests the UpdateIssue tool handler
func TestUpdateIssueHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockIssues, ctrl := setupMockClientForIssues(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	updateIssueTool, handler := UpdateIssue(mockGetClient, nil)

	projectID := "group/project"
	issueIid := 1.0
	timeNow := time.Now()

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
			name: "Success - Update Issue title",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  issueIid,
				"title":     "Updated Title",
			},
			mockSetup: func() {
				expectedIssue := &gl.Issue{
					ID:          123,
					IID:         1,
					ProjectID:   456,
					Title:       "Updated Title",
					Description: "Original description",
					State:       "opened",
					UpdatedAt:   &timeNow,
				}
				mockIssues.EXPECT().
					UpdateIssue(projectID, 1, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gl.UpdateIssueOptions, _ ...gl.RequestOptionFunc) (*gl.Issue, *gl.Response, error) {
						assert.Equal(t, "Updated Title", *opts.Title)
						return expectedIssue, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: &gl.Issue{
				ID:          123,
				IID:         1,
				ProjectID:   456,
				Title:       "Updated Title",
				Description: "Original description",
				State:       "opened",
				UpdatedAt:   &timeNow,
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Update Issue with stateEvent",
			args: map[string]any{
				"projectId":  projectID,
				"issueIid":   issueIid,
				"stateEvent": "close",
			},
			mockSetup: func() {
				expectedIssue := &gl.Issue{
					ID:        123,
					IID:       1,
					ProjectID: 456,
					Title:     "Test Issue",
					State:     "closed",
					UpdatedAt: &timeNow,
				}
				mockIssues.EXPECT().
					UpdateIssue(projectID, 1, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gl.UpdateIssueOptions, _ ...gl.RequestOptionFunc) (*gl.Issue, *gl.Response, error) {
						assert.Equal(t, "close", *opts.StateEvent)
						return expectedIssue, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: &gl.Issue{
				ID:        123,
				IID:       1,
				ProjectID: 456,
				Title:     "Test Issue",
				State:     "closed",
				UpdatedAt: &timeNow,
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Error - Missing projectId",
			args: map[string]any{
				"issueIid": issueIid,
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: projectId",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Missing issueIid",
			args: map[string]any{
				"projectId": projectID,
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: issueIid",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Issue Not Found (404)",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  999.0,
			},
			mockSetup: func() {
				mockIssues.EXPECT().
					UpdateIssue(projectID, 999, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Issue Not Found"))
			},
			expectedResult:      "issue 999 in project \"group/project\" not found or access denied (404)",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - GitLab API Error (500)",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  issueIid,
			},
			mockSetup: func() {
				mockIssues.EXPECT().
					UpdateIssue(projectID, 1, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to update issue",
		},
		{
			name: "Success - Update multiple fields",
			args: map[string]any{
				"projectId":  projectID,
				"issueIid":   issueIid,
				"title":       "Updated Title",
				"description": "Updated description",
				"labels":      "bug,enhancement",
			},
			mockSetup: func() {
				expectedIssue := &gl.Issue{
					ID:          123,
					IID:         1,
					ProjectID:   456,
					Title:       "Updated Title",
					Description: "Updated description",
					State:       "opened",
					UpdatedAt:   &timeNow,
				}
				mockIssues.EXPECT().
					UpdateIssue(projectID, 1, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gl.UpdateIssueOptions, _ ...gl.RequestOptionFunc) (*gl.Issue, *gl.Response, error) {
						assert.Equal(t, "Updated Title", *opts.Title)
						assert.Equal(t, "Updated description", *opts.Description)
						require.NotNil(t, opts.Labels)
						assert.ElementsMatch(t, gl.LabelOptions{"bug", "enhancement"}, *opts.Labels)
						return expectedIssue, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: &gl.Issue{
				ID:          123,
				IID:         1,
				ProjectID:   456,
				Title:       "Updated Title",
				Description: "Updated description",
				State:       "opened",
				UpdatedAt:   &timeNow,
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Update with stateEvent reopen",
			args: map[string]any{
				"projectId":  projectID,
				"issueIid":   issueIid,
				"stateEvent": "reopen",
			},
			mockSetup: func() {
				expectedIssue := &gl.Issue{
					ID:        123,
					IID:       1,
					ProjectID: 456,
					Title:     "Test Issue",
					State:     "opened",
					UpdatedAt: &timeNow,
				}
				mockIssues.EXPECT().
					UpdateIssue(projectID, 1, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gl.UpdateIssueOptions, _ ...gl.RequestOptionFunc) (*gl.Issue, *gl.Response, error) {
						assert.Equal(t, "reopen", *opts.StateEvent)
						return expectedIssue, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: &gl.Issue{
				ID:        123,
				IID:       1,
				ProjectID: 456,
				Title:     "Test Issue",
				State:     "opened",
				UpdatedAt: &timeNow,
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Update with milestoneId",
			args: map[string]any{
				"projectId":   projectID,
				"issueIid":    issueIid,
				"milestoneId": 10.0,
			},
			mockSetup: func() {
				expectedIssue := &gl.Issue{
					ID:        123,
					IID:       1,
					ProjectID: 456,
					Title:     "Test Issue",
					State:     "opened",
					UpdatedAt: &timeNow,
				}
				mockIssues.EXPECT().
					UpdateIssue(projectID, 1, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gl.UpdateIssueOptions, _ ...gl.RequestOptionFunc) (*gl.Issue, *gl.Response, error) {
						require.NotNil(t, opts.MilestoneID)
						assert.Equal(t, 10, *opts.MilestoneID)
						return expectedIssue, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: &gl.Issue{
				ID:        123,
				IID:       1,
				ProjectID: 456,
				Title:     "Test Issue",
				State:     "opened",
				UpdatedAt: &timeNow,
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Update with assigneeIds",
			args: map[string]any{
				"projectId":   projectID,
				"issueIid":    issueIid,
				"assigneeIds": "123,456",
			},
			mockSetup: func() {
				expectedIssue := &gl.Issue{
					ID:        123,
					IID:       1,
					ProjectID: 456,
					Title:     "Test Issue",
					State:     "opened",
					UpdatedAt: &timeNow,
				}
				mockIssues.EXPECT().
					UpdateIssue(projectID, 1, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gl.UpdateIssueOptions, _ ...gl.RequestOptionFunc) (*gl.Issue, *gl.Response, error) {
						require.NotNil(t, opts.AssigneeIDs)
						assert.Equal(t, []int{123, 456}, *opts.AssigneeIDs)
						return expectedIssue, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: &gl.Issue{
				ID:        123,
				IID:       1,
				ProjectID: 456,
				Title:     "Test Issue",
				State:     "opened",
				UpdatedAt: &timeNow,
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Error - Forbidden (403)",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  issueIid,
				"title":     "Updated Title",
			},
			mockSetup: func() {
				mockIssues.EXPECT().
					UpdateIssue(projectID, 1, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 403}}, errors.New("gitlab: 403 Forbidden"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to update issue",
		},
		{
			name: "Error - Unauthorized (401)",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  issueIid,
				"title":     "Updated Title",
			},
			mockSetup: func() {
				mockIssues.EXPECT().
					UpdateIssue(projectID, 1, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("gitlab: 401 Unauthorized"))
			},
			expectedResult:      "Authentication failed (401)",
			expectResultError:   true,
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
					Name:      updateIssueTool.Name,
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
					expectedIssue, ok := tc.expectedResult.(*gl.Issue)
					require.True(t, ok)
					expectedJSON, err := json.Marshal(expectedIssue)
					require.NoError(t, err)
					assert.JSONEq(t, string(expectedJSON), textContent.Text)
				}
			}
		})
	}
}

// TestCreateIssueCommentHandler tests the CreateIssueComment tool handler
func TestCreateIssueCommentHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockNotes, ctrl := setupMockClientForNotes(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	createCommentTool, handler := CreateIssueComment(mockGetClient, nil)

	projectID := "group/project"
	issueIid := 1.0
	timeNow := time.Now()

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
			name: "Success - Create Issue Comment",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  issueIid,
				"body":      "This is a comment",
			},
			mockSetup: func() {
				expectedNote := &gl.Note{
					ID:   123,
					Body: "This is a comment",
					Author: gl.NoteAuthor{
						Name: "Test User",
					},
					CreatedAt: &timeNow,
				}
				mockNotes.EXPECT().
					CreateIssueNote(projectID, 1, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gl.CreateIssueNoteOptions, _ ...gl.RequestOptionFunc) (*gl.Note, *gl.Response, error) {
						assert.Equal(t, "This is a comment", *opts.Body)
						return expectedNote, &gl.Response{Response: &http.Response{StatusCode: 201}}, nil
					})
			},
			expectedResult: &gl.Note{
				ID:   123,
				Body: "This is a comment",
				Author: gl.NoteAuthor{
					Name: "Test User",
				},
				CreatedAt: &timeNow,
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Error - Missing projectId",
			args: map[string]any{
				"issueIid": issueIid,
				"body":     "Comment",
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: projectId",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Missing body",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  issueIid,
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: body",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Issue Not Found (404)",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  999.0,
				"body":      "Comment",
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					CreateIssueNote(projectID, 999, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Issue Not Found"))
			},
			expectedResult:      "issue 999 in project \"group/project\" not found or access denied (404)",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - GitLab API Error (500)",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  issueIid,
				"body":      "Comment",
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					CreateIssueNote(projectID, 1, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to create comment issue",
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
					Name:      createCommentTool.Name,
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
					expectedNote, ok := tc.expectedResult.(*gl.Note)
					require.True(t, ok)
					expectedJSON, err := json.Marshal(expectedNote)
					require.NoError(t, err)
					assert.JSONEq(t, string(expectedJSON), textContent.Text)
				}
			}
		})
	}
}

// TestUpdateIssueCommentHandler tests the UpdateIssueComment tool handler
func TestUpdateIssueCommentHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockNotes, ctrl := setupMockClientForNotes(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	updateCommentTool, handler := UpdateIssueComment(mockGetClient, nil)

	projectID := "group/project"
	issueIid := 1.0
	noteID := 123.0
	timeNow := time.Now()

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
			name: "Success - Update Issue Comment",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  issueIid,
				"noteId":    noteID,
				"body":      "Updated comment",
			},
			mockSetup: func() {
				expectedNote := &gl.Note{
					ID:   123,
					Body: "Updated comment",
					Author: gl.NoteAuthor{
						Name: "Test User",
					},
					UpdatedAt: &timeNow,
				}
				mockNotes.EXPECT().
					UpdateIssueNote(projectID, 1, 123, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, _ int, opts *gl.UpdateIssueNoteOptions, _ ...gl.RequestOptionFunc) (*gl.Note, *gl.Response, error) {
						assert.Equal(t, "Updated comment", *opts.Body)
						return expectedNote, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: &gl.Note{
				ID:   123,
				Body: "Updated comment",
				Author: gl.NoteAuthor{
					Name: "Test User",
				},
				UpdatedAt: &timeNow,
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Error - Missing noteId",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  issueIid,
				"body":      "Comment",
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: noteId",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Issue or Note Not Found (404)",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  999.0,
				"noteId":    999.0,
				"body":      "Comment",
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					UpdateIssueNote(projectID, 999, 999, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Not Found"))
			},
			expectedResult:      "issue 999 or note 999 in project \"group/project\" not found or access denied (404)",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - GitLab API Error (500)",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  issueIid,
				"noteId":    noteID,
				"body":      "Comment",
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					UpdateIssueNote(projectID, 1, 123, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to update comment",
		},
		{
			name: "Success - Update with special characters",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  issueIid,
				"noteId":    noteID,
				"body":      "Updated comment with <html> & \"quotes\" and 'apostrophes'",
			},
			mockSetup: func() {
				expectedNote := &gl.Note{
					ID:   123,
					Body: "Updated comment with <html> & \"quotes\" and 'apostrophes'",
					Author: gl.NoteAuthor{
						Name: "Test User",
					},
					UpdatedAt: &timeNow,
				}
				mockNotes.EXPECT().
					UpdateIssueNote(projectID, 1, 123, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, _ int, opts *gl.UpdateIssueNoteOptions, _ ...gl.RequestOptionFunc) (*gl.Note, *gl.Response, error) {
						assert.Equal(t, "Updated comment with <html> & \"quotes\" and 'apostrophes'", *opts.Body)
						return expectedNote, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: &gl.Note{
				ID:   123,
				Body: "Updated comment with <html> & \"quotes\" and 'apostrophes'",
				Author: gl.NoteAuthor{
					Name: "Test User",
				},
				UpdatedAt: &timeNow,
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Error - Empty body validation",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  issueIid,
				"noteId":    noteID,
				"body":      "",
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: required parameter 'body' cannot be empty or zero value",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Forbidden (403)",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  issueIid,
				"noteId":    noteID,
				"body":      "Comment",
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					UpdateIssueNote(projectID, 1, 123, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 403}}, errors.New("gitlab: 403 Forbidden"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to update comment",
		},
		{
			name: "Error - Unauthorized (401)",
			args: map[string]any{
				"projectId": projectID,
				"issueIid":  issueIid,
				"noteId":    noteID,
				"body":      "Comment",
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					UpdateIssueNote(projectID, 1, 123, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("gitlab: 401 Unauthorized"))
			},
			expectedResult:      "Authentication failed (401). Your GitLab token may be expired. Please update it using the updateToken tool.",
			expectResultError:   true,
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
					Name:      updateCommentTool.Name,
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
					expectedNote, ok := tc.expectedResult.(*gl.Note)
					require.True(t, ok)
					expectedJSON, err := json.Marshal(expectedNote)
					require.NoError(t, err)
					assert.JSONEq(t, string(expectedJSON), textContent.Text)
				}
			}
		})
	}
}
