package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/translations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gl "gitlab.com/gitlab-org/api/client-go"
)

// ListPipelineJobs defines the MCP tool for listing jobs in a pipeline.
func ListPipelineJobs(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"listPipelineJobs",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_LIST_PIPELINE_JOBS_DESCRIPTION)),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List Pipeline Jobs",
			ReadOnlyHint: true,
		}),
		mcp.WithString("projectId",
			mcp.Required(),
			mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
		),
		mcp.WithNumber("pipelineId",
			mcp.Required(),
			mcp.Description("The ID of the pipeline."),
		),
		WithPagination(),
	),
	func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse parameters
		projectIDStr, err := requiredParam[string](&request, "projectId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		pipelineIdFloat, err := requiredParam[float64](&request, "pipelineId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		page, perPage, err := OptionalPaginationParams(&request)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		// Convert pipelineId to integer
		pipelineId := int(pipelineIdFloat)

		// Construct GitLab API options
		opts := &gl.ListJobsOptions{
			ListOptions: gl.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		}

		// Obtain GitLab client
		glClient, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitLab client: %w", err)
		}

		// Call GitLab API
		jobs, resp, err := glClient.Jobs.ListPipelineJobs(projectIDStr, pipelineId, opts, gl.WithContext(ctx))

		// Handle API errors
		if err != nil {
			result, apiErr := HandleListAPIError(err, resp, fmt.Sprintf("jobs for pipeline %d in project %q", pipelineId, projectIDStr))
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		// Marshal and return success
		if len(jobs) == 0 {
			return mcp.NewToolResultText("[]"), nil
		}

		data, err := json.Marshal(jobs)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal job list data: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// GetPipelineJob defines the MCP tool for getting details of a specific job.
func GetPipelineJob(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"getPipelineJob",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_GET_PIPELINE_JOB_DESCRIPTION)),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Pipeline Job",
			ReadOnlyHint: true,
		}),
		mcp.WithString("projectId",
			mcp.Required(),
			mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
		),
		mcp.WithNumber("jobId",
			mcp.Required(),
			mcp.Description("The ID of the job."),
		),
	),
	func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse parameters
		projectIDStr, err := requiredParam[string](&request, "projectId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		jobIdFloat, err := requiredParam[float64](&request, "jobId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		// Convert jobId to integer
		jobId := int(jobIdFloat)

		// Obtain GitLab client
		glClient, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitLab client: %w", err)
		}

		// Call GitLab API
		job, resp, err := glClient.Jobs.GetJob(projectIDStr, jobId, gl.WithContext(ctx))

		// Handle API errors
		if err != nil {
			result, apiErr := HandleAPIError(err, resp, fmt.Sprintf("job %d in project %q", jobId, projectIDStr))
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		// Marshal and return success
		data, err := json.Marshal(job)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal job data: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// GetPipelineJobTrace defines the MCP tool for getting the log trace of a job.
func GetPipelineJobTrace(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"getPipelineJobTrace",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_GET_PIPELINE_JOB_TRACE_DESCRIPTION)),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Pipeline Job Trace",
			ReadOnlyHint: true,
		}),
		mcp.WithString("projectId",
			mcp.Required(),
			mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
		),
		mcp.WithNumber("jobId",
			mcp.Required(),
			mcp.Description("The ID of the job."),
		),
	),
	func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse parameters
		projectIDStr, err := requiredParam[string](&request, "projectId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		jobIdFloat, err := requiredParam[float64](&request, "jobId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		// Convert jobId to integer
		jobId := int(jobIdFloat)

		// Obtain GitLab client
		glClient, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitLab client: %w", err)
		}

		// Call GitLab API to get trace
		traceReader, resp, err := glClient.Jobs.GetTraceFile(projectIDStr, jobId, gl.WithContext(ctx))

		// Handle API errors
		if err != nil {
			result, apiErr := HandleAPIError(err, resp, fmt.Sprintf("trace for job %d in project %q", jobId, projectIDStr))
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		// Read trace content
		traceBytes, err := io.ReadAll(traceReader)
		if err != nil {
			return nil, fmt.Errorf("failed to read job trace: %w", err)
		}

		// Return trace as text
		return mcp.NewToolResultText(string(traceBytes)), nil
	}
}

// RetryPipelineJob defines the MCP tool for retrying a failed job.
func RetryPipelineJob(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"retryPipelineJob",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_RETRY_PIPELINE_JOB_DESCRIPTION)),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Retry Pipeline Job",
			ReadOnlyHint: false,
		}),
		mcp.WithString("projectId",
			mcp.Required(),
			mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
		),
		mcp.WithNumber("jobId",
			mcp.Required(),
			mcp.Description("The ID of the job to retry."),
		),
	),
	func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse parameters
		projectIDStr, err := requiredParam[string](&request, "projectId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		jobIdFloat, err := requiredParam[float64](&request, "jobId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		// Convert jobId to integer
		jobId := int(jobIdFloat)

		// Obtain GitLab client
		glClient, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitLab client: %w", err)
		}

		// Call GitLab API to retry job
		job, resp, err := glClient.Jobs.RetryJob(projectIDStr, jobId, gl.WithContext(ctx))

		// Handle API errors
		if err != nil {
			result, apiErr := HandleCreateUpdateAPIError(err, resp, fmt.Sprintf("job %d in project %q", jobId, projectIDStr), "retry")
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		// Marshal and return success
		data, err := json.Marshal(job)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal retried job data: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// PlayPipelineJob defines the MCP tool for triggering a manual job.
func PlayPipelineJob(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"playPipelineJob",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_PLAY_PIPELINE_JOB_DESCRIPTION)),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Play Pipeline Job",
			ReadOnlyHint: false,
		}),
		mcp.WithString("projectId",
			mcp.Required(),
			mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
		),
		mcp.WithNumber("jobId",
			mcp.Required(),
			mcp.Description("The ID of the job to play."),
		),
	),
	func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse parameters
		projectIDStr, err := requiredParam[string](&request, "projectId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		jobIdFloat, err := requiredParam[float64](&request, "jobId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		// Convert jobId to integer
		jobId := int(jobIdFloat)

		// Obtain GitLab client
		glClient, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitLab client: %w", err)
		}

		// Call GitLab API to play job
		job, resp, err := glClient.Jobs.PlayJob(projectIDStr, jobId, &gl.PlayJobOptions{}, gl.WithContext(ctx))

		// Handle API errors
		if err != nil {
			result, apiErr := HandleCreateUpdateAPIError(err, resp, fmt.Sprintf("job %d in project %q", jobId, projectIDStr), "play")
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		// Marshal and return success
		data, err := json.Marshal(job)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal played job data: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// CancelPipeline defines the MCP tool for canceling a running pipeline.
func CancelPipeline(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"cancelPipeline",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_CANCEL_PIPELINE_DESCRIPTION)),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Cancel Pipeline",
			ReadOnlyHint: false,
		}),
		mcp.WithString("projectId",
			mcp.Required(),
			mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
		),
		mcp.WithNumber("pipelineId",
			mcp.Required(),
			mcp.Description("The ID of the pipeline to cancel."),
		),
	),
	func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse parameters
		projectIDStr, err := requiredParam[string](&request, "projectId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		pipelineIdFloat, err := requiredParam[float64](&request, "pipelineId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		// Convert pipelineId to integer
		pipelineId := int(pipelineIdFloat)

		// Obtain GitLab client
		glClient, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitLab client: %w", err)
		}

		// Call GitLab API to cancel pipeline
		pipeline, resp, err := glClient.Pipelines.CancelPipelineBuild(projectIDStr, pipelineId, gl.WithContext(ctx))

		// Handle API errors
		if err != nil {
			result, apiErr := HandleCreateUpdateAPIError(err, resp, fmt.Sprintf("pipeline %d in project %q", pipelineId, projectIDStr), "cancel")
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		// Marshal and return success
		data, err := json.Marshal(pipeline)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal canceled pipeline data: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// RetryPipeline defines the MCP tool for retrying all jobs in a failed pipeline.
func RetryPipeline(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"retryPipeline",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_RETRY_PIPELINE_DESCRIPTION)),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Retry Pipeline",
			ReadOnlyHint: false,
		}),
		mcp.WithString("projectId",
			mcp.Required(),
			mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
		),
		mcp.WithNumber("pipelineId",
			mcp.Required(),
			mcp.Description("The ID of the pipeline to retry."),
		),
	),
	func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse parameters
		projectIDStr, err := requiredParam[string](&request, "projectId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		pipelineIdFloat, err := requiredParam[float64](&request, "pipelineId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		// Convert pipelineId to integer
		pipelineId := int(pipelineIdFloat)

		// Obtain GitLab client
		glClient, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitLab client: %w", err)
		}

		// Call GitLab API to retry pipeline
		pipeline, resp, err := glClient.Pipelines.RetryPipelineBuild(projectIDStr, pipelineId, gl.WithContext(ctx))

		// Handle API errors
		if err != nil {
			result, apiErr := HandleCreateUpdateAPIError(err, resp, fmt.Sprintf("pipeline %d in project %q", pipelineId, projectIDStr), "retry")
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		// Marshal and return success
		data, err := json.Marshal(pipeline)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal retried pipeline data: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
