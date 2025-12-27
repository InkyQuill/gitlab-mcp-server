package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gl "gitlab.com/gitlab-org/api/client-go"

	"github.com/InkyQuill/gitlab-mcp-server/internal/toolsnaps"
	"go.uber.org/mock/gomock"
)

// TestCreateMilestoneHandler tests the CreateMilestone tool handler
func TestCreateMilestoneHandler(t *testing.T) {
	// Tool schema snapshot test
	tool, _ := CreateMilestone(nil, nil)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockMilestones, ctrl := setupMockClientForMilestones(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	createMilestoneTool, handler := CreateMilestone(mockGetClient, nil)

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
			name: "Success - Create Milestone with minimal fields",
			args: map[string]any{
				"projectId": projectID,
				"title":     "Sprint 1",
			},
			mockSetup: func() {
				expectedMilestone := &gl.Milestone{
					ID:          123,
					IID:         1,
					ProjectID:   456,
					Title:       "Sprint 1",
					Description: "",
					State:       "active",
					CreatedAt:   &timeNow,
				}
				mockMilestones.EXPECT().
					CreateMilestone(projectID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gl.CreateMilestoneOptions, _ ...gl.RequestOptionFunc) (*gl.Milestone, *gl.Response, error) {
						assert.Equal(t, "Sprint 1", *opts.Title)
						return expectedMilestone, &gl.Response{Response: &http.Response{StatusCode: 201}}, nil
					})
			},
			expectedResult: &gl.Milestone{
				ID:          123,
				IID:         1,
				ProjectID:   456,
				Title:       "Sprint 1",
				Description: "",
				State:       "active",
				CreatedAt:   &timeNow,
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Create Milestone with all fields",
			args: map[string]any{
				"projectId":   projectID,
				"title":       "Sprint 2",
				"description": "Second sprint milestone",
				"dueDate":     "2024-12-31",
				"startDate":   "2024-12-01",
			},
			mockSetup: func() {
				expectedMilestone := &gl.Milestone{
					ID:          124,
					IID:         2,
					ProjectID:   456,
					Title:       "Sprint 2",
					Description: "Second sprint milestone",
					State:       "active",
					CreatedAt:   &timeNow,
				}
				mockMilestones.EXPECT().
					CreateMilestone(projectID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gl.CreateMilestoneOptions, _ ...gl.RequestOptionFunc) (*gl.Milestone, *gl.Response, error) {
						assert.Equal(t, "Sprint 2", *opts.Title)
						assert.Equal(t, "Second sprint milestone", *opts.Description)
						assert.NotNil(t, opts.DueDate)
						assert.NotNil(t, opts.StartDate)
						return expectedMilestone, &gl.Response{Response: &http.Response{StatusCode: 201}}, nil
					})
			},
			expectedResult: &gl.Milestone{
				ID:          124,
				IID:         2,
				ProjectID:   456,
				Title:       "Sprint 2",
				Description: "Second sprint milestone",
				State:       "active",
				CreatedAt:   &timeNow,
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Error - Missing projectId",
			args: map[string]any{
				"title": "Sprint 1",
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
				"title":     "Sprint 1",
				"dueDate":   "invalid-date",
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: dueDate must be in YYYY-MM-DD format",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Project Not Found (404)",
			args: map[string]any{
				"projectId": "nonexistent",
				"title":     "Sprint 1",
			},
			mockSetup: func() {
				mockMilestones.EXPECT().
					CreateMilestone("nonexistent", gomock.Any(), gomock.Any()).
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
				"title":     "Sprint 1",
			},
			mockSetup: func() {
				mockMilestones.EXPECT().
					CreateMilestone(projectID, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to create milestone project",
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
					Name:      createMilestoneTool.Name,
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
					expectedMilestone, ok := tc.expectedResult.(*gl.Milestone)
					require.True(t, ok)
					expectedJSON, err := json.Marshal(expectedMilestone)
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
		_, handler := CreateMilestone(errorGetClientFn, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      createMilestoneTool.Name,
				Arguments: map[string]any{"projectId": projectID, "title": "Test"},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to initialize GitLab client")
		assert.Nil(t, result)
	})
}

// TestUpdateMilestoneHandler tests the UpdateMilestone tool handler
func TestUpdateMilestoneHandler(t *testing.T) {
	// Tool schema snapshot test
	tool, _ := UpdateMilestone(nil, nil)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockMilestones, ctrl := setupMockClientForMilestones(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	updateMilestoneTool, handler := UpdateMilestone(mockGetClient, nil)

	projectID := "group/project"
	milestoneID := 1.0
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
			name: "Success - Update Milestone title",
			args: map[string]any{
				"projectId":   projectID,
				"milestoneId": milestoneID,
				"title":       "Updated Sprint 1",
			},
			mockSetup: func() {
				expectedMilestone := &gl.Milestone{
					ID:        123,
					IID:       1,
					ProjectID: 456,
					Title:     "Updated Sprint 1",
					State:     "active",
					UpdatedAt: &timeNow,
				}
				mockMilestones.EXPECT().
					UpdateMilestone(projectID, 1, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gl.UpdateMilestoneOptions, _ ...gl.RequestOptionFunc) (*gl.Milestone, *gl.Response, error) {
						assert.Equal(t, "Updated Sprint 1", *opts.Title)
						return expectedMilestone, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: &gl.Milestone{
				ID:        123,
				IID:       1,
				ProjectID: 456,
				Title:     "Updated Sprint 1",
				State:     "active",
				UpdatedAt: &timeNow,
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Update Milestone with stateEvent",
			args: map[string]any{
				"projectId":   projectID,
				"milestoneId": milestoneID,
				"stateEvent":  "close",
			},
			mockSetup: func() {
				expectedMilestone := &gl.Milestone{
					ID:        123,
					IID:       1,
					ProjectID: 456,
					Title:     "Sprint 1",
					State:     "closed",
					UpdatedAt: &timeNow,
				}
				mockMilestones.EXPECT().
					UpdateMilestone(projectID, 1, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gl.UpdateMilestoneOptions, _ ...gl.RequestOptionFunc) (*gl.Milestone, *gl.Response, error) {
						assert.Equal(t, "close", *opts.StateEvent)
						return expectedMilestone, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: &gl.Milestone{
				ID:        123,
				IID:       1,
				ProjectID: 456,
				Title:     "Sprint 1",
				State:     "closed",
				UpdatedAt: &timeNow,
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Error - Missing projectId",
			args: map[string]any{
				"milestoneId": milestoneID,
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: projectId",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Missing milestoneId",
			args: map[string]any{
				"projectId": projectID,
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: milestoneId",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Milestone Not Found (404)",
			args: map[string]any{
				"projectId":   projectID,
				"milestoneId": 999.0,
			},
			mockSetup: func() {
				mockMilestones.EXPECT().
					UpdateMilestone(projectID, 999, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Milestone Not Found"))
			},
			expectedResult:      "milestone 999 in project \"group/project\" not found or access denied (404)",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - GitLab API Error (500)",
			args: map[string]any{
				"projectId":   projectID,
				"milestoneId": milestoneID,
			},
			mockSetup: func() {
				mockMilestones.EXPECT().
					UpdateMilestone(projectID, 1, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to update milestone",
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
					Name:      updateMilestoneTool.Name,
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
					expectedMilestone, ok := tc.expectedResult.(*gl.Milestone)
					require.True(t, ok)
					expectedJSON, err := json.Marshal(expectedMilestone)
					require.NoError(t, err)
					assert.JSONEq(t, string(expectedJSON), textContent.Text)
				}
			}
		})
	}
}

// TestGetMilestoneHandler tests the GetMilestone tool handler
func TestGetMilestoneHandler(t *testing.T) {
	// Tool schema snapshot test
	tool, _ := GetMilestone(nil, nil)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockMilestones, ctrl := setupMockClientForMilestones(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	getMilestoneTool, handler := GetMilestone(mockGetClient, nil)

	projectID := "group/project"
	milestoneID := 1.0
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
			name: "Success - Get Milestone",
			args: map[string]any{
				"projectId":   projectID,
				"milestoneId": milestoneID,
			},
			mockSetup: func() {
				expectedMilestone := &gl.Milestone{
					ID:          123,
					IID:         1,
					ProjectID:   456,
					Title:       "Sprint 1",
					Description: "First sprint",
					State:       "active",
					CreatedAt:   &timeNow,
				}
				mockMilestones.EXPECT().
					GetMilestone(projectID, 1, nil, gomock.Any()).
					Return(expectedMilestone, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: &gl.Milestone{
				ID:          123,
				IID:         1,
				ProjectID:   456,
				Title:       "Sprint 1",
				Description: "First sprint",
				State:       "active",
				CreatedAt:   &timeNow,
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Error - Missing projectId",
			args: map[string]any{
				"milestoneId": milestoneID,
			},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: projectId",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Milestone Not Found (404)",
			args: map[string]any{
				"projectId":   projectID,
				"milestoneId": 999.0,
			},
			mockSetup: func() {
				mockMilestones.EXPECT().
					GetMilestone(projectID, 999, nil, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Milestone Not Found"))
			},
			expectedResult:      "milestone 999 in project \"group/project\" not found or access denied (404)",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - GitLab API Error (500)",
			args: map[string]any{
				"projectId":   projectID,
				"milestoneId": milestoneID,
			},
			mockSetup: func() {
				mockMilestones.EXPECT().
					GetMilestone(projectID, 1, nil, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to process milestone",
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
					Name:      getMilestoneTool.Name,
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
					expectedMilestone, ok := tc.expectedResult.(*gl.Milestone)
					require.True(t, ok)
					expectedJSON, err := json.Marshal(expectedMilestone)
					require.NoError(t, err)
					assert.JSONEq(t, string(expectedJSON), textContent.Text)
				}
			}
		})
	}
}

// TestListMilestonesHandler tests the ListMilestones tool handler
func TestListMilestonesHandler(t *testing.T) {
	// Tool schema snapshot test
	tool, _ := ListMilestones(nil, nil)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockMilestones, ctrl := setupMockClientForMilestones(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	listMilestonesTool, handler := ListMilestones(mockGetClient, nil)

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
			name: "Success - List Milestones",
			args: map[string]any{
				"projectId": projectID,
			},
			mockSetup: func() {
				expectedMilestones := []*gl.Milestone{
					{
						ID:        123,
						IID:       1,
						ProjectID: 456,
						Title:     "Sprint 1",
						State:     "active",
						CreatedAt: &timeNow,
					},
					{
						ID:        124,
						IID:       2,
						ProjectID: 456,
						Title:     "Sprint 2",
						State:     "active",
						CreatedAt: &timeNow,
					},
				}
				mockMilestones.EXPECT().
					ListMilestones(projectID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gl.ListMilestonesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Milestone, *gl.Response, error) {
						assert.Equal(t, 1, opts.Page)
						assert.Equal(t, DefaultPerPage, opts.PerPage)
						return expectedMilestones, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Milestone{
				{
					ID:        123,
					IID:       1,
					ProjectID: 456,
					Title:     "Sprint 1",
					State:     "active",
					CreatedAt: &timeNow,
				},
				{
					ID:        124,
					IID:       2,
					ProjectID: 456,
					Title:     "Sprint 2",
					State:     "active",
					CreatedAt: &timeNow,
				},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - List Milestones with filters",
			args: map[string]any{
				"projectId": projectID,
				"state":     "active",
				"search":    "Sprint",
				"page":      2.0,
				"per_page":  5.0,
			},
			mockSetup: func() {
				expectedMilestones := []*gl.Milestone{
					{
						ID:        125,
						IID:       3,
						ProjectID: 456,
						Title:     "Sprint 3",
						State:     "active",
						CreatedAt: &timeNow,
					},
				}
				mockMilestones.EXPECT().
					ListMilestones(projectID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, opts *gl.ListMilestonesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Milestone, *gl.Response, error) {
						assert.Equal(t, "active", *opts.State)
						assert.Equal(t, "Sprint", *opts.Search)
						assert.Equal(t, 2, opts.Page)
						assert.Equal(t, 5, opts.PerPage)
						return expectedMilestones, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Milestone{
				{
					ID:        125,
					IID:       3,
					ProjectID: 456,
					Title:     "Sprint 3",
					State:     "active",
					CreatedAt: &timeNow,
				},
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name: "Success - Empty Milestones List",
			args: map[string]any{
				"projectId": projectID,
				"state":     "closed",
			},
			mockSetup: func() {
				mockMilestones.EXPECT().
					ListMilestones(projectID, gomock.Any(), gomock.Any()).
					Return([]*gl.Milestone{}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult:      "[]",
			expectResultError:   false,
			expectInternalError: false,
		},
		{
			name:                "Error - Missing projectId",
			args:                map[string]any{},
			mockSetup:           func() {},
			expectedResult:      "Validation Error: missing required parameter: projectId",
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name: "Error - Project Not Found (404)",
			args: map[string]any{
				"projectId": "nonexistent",
			},
			mockSetup: func() {
				mockMilestones.EXPECT().
					ListMilestones("nonexistent", gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Project Not Found"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to list milestones from project \"nonexistent\"",
		},
		{
			name: "Error - GitLab API Error (500)",
			args: map[string]any{
				"projectId": projectID,
			},
			mockSetup: func() {
				mockMilestones.EXPECT().
					ListMilestones(projectID, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:      nil,
			expectResultError:   false,
			expectInternalError: true,
			errorContains:       "failed to list milestones from project",
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
					Name:      listMilestonesTool.Name,
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
						// Special case for empty array
						assert.Equal(t, expectedStr, textContent.Text)
					} else {
						expectedMilestones, ok := tc.expectedResult.([]*gl.Milestone)
						require.True(t, ok)
						expectedJSON, err := json.Marshal(expectedMilestones)
						require.NoError(t, err)
						assert.JSONEq(t, string(expectedJSON), textContent.Text)
					}
				}
			}
		})
	}

	t.Run("Error - Client Initialization Error", func(t *testing.T) {
		errorGetClientFn := func(_ context.Context) (*gl.Client, error) {
			return nil, fmt.Errorf("mock init error")
		}
		_, handler := ListMilestones(errorGetClientFn, nil)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      listMilestonesTool.Name,
				Arguments: map[string]any{"projectId": projectID},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to initialize GitLab client")
		assert.Nil(t, result)
	})
}
