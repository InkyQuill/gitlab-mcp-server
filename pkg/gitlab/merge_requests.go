package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gl "gitlab.com/gitlab-org/api/client-go" // GitLab client library
)

// GetMergeRequest defines the MCP tool for retrieving details of a specific merge request.
func GetMergeRequest(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"getMergeRequest",
			mcp.WithDescription("Retrieves details for a specific GitLab merge request."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title:        "Get GitLab Merge Request",
				ReadOnlyHint: true,
			}),
			mcp.WithString("projectId",
				mcp.Required(),
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
			),
			mcp.WithNumber("mergeRequestIid",
				mcp.Required(),
				mcp.Description("The IID (internal ID, integer) of the merge request within the project."),
			),
		),
		// Handler function implementation
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// --- Parse required parameters
			projectID, err := requiredParam[string](&request, "projectId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			mrIidFloat, err := requiredParam[float64](&request, "mergeRequestIid")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			mrIid := int(mrIidFloat) // Convert float64 to int for API call
			// Check if conversion lost precision
			if float64(mrIid) != mrIidFloat {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: mergeRequestIid %v is not a valid integer", mrIidFloat)), nil
			}

			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get GitLab client: %w", err)
			}

			// --- Call GitLab API
			mr, resp, err := glClient.MergeRequests.GetMergeRequest(projectID, mrIid, nil, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				code := http.StatusInternalServerError
				if resp != nil {
					code = resp.StatusCode
				}
				if code == http.StatusNotFound {
					msg := fmt.Sprintf("merge request %d not found in project %q or access denied (%d)", mrIid, projectID, code)
					return mcp.NewToolResultError(msg), nil
				}
				return nil, fmt.Errorf("failed to get merge request %d from project %q: %w (status: %d)", mrIid, projectID, err, code)
			}

			// --- Marshal and return success
			data, err := json.Marshal(mr)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal merge request data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}

// GetMergeRequestComments defines the MCP tool for retrieving comments/notes for a specific merge request.
func GetMergeRequestComments(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"getMergeRequestComments",
			mcp.WithDescription("Retrieves comments or notes from a specific merge request in a GitLab project."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title:        "Get Merge Request Comments",
				ReadOnlyHint: true,
			}),
			// Required parameters
			mcp.WithString("projectId",
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
				mcp.Required(),
			),
			mcp.WithNumber("mergeRequestIid",
				mcp.Description("The IID (internal ID, integer) of the merge request within the project."),
				mcp.Required(),
			),
			// Add standard MCP pagination parameters
			WithPagination(),
		),
		// Handler function implementation
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// --- Parse parameters
			projectID, err := requiredParam[string](&request, "projectId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			mrIidFloat, err := requiredParam[float64](&request, "mergeRequestIid")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			mrIid := int(mrIidFloat) // Convert float64 to int for API call
			// Check if conversion lost precision
			if float64(mrIid) != mrIidFloat {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: mergeRequestIid %v is not a valid integer", mrIidFloat)), nil
			}

			// Get pagination parameters
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
			opts := &gl.ListMergeRequestNotesOptions{
				ListOptions: gl.ListOptions{
					Page:    page,
					PerPage: perPage,
				},
			}

			// --- Call GitLab API
			notes, resp, err := glClient.Notes.ListMergeRequestNotes(projectID, mrIid, opts, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				code := http.StatusInternalServerError
				if resp != nil {
					code = resp.StatusCode
				}
				if code == http.StatusNotFound {
					msg := fmt.Sprintf("merge request %d not found in project %q or access denied (%d)", mrIid, projectID, code)
					return mcp.NewToolResultError(msg), nil
				}
				return nil, fmt.Errorf("failed to get comments for merge request %d from project %q: %w (status: %d)", mrIid, projectID, err, code)
			}

			// --- Marshal and return success
			// Handle empty list gracefully
			if len(notes) == 0 {
				return mcp.NewToolResultText("[]"), nil // Return empty JSON array
			}

			data, err := json.Marshal(notes)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal merge request comments data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}

// ListMergeRequests defines the MCP tool for listing merge requests with pagination and filtering.
func ListMergeRequests(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"listMergeRequests",
			mcp.WithDescription("Lists merge requests for a GitLab project with filtering and pagination options."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title:        "List GitLab Merge Requests",
				ReadOnlyHint: true,
			}),
			// Required parameters
			mcp.WithString("projectId",
				mcp.Required(),
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
			),
			// Optional filtering parameters
			mcp.WithString("state",
				mcp.Description("Return merge requests with the specified state ('opened', 'closed', 'locked', 'merged', or 'all'). Default: 'all'."),
				mcp.Enum("opened", "closed", "locked", "merged", "all"),
			),
			mcp.WithString("scope",
				mcp.Description("Return merge requests for the specified scope ('created_by_me', 'assigned_to_me', or 'all'). Default: 'all'."),
				mcp.Enum("created_by_me", "assigned_to_me", "all"),
			),
			mcp.WithString("author_id",
				mcp.Description("Return merge requests created by the specified user ID."),
			),
			mcp.WithString("assignee_id",
				mcp.Description("Return merge requests assigned to the specified user ID."),
			),
			mcp.WithString("labels",
				mcp.Description("Return merge requests matching the comma-separated list of labels."),
			),
			mcp.WithString("milestone",
				mcp.Description("Return merge requests for the specified milestone title."),
			),
			mcp.WithString("search",
				mcp.Description("Return merge requests matching the search query in their title or description."),
			),
			mcp.WithString("created_after",
				mcp.Description("Return merge requests created on or after the given datetime (ISO 8601 format)."),
			),
			mcp.WithString("created_before",
				mcp.Description("Return merge requests created on or before the given datetime (ISO 8601 format)."),
			),
			mcp.WithString("updated_after",
				mcp.Description("Return merge requests updated on or after the given datetime (ISO 8601 format)."),
			),
			mcp.WithString("updated_before",
				mcp.Description("Return merge requests updated on or before the given datetime (ISO 8601 format)."),
			),
			mcp.WithString("sort",
				mcp.Description("Return merge requests sorted in the specified order ('asc' or 'desc'). Default: 'desc'."),
				mcp.Enum("asc", "desc"),
			),
			mcp.WithString("order_by",
				mcp.Description("Return merge requests ordered by the specified field ('created_at', 'updated_at', or 'title'). Default: 'created_at'."),
				mcp.Enum("created_at", "updated_at", "title"),
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

			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
			}

			// --- Parse optional filtering parameters
			opts := &gl.ListProjectMergeRequestsOptions{}

			// Get pagination parameters
			page, perPage, err := OptionalPaginationParams(&request)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			opts.Page = page
			opts.PerPage = perPage

			// String parameters
			if state, err := OptionalParam[string](&request, "state"); err == nil && state != "" {
				opts.State = &state
			}

			if scope, err := OptionalParam[string](&request, "scope"); err == nil && scope != "" {
				opts.Scope = &scope
			}

			if authorID, err := OptionalParam[string](&request, "author_id"); err == nil && authorID != "" {
				// Convert string to int
				id, err := strconv.Atoi(authorID)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Validation Error: author_id must be a valid integer: %v", err)), nil
				}
				opts.AuthorID = &id
			}

			if assigneeID, err := OptionalParam[string](&request, "assignee_id"); err == nil && assigneeID != "" {
				// Convert string to int and wrap with AssigneeID
				id, err := strconv.Atoi(assigneeID)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Validation Error: assignee_id must be a valid integer: %v", err)), nil
				}
				opts.AssigneeID = gl.AssigneeID(id)
			}

			if labels, err := OptionalParam[string](&request, "labels"); err == nil && labels != "" {
				// Convert to LabelOptions ([]string)
				labelsList := strings.Split(labels, ",")
				labelOpts := gl.LabelOptions(labelsList)
				opts.Labels = &labelOpts
			}

			if milestone, err := OptionalParam[string](&request, "milestone"); err == nil && milestone != "" {
				opts.Milestone = &milestone
			}

			if search, err := OptionalParam[string](&request, "search"); err == nil && search != "" {
				opts.Search = &search
			}

			// Handle time parameters - parse ISO 8601 strings to time.Time
			if createdAfter, err := OptionalParam[string](&request, "created_after"); err == nil && createdAfter != "" {
				t, err := time.Parse(time.RFC3339, createdAfter)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Validation Error: created_after must be a valid ISO 8601 datetime: %v", err)), nil
				}
				opts.CreatedAfter = &t
			}

			if createdBefore, err := OptionalParam[string](&request, "created_before"); err == nil && createdBefore != "" {
				t, err := time.Parse(time.RFC3339, createdBefore)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Validation Error: created_before must be a valid ISO 8601 datetime: %v", err)), nil
				}
				opts.CreatedBefore = &t
			}

			if updatedAfter, err := OptionalParam[string](&request, "updated_after"); err == nil && updatedAfter != "" {
				t, err := time.Parse(time.RFC3339, updatedAfter)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Validation Error: updated_after must be a valid ISO 8601 datetime: %v", err)), nil
				}
				opts.UpdatedAfter = &t
			}

			if updatedBefore, err := OptionalParam[string](&request, "updated_before"); err == nil && updatedBefore != "" {
				t, err := time.Parse(time.RFC3339, updatedBefore)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Validation Error: updated_before must be a valid ISO 8601 datetime: %v", err)), nil
				}
				opts.UpdatedBefore = &t
			}

			if sort, err := OptionalParam[string](&request, "sort"); err == nil && sort != "" {
				opts.Sort = &sort
			}

			if orderBy, err := OptionalParam[string](&request, "order_by"); err == nil && orderBy != "" {
				opts.OrderBy = &orderBy
			}

			// --- Call GitLab API
			mrs, resp, err := glClient.MergeRequests.ListProjectMergeRequests(projectID, opts, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				code := http.StatusInternalServerError
				if resp != nil {
					code = resp.StatusCode
				}
				if code == http.StatusNotFound {
					msg := fmt.Sprintf("project %q not found or access denied (%d)", projectID, code)
					return mcp.NewToolResultError(msg), nil
				}
				return nil, fmt.Errorf("failed to list merge requests for project %q: %w (status: %d)", projectID, err, code)
			}

			// --- Marshal and return success
			// Handle empty list gracefully
			if len(mrs) == 0 {
				return mcp.NewToolResultText("[]"), nil // Return empty JSON array
			}

			data, err := json.Marshal(mrs)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal merge requests data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}

// CreateMergeRequest defines the MCP tool for creating a new GitLab merge request.
func CreateMergeRequest(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"createMergeRequest",
			mcp.WithDescription("Creates a new merge request in a GitLab project."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title: "Create GitLab Merge Request",
			}),
			// Required parameters
			mcp.WithString("projectId",
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
				mcp.Required(),
			),
			mcp.WithString("sourceBranch",
				mcp.Description("The source branch name."),
				mcp.Required(),
			),
			mcp.WithString("targetBranch",
				mcp.Description("The target branch name."),
				mcp.Required(),
			),
			mcp.WithString("title",
				mcp.Description("The title of the merge request."),
				mcp.Required(),
			),
			// Optional parameters
			mcp.WithString("description",
				mcp.Description("The description of the merge request."),
			),
			mcp.WithString("labels",
				mcp.Description("Comma-separated list of label names to apply to the merge request."),
			),
			mcp.WithString("assigneeIds",
				mcp.Description("Comma-separated list of user IDs to assign the merge request to."),
			),
			mcp.WithNumber("milestoneId",
				mcp.Description("The ID of the milestone to associate the merge request with."),
			),
			mcp.WithBoolean("removeSourceBranch",
				mcp.Description("Flag indicating if the source branch should be removed after merge."),
			),
			mcp.WithBoolean("squash",
				mcp.Description("Flag indicating if commits should be squashed into a single commit on merge."),
			),
		),
		// Handler function implementation
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// --- Parse required parameters
			projectID, err := requiredParam[string](&request, "projectId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			sourceBranch, err := requiredParam[string](&request, "sourceBranch")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			targetBranch, err := requiredParam[string](&request, "targetBranch")
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

			labels, err := OptionalParam[string](&request, "labels")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			assigneeIdsStr, err := OptionalParam[string](&request, "assigneeIds")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			milestoneIDFloat, err := OptionalParam[float64](&request, "milestoneId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			removeSourceBranch, err := OptionalBoolParam(&request, "removeSourceBranch")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			squash, err := OptionalBoolParam(&request, "squash")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
			}

			// --- Construct GitLab API options
			opts := &gl.CreateMergeRequestOptions{
				SourceBranch: &sourceBranch,
				TargetBranch: &targetBranch,
				Title:        &title,
			}

			if description != "" {
				opts.Description = &description
			}

			if labels != "" {
				labelSlice := strings.Split(labels, ",")
				// Trim whitespace from each label
				for i, label := range labelSlice {
					labelSlice[i] = strings.TrimSpace(label)
				}
				labelOpts := gl.LabelOptions(labelSlice)
				opts.Labels = &labelOpts
			}

			if assigneeIdsStr != "" {
				assigneeIdsList := strings.Split(assigneeIdsStr, ",")
				assigneeIds := make([]int, 0, len(assigneeIdsList))
				for _, idStr := range assigneeIdsList {
					idStr = strings.TrimSpace(idStr)
					if idStr == "" {
						continue
					}
					id, err := strconv.Atoi(idStr)
					if err != nil {
						return mcp.NewToolResultError(fmt.Sprintf("Validation Error: invalid assignee ID %q: %v", idStr, err)), nil
					}
					assigneeIds = append(assigneeIds, id)
				}
				if len(assigneeIds) > 0 {
					opts.AssigneeIDs = &assigneeIds
				}
			}

			if milestoneIDFloat != 0 {
				milestoneID := int(milestoneIDFloat)
				if float64(milestoneID) != milestoneIDFloat {
					return mcp.NewToolResultError(fmt.Sprintf("Validation Error: milestoneId %v is not a valid integer", milestoneIDFloat)), nil
				}
				opts.MilestoneID = &milestoneID
			}

			if removeSourceBranch != nil {
				opts.RemoveSourceBranch = removeSourceBranch
			}

			if squash != nil {
				opts.Squash = squash
			}

			// --- Call GitLab API
			mr, resp, err := glClient.MergeRequests.CreateMergeRequest(projectID, opts, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				code := http.StatusInternalServerError
				if resp != nil {
					code = resp.StatusCode
				}
				if code == http.StatusNotFound {
					return mcp.NewToolResultError(fmt.Sprintf("project %q not found or access denied (%d)", projectID, code)), nil
				}
				if code == http.StatusBadRequest || code == http.StatusUnprocessableEntity {
					return mcp.NewToolResultError(fmt.Sprintf("failed to create merge request: %v (status: %d)", err, code)), nil
				}
				return nil, fmt.Errorf("failed to create merge request in project %q: %w (status: %d)", projectID, err, code)
			}

			// --- Marshal and return success
			data, err := json.Marshal(mr)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal merge request data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}

// UpdateMergeRequest defines the MCP tool for updating an existing GitLab merge request.
func UpdateMergeRequest(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"updateMergeRequest",
			mcp.WithDescription("Updates an existing merge request in a GitLab project."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title: "Update GitLab Merge Request",
			}),
			// Required parameters
			mcp.WithString("projectId",
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
				mcp.Required(),
			),
			mcp.WithNumber("mergeRequestIid",
				mcp.Description("The IID (internal ID, integer) of the merge request within the project."),
				mcp.Required(),
			),
			// Optional parameters
			mcp.WithString("title",
				mcp.Description("The title of the merge request."),
			),
			mcp.WithString("description",
				mcp.Description("The description of the merge request."),
			),
			mcp.WithString("targetBranch",
				mcp.Description("The target branch name."),
			),
			mcp.WithString("labels",
				mcp.Description("Comma-separated list of label names to apply to the merge request."),
			),
			mcp.WithString("assigneeIds",
				mcp.Description("Comma-separated list of user IDs to assign the merge request to."),
			),
			mcp.WithNumber("milestoneId",
				mcp.Description("The ID of the milestone to associate the merge request with."),
			),
			mcp.WithString("stateEvent",
				mcp.Description("The state event to perform on the merge request (close, reopen, merge)."),
				mcp.Enum("close", "reopen", "merge"),
			),
			mcp.WithBoolean("removeSourceBranch",
				mcp.Description("Flag indicating if the source branch should be removed after merge."),
			),
			mcp.WithBoolean("squash",
				mcp.Description("Flag indicating if commits should be squashed into a single commit on merge."),
			),
		),
		// Handler function implementation
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// --- Parse required parameters
			projectID, err := requiredParam[string](&request, "projectId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			mrIidFloat, err := requiredParam[float64](&request, "mergeRequestIid")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			mrIid := int(mrIidFloat)
			if float64(mrIid) != mrIidFloat {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: mergeRequestIid %v is not a valid integer", mrIidFloat)), nil
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

			targetBranch, err := OptionalParam[string](&request, "targetBranch")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			labels, err := OptionalParam[string](&request, "labels")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			assigneeIdsStr, err := OptionalParam[string](&request, "assigneeIds")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			milestoneIDFloat, err := OptionalParam[float64](&request, "milestoneId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			stateEvent, err := OptionalParam[string](&request, "stateEvent")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			removeSourceBranch, err := OptionalBoolParam(&request, "removeSourceBranch")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			squash, err := OptionalBoolParam(&request, "squash")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
			}

			// --- Construct GitLab API options
			opts := &gl.UpdateMergeRequestOptions{}

			if title != "" {
				opts.Title = &title
			}

			if description != "" {
				opts.Description = &description
			}

			if targetBranch != "" {
				opts.TargetBranch = &targetBranch
			}

			if labels != "" {
				labelSlice := strings.Split(labels, ",")
				// Trim whitespace from each label
				for i, label := range labelSlice {
					labelSlice[i] = strings.TrimSpace(label)
				}
				labelOpts := gl.LabelOptions(labelSlice)
				opts.Labels = &labelOpts
			}

			if assigneeIdsStr != "" {
				assigneeIdsList := strings.Split(assigneeIdsStr, ",")
				assigneeIds := make([]int, 0, len(assigneeIdsList))
				for _, idStr := range assigneeIdsList {
					idStr = strings.TrimSpace(idStr)
					if idStr == "" {
						continue
					}
					id, err := strconv.Atoi(idStr)
					if err != nil {
						return mcp.NewToolResultError(fmt.Sprintf("Validation Error: invalid assignee ID %q: %v", idStr, err)), nil
					}
					assigneeIds = append(assigneeIds, id)
				}
				if len(assigneeIds) > 0 {
					opts.AssigneeIDs = &assigneeIds
				}
			}

			if milestoneIDFloat != 0 {
				milestoneID := int(milestoneIDFloat)
				if float64(milestoneID) != milestoneIDFloat {
					return mcp.NewToolResultError(fmt.Sprintf("Validation Error: milestoneId %v is not a valid integer", milestoneIDFloat)), nil
				}
				opts.MilestoneID = &milestoneID
			}

			if stateEvent != "" {
				opts.StateEvent = &stateEvent
			}

			if removeSourceBranch != nil {
				opts.RemoveSourceBranch = removeSourceBranch
			}

			if squash != nil {
				opts.Squash = squash
			}

			// --- Call GitLab API
			mr, resp, err := glClient.MergeRequests.UpdateMergeRequest(projectID, mrIid, opts, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				code := http.StatusInternalServerError
				if resp != nil {
					code = resp.StatusCode
				}
				if code == http.StatusNotFound {
					return mcp.NewToolResultError(fmt.Sprintf("merge request %d not found in project %q or access denied (%d)", mrIid, projectID, code)), nil
				}
				if code == http.StatusBadRequest || code == http.StatusUnprocessableEntity {
					return mcp.NewToolResultError(fmt.Sprintf("failed to update merge request: %v (status: %d)", err, code)), nil
				}
				return nil, fmt.Errorf("failed to update merge request %d in project %q: %w (status: %d)", mrIid, projectID, err, code)
			}

			// --- Marshal and return success
			data, err := json.Marshal(mr)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal merge request data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}

// CreateMergeRequestComment defines the MCP tool for creating a comment on a GitLab merge request.
func CreateMergeRequestComment(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"createMergeRequestComment",
			mcp.WithDescription("Creates a comment (note) on a specific GitLab merge request."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title: "Create Merge Request Comment",
			}),
			// Required parameters
			mcp.WithString("projectId",
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
				mcp.Required(),
			),
			mcp.WithNumber("mergeRequestIid",
				mcp.Description("The IID (internal ID, integer) of the merge request within the project."),
				mcp.Required(),
			),
			mcp.WithString("body",
				mcp.Description("The content of the comment."),
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

			mrIidFloat, err := requiredParam[float64](&request, "mergeRequestIid")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			mrIid := int(mrIidFloat)
			if float64(mrIid) != mrIidFloat {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: mergeRequestIid %v is not a valid integer", mrIidFloat)), nil
			}

			body, err := requiredParam[string](&request, "body")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
			}

			// --- Construct GitLab API options
			opts := &gl.CreateMergeRequestNoteOptions{
				Body: &body,
			}

			// --- Call GitLab API
			note, resp, err := glClient.Notes.CreateMergeRequestNote(projectID, mrIid, opts, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				code := http.StatusInternalServerError
				if resp != nil {
					code = resp.StatusCode
				}
				if code == http.StatusNotFound {
					return mcp.NewToolResultError(fmt.Sprintf("merge request %d not found in project %q or access denied (%d)", mrIid, projectID, code)), nil
				}
				if code == http.StatusBadRequest || code == http.StatusUnprocessableEntity {
					return mcp.NewToolResultError(fmt.Sprintf("failed to create comment: %v (status: %d)", err, code)), nil
				}
				return nil, fmt.Errorf("failed to create comment on merge request %d in project %q: %w (status: %d)", mrIid, projectID, err, code)
			}

			// --- Marshal and return success
			data, err := json.Marshal(note)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal comment data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}

// UpdateMergeRequestComment defines the MCP tool for updating a comment on a GitLab merge request.
func UpdateMergeRequestComment(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"updateMergeRequestComment",
			mcp.WithDescription("Updates an existing comment (note) on a specific GitLab merge request."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title: "Update Merge Request Comment",
			}),
			// Required parameters
			mcp.WithString("projectId",
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
				mcp.Required(),
			),
			mcp.WithNumber("mergeRequestIid",
				mcp.Description("The IID (internal ID, integer) of the merge request within the project."),
				mcp.Required(),
			),
			mcp.WithNumber("noteId",
				mcp.Description("The ID of the note (comment) to update."),
				mcp.Required(),
			),
			mcp.WithString("body",
				mcp.Description("The updated content of the comment."),
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

			mrIidFloat, err := requiredParam[float64](&request, "mergeRequestIid")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			mrIid := int(mrIidFloat)
			if float64(mrIid) != mrIidFloat {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: mergeRequestIid %v is not a valid integer", mrIidFloat)), nil
			}

			noteIDFloat, err := requiredParam[float64](&request, "noteId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			noteID := int(noteIDFloat)
			if float64(noteID) != noteIDFloat {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: noteId %v is not a valid integer", noteIDFloat)), nil
			}

			body, err := requiredParam[string](&request, "body")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
			}

			// --- Construct GitLab API options
			opts := &gl.UpdateMergeRequestNoteOptions{
				Body: &body,
			}

			// --- Call GitLab API
			note, resp, err := glClient.Notes.UpdateMergeRequestNote(projectID, mrIid, noteID, opts, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				code := http.StatusInternalServerError
				if resp != nil {
					code = resp.StatusCode
				}
				if code == http.StatusNotFound {
					return mcp.NewToolResultError(fmt.Sprintf("merge request %d or note %d not found in project %q or access denied (%d)", mrIid, noteID, projectID, code)), nil
				}
				if code == http.StatusBadRequest || code == http.StatusUnprocessableEntity {
					return mcp.NewToolResultError(fmt.Sprintf("failed to update comment: %v (status: %d)", err, code)), nil
				}
				return nil, fmt.Errorf("failed to update comment %d on merge request %d in project %q: %w (status: %d)", noteID, mrIid, projectID, err, code)
			}

			// --- Marshal and return success
			data, err := json.Marshal(note)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal comment data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}
