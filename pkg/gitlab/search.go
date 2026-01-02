package gitlab

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/translations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gl "gitlab.com/gitlab-org/api/client-go"
)

// Search defines the unified MCP tool for searching across GitLab resources.
// This consolidates 19 previous search tools into one with resourceType and scope parameters.
func Search(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"search",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_SEARCH_DESCRIPTION)),
		mcp.WithString("resourceType",
			mcp.Description("The type of resource to search for"),
			mcp.Required(),
			mcp.Enum("projects", "issues", "merge_requests", "blobs", "commits", "milestones", "snippet_titles", "snippet_blobs", "wiki_blobs", "notes"),
		),
		mcp.WithString("search",
			mcp.Description("The search query string"),
			mcp.Required(),
		),
		mcp.WithString("scope",
			mcp.Description("The scope of the search (global, group, or project)"),
			mcp.Enum("global", "group", "project"),
		),
		mcp.WithString("gid",
			mcp.Description("The ID or URL-encoded path of the group (required when scope='group')"),
		),
		mcp.WithString("pid",
			mcp.Description("The ID or URL-encoded path of the project (required when scope='project')"),
		),
		mcp.WithNumber("page",
			mcp.Description("The page number to retrieve (default: 1)"),
		),
		mcp.WithNumber("per_page",
			mcp.Description("The number of results per page (default: 20, max: 100)"),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Search GitLab Resources",
			ReadOnlyHint: true,
		}),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
		}

		// Parse required parameters
		resourceType, err := requiredParam[string](&req, "resourceType")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		searchQuery, err := requiredParam[string](&req, "search")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		// Parse optional scope parameter (defaults to "global")
		scope, err := OptionalParam[string](&req, "scope")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		if scope == "" {
			scope = "global"
		}

		// Parse scope-specific parameters
		gid, err := OptionalParam[string](&req, "gid")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		pid, err := OptionalParam[string](&req, "pid")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		// Validate scope-specific parameters
		if scope == "group" && gid == "" {
			return mcp.NewToolResultError("Validation Error: gid is required when scope='group'"), nil
		}
		if scope == "project" && pid == "" {
			return mcp.NewToolResultError("Validation Error: pid is required when scope='project'"), nil
		}

		page, perPage, err := OptionalPaginationParams(&req)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		opts := &gl.SearchOptions{
			ListOptions: gl.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		}

		// Route to appropriate GitLab API method based on resourceType and scope
		var results interface{}
		var resp *gl.Response
		var apiErr error

		switch resourceType {
		case "projects":
			switch scope {
			case "global":
				results, resp, apiErr = client.Search.Projects(searchQuery, opts, gl.WithContext(ctx))
			case "group":
				results, resp, apiErr = client.Search.ProjectsByGroup(gid, searchQuery, opts, gl.WithContext(ctx))
			default:
				return mcp.NewToolResultError("Validation Error: projects search only supports global or group scope"), nil
			}

		case "issues":
			switch scope {
			case "global":
				results, resp, apiErr = client.Search.Issues(searchQuery, opts, gl.WithContext(ctx))
			case "group":
				results, resp, apiErr = client.Search.IssuesByGroup(gid, searchQuery, opts, gl.WithContext(ctx))
			case "project":
				results, resp, apiErr = client.Search.IssuesByProject(pid, searchQuery, opts, gl.WithContext(ctx))
			}

		case "merge_requests":
			switch scope {
			case "global":
				results, resp, apiErr = client.Search.MergeRequests(searchQuery, opts, gl.WithContext(ctx))
			case "group":
				results, resp, apiErr = client.Search.MergeRequestsByGroup(gid, searchQuery, opts, gl.WithContext(ctx))
			case "project":
				results, resp, apiErr = client.Search.MergeRequestsByProject(pid, searchQuery, opts, gl.WithContext(ctx))
			}

		case "blobs":
			switch scope {
			case "global":
				results, resp, apiErr = client.Search.Blobs(searchQuery, opts, gl.WithContext(ctx))
			case "group":
				results, resp, apiErr = client.Search.BlobsByGroup(gid, searchQuery, opts, gl.WithContext(ctx))
			case "project":
				results, resp, apiErr = client.Search.BlobsByProject(pid, searchQuery, opts, gl.WithContext(ctx))
			}

		case "commits":
			switch scope {
			case "global":
				results, resp, apiErr = client.Search.Commits(searchQuery, opts, gl.WithContext(ctx))
			case "project":
				results, resp, apiErr = client.Search.CommitsByProject(pid, searchQuery, opts, gl.WithContext(ctx))
			default:
				return mcp.NewToolResultError("Validation Error: commits search only supports global or project scope"), nil
			}

		case "milestones":
			switch scope {
			case "global":
				results, resp, apiErr = client.Search.Milestones(searchQuery, opts, gl.WithContext(ctx))
			case "group":
				results, resp, apiErr = client.Search.MilestonesByGroup(gid, searchQuery, opts, gl.WithContext(ctx))
			case "project":
				results, resp, apiErr = client.Search.MilestonesByProject(pid, searchQuery, opts, gl.WithContext(ctx))
			}

		case "snippet_titles":
			if scope != "global" {
				return mcp.NewToolResultError("Validation Error: snippet_titles search only supports global scope"), nil
			}
			results, resp, apiErr = client.Search.SnippetTitles(searchQuery, opts, gl.WithContext(ctx))

		case "snippet_blobs":
			if scope != "global" {
				return mcp.NewToolResultError("Validation Error: snippet_blobs search only supports global scope"), nil
			}
			results, resp, apiErr = client.Search.SnippetBlobs(searchQuery, opts, gl.WithContext(ctx))

		case "wiki_blobs":
			if scope != "global" {
				return mcp.NewToolResultError("Validation Error: wiki_blobs search only supports global scope"), nil
			}
			results, resp, apiErr = client.Search.WikiBlobs(searchQuery, opts, gl.WithContext(ctx))

		case "notes":
			if scope != "project" {
				return mcp.NewToolResultError("Validation Error: notes search only supports project scope"), nil
			}
			results, resp, apiErr = client.Search.NotesByProject(pid, searchQuery, opts, gl.WithContext(ctx))

		default:
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: unsupported resourceType '%s'", resourceType)), nil
		}

		// Handle API errors
		if apiErr != nil {
			result, err := HandleListAPIError(apiErr, resp, fmt.Sprintf("%s (scope=%s)", resourceType, scope))
			if result != nil {
				return result, nil
			}
			return nil, err
		}

		// Handle empty results
		if results == nil || (fmt.Sprintf("%v", results) == "[]") {
			return mcp.NewToolResultText("[]"), nil
		}

		// Marshal and return results
		jsonData, err := json.Marshal(results)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal search results: %w", err)
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
