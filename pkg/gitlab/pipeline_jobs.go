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

// PipelineJob defines the consolidated MCP tool for managing pipeline jobs (list, get, trace).
func PipelineJob(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"pipelineJob",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_PIPELINE_JOB_DESCRIPTION)),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title: "Manage Pipeline Job",
		}),
		// Required parameters
		mcp.WithString("action",
			mcp.Description("The action to perform on the pipeline job."),
			mcp.Required(),
			mcp.Enum("list", "get", "trace"),
		),
		mcp.WithString("projectId",
			mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
			mcp.Required(),
		),
		// Conditional required parameters
		mcp.WithNumber("pipelineId",
			mcp.Description("The ID of the pipeline (required for list action)."),
		),
		mcp.WithNumber("jobId",
			mcp.Description("The ID of the job (required for get/trace actions)."),
		),
		// Optional parameters
		WithPagination(),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// --- Parse required parameters
		action, err := requiredParam[string](&request, "action")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		projectID, err := requiredParam[string](&request, "projectId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		// --- Obtain GitLab client
		glClient, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
		}

		// --- Switch on action
		switch action {
		case "list":
			pipelineIdFloat, err := requiredParam[float64](&request, "pipelineId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			pipelineId := int(pipelineIdFloat)

			page, perPage, err := OptionalPaginationParams(&request)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			opts := &gl.ListJobsOptions{
				ListOptions: gl.ListOptions{
					Page:    page,
					PerPage: perPage,
				},
			}

			jobs, resp, err := glClient.Jobs.ListPipelineJobs(projectID, pipelineId, opts, gl.WithContext(ctx))
			if err != nil {
				result, apiErr := HandleListAPIError(err, resp, fmt.Sprintf("jobs for pipeline %d in project %q", pipelineId, projectID))
				if result != nil {
					return result, nil
				}
				return nil, apiErr
			}

			if len(jobs) == 0 {
				return mcp.NewToolResultText("[]"), nil
			}

			data, err := json.Marshal(jobs)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal job list data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil

		case "get":
			jobIdFloat, err := requiredParam[float64](&request, "jobId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			jobId := int(jobIdFloat)

			job, resp, err := glClient.Jobs.GetJob(projectID, jobId, gl.WithContext(ctx))
			if err != nil {
				result, apiErr := HandleAPIError(err, resp, fmt.Sprintf("job %d in project %q", jobId, projectID))
				if result != nil {
					return result, nil
				}
				return nil, apiErr
			}

			data, err := json.Marshal(job)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal job data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil

		case "trace":
			jobIdFloat, err := requiredParam[float64](&request, "jobId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			jobId := int(jobIdFloat)

			traceReader, resp, err := glClient.Jobs.GetTraceFile(projectID, jobId, gl.WithContext(ctx))
			if err != nil {
				result, apiErr := HandleAPIError(err, resp, fmt.Sprintf("trace for job %d in project %q", jobId, projectID))
				if result != nil {
					return result, nil
				}
				return nil, apiErr
			}

			traceBytes, err := io.ReadAll(traceReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read job trace: %w", err)
			}

			return mcp.NewToolResultText(string(traceBytes)), nil

		default:
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: invalid action %q (must be list, get, or trace)", action)), nil
		}
	}
}

// Pipeline defines the consolidated MCP tool for controlling pipelines (cancel, retry).
func Pipeline(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"pipeline",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_PIPELINE_DESCRIPTION)),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title: "Control Pipeline",
		}),
		// Required parameters
		mcp.WithString("action",
			mcp.Description("The action to perform on the pipeline."),
			mcp.Required(),
			mcp.Enum("cancel", "retry"),
		),
		mcp.WithString("projectId",
			mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
			mcp.Required(),
		),
		mcp.WithNumber("pipelineId",
			mcp.Description("The ID of the pipeline."),
			mcp.Required(),
		),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// --- Parse required parameters
		action, err := requiredParam[string](&request, "action")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		projectID, err := requiredParam[string](&request, "projectId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		pipelineIdFloat, err := requiredParam[float64](&request, "pipelineId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		pipelineId := int(pipelineIdFloat)

		// --- Obtain GitLab client
		glClient, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
		}

		// --- Switch on action
		switch action {
		case "cancel":
			pipeline, resp, err := glClient.Pipelines.CancelPipelineBuild(projectID, pipelineId, gl.WithContext(ctx))
			if err != nil {
				result, apiErr := HandleCreateUpdateAPIError(err, resp, fmt.Sprintf("pipeline %d in project %q", pipelineId, projectID), "cancel")
				if result != nil {
					return result, nil
				}
				return nil, apiErr
			}

			data, err := json.Marshal(pipeline)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal canceled pipeline data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil

		case "retry":
			pipeline, resp, err := glClient.Pipelines.RetryPipelineBuild(projectID, pipelineId, gl.WithContext(ctx))
			if err != nil {
				result, apiErr := HandleCreateUpdateAPIError(err, resp, fmt.Sprintf("pipeline %d in project %q", pipelineId, projectID), "retry")
				if result != nil {
					return result, nil
				}
				return nil, apiErr
			}

			data, err := json.Marshal(pipeline)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal retried pipeline data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil

		default:
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: invalid action %q (must be cancel or retry)", action)), nil
		}
	}
}
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
