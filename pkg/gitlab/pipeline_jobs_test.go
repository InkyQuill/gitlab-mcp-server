package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/mark3labs/mcp-go/mcp"
	gl "gitlab.com/gitlab-org/api/client-go"

	"github.com/InkyQuill/gitlab-mcp-server/internal/toolsnaps"
)

// TestListPipelineJobsHandler tests the listPipelineJobs tool
func TestListPipelineJobsHandler(t *testing.T) {
	// Tool schema snapshot test
	listPipelineJobsTool, _ := ListPipelineJobs(nil, nil)
	require.NoError(t, toolsnaps.Test(listPipelineJobsTool.Name, listPipelineJobsTool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockJobs, ctrl := setupMockClientForJobs(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	listPipelineJobsTool, listPipelineJobsHandler := ListPipelineJobs(mockGetClient, nil)

	projectID := "group/project"
	pipelineID := 12345

	createJob := func(id int, name string, status string) *gl.Job {
		return &gl.Job{
			ID:     id,
			Name:   name,
			Status: status,
		}
	}

	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectedResult     []*gl.Job
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - List Jobs - No Filters",
			inputArgs: map[string]any{
				"projectId":  projectID,
				"pipelineId": float64(pipelineID),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					ListPipelineJobs(projectID, pipelineID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ interface{}, pipelineID int, opts *gl.ListJobsOptions, _ ...gl.RequestOptionFunc) ([]*gl.Job, *gl.Response, error) {
						assert.Equal(t, 1, opts.Page)
						return []*gl.Job{
							createJob(1, "build", "success"),
							createJob(2, "test", "running"),
						}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Job{
				createJob(1, "build", "success"),
				createJob(2, "test", "running"),
			},
		},
		{
			name: "Success - List Jobs - With Pagination",
			inputArgs: map[string]any{
				"projectId":  projectID,
				"pipelineId": float64(pipelineID),
				"page":       2,
				"per_page":   1,
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					ListPipelineJobs(projectID, pipelineID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ interface{}, pipelineID int, opts *gl.ListJobsOptions, _ ...gl.RequestOptionFunc) ([]*gl.Job, *gl.Response, error) {
						assert.Equal(t, 2, opts.Page)
						assert.Equal(t, 1, opts.PerPage)
						return []*gl.Job{createJob(2, "test", "running")}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Job{createJob(2, "test", "running")},
		},
		{
			name: "Success - Empty List",
			inputArgs: map[string]any{
				"projectId":  projectID,
				"pipelineId": float64(pipelineID),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					ListPipelineJobs(projectID, pipelineID, gomock.Any(), gomock.Any()).
					Return([]*gl.Job{}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: []*gl.Job{},
		},
		{
			name: "Error - Pipeline Not Found (404)",
			inputArgs: map[string]any{
				"projectId":  "nonexistent/project",
				"pipelineId": float64(99999),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					ListPipelineJobs("nonexistent/project", 99999, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("404 Not Found"))
			},
			expectHandlerError: true,
			errorContains:      "failed to list jobs",
		},
		{
			name: "Error - GitLab API Error (500)",
			inputArgs: map[string]any{
				"projectId":  projectID,
				"pipelineId": float64(pipelineID),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					ListPipelineJobs(projectID, pipelineID, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("500 Internal Server Error"))
			},
			expectHandlerError: true,
			errorContains:      "failed to list jobs",
		},
		{
			name:               "Error - Missing projectId",
			inputArgs:          map[string]any{"pipelineId": float64(pipelineID)},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: projectId",
		},
		{
			name:               "Error - Missing pipelineId",
			inputArgs:          map[string]any{"projectId": projectID},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: pipelineId",
		},
		{
			name: "Error - Unauthorized (401)",
			inputArgs: map[string]any{
				"projectId":  projectID,
				"pipelineId": float64(pipelineID),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					ListPipelineJobs(projectID, pipelineID, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("401 Unauthorized"))
			},
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name: "Error - Network Error",
			inputArgs: map[string]any{
				"projectId":  projectID,
				"pipelineId": float64(pipelineID),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					ListPipelineJobs(projectID, pipelineID, gomock.Any(), gomock.Any()).
					Return(nil, nil, errors.New("network error"))
			},
			expectHandlerError: true,
			errorContains:      "failed to list jobs",
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
					Name:      listPipelineJobsTool.Name,
					Arguments: tt.inputArgs,
				},
			}

			result, err := listPipelineJobsHandler(ctx, request)

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
				var jobs []*gl.Job
				err := json.Unmarshal([]byte(textContent.Text), &jobs)
				require.NoError(t, err)
				assert.Equal(t, len(tt.expectedResult), len(jobs))
			}
		})
	}
}

// TestGetPipelineJobHandler tests the getPipelineJob tool
func TestGetPipelineJobHandler(t *testing.T) {
	getPipelineJobTool, _ := GetPipelineJob(nil, nil)
	require.NoError(t, toolsnaps.Test(getPipelineJobTool.Name, getPipelineJobTool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockJobs, ctrl := setupMockClientForJobs(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	getPipelineJobTool, getPipelineJobHandler := GetPipelineJob(mockGetClient, nil)

	projectID := "group/project"
	jobID := 123

	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - Get Job",
			inputArgs: map[string]any{
				"projectId": projectID,
				"jobId":     float64(jobID),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					GetJob(projectID, jobID, gomock.Any()).
					Return(&gl.Job{
						ID:     jobID,
						Name:   "build",
						Status: "success",
					}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name: "Error - Job Not Found (404)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"jobId":     float64(99999),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					GetJob(projectID, 99999, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("404 Not Found"))
			},
			expectResultError: true,
			errorContains:     "not found or access denied (404)",
		},
		{
			name:               "Error - Missing jobId",
			inputArgs:          map[string]any{"projectId": projectID},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: jobId",
		},
		{
			name:               "Error - Missing projectId",
			inputArgs:          map[string]any{"jobId": float64(jobID)},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: projectId",
		},
		{
			name: "Error - Unauthorized (401)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"jobId":     float64(jobID),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					GetJob(projectID, jobID, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("401 Unauthorized"))
			},
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
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
					Name:      getPipelineJobTool.Name,
					Arguments: tt.inputArgs,
				},
			}

			result, err := getPipelineJobHandler(ctx, request)

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

// TestGetPipelineJobTraceHandler tests the getPipelineJobTrace tool
func TestGetPipelineJobTraceHandler(t *testing.T) {
	getPipelineJobTraceTool, _ := GetPipelineJobTrace(nil, nil)
	require.NoError(t, toolsnaps.Test(getPipelineJobTraceTool.Name, getPipelineJobTraceTool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockJobs, ctrl := setupMockClientForJobs(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	getPipelineJobTraceTool, getPipelineJobTraceHandler := GetPipelineJobTrace(mockGetClient, nil)

	projectID := "group/project"
	jobID := 123
	traceContent := "Running build...\nBuild successful!\n"

	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
		expectedTrace      bool
	}{
		{
			name: "Success - Get Job Trace",
			inputArgs: map[string]any{
				"projectId": projectID,
				"jobId":     float64(jobID),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					GetTraceFile(projectID, jobID, gomock.Any()).
					Return(bytes.NewReader([]byte(traceContent)), &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedTrace: true,
		},
		{
			name: "Error - Job Not Found (404)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"jobId":     float64(99999),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					GetTraceFile(projectID, 99999, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("404 Not Found"))
			},
			expectResultError: true,
			errorContains:     "not found or access denied (404)",
		},
		{
			name:               "Error - Missing jobId",
			inputArgs:          map[string]any{"projectId": projectID},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: jobId",
		},
		{
			name:               "Error - Missing projectId",
			inputArgs:          map[string]any{"jobId": float64(jobID)},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: projectId",
		},
		{
			name: "Error - Unauthorized (401)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"jobId":     float64(jobID),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					GetTraceFile(projectID, jobID, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("401 Unauthorized"))
			},
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name: "Error - Server Error (500)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"jobId":     float64(jobID),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					GetTraceFile(projectID, jobID, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("500 Internal Server Error"))
			},
			expectHandlerError: true,
			errorContains:      "failed to process trace",
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
					Name:      getPipelineJobTraceTool.Name,
					Arguments: tt.inputArgs,
				},
			}

			result, err := getPipelineJobTraceHandler(ctx, request)

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

			if tt.expectedTrace {
				assert.Contains(t, textContent.Text, "Build successful")
			}
		})
	}
}

// TestRetryPipelineJobHandler tests the retryPipelineJob tool
func TestRetryPipelineJobHandler(t *testing.T) {
	retryPipelineJobTool, _ := RetryPipelineJob(nil, nil)
	require.NoError(t, toolsnaps.Test(retryPipelineJobTool.Name, retryPipelineJobTool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockJobs, ctrl := setupMockClientForJobs(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	retryPipelineJobTool, retryPipelineJobHandler := RetryPipelineJob(mockGetClient, nil)

	projectID := "group/project"
	jobID := 123

	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - Retry Job",
			inputArgs: map[string]any{
				"projectId": projectID,
				"jobId":     float64(jobID),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					RetryJob(projectID, jobID, gomock.Any()).
					Return(&gl.Job{
						ID:     jobID,
						Name:   "build",
						Status: "pending",
					}, &gl.Response{Response: &http.Response{StatusCode: 201}}, nil)
			},
		},
		{
			name: "Error - Job Not Found (404)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"jobId":     float64(99999),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					RetryJob(projectID, 99999, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("404 Not Found"))
			},
			expectResultError: true,
			errorContains:     "not found or access denied (404)",
		},
		{
			name:               "Error - Missing jobId",
			inputArgs:          map[string]any{"projectId": projectID},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: jobId",
		},
		{
			name:               "Error - Missing projectId",
			inputArgs:          map[string]any{"jobId": float64(jobID)},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: projectId",
		},
		{
			name: "Error - Unauthorized (401)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"jobId":     float64(jobID),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					RetryJob(projectID, jobID, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("401 Unauthorized"))
			},
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name: "Error - Cannot Retry (400)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"jobId":     float64(jobID),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					RetryJob(projectID, jobID, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 400}}, errors.New("job cannot be retried"))
			},
			expectResultError: true,
			errorContains:     "failed to retry",
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
					Name:      retryPipelineJobTool.Name,
					Arguments: tt.inputArgs,
				},
			}

			result, err := retryPipelineJobHandler(ctx, request)

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

// TestPlayPipelineJobHandler tests the playPipelineJob tool
func TestPlayPipelineJobHandler(t *testing.T) {
	playPipelineJobTool, _ := PlayPipelineJob(nil, nil)
	require.NoError(t, toolsnaps.Test(playPipelineJobTool.Name, playPipelineJobTool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockJobs, ctrl := setupMockClientForJobs(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	playPipelineJobTool, playPipelineJobHandler := PlayPipelineJob(mockGetClient, nil)

	projectID := "group/project"
	jobID := 123

	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - Play Manual Job",
			inputArgs: map[string]any{
				"projectId": projectID,
				"jobId":     float64(jobID),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					PlayJob(projectID, jobID, gomock.Any(), gomock.Any()).
					Return(&gl.Job{
						ID:     jobID,
						Name:   "deploy",
						Status: "pending",
					}, &gl.Response{Response: &http.Response{StatusCode: 201}}, nil)
			},
		},
		{
			name: "Error - Job Not Found (404)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"jobId":     float64(99999),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					PlayJob(projectID, 99999, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("404 Not Found"))
			},
			expectResultError: true,
			errorContains:     "not found or access denied (404)",
		},
		{
			name:               "Error - Missing jobId",
			inputArgs:          map[string]any{"projectId": projectID},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: jobId",
		},
		{
			name:               "Error - Missing projectId",
			inputArgs:          map[string]any{"jobId": float64(jobID)},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: projectId",
		},
		{
			name: "Error - Unauthorized (401)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"jobId":     float64(jobID),
			},
			mockSetup: func() {
				mockJobs.EXPECT().
					PlayJob(projectID, jobID, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("401 Unauthorized"))
			},
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
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
					Name:      playPipelineJobTool.Name,
					Arguments: tt.inputArgs,
				},
			}

			result, err := playPipelineJobHandler(ctx, request)

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

// TestCancelPipelineHandler tests the cancelPipeline tool
func TestCancelPipelineHandler(t *testing.T) {
	cancelPipelineTool, _ := CancelPipeline(nil, nil)
	require.NoError(t, toolsnaps.Test(cancelPipelineTool.Name, cancelPipelineTool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockPipelines, ctrl := setupMockClientForPipelines(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	cancelPipelineTool, cancelPipelineHandler := CancelPipeline(mockGetClient, nil)

	projectID := "group/project"
	pipelineID := 12345

	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - Cancel Pipeline",
			inputArgs: map[string]any{
				"projectId":  projectID,
				"pipelineId": float64(pipelineID),
			},
			mockSetup: func() {
				mockPipelines.EXPECT().
					CancelPipelineBuild(projectID, pipelineID, gomock.Any()).
					Return(&gl.Pipeline{
						ID:     pipelineID,
						Status: "canceled",
					}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
		},
		{
			name: "Error - Pipeline Not Found (404)",
			inputArgs: map[string]any{
				"projectId":  projectID,
				"pipelineId": float64(99999),
			},
			mockSetup: func() {
				mockPipelines.EXPECT().
					CancelPipelineBuild(projectID, 99999, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("404 Not Found"))
			},
			expectResultError: true,
			errorContains:     "not found or access denied (404)",
		},
		{
			name:               "Error - Missing pipelineId",
			inputArgs:          map[string]any{"projectId": projectID},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: pipelineId",
		},
		{
			name:               "Error - Missing projectId",
			inputArgs:          map[string]any{"pipelineId": float64(pipelineID)},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: projectId",
		},
		{
			name: "Error - Unauthorized (401)",
			inputArgs: map[string]any{
				"projectId":  projectID,
				"pipelineId": float64(pipelineID),
			},
			mockSetup: func() {
				mockPipelines.EXPECT().
					CancelPipelineBuild(projectID, pipelineID, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("401 Unauthorized"))
			},
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name: "Error - Cannot Cancel (400)",
			inputArgs: map[string]any{
				"projectId":  projectID,
				"pipelineId": float64(pipelineID),
			},
			mockSetup: func() {
				mockPipelines.EXPECT().
					CancelPipelineBuild(projectID, pipelineID, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 400}}, errors.New("pipeline cannot be canceled"))
			},
			expectResultError: true,
			errorContains:     "failed to cancel",
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
					Name:      cancelPipelineTool.Name,
					Arguments: tt.inputArgs,
				},
			}

			result, err := cancelPipelineHandler(ctx, request)

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

// TestRetryPipelineHandler tests the retryPipeline tool
func TestRetryPipelineHandler(t *testing.T) {
	retryPipelineTool, _ := RetryPipeline(nil, nil)
	require.NoError(t, toolsnaps.Test(retryPipelineTool.Name, retryPipelineTool), "tool schema should match snapshot")

	ctx := context.Background()
	mockClient, mockPipelines, ctrl := setupMockClientForPipelines(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	retryPipelineTool, retryPipelineHandler := RetryPipeline(mockGetClient, nil)

	projectID := "group/project"
	pipelineID := 12345

	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - Retry Pipeline",
			inputArgs: map[string]any{
				"projectId":  projectID,
				"pipelineId": float64(pipelineID),
			},
			mockSetup: func() {
				mockPipelines.EXPECT().
					RetryPipelineBuild(projectID, pipelineID, gomock.Any()).
					Return(&gl.Pipeline{
						ID:     pipelineID,
						Status: "pending",
					}, &gl.Response{Response: &http.Response{StatusCode: 201}}, nil)
			},
		},
		{
			name: "Error - Pipeline Not Found (404)",
			inputArgs: map[string]any{
				"projectId":  projectID,
				"pipelineId": float64(99999),
			},
			mockSetup: func() {
				mockPipelines.EXPECT().
					RetryPipelineBuild(projectID, 99999, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("404 Not Found"))
			},
			expectResultError: true,
			errorContains:     "not found or access denied (404)",
		},
		{
			name:               "Error - Missing pipelineId",
			inputArgs:          map[string]any{"projectId": projectID},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: pipelineId",
		},
		{
			name:               "Error - Missing projectId",
			inputArgs:          map[string]any{"pipelineId": float64(pipelineID)},
			mockSetup:          func() {},
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: projectId",
		},
		{
			name: "Error - Unauthorized (401)",
			inputArgs: map[string]any{
				"projectId":  projectID,
				"pipelineId": float64(pipelineID),
			},
			mockSetup: func() {
				mockPipelines.EXPECT().
					RetryPipelineBuild(projectID, pipelineID, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 401}}, errors.New("401 Unauthorized"))
			},
			expectResultError: true,
			errorContains:     "Authentication failed (401)",
		},
		{
			name: "Error - Cannot Retry (400)",
			inputArgs: map[string]any{
				"projectId":  projectID,
				"pipelineId": float64(pipelineID),
			},
			mockSetup: func() {
				mockPipelines.EXPECT().
					RetryPipelineBuild(projectID, pipelineID, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 400}}, errors.New("pipeline cannot be retried"))
			},
			expectResultError: true,
			errorContains:     "failed to retry",
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
					Name:      retryPipelineTool.Name,
					Arguments: tt.inputArgs,
				},
			}

			result, err := retryPipelineHandler(ctx, request)

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
