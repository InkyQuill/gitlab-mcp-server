package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gl "gitlab.com/gitlab-org/api/client-go"

	"github.com/InkyQuill/gitlab-mcp-server/internal/toolsnaps"
	"go.uber.org/mock/gomock"
)

// TestGetMergeRequestHandler tests the GetMergeRequest tool
func TestGetMergeRequestHandler(t *testing.T) {
	// Tool schema snapshot test
	tool, _ := GetMergeRequest(nil, nil)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockMRs, ctrl := setupMockClientForMergeRequests(t)
	defer ctrl.Finish()

	// Mock getClient function for merge request tests
	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	// Define the Tool and Handler
	getMergeRequestTool, getMergeRequestHandler := GetMergeRequest(mockGetClient, nil)

	// Test data
	projectID := "group/project"
	mrIID := 1.0 // MCP number type maps to float64
	timeNow := time.Now()

	// Create a sample merge request for testing
	sampleMR := &gl.MergeRequest{
		BasicMergeRequest: gl.BasicMergeRequest{
			ID:           123,
			IID:          int(mrIID),
			ProjectID:    456,
			Title:        "Implement feature X",
			Description:  "This adds feature X which does Y",
			State:        "opened",
			CreatedAt:    &timeNow,
			WebURL:       fmt.Sprintf("https://gitlab.com/%s/merge_requests/%d", projectID, int(mrIID)),
			SourceBranch: "feature-x",
			TargetBranch: "main",
		},
	}

	// Test cases
	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectedResult     interface{}
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - Get MR by ID",
			inputArgs: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": mrIID,
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					GetMergeRequest(projectID, int(mrIID), nil, gomock.Any()).
					Return(sampleMR, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: sampleMR,
		},
		{
			name: "Error - MR Not Found (404)",
			inputArgs: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": 999.0,
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					GetMergeRequest(projectID, 999, nil, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Not Found"))
			},
			expectResultError: true,
			errorContains:     "merge request 999 in project \"group/project\" not found or access denied (404)",
		},
		{
			name: "Error - GitLab API Error (500)",
			inputArgs: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": mrIID,
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					GetMergeRequest(projectID, int(mrIID), nil, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectHandlerError: true,
			errorContains:      "failed to process merge request",
		},
		{
			name: "Error - Missing projectId parameter",
			inputArgs: map[string]any{
				"mergeRequestIid": mrIID,
			},
			mockSetup:         func() { /* No API call expected */ },
			expectResultError: true,
			errorContains:     "Validation Error: missing required parameter: projectId",
		},
		{
			name: "Error - Missing mergeRequestIid parameter",
			inputArgs: map[string]any{
				"projectId": projectID,
			},
			mockSetup:         func() { /* No API call expected */ },
			expectResultError: true,
			errorContains:     "Validation Error: missing required parameter: mergeRequestIid",
		},
		{
			name: "Error - Invalid mergeRequestIid (not integer)",
			inputArgs: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": 1.5, // Non-integer value
			},
			mockSetup:         func() { /* No API call expected */ },
			expectResultError: true,
			errorContains:     "Validation Error: mergeRequestIid 1.5 is not a valid integer",
		},
	}

	// Run test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()

			// Create the request
			req := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      getMergeRequestTool.Name,
					Arguments: tc.inputArgs,
				},
			}

			// Call the handler
			result, err := getMergeRequestHandler(ctx, req)

			// Verify results
			if tc.expectHandlerError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				textContent := getTextResult(t, result)

				if tc.expectResultError {
					assert.Contains(t, textContent.Text, tc.errorContains, "Error message mismatch")
				} else {
					// For successful responses, verify the returned data
					var actualMR gl.MergeRequest
					err = json.Unmarshal([]byte(textContent.Text), &actualMR)
					require.NoError(t, err, "Failed to unmarshal actual result JSON")

					// Marshal both expected and actual for JSONEq comparison
					expectedJSON, _ := json.Marshal(tc.expectedResult)
					actualJSON, _ := json.Marshal(actualMR)
					assert.JSONEq(t, string(expectedJSON), string(actualJSON), "Merge request content mismatch")
				}
			}
		})
	}

	// Test for client initialization error
	t.Run("Error - Client Initialization Error", func(t *testing.T) {
		errorGetClientFn := func(_ context.Context) (*gl.Client, error) {
			return nil, fmt.Errorf("mock init error")
		}
		_, handler := GetMergeRequest(errorGetClientFn, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name: getMergeRequestTool.Name,
				Arguments: map[string]any{
					"projectId":       projectID,
					"mergeRequestIid": mrIID,
				},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get GitLab client")
		assert.Nil(t, result)
	})
}

// TestGetMergeRequestCommentsHandler tests the GetMergeRequestComments tool
func TestGetMergeRequestCommentsHandler(t *testing.T) {
	// Tool schema snapshot test
	tool, _ := MergeRequestComment(nil, nil)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	ctx := context.Background()

	// --- Setup Mock Client and GetClientFn once ---
	mockClient, mockNotes, ctrl := setupMockClientForNotes(t)
	defer ctrl.Finish()

	// Create the mock getClient function once, capturing the mockClient
	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	// --- Define the Tool and Handler once ---
	getMRCommentsTool, handler := MergeRequestComment(mockGetClient, nil)

	// Define common test data
	projectID := "group/project"
	mrIid := 1.0 // MCP number type maps to float64

	// Create time values to use in tests
	timeNow := time.Now()
	time24HoursAgo := timeNow.Add(-24 * time.Hour)
	time12HoursAgo := timeNow.Add(-12 * time.Hour)
	time6HoursAgo := timeNow.Add(-6 * time.Hour)

	// --- Test Cases ---
	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectedResult     interface{} // Either []*gl.Note for success or string for error message
		expectHandlerError bool        // Whether the handler itself should return an error
		expectResultError  bool        // Whether the returned mcp.CallToolResult should represent an error
		errorContains      string      // Substring to check in the error message
	}{
		// --- Success Cases ---
		{
			name: "Success - Get Merge Request Comments",
			inputArgs: map[string]any{
				"action":          "list",
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
			},
			mockSetup: func() {
				// Create expected notes for the response with correct fields
				expectedNotes := []*gl.Note{
					{
						ID:   123,
						Body: "This looks good to me",
						Author: gl.NoteAuthor{
							Name: "Test User",
						},
						CreatedAt: &time24HoursAgo,
					},
					{
						ID:   124,
						Body: "I have a suggestion for this line",
						Author: gl.NoteAuthor{
							Name: "Another User",
						},
						CreatedAt: &time12HoursAgo,
					},
				}

				mockNotes.EXPECT().
					ListMergeRequestNotes(projectID, int(mrIid), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gl.ListMergeRequestNotesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Note, *gl.Response, error) {
						// Verify pagination settings
						assert.Equal(t, 1, opts.Page)
						assert.Equal(t, DefaultPerPage, opts.PerPage)

						return expectedNotes, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Note{
				{
					ID:   123,
					Body: "This looks good to me",
					Author: gl.NoteAuthor{
						Name: "Test User",
					},
					CreatedAt: &time24HoursAgo,
				},
				{
					ID:   124,
					Body: "I have a suggestion for this line",
					Author: gl.NoteAuthor{
						Name: "Another User",
					},
					CreatedAt: &time12HoursAgo,
				},
			},
		},
		{
			name: "Success - Merge Request Comments With Pagination",
			inputArgs: map[string]any{
				"action":          "list",
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
				"page":            2,
				"per_page":        5,
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
					ListMergeRequestNotes(projectID, int(mrIid), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gl.ListMergeRequestNotesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Note, *gl.Response, error) {
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
		},
		{
			name: "Success - Empty Comments",
			inputArgs: map[string]any{
				"action":          "list",
				"projectId":       projectID,
				"mergeRequestIid": 2.0, // Different MR with no comments
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					ListMergeRequestNotes(projectID, 2, gomock.Any(), gomock.Any()).
					Return([]*gl.Note{}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: []*gl.Note{}, // Empty array
		},
		// --- Error Cases ---
		{
			name: "Error - Merge Request Not Found (404)",
			inputArgs: map[string]any{
				"action":          "list",
				"projectId":       projectID,
				"mergeRequestIid": 999.0,
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					ListMergeRequestNotes(projectID, 999, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Merge Request Not Found"))
			},
			expectResultError: true,
			errorContains:     "merge request 999 in project \"group/project\" not found or access denied (404)",
		},
		{
			name: "Error - GitLab API Error (500)",
			inputArgs: map[string]any{
				"action":          "list",
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					ListMergeRequestNotes(projectID, int(mrIid), gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectHandlerError: true,
			errorContains:      "failed to process comments for merge request",
		},
		{
			name: "Error - Missing projectId parameter",
			inputArgs: map[string]any{
				"mergeRequestIid": mrIid,
			}, // Missing projectId
			mockSetup:         func() { /* No API call expected */ },
			expectResultError: true,
			errorContains:     "Validation Error: missing required parameter: projectId",
		},
		{
			name: "Error - Missing mergeRequestIid parameter",
			inputArgs: map[string]any{
				"projectId": projectID,
			}, // Missing mergeRequestIid
			mockSetup:         func() { /* No API call expected */ },
			expectResultError: true,
			errorContains:     "Validation Error: missing required parameter: mergeRequestIid",
		},
		{
			name: "Error - Invalid mergeRequestIid (not integer)",
			inputArgs: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": 1.5,
			}, // Non-integer float
			mockSetup:         func() { /* No API call expected */ },
			expectResultError: true,
			errorContains:     "Validation Error: mergeRequestIid 1.5 is not a valid integer",
		},
	}

	// --- Run Test Cases ---
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
					Name:      getMRCommentsTool.Name,
					Arguments: tc.inputArgs,
				},
			}

			// Execute handler
			result, err := handler(ctx, request)

			// Validate results following the pattern
			if tc.expectHandlerError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				textContent := getTextResult(t, result)

				if tc.expectResultError {
					assert.Contains(t, textContent.Text, tc.errorContains, "Error message mismatch")
				} else {
					// Handle special case for empty array
					if notes, ok := tc.expectedResult.([]*gl.Note); ok && len(notes) == 0 {
						assert.Equal(t, "[]", textContent.Text, "Empty array mismatch")
					} else {
						// Unmarshal expected and actual results
						var actualNotes []*gl.Note
						err = json.Unmarshal([]byte(textContent.Text), &actualNotes)
						require.NoError(t, err, "Failed to unmarshal actual result JSON")

						// Compare lengths first
						expectedNotes, _ := tc.expectedResult.([]*gl.Note)
						require.Equal(t, len(expectedNotes), len(actualNotes), "Number of notes mismatch")

						// Compare content using JSONEq
						expectedJSON, _ := json.Marshal(tc.expectedResult)
						actualJSON, _ := json.Marshal(actualNotes)
						assert.JSONEq(t, string(expectedJSON), string(actualJSON), "Notes content mismatch")
					}
				}
			}
		})
	}

	// Test case for Client Initialization Error (outside the loop)
	t.Run("Error - Client Initialization Error", func(t *testing.T) {
		// Define a GetClientFn that returns an error
		errorGetClientFn := func(_ context.Context) (*gl.Client, error) {
			return nil, fmt.Errorf("mock init error")
		}
		_, handler := MergeRequestComment(errorGetClientFn, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name: getMRCommentsTool.Name,
				Arguments: map[string]any{
					"action":          "list",
					"projectId":       projectID,
					"mergeRequestIid": mrIid,
				},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize GitLab client")
		assert.Nil(t, result)
	})
}

// TestListMergeRequestsHandler tests the ListMergeRequests tool
func TestListMergeRequestsHandler(t *testing.T) {
	// Tool schema snapshot test
	tool, _ := ListMergeRequests(nil, nil)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockMRs, ctrl := setupMockClientForMergeRequests(t)
	defer ctrl.Finish()

	// Mock getClient function for merge request tests
	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	// Define the Tool and Handler
	listMergeRequestsTool, listMergeRequestsHandler := ListMergeRequests(mockGetClient, nil)

	// Test data
	projectID := "group/project"
	timeNow := time.Now()
	time24HoursAgo := timeNow.Add(-24 * time.Hour)

	// Sample merge requests for testing
	sampleMRs := []*gl.MergeRequest{
		{
			BasicMergeRequest: gl.BasicMergeRequest{
				ID:           123,
				IID:          1,
				ProjectID:    456,
				Title:        "Implement feature X",
				Description:  "This adds feature X which does Y",
				State:        "opened",
				CreatedAt:    &time24HoursAgo,
				SourceBranch: "feature-x",
				TargetBranch: "main",
			},
		},
		{
			BasicMergeRequest: gl.BasicMergeRequest{
				ID:           124,
				IID:          2,
				ProjectID:    456,
				Title:        "Fix bug Z",
				Description:  "This fixes critical bug Z",
				State:        "opened",
				CreatedAt:    &timeNow,
				SourceBranch: "fix-bug-z",
				TargetBranch: "main",
			},
		},
	}

	// Test cases
	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectedResult     interface{}
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - List Project MRs",
			inputArgs: map[string]any{
				"projectId": projectID,
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(proj any, opts *gl.ListProjectMergeRequestsOptions, _ ...gl.RequestOptionFunc) ([]*gl.BasicMergeRequest, *gl.Response, error) {
						// Verify default pagination
						assert.Equal(t, projectID, proj)
						assert.Equal(t, 1, opts.Page)
						assert.Equal(t, DefaultPerPage, opts.PerPage)

						return convertToBasicMRs(sampleMRs), &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: sampleMRs,
		},
		{
			name: "Success - List Project MRs with Filtering",
			inputArgs: map[string]any{
				"projectId": projectID,
				"state":     "opened",
				"labels":    "bug,critical",
				"sort":      "desc",
				"order_by":  "created_at",
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(proj any, opts *gl.ListProjectMergeRequestsOptions, _ ...gl.RequestOptionFunc) ([]*gl.BasicMergeRequest, *gl.Response, error) {
						// Verify filter params
						assert.Equal(t, projectID, proj)
						assert.Equal(t, "opened", *opts.State)
						assert.ElementsMatch(t, gl.LabelOptions{"bug", "critical"}, *opts.Labels)
						assert.Equal(t, "desc", *opts.Sort)
						assert.Equal(t, "created_at", *opts.OrderBy)

						return convertToBasicMRs([]*gl.MergeRequest{sampleMRs[1]}), &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.MergeRequest{sampleMRs[1]},
		},
		{
			name: "Success - List Project MRs with Pagination",
			inputArgs: map[string]any{
				"projectId": projectID,
				"page":      2,
				"per_page":  5,
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(proj any, opts *gl.ListProjectMergeRequestsOptions, _ ...gl.RequestOptionFunc) ([]*gl.BasicMergeRequest, *gl.Response, error) {
						// Verify pagination
						assert.Equal(t, projectID, proj)
						assert.Equal(t, 2, opts.Page)
						assert.Equal(t, 5, opts.PerPage)

						return convertToBasicMRs([]*gl.MergeRequest{sampleMRs[1]}), &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.MergeRequest{sampleMRs[1]},
		},
		{
			name: "Success - Empty MRs List",
			inputArgs: map[string]any{
				"projectId": projectID,
				"state":     "merged", // Filter that returns no results
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]*gl.BasicMergeRequest{}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: []*gl.MergeRequest{},
		},
		{
			name: "Error - Project Not Found (404)",
			inputArgs: map[string]any{
				"projectId": "nonexistent/project",
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Project Not Found"))
			},
			expectResultError:  false,
			expectHandlerError: true,
			errorContains:      "failed to list merge requests for project \"nonexistent/project\"",
		},
		{
			name: "Error - GitLab API Error (500)",
			inputArgs: map[string]any{
				"projectId": projectID,
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectHandlerError: true,
			errorContains:      "failed to list merge requests",
		},
		{
			name:              "Error - Missing projectId parameter",
			inputArgs:         map[string]any{},
			mockSetup:         func() { /* No API call expected */ },
			expectResultError: true,
			errorContains:     "Validation Error: missing required parameter: projectId",
		},
		{
			name: "Error - Invalid date format",
			inputArgs: map[string]any{
				"projectId":     projectID,
				"created_after": "not-a-date",
			},
			mockSetup:         func() { /* No API call expected */ },
			expectResultError: true,
			errorContains:     "must be a valid ISO 8601 datetime",
		},
		{
			name: "Success - Filter by milestone",
			inputArgs: map[string]any{
				"projectId": projectID,
				"milestone": "v1.0",
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(proj any, opts *gl.ListProjectMergeRequestsOptions, _ ...gl.RequestOptionFunc) ([]*gl.BasicMergeRequest, *gl.Response, error) {
						require.NotNil(t, opts.Milestone)
						assert.Equal(t, "v1.0", *opts.Milestone)
						return convertToBasicMRs([]*gl.MergeRequest{sampleMRs[0]}), &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.MergeRequest{sampleMRs[0]},
		},
		{
			name: "Success - Filter by scope",
			inputArgs: map[string]any{
				"projectId": projectID,
				"scope":     "created_by_me",
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(proj any, opts *gl.ListProjectMergeRequestsOptions, _ ...gl.RequestOptionFunc) ([]*gl.BasicMergeRequest, *gl.Response, error) {
						require.NotNil(t, opts.Scope)
						assert.Equal(t, "created_by_me", *opts.Scope)
						return convertToBasicMRs([]*gl.MergeRequest{sampleMRs[0]}), &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.MergeRequest{sampleMRs[0]},
		},
		{
			name: "Success - Filter by author ID",
			inputArgs: map[string]any{
				"projectId": projectID,
				"author_id": "123",
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(proj any, opts *gl.ListProjectMergeRequestsOptions, _ ...gl.RequestOptionFunc) ([]*gl.BasicMergeRequest, *gl.Response, error) {
						require.NotNil(t, opts.AuthorID)
						assert.Equal(t, 123, *opts.AuthorID)
						return convertToBasicMRs([]*gl.MergeRequest{sampleMRs[0]}), &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.MergeRequest{sampleMRs[0]},
		},
		{
			name: "Success - Filter by assignee ID",
			inputArgs: map[string]any{
				"projectId":   projectID,
				"assignee_id": "456",
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(proj any, opts *gl.ListProjectMergeRequestsOptions, _ ...gl.RequestOptionFunc) ([]*gl.BasicMergeRequest, *gl.Response, error) {
						require.NotNil(t, opts.AssigneeID)
						// AssigneeIDValue is a custom type, just verify it's set
						return convertToBasicMRs([]*gl.MergeRequest{sampleMRs[1]}), &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.MergeRequest{sampleMRs[1]},
		},
		{
			name: "Success - Filter by search",
			inputArgs: map[string]any{
				"projectId": projectID,
				"search":    "feature",
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(proj any, opts *gl.ListProjectMergeRequestsOptions, _ ...gl.RequestOptionFunc) ([]*gl.BasicMergeRequest, *gl.Response, error) {
						require.NotNil(t, opts.Search)
						assert.Equal(t, "feature", *opts.Search)
						return convertToBasicMRs([]*gl.MergeRequest{sampleMRs[0]}), &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.MergeRequest{sampleMRs[0]},
		},
		{
			name: "Success - Filter by date range (created_after)",
			inputArgs: map[string]any{
				"projectId":     projectID,
				"created_after": "2024-01-01T00:00:00Z",
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(proj any, opts *gl.ListProjectMergeRequestsOptions, _ ...gl.RequestOptionFunc) ([]*gl.BasicMergeRequest, *gl.Response, error) {
						require.NotNil(t, opts.CreatedAfter)
						assert.Equal(t, "2024-01-01T00:00:00Z", opts.CreatedAfter.Format(time.RFC3339))
						return convertToBasicMRs(sampleMRs), &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: sampleMRs,
		},
		{
			name: "Success - Filter by date range (created_before)",
			inputArgs: map[string]any{
				"projectId":      projectID,
				"created_before": "2024-12-31T23:59:59Z",
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(proj any, opts *gl.ListProjectMergeRequestsOptions, _ ...gl.RequestOptionFunc) ([]*gl.BasicMergeRequest, *gl.Response, error) {
						require.NotNil(t, opts.CreatedBefore)
						assert.Equal(t, "2024-12-31T23:59:59Z", opts.CreatedBefore.Format(time.RFC3339))
						return convertToBasicMRs([]*gl.MergeRequest{sampleMRs[0]}), &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.MergeRequest{sampleMRs[0]},
		},
		{
			name: "Success - Filter by date range (updated_after)",
			inputArgs: map[string]any{
				"projectId":     projectID,
				"updated_after": "2024-01-01T00:00:00Z",
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(proj any, opts *gl.ListProjectMergeRequestsOptions, _ ...gl.RequestOptionFunc) ([]*gl.BasicMergeRequest, *gl.Response, error) {
						require.NotNil(t, opts.UpdatedAfter)
						assert.Equal(t, "2024-01-01T00:00:00Z", opts.UpdatedAfter.Format(time.RFC3339))
						return convertToBasicMRs([]*gl.MergeRequest{sampleMRs[1]}), &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.MergeRequest{sampleMRs[1]},
		},
		{
			name: "Success - Filter by date range (updated_before)",
			inputArgs: map[string]any{
				"projectId":      projectID,
				"updated_before": "2024-12-31T23:59:59Z",
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(proj any, opts *gl.ListProjectMergeRequestsOptions, _ ...gl.RequestOptionFunc) ([]*gl.BasicMergeRequest, *gl.Response, error) {
						require.NotNil(t, opts.UpdatedBefore)
						assert.Equal(t, "2024-12-31T23:59:59Z", opts.UpdatedBefore.Format(time.RFC3339))
						return convertToBasicMRs([]*gl.MergeRequest{sampleMRs[0]}), &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.MergeRequest{sampleMRs[0]},
		},
		{
			name: "Success - Combined multiple filters",
			inputArgs: map[string]any{
				"projectId":     projectID,
				"state":         "opened",
				"labels":        "bug",
				"milestone":     "v1.0",
				"search":        "fix",
				"author_id":     "123",
				"assignee_id":   "456",
				"created_after": "2024-01-01T00:00:00Z",
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(proj any, opts *gl.ListProjectMergeRequestsOptions, _ ...gl.RequestOptionFunc) ([]*gl.BasicMergeRequest, *gl.Response, error) {
						require.NotNil(t, opts.State)
						assert.Equal(t, "opened", *opts.State)
						require.NotNil(t, opts.Labels)
						assert.ElementsMatch(t, gl.LabelOptions{"bug"}, *opts.Labels)
						require.NotNil(t, opts.Milestone)
						assert.Equal(t, "v1.0", *opts.Milestone)
						require.NotNil(t, opts.Search)
						assert.Equal(t, "fix", *opts.Search)
						require.NotNil(t, opts.AuthorID)
						assert.Equal(t, 123, *opts.AuthorID)
						require.NotNil(t, opts.AssigneeID)
						// AssigneeIDValue is a custom type, just verify it's set
						require.NotNil(t, opts.CreatedAfter)
						assert.Equal(t, "2024-01-01T00:00:00Z", opts.CreatedAfter.Format(time.RFC3339))
						return convertToBasicMRs([]*gl.MergeRequest{sampleMRs[1]}), &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.MergeRequest{sampleMRs[1]},
		},
		{
			name: "Success - Long Description Truncation",
			inputArgs: map[string]any{
				"projectId": projectID,
			},
			mockSetup: func() {
				longDesc := strings.Repeat("b", 500)
				mrWithLongDesc := &gl.MergeRequest{
					BasicMergeRequest: gl.BasicMergeRequest{
						ID:           125,
						IID:          3,
						ProjectID:    456,
						Title:        "MR with long description",
						Description:  longDesc,
						State:        "opened",
						SourceBranch: "feature",
						TargetBranch: "main",
					},
				}
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(convertToBasicMRs([]*gl.MergeRequest{mrWithLongDesc}), &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: []*gl.MergeRequest{
				{
					BasicMergeRequest: gl.BasicMergeRequest{
						ID:           125,
						IID:          3,
						ProjectID:    456,
						Title:        "MR with long description",
						Description:  strings.Repeat("b", 300) + "...",
						State:        "opened",
						SourceBranch: "feature",
						TargetBranch: "main",
					},
				},
			},
		},
	}

	// Run test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()

			// Create the request
			req := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      listMergeRequestsTool.Name,
					Arguments: tc.inputArgs,
				},
			}

			// Call the handler
			result, err := listMergeRequestsHandler(ctx, req)

			// Verify results
			if tc.expectHandlerError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				textContent := getTextResult(t, result)

				if tc.expectResultError {
					assert.Contains(t, textContent.Text, tc.errorContains, "Error message mismatch")
				} else {
					// For empty lists, check for empty PaginatedResponse
					mrList, ok := tc.expectedResult.([]*gl.MergeRequest)
					if ok && len(mrList) == 0 {
						var emptyResp PaginatedResponse
						err = json.Unmarshal([]byte(textContent.Text), &emptyResp)
						require.NoError(t, err, "Failed to unmarshal empty response")
						assert.Empty(t, emptyResp.Items, "Empty response should have empty items")
					} else {
						// For successful responses with data, verify key fields
						expectedMRs, ok := tc.expectedResult.([]*gl.MergeRequest)
						require.True(t, ok, "Expected result should be []*gl.MergeRequest")

						// Unmarshal actual response as PaginatedResponse
						var actualResp PaginatedResponse
						err = json.Unmarshal([]byte(textContent.Text), &actualResp)
						require.NoError(t, err, "Failed to unmarshal actual response as PaginatedResponse")

						// Convert items to []interface{} first, then to []map for field checking
						actualItemsSlice, ok := actualResp.Items.([]interface{})
						require.True(t, ok, "Items should be []interface{}")

						// Check that we have the expected number of MRs
						assert.Equal(t, len(expectedMRs), len(actualItemsSlice), "Number of MRs should match")

						// Verify key fields are present in each MR
						for i, expectedMR := range expectedMRs {
							if i < len(actualItemsSlice) {
								actualMR, ok := actualItemsSlice[i].(map[string]interface{})
								require.True(t, ok, "Item should be map[string]interface{}")

								assert.Equal(t, expectedMR.ID, int(actualMR["id"].(float64)), "ID should match")
								assert.Equal(t, expectedMR.IID, int(actualMR["iid"].(float64)), "IID should match")
								assert.Equal(t, expectedMR.Title, actualMR["title"], "Title should match")

								// Verify that unwanted fields are removed
								assert.NotContains(t, actualMR, "_links", "Should not contain _links")
								assert.NotContains(t, actualMR, "web_url", "Should not contain web_url")
								assert.NotContains(t, actualMR, "diff_refs", "Should not contain diff_refs")
								assert.NotContains(t, actualMR, "label_details", "Should not contain label_details")
							}
						}
					}
				}
			}
		})
	}

	// Test for client initialization error
	t.Run("Error - Client Initialization Error", func(t *testing.T) {
		errorGetClientFn := func(_ context.Context) (*gl.Client, error) {
			return nil, fmt.Errorf("mock init error")
		}
		_, handler := ListMergeRequests(errorGetClientFn, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name: listMergeRequestsTool.Name,
				Arguments: map[string]any{
					"projectId": projectID,
				},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize GitLab client")
		assert.Nil(t, result)
	})
}

func convertToBasicMRs(mrs []*gl.MergeRequest) []*gl.BasicMergeRequest {
	var basicMRs []*gl.BasicMergeRequest
	for _, mr := range mrs {
		basicMRs = append(basicMRs, &gl.BasicMergeRequest{
			ID:           mr.ID,
			IID:          mr.IID,
			ProjectID:    mr.ProjectID,
			Title:        mr.Title,
			Description:  mr.Description,
			State:        mr.State,
			CreatedAt:    mr.CreatedAt,
			UpdatedAt:    mr.UpdatedAt,
			SourceBranch: mr.SourceBranch,
			TargetBranch: mr.TargetBranch,
		})
	}
	return basicMRs
}

// TestCreateMergeRequestHandler tests the CreateMergeRequest tool handler
func TestCreateMergeRequestHandler(t *testing.T) {
	// Tool schema snapshot test
	tool, _ := CreateMergeRequest(nil, nil)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockMRs, ctrl := setupMockClientForMergeRequests(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	createMRTool, handler := CreateMergeRequest(mockGetClient, nil)

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
			name: "Success - Create Merge Request with minimal fields",
			args: map[string]any{
				"projectId":    projectID,
				"sourceBranch": "feature-branch",
				"targetBranch": "main",
				"title":        "New MR",
			},
			mockSetup: func() {
				expectedMR := &gl.MergeRequest{
					BasicMergeRequest: gl.BasicMergeRequest{
						ID:           123,
						IID:          1,
						ProjectID:    456,
						Title:        "New MR",
						SourceBranch: "feature-branch",
						TargetBranch: "main",
						State:        "opened",
						CreatedAt:    &timeNow,
					},
				}
				mockMRs.EXPECT().
					CreateMergeRequest(projectID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gl.CreateMergeRequestOptions, _ ...gl.RequestOptionFunc) (*gl.MergeRequest, *gl.Response, error) {
						assert.Equal(t, "New MR", *opts.Title)
						assert.Equal(t, "feature-branch", *opts.SourceBranch)
						assert.Equal(t, "main", *opts.TargetBranch)
						return expectedMR, &gl.Response{Response: &http.Response{StatusCode: 201}}, nil
					})
			},
			expectedResult: &gl.MergeRequest{
				BasicMergeRequest: gl.BasicMergeRequest{
					ID:           123,
					IID:          1,
					ProjectID:    456,
					Title:        "New MR",
					SourceBranch: "feature-branch",
					TargetBranch: "main",
					State:        "opened",
					CreatedAt:    &timeNow,
				},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Create Merge Request with all fields",
			args: map[string]any{
				"projectId":          projectID,
				"sourceBranch":       "feature-branch",
				"targetBranch":       "main",
				"title":              "Complete MR",
				"description":        "This is a complete MR",
				"labels":             "bug,critical",
				"assigneeIds":        "1,2",
				"milestoneId":        5.0,
				"removeSourceBranch": true,
				"squash":             true,
			},
			mockSetup: func() {
				expectedMR := &gl.MergeRequest{
					BasicMergeRequest: gl.BasicMergeRequest{
						ID:           124,
						IID:          2,
						ProjectID:    456,
						Title:        "Complete MR",
						Description:  "This is a complete MR",
						SourceBranch: "feature-branch",
						TargetBranch: "main",
						State:        "opened",
						CreatedAt:    &timeNow,
					},
				}
				mockMRs.EXPECT().
					CreateMergeRequest(projectID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gl.CreateMergeRequestOptions, _ ...gl.RequestOptionFunc) (*gl.MergeRequest, *gl.Response, error) {
						assert.Equal(t, "Complete MR", *opts.Title)
						assert.Equal(t, "This is a complete MR", *opts.Description)
						assert.NotNil(t, opts.Labels)
						assert.NotNil(t, opts.AssigneeIDs)
						assert.Equal(t, 2, len(*opts.AssigneeIDs))
						assert.Equal(t, 5, *opts.MilestoneID)
						assert.NotNil(t, opts.RemoveSourceBranch)
						assert.True(t, *opts.RemoveSourceBranch)
						assert.NotNil(t, opts.Squash)
						assert.True(t, *opts.Squash)
						return expectedMR, &gl.Response{Response: &http.Response{StatusCode: 201}}, nil
					})
			},
			expectedResult: &gl.MergeRequest{
				BasicMergeRequest: gl.BasicMergeRequest{
					ID:           124,
					IID:          2,
					ProjectID:    456,
					Title:        "Complete MR",
					Description:  "This is a complete MR",
					SourceBranch: "feature-branch",
					TargetBranch: "main",
					State:        "opened",
					CreatedAt:    &timeNow,
				},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Error - Missing projectId",
			args: map[string]any{
				"sourceBranch": "feature-branch",
				"targetBranch": "main",
				"title":        "New MR",
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: projectId",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Missing sourceBranch",
			args: map[string]any{
				"projectId":    projectID,
				"targetBranch": "main",
				"title":        "New MR",
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: sourceBranch",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Project Not Found (404)",
			args: map[string]any{
				"projectId":    "nonexistent",
				"sourceBranch": "feature-branch",
				"targetBranch": "main",
				"title":        "New MR",
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					CreateMergeRequest("nonexistent", gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Project Not Found"))
			},
			expectedResult:      "project \"nonexistent\" not found or access denied (404)",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - GitLab API Error (500)",
			args: map[string]any{
				"projectId":    projectID,
				"sourceBranch": "feature-branch",
				"targetBranch": "main",
				"title":        "New MR",
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					CreateMergeRequest(projectID, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to create merge request project",
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
					Name:      createMRTool.Name,
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
					expectedMR, ok := tc.expectedResult.(*gl.MergeRequest)
					require.True(t, ok)
					expectedJSON, err := json.Marshal(expectedMR)
					require.NoError(t, err)
					assert.JSONEq(t, string(expectedJSON), textContent.Text)
				}
			}
		})
	}
}

// TestUpdateMergeRequestHandler tests the UpdateMergeRequest tool handler
func TestUpdateMergeRequestHandler(t *testing.T) {
	// Tool schema snapshot test
	tool, _ := UpdateMergeRequest(nil, nil)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockMRs, ctrl := setupMockClientForMergeRequests(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	updateMRTool, handler := UpdateMergeRequest(mockGetClient, nil)

	projectID := "group/project"
	mrIid := 1.0
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
			name: "Success - Update Merge Request title",
			args: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
				"title":           "Updated MR Title",
			},
			mockSetup: func() {
				expectedMR := &gl.MergeRequest{
					BasicMergeRequest: gl.BasicMergeRequest{
						ID:        123,
						IID:       1,
						ProjectID: 456,
						Title:     "Updated MR Title",
						State:     "opened",
						UpdatedAt: &timeNow,
					},
				}
				mockMRs.EXPECT().
					UpdateMergeRequest(projectID, 1, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gl.UpdateMergeRequestOptions, _ ...gl.RequestOptionFunc) (*gl.MergeRequest, *gl.Response, error) {
						assert.Equal(t, "Updated MR Title", *opts.Title)
						return expectedMR, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: &gl.MergeRequest{
				BasicMergeRequest: gl.BasicMergeRequest{
					ID:        123,
					IID:       1,
					ProjectID: 456,
					Title:     "Updated MR Title",
					State:     "opened",
					UpdatedAt: &timeNow,
				},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Update Merge Request with stateEvent",
			args: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
				"stateEvent":      "close",
			},
			mockSetup: func() {
				expectedMR := &gl.MergeRequest{
					BasicMergeRequest: gl.BasicMergeRequest{
						ID:        123,
						IID:       1,
						ProjectID: 456,
						Title:     "Test MR",
						State:     "closed",
						UpdatedAt: &timeNow,
					},
				}
				mockMRs.EXPECT().
					UpdateMergeRequest(projectID, 1, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gl.UpdateMergeRequestOptions, _ ...gl.RequestOptionFunc) (*gl.MergeRequest, *gl.Response, error) {
						assert.Equal(t, "close", *opts.StateEvent)
						return expectedMR, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: &gl.MergeRequest{
				BasicMergeRequest: gl.BasicMergeRequest{
					ID:        123,
					IID:       1,
					ProjectID: 456,
					Title:     "Test MR",
					State:     "closed",
					UpdatedAt: &timeNow,
				},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Error - Missing projectId",
			args: map[string]any{
				"mergeRequestIid": mrIid,
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: projectId",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - MR Not Found (404)",
			args: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": 999.0,
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					UpdateMergeRequest(projectID, 999, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 MR Not Found"))
			},
			expectedResult:      "merge request 999 in project \"group/project\" not found or access denied (404)",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - GitLab API Error (500)",
			args: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					UpdateMergeRequest(projectID, 1, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to update merge request",
		},
		{
			name: "Error - Invalid assignee IDs format",
			args: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
				"assigneeIds":     "123,abc,456",
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: invalid assignee ID \"abc\"",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Invalid milestoneId (not integer)",
			args: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
				"milestoneId":     1.5,
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: milestoneId 1.5 is not a valid integer",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Invalid mergeRequestIid (not integer)",
			args: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": 1.5,
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: mergeRequestIid 1.5 is not a valid integer",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Success - Update with all optional fields",
			args: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
				"title":           "Updated MR Title",
				"description":     "Updated description",
				"labels":          "bug,enhancement",
				"assigneeIds":     "123,456",
				"milestoneId":     5.0,
			},
			mockSetup: func() {
				expectedMR := &gl.MergeRequest{
					BasicMergeRequest: gl.BasicMergeRequest{
						ID:        123,
						IID:       1,
						ProjectID: 456,
						Title:     "Updated MR Title",
						UpdatedAt: &timeNow,
					},
				}
				mockMRs.EXPECT().
					UpdateMergeRequest(projectID, 1, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gl.UpdateMergeRequestOptions, _ ...gl.RequestOptionFunc) (*gl.MergeRequest, *gl.Response, error) {
						assert.Equal(t, "Updated MR Title", *opts.Title)
						assert.Equal(t, "Updated description", *opts.Description)
						assert.NotNil(t, opts.Labels)
						assert.NotNil(t, opts.AssigneeIDs)
						assert.Equal(t, 5, *opts.MilestoneID)
						return expectedMR, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: &gl.MergeRequest{
				BasicMergeRequest: gl.BasicMergeRequest{
					ID:        123,
					IID:       1,
					ProjectID: 456,
					Title:     "Updated MR Title",
					UpdatedAt: &timeNow,
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
					Name:      updateMRTool.Name,
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
					expectedMR, ok := tc.expectedResult.(*gl.MergeRequest)
					require.True(t, ok)
					expectedJSON, err := json.Marshal(expectedMR)
					require.NoError(t, err)
					assert.JSONEq(t, string(expectedJSON), textContent.Text)
				}
			}
		})
	}
}

// TestCreateMergeRequestCommentHandler tests the CreateMergeRequestComment tool handler
func TestCreateMergeRequestCommentHandler(t *testing.T) {
	// Tool schema snapshot test
	tool, _ := MergeRequestComment(nil, nil)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockNotes, ctrl := setupMockClientForNotes(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	createCommentTool, handler := MergeRequestComment(mockGetClient, nil)

	projectID := "group/project"
	mrIid := 1.0
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
			name: "Success - Create Merge Request Comment",
			args: map[string]any{
				"action":          "create",
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
				"body":            "This is a comment on MR",
			},
			mockSetup: func() {
				expectedNote := &gl.Note{
					ID:   123,
					Body: "This is a comment on MR",
					Author: gl.NoteAuthor{
						Name: "Test User",
					},
					CreatedAt: &timeNow,
				}
				mockNotes.EXPECT().
					CreateMergeRequestNote(projectID, 1, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gl.CreateMergeRequestNoteOptions, _ ...gl.RequestOptionFunc) (*gl.Note, *gl.Response, error) {
						assert.Equal(t, "This is a comment on MR", *opts.Body)
						return expectedNote, &gl.Response{Response: &http.Response{StatusCode: 201}}, nil
					})
			},
			expectedResult: &gl.Note{
				ID:   123,
				Body: "This is a comment on MR",
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
				"mergeRequestIid": mrIid,
				"body":            "Comment",
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: projectId",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - MR Not Found (404)",
			args: map[string]any{
				"action":          "create",
				"projectId":       projectID,
				"mergeRequestIid": 999.0,
				"body":            "Comment",
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					CreateMergeRequestNote(projectID, 999, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 MR Not Found"))
			},
			expectedResult:      "merge request 999 in project \"group/project\" not found or access denied (404)",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - GitLab API Error (500)",
			args: map[string]any{
				"action":          "create",
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
				"body":            "Comment",
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					CreateMergeRequestNote(projectID, 1, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to create comment merge request",
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

// TestUpdateMergeRequestCommentHandler tests the UpdateMergeRequestComment tool handler
func TestUpdateMergeRequestCommentHandler(t *testing.T) {
	// Tool schema snapshot test
	tool, _ := MergeRequestComment(nil, nil)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockNotes, ctrl := setupMockClientForNotes(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	updateCommentTool, handler := MergeRequestComment(mockGetClient, nil)

	projectID := "group/project"
	mrIid := 1.0
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
			name: "Success - Update Merge Request Comment",
			args: map[string]any{
				"action":          "update",
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
				"noteId":          noteID,
				"body":            "Updated comment on MR",
			},
			mockSetup: func() {
				expectedNote := &gl.Note{
					ID:   123,
					Body: "Updated comment on MR",
					Author: gl.NoteAuthor{
						Name: "Test User",
					},
					UpdatedAt: &timeNow,
				}
				mockNotes.EXPECT().
					UpdateMergeRequestNote(projectID, 1, 123, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, _ int, opts *gl.UpdateMergeRequestNoteOptions, _ ...gl.RequestOptionFunc) (*gl.Note, *gl.Response, error) {
						assert.Equal(t, "Updated comment on MR", *opts.Body)
						return expectedNote, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: &gl.Note{
				ID:   123,
				Body: "Updated comment on MR",
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
				"action":          "update",
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
				"body":            "Comment",
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: noteId",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - MR or Note Not Found (404)",
			args: map[string]any{
				"action":          "update",
				"projectId":       projectID,
				"mergeRequestIid": 999.0,
				"noteId":          999.0,
				"body":            "Comment",
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					UpdateMergeRequestNote(projectID, 999, 999, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Not Found"))
			},
			expectedResult:      "merge request 999 or note 999 in project \"group/project\" not found or access denied (404)",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - GitLab API Error (500)",
			args: map[string]any{
				"action":          "update",
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
				"noteId":          noteID,
				"body":            "Comment",
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					UpdateMergeRequestNote(projectID, 1, 123, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to update comment",
		},
		{
			name: "Error - Missing body",
			args: map[string]any{
				"action":          "update",
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
				"noteId":          noteID,
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: body",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Invalid mergeRequestIid (not integer)",
			args: map[string]any{
				"action":          "update",
				"projectId":       projectID,
				"mergeRequestIid": 1.5,
				"noteId":          noteID,
				"body":            "Comment",
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: mergeRequestIid 1.5 is not a valid integer",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Invalid noteId (not integer)",
			args: map[string]any{
				"action":          "update",
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
				"noteId":          123.5,
				"body":            "Comment",
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: noteId 123.5 is not a valid integer",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Success - Update with special characters",
			args: map[string]any{
				"action":          "update",
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
				"noteId":          noteID,
				"body":            "Comment with <html> & \"quotes\" and 'apostrophes'",
			},
			mockSetup: func() {
				expectedNote := &gl.Note{
					ID:   123,
					Body: "Comment with <html> & \"quotes\" and 'apostrophes'",
					Author: gl.NoteAuthor{
						Name: "Test User",
					},
					UpdatedAt: &timeNow,
				}
				mockNotes.EXPECT().
					UpdateMergeRequestNote(projectID, 1, 123, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, _ int, opts *gl.UpdateMergeRequestNoteOptions, _ ...gl.RequestOptionFunc) (*gl.Note, *gl.Response, error) {
						assert.Equal(t, "Comment with <html> & \"quotes\" and 'apostrophes'", *opts.Body)
						return expectedNote, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: &gl.Note{
				ID:   123,
				Body: "Comment with <html> & \"quotes\" and 'apostrophes'",
				Author: gl.NoteAuthor{
					Name: "Test User",
				},
				UpdatedAt: &timeNow,
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Update with markdown",
			args: map[string]any{
				"action":          "update",
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
				"noteId":          noteID,
				"body":            "## Markdown\n\n- Item 1\n- Item 2\n\n[Link](https://example.com)",
			},
			mockSetup: func() {
				expectedNote := &gl.Note{
					ID:   123,
					Body: "## Markdown\n\n- Item 1\n- Item 2\n\n[Link](https://example.com)",
					Author: gl.NoteAuthor{
						Name: "Test User",
					},
					UpdatedAt: &timeNow,
				}
				mockNotes.EXPECT().
					UpdateMergeRequestNote(projectID, 1, 123, gomock.Any(), gomock.Any()).
					Return(expectedNote, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: &gl.Note{
				ID:   123,
				Body: "## Markdown\n\n- Item 1\n- Item 2\n\n[Link](https://example.com)",
				Author: gl.NoteAuthor{
					Name: "Test User",
				},
				UpdatedAt: &timeNow,
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Error - Permission Denied (403)",
			args: map[string]any{
				"action":          "update",
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
				"noteId":          noteID,
				"body":            "Comment",
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					UpdateMergeRequestNote(projectID, 1, 123, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 403}}, errors.New("gitlab: 403 Forbidden"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to update comment merge request 1 or note 123 in project \"group/project\"",
		},
		{
			name: "Error - Empty Body",
			args: map[string]any{
				"action":          "update",
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
				"noteId":          noteID,
				"body":            "",
			},
			mockSetup:           func() {}, // No API call expected - validation fails first
			expectedResult:      "Validation Error: required parameter 'body' cannot be empty or zero value",
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
