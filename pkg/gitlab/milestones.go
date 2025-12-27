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

// CreateMilestone defines the MCP tool for creating a new GitLab milestone.
func CreateMilestone(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"createMilestone",
			mcp.WithDescription(translations.Translate(t, translations.TOOL_CREATE_MILESTONE_DESCRIPTION)),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title: "Create GitLab Milestone",
			}),
			// Required parameters
			mcp.WithString("projectId",
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
				mcp.Required(),
			),
			mcp.WithString("title",
				mcp.Description("The title of the milestone."),
				mcp.Required(),
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
		),
		// Handler function implementation
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// --- Parse required parameters
			projectID, err := requiredParam[string](&request, "projectId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			title, err := requiredParam[string](&request, "title")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Parse optional parameters
			description, err := OptionalParam[string](&request, "description")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			dueDateStr, err := OptionalParam[string](&request, "dueDate")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			startDateStr, err := OptionalParam[string](&request, "startDate")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
			}

			// --- Construct GitLab API options
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

			// --- Call GitLab API
			milestone, resp, err := glClient.Milestones.CreateMilestone(projectID, opts, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				result, apiErr := HandleCreateUpdateAPIError(err, resp, fmt.Sprintf("project %q", projectID), "create milestone")
				if result != nil {
					return result, nil
				}
				return nil, apiErr
			}

			// --- Marshal and return success
			data, err := json.Marshal(milestone)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal milestone data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}

// UpdateMilestone defines the MCP tool for updating an existing GitLab milestone.
func UpdateMilestone(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"updateMilestone",
			mcp.WithDescription(translations.Translate(t, translations.TOOL_UPDATE_MILESTONE_DESCRIPTION)),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title: "Update GitLab Milestone",
			}),
			// Required parameters
			mcp.WithString("projectId",
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
				mcp.Required(),
			),
			mcp.WithNumber("milestoneId",
				mcp.Description("The ID of the milestone."),
				mcp.Required(),
			),
			// Optional parameters
			mcp.WithString("title",
				mcp.Description("The title of the milestone."),
			),
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
				mcp.Description("The state event to perform on the milestone (activate, close)."),
				mcp.Enum("activate", "close"),
			),
		),
		// Handler function implementation
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// --- Parse required parameters
			projectID, err := requiredParam[string](&request, "projectId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			milestoneIDFloat, err := requiredParam[float64](&request, "milestoneId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			milestoneID := int(milestoneIDFloat)
			if float64(milestoneID) != milestoneIDFloat {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: milestoneId %v is not a valid integer", milestoneIDFloat)), nil
			}

			// --- Parse optional parameters
			title, err := OptionalParam[string](&request, "title")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			description, err := OptionalParam[string](&request, "description")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			dueDateStr, err := OptionalParam[string](&request, "dueDate")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			startDateStr, err := OptionalParam[string](&request, "startDate")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			stateEvent, err := OptionalParam[string](&request, "stateEvent")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
			}

			// --- Construct GitLab API options
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

			// --- Call GitLab API
			milestone, resp, err := glClient.Milestones.UpdateMilestone(projectID, milestoneID, opts, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				result, apiErr := HandleCreateUpdateAPIError(err, resp, fmt.Sprintf("milestone %d in project %q", milestoneID, projectID), "update milestone")
				if result != nil {
					return result, nil
				}
				return nil, apiErr
			}

			// --- Marshal and return success
			data, err := json.Marshal(milestone)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal milestone data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}

// GetMilestone defines the MCP tool for retrieving a specific GitLab milestone.
func GetMilestone(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"getMilestone",
			mcp.WithDescription(translations.Translate(t, translations.TOOL_GET_MILESTONE_DESCRIPTION)),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title:        "Get GitLab Milestone",
				ReadOnlyHint: true,
			}),
			// Required parameters
			mcp.WithString("projectId",
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
				mcp.Required(),
			),
			mcp.WithNumber("milestoneId",
				mcp.Description("The ID of the milestone."),
				mcp.Required(),
			),
		),
		// Handler function implementation
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// --- Parse required parameters
			projectID, err := requiredParam[string](&request, "projectId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			milestoneIDFloat, err := requiredParam[float64](&request, "milestoneId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			milestoneID := int(milestoneIDFloat)
			if float64(milestoneID) != milestoneIDFloat {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: milestoneId %v is not a valid integer", milestoneIDFloat)), nil
			}

			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
			}

			// --- Call GitLab API
			milestone, resp, err := glClient.Milestones.GetMilestone(projectID, milestoneID, nil, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				result, apiErr := HandleAPIError(err, resp, fmt.Sprintf("milestone %d in project %q", milestoneID, projectID))
				if result != nil {
					return result, nil
				}
				return nil, apiErr
			}

			// --- Marshal and return success
			data, err := json.Marshal(milestone)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal milestone data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
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

			// --- Marshal and return success
			data, err := json.Marshal(milestones)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal milestones list: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}
