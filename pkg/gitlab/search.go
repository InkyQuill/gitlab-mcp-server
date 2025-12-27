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

// SearchProjects defines the MCP tool for searching projects globally.
func SearchProjects(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"searchProjects",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_SEARCH_PROJECTS_DESCRIPTION)),
		mcp.WithString("search",
			mcp.Description("The search query string"),
			mcp.Required(),
		),
		mcp.WithNumber("page",
			mcp.Description("The page number to retrieve (default: 1)"),
		),
		mcp.WithNumber("per_page",
			mcp.Description("The number of results per page (default: 20, max: 100)"),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Search GitLab Projects",
			ReadOnlyHint: true,
		}),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
		}

		searchQuery, err := requiredParam[string](&req, "search")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
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

		projects, resp, err := client.Search.Projects(searchQuery, opts, gl.WithContext(ctx))
		if err != nil {
			result, apiErr := HandleListAPIError(err, resp, "projects")
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		if len(projects) == 0 {
			return mcp.NewToolResultText("[]"), nil
		}

		jsonData, err := json.Marshal(projects)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal projects data: %w", err)
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// SearchIssues defines the MCP tool for searching issues globally.
func SearchIssues(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"searchIssues",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_SEARCH_ISSUES_DESCRIPTION)),
		mcp.WithString("search",
			mcp.Description("The search query string"),
			mcp.Required(),
		),
		mcp.WithNumber("page",
			mcp.Description("The page number to retrieve (default: 1)"),
		),
		mcp.WithNumber("per_page",
			mcp.Description("The number of results per page (default: 20, max: 100)"),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Search GitLab Issues",
			ReadOnlyHint: true,
		}),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
		}

		searchQuery, err := requiredParam[string](&req, "search")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
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

		issues, resp, err := client.Search.Issues(searchQuery, opts, gl.WithContext(ctx))
		if err != nil {
			result, apiErr := HandleListAPIError(err, resp, "issues")
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		if len(issues) == 0 {
			return mcp.NewToolResultText("[]"), nil
		}

		jsonData, err := json.Marshal(issues)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal issues data: %w", err)
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// SearchMergeRequests defines the MCP tool for searching merge requests globally.
func SearchMergeRequests(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"searchMergeRequests",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_SEARCH_MERGE_REQUESTS_DESCRIPTION)),
		mcp.WithString("search",
			mcp.Description("The search query string"),
			mcp.Required(),
		),
		mcp.WithNumber("page",
			mcp.Description("The page number to retrieve (default: 1)"),
		),
		mcp.WithNumber("per_page",
			mcp.Description("The number of results per page (default: 20, max: 100)"),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Search GitLab Merge Requests",
			ReadOnlyHint: true,
		}),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
		}

		searchQuery, err := requiredParam[string](&req, "search")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
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

		mrs, resp, err := client.Search.MergeRequests(searchQuery, opts, gl.WithContext(ctx))
		if err != nil {
			result, apiErr := HandleListAPIError(err, resp, "merge requests")
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		if len(mrs) == 0 {
			return mcp.NewToolResultText("[]"), nil
		}

		jsonData, err := json.Marshal(mrs)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal merge requests data: %w", err)
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// SearchBlobs defines the MCP tool for searching code blobs globally.
func SearchBlobs(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"searchBlobs",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_SEARCH_BLOBS_DESCRIPTION)),
		mcp.WithString("search",
			mcp.Description("The search query string for code search"),
			mcp.Required(),
		),
		mcp.WithNumber("page",
			mcp.Description("The page number to retrieve (default: 1)"),
		),
		mcp.WithNumber("per_page",
			mcp.Description("The number of results per page (default: 20, max: 100)"),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Search GitLab Code",
			ReadOnlyHint: true,
		}),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
		}

		searchQuery, err := requiredParam[string](&req, "search")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
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

		blobs, resp, err := client.Search.Blobs(searchQuery, opts, gl.WithContext(ctx))
		if err != nil {
			result, apiErr := HandleListAPIError(err, resp, "code blobs")
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		if len(blobs) == 0 {
			return mcp.NewToolResultText("[]"), nil
		}

		jsonData, err := json.Marshal(blobs)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal code blobs data: %w", err)
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// SearchCommits defines the MCP tool for searching commits globally.
func SearchCommits(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"searchCommits",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_SEARCH_COMMITS_DESCRIPTION)),
		mcp.WithString("search",
			mcp.Description("The search query string"),
			mcp.Required(),
		),
		mcp.WithNumber("page",
			mcp.Description("The page number to retrieve (default: 1)"),
		),
		mcp.WithNumber("per_page",
			mcp.Description("The number of results per page (default: 20, max: 100)"),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Search GitLab Commits",
			ReadOnlyHint: true,
		}),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
		}

		searchQuery, err := requiredParam[string](&req, "search")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
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

		commits, resp, err := client.Search.Commits(searchQuery, opts, gl.WithContext(ctx))
		if err != nil {
			result, apiErr := HandleListAPIError(err, resp, "commits")
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		if len(commits) == 0 {
			return mcp.NewToolResultText("[]"), nil
		}

		jsonData, err := json.Marshal(commits)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal commits data: %w", err)
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
