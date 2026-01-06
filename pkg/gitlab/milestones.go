package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/InkyQuill/gitlab-mcp-server/pkg/translations"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gl "gitlab.com/gitlab-org/api/client-go"
)

// Milestone defines the consolidated MCP tool for managing GitLab milestones (get, create, update).
func Milestone(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"milestone",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_MILESTONE_DESCRIPTION)),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title: "Manage GitLab Milestone",
		}),
		// Required parameters
		mcp.WithString("action",
			mcp.Description("The action to perform on the milestone."),
			mcp.Required(),
			mcp.Enum("get", "create", "update"),
		),
		mcp.WithString("projectId",
			mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
			mcp.Required(),
		),
		// Conditional required parameters
		mcp.WithNumber("milestoneId",
			mcp.Description("The ID of the milestone (required for get/update)."),
		),
		mcp.WithString("title",
			mcp.Description("The title of the milestone (required for create, optional for update)."),
		),
		// Optional parameters
		mcp.WithString("description",
			mcp.Description("The description of the milestone."),
		),
		mcp.WithString("dueDate",
			mcp.Description("The due date of the milestone (ISO 8601 format: YYYY-MM-DD)."),
		),
		mcp.WithString("startDate",
			mcp.Description("The start date of the milestone (ISO 8601 format: YYYY-MM-DD)."),
		),
		mcp.WithString("stateEvent",
			mcp.Description("The state event to perform on the milestone (activate, close). Only for update action."),
			mcp.Enum("activate", "close"),
		),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return milestoneHandler(ctx, request, getClient)
	}
}

// milestoneHandler is the handler function for the Milestone tool
func milestoneHandler(ctx context.Context, request mcp.CallToolRequest, getClient GetClientFn) (*mcp.CallToolResult, error) {
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
	case "get":
		return milestoneGet(ctx, &request, glClient, projectID)
	case "create":
		return milestoneCreate(ctx, &request, glClient, projectID)
	case "update":
		return milestoneUpdate(ctx, &request, glClient, projectID)
	default:
		return mcp.NewToolResultError(fmt.Sprintf("Validation Error: invalid action %q (must be get, create, or update)", action)), nil
	}
}

// milestoneGet handles the get action
func milestoneGet(ctx context.Context, request *mcp.CallToolRequest, glClient *gl.Client, projectID string) (*mcp.CallToolResult, error) {
	// Parse milestoneId
	milestoneIDFloat, err := requiredParam[float64](request, "milestoneId")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
	}
	milestoneID := int(milestoneIDFloat)
	if float64(milestoneID) != milestoneIDFloat {
		return mcp.NewToolResultError(fmt.Sprintf("Validation Error: milestoneId %v is not a valid integer", milestoneIDFloat)), nil
	}

	// Call GitLab API
	milestone, resp, err := glClient.Milestones.GetMilestone(projectID, milestoneID, nil, gl.WithContext(ctx))

	// Handle API errors
	if err != nil {
		result, apiErr := HandleAPIError(err, resp, fmt.Sprintf("milestone %d in project %q", milestoneID, projectID))
		if result != nil {
			return result, nil
		}
		return nil, apiErr
	}

	// Marshal and return success
	data, err := json.Marshal(milestone)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal milestone data: %w", err)
	}
	return mcp.NewToolResultText(string(data)), nil
}

// milestoneCreate handles the create action
func milestoneCreate(ctx context.Context, request *mcp.CallToolRequest, glClient *gl.Client, projectID string) (*mcp.CallToolResult, error) {
	// Parse title
	title, err := requiredParam[string](request, "title")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
	}

	// Parse optional parameters
	description, err := OptionalParam[string](request, "description")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
	}

	dueDateStr, err := OptionalParam[string](request, "dueDate")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
	}

	startDateStr, err := OptionalParam[string](request, "startDate")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
	}

	// Construct GitLab API options
	opts := &gl.CreateMilestoneOptions{
		Title: &title,
	}

	if description != "" {
		opts.Description = &description
	}

	if dueDateStr != "" {
		dueDate, err := time.Parse("2006-01-02", dueDateStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: dueDate must be in YYYY-MM-DD format, got %q: %v", dueDateStr, err)), nil
		}
		isoTime := gl.ISOTime(dueDate)
		opts.DueDate = &isoTime
	}

	if startDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: startDate must be in YYYY-MM-DD format, got %q: %v", startDateStr, err)), nil
		}
		isoTime := gl.ISOTime(startDate)
		opts.StartDate = &isoTime
	}

	// Call GitLab API
	milestone, resp, err := glClient.Milestones.CreateMilestone(projectID, opts, gl.WithContext(ctx))

	// Handle API errors
	if err != nil {
		result, apiErr := HandleCreateUpdateAPIError(err, resp, fmt.Sprintf("project %q", projectID), "create milestone")
		if result != nil {
			return result, nil
		}
		return nil, apiErr
	}

	// Marshal and return success
	data, err := json.Marshal(milestone)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal milestone data: %w", err)
	}
	return mcp.NewToolResultText(string(data)), nil
}

// milestoneUpdate handles the update action
func milestoneUpdate(ctx context.Context, request *mcp.CallToolRequest, glClient *gl.Client, projectID string) (*mcp.CallToolResult, error) {
	// Parse milestoneId
	milestoneIDFloat, err := requiredParam[float64](request, "milestoneId")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
	}
	milestoneID := int(milestoneIDFloat)
	if float64(milestoneID) != milestoneIDFloat {
		return mcp.NewToolResultError(fmt.Sprintf("Validation Error: milestoneId %v is not a valid integer", milestoneIDFloat)), nil
	}

	// Parse optional parameters
	title, err := OptionalParam[string](request, "title")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
	}

	description, err := OptionalParam[string](request, "description")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
	}

	dueDateStr, err := OptionalParam[string](request, "dueDate")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
	}

	startDateStr, err := OptionalParam[string](request, "startDate")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
	}

	stateEvent, err := OptionalParam[string](request, "stateEvent")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
	}

	// Construct GitLab API options
	opts := &gl.UpdateMilestoneOptions{}

	if title != "" {
		opts.Title = &title
	}

	if description != "" {
		opts.Description = &description
	}

	if dueDateStr != "" {
		dueDate, err := time.Parse("2006-01-02", dueDateStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: dueDate must be in YYYY-MM-DD format, got %q: %v", dueDateStr, err)), nil
		}
		isoTime := gl.ISOTime(dueDate)
		opts.DueDate = &isoTime
	}

	if startDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: startDate must be in YYYY-MM-DD format, got %q: %v", startDateStr, err)), nil
		}
		isoTime := gl.ISOTime(startDate)
		opts.StartDate = &isoTime
	}

	if stateEvent != "" {
		opts.StateEvent = &stateEvent
	}

	// Call GitLab API
	milestone, resp, err := glClient.Milestones.UpdateMilestone(projectID, milestoneID, opts, gl.WithContext(ctx))

	// Handle API errors
	if err != nil {
		result, apiErr := HandleCreateUpdateAPIError(err, resp, fmt.Sprintf("milestone %d in project %q", milestoneID, projectID), "update milestone")
		if result != nil {
			return result, nil
		}
		return nil, apiErr
	}

	// Marshal and return success
	data, err := json.Marshal(milestone)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal milestone data: %w", err)
	}
	return mcp.NewToolResultText(string(data)), nil
}

// ListMilestones defines the MCP tool for listing milestones in a GitLab project.
func ListMilestones(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"listMilestones",
			mcp.WithDescription(translations.Translate(t, translations.TOOL_LIST_MILESTONES_DESCRIPTION)),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title:        "List GitLab Milestones",
				ReadOnlyHint: true,
			}),
			// Required parameters
			mcp.WithString("projectId",
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
				mcp.Required(),
			),
			// Optional filtering parameters
			mcp.WithString("state",
				mcp.Description("Return milestones with the specified state (active, closed, all)."),
				mcp.Enum("active", "closed", "all"),
			),
			mcp.WithString("search",
				mcp.Description("Search milestones against their title and description."),
			),
			// Add standard MCP pagination parameters
			WithPagination(),
		),
		// Handler function implementation
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// --- Parse required parameters
			projectID, err := requiredParam[string](&request, "projectId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Parse optional filtering parameters
			state, err := OptionalParam[string](&request, "state")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			search, err := OptionalParam[string](&request, "search")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Parse pagination parameters
			page, perPage, err := OptionalPaginationParams(&request)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
			}

			// --- Construct GitLab API options
			opts := &gl.ListMilestonesOptions{
				ListOptions: gl.ListOptions{
					Page:    page,
					PerPage: perPage,
				},
			}

			if state != "" {
				opts.State = &state
			}

			if search != "" {
				opts.Search = &search
			}

			// --- Call GitLab API
			milestones, resp, err := glClient.Milestones.ListMilestones(projectID, opts, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				result, apiErr := HandleListAPIError(err, resp, fmt.Sprintf("milestones from project %q", projectID))
				if result != nil {
					return result, nil
				}
				return nil, apiErr
			}

			// --- Handle empty result gracefully
			if len(milestones) == 0 {
				return mcp.NewToolResultText("[]"), nil // Return empty JSON array
			}

			// --- Truncate long text fields for list operations
			truncator := NewTextTruncator(MaxFieldLength)
			truncatedMilestones, err := truncator.TruncateListResponse(milestones, MilestoneFields)
			if err != nil {
				return nil, fmt.Errorf("failed to truncate milestones: %w", err)
			}

			// --- Marshal and return success
			data, err := json.Marshal(truncatedMilestones)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal milestones list: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}
