package gitlab

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/LuisCusihuaman/gitlab-mcp-server/pkg/translations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gl "gitlab.com/gitlab-org/api/client-go" // GitLab client library
)

// GetProjectBranches defines the MCP tool for listing branches in a project.
func GetProjectBranches(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"getProjectBranches",
			mcp.WithDescription(translations.Translate(t, translations.TOOL_GET_PROJECT_BRANCHES_DESCRIPTION)),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title:        "List Project Branches",
				ReadOnlyHint: true,
			}),
			mcp.WithString("projectId",
				mcp.Required(),
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
			),
			mcp.WithString("search",
				mcp.Description("Return list of branches matching the search criteria."),
			),
			// Add standard MCP pagination parameters
			WithPagination(),
		),
		// Handler function implementation
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// --- Parse parameters
			projectIDStr, err := requiredParam[string](&request, "projectId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			search, err := OptionalParam[string](&request, "search")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			page, perPage, err := OptionalPaginationParams(&request)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Construct GitLab API options
			opts := &gl.ListBranchesOptions{
				ListOptions: gl.ListOptions{
					Page:    page,
					PerPage: perPage,
				},
			}
			if search != "" {
				opts.Search = &search
			}

			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get GitLab client: %w", err)
			}

			// --- Call GitLab API
			branches, resp, err := glClient.Branches.ListBranches(projectIDStr, opts, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				result, apiErr := HandleListAPIError(err, resp, fmt.Sprintf("branches for project %q", projectIDStr))
				if result != nil {
					return result, nil
				}
				return nil, apiErr
			}

			// --- Marshal and return success
			// Handle empty list gracefully
			if len(branches) == 0 {
				return mcp.NewToolResultText("[]"), nil // Return empty JSON array
			}

			data, err := json.Marshal(branches)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal branch list data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}
