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

// ListRepositoryTags defines the MCP tool for listing all tags in a repository.
func ListRepositoryTags(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"listRepositoryTags",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_LIST_REPOSITORY_TAGS_DESCRIPTION)),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List Repository Tags",
			ReadOnlyHint: true,
		}),
		mcp.WithString("projectId",
			mcp.Required(),
			mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
		),
		mcp.WithString("search",
			mcp.Description("Return list of tags matching the search criteria."),
		),
		WithPagination(),
	),
	func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse parameters
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

		// Construct GitLab API options
		opts := &gl.ListTagsOptions{
			ListOptions: gl.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		}
		if search != "" {
			opts.Search = &search
		}

		// Obtain GitLab client
		glClient, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitLab client: %w", err)
		}

		// Call GitLab API
		tags, resp, err := glClient.Tags.ListTags(projectIDStr, opts, gl.WithContext(ctx))

		// Handle API errors
		if err != nil {
			result, apiErr := HandleListAPIError(err, resp, fmt.Sprintf("tags for project %q", projectIDStr))
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		// Marshal and return success
		if len(tags) == 0 {
			return mcp.NewToolResultText("[]"), nil
		}

		data, err := json.Marshal(tags)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tag list data: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// GetRepositoryTag defines the MCP tool for getting details of a specific tag.
func GetRepositoryTag(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"getRepositoryTag",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_GET_REPOSITORY_TAG_DESCRIPTION)),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Repository Tag",
			ReadOnlyHint: true,
		}),
		mcp.WithString("projectId",
			mcp.Required(),
			mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
		),
		mcp.WithString("tagName",
			mcp.Required(),
			mcp.Description("The name of the tag."),
		),
	),
	func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse parameters
		projectIDStr, err := requiredParam[string](&request, "projectId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		tagName, err := requiredParam[string](&request, "tagName")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		// Obtain GitLab client
		glClient, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitLab client: %w", err)
		}

		// Call GitLab API
		tag, resp, err := glClient.Tags.GetTag(projectIDStr, tagName, gl.WithContext(ctx))

		// Handle API errors
		if err != nil {
			result, apiErr := HandleAPIError(err, resp, fmt.Sprintf("tag %q in project %q", tagName, projectIDStr))
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		// Marshal and return success
		data, err := json.Marshal(tag)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tag data: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// CreateRepositoryTag defines the MCP tool for creating a new tag.
func CreateRepositoryTag(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"createRepositoryTag",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_CREATE_REPOSITORY_TAG_DESCRIPTION)),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Create Repository Tag",
			ReadOnlyHint: false,
		}),
		mcp.WithString("projectId",
			mcp.Required(),
			mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
		),
		mcp.WithString("tagName",
			mcp.Required(),
			mcp.Description("The name of the tag."),
		),
		mcp.WithString("ref",
			mcp.Required(),
			mcp.Description("The branch name or commit SHA to create the tag from."),
		),
		mcp.WithString("message",
			mcp.Description("The message for the tag annotation (creates annotated tag)."),
		),
	),
	func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse parameters
		projectIDStr, err := requiredParam[string](&request, "projectId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		tagName, err := requiredParam[string](&request, "tagName")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		ref, err := requiredParam[string](&request, "ref")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		message, err := OptionalParam[string](&request, "message")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		// Obtain GitLab client
		glClient, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitLab client: %w", err)
		}

		// Construct GitLab API options
		opts := &gl.CreateTagOptions{
			TagName: &tagName,
			Ref:     &ref,
		}
		if message != "" {
			opts.Message = &message
		}

		// Call GitLab API
		tag, resp, err := glClient.Tags.CreateTag(projectIDStr, opts, gl.WithContext(ctx))

		// Handle API errors
		if err != nil {
			result, apiErr := HandleCreateUpdateAPIError(err, resp, fmt.Sprintf("tag %q in project %q", tagName, projectIDStr), "create")
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		// Marshal and return success
		data, err := json.Marshal(tag)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal created tag data: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// DeleteRepositoryTag defines the MCP tool for deleting a tag.
func DeleteRepositoryTag(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"deleteRepositoryTag",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_DELETE_REPOSITORY_TAG_DESCRIPTION)),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Delete Repository Tag",
			ReadOnlyHint: false,
		}),
		mcp.WithString("projectId",
			mcp.Required(),
			mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
		),
		mcp.WithString("tagName",
			mcp.Required(),
			mcp.Description("The name of the tag to delete."),
		),
	),
	func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse parameters
		projectIDStr, err := requiredParam[string](&request, "projectId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		tagName, err := requiredParam[string](&request, "tagName")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		// Obtain GitLab client
		glClient, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitLab client: %w", err)
		}

		// Call GitLab API
		resp, err := glClient.Tags.DeleteTag(projectIDStr, tagName, gl.WithContext(ctx))

		// Handle API errors
		if err != nil {
			result, apiErr := HandleAPIError(err, resp, fmt.Sprintf("tag %q in project %q", tagName, projectIDStr))
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		// Return success message
		return mcp.NewToolResultText(fmt.Sprintf(`{"message":"Tag %q successfully deleted from project %q"}`, tagName, projectIDStr)), nil
	}
}

// GetTagCommit defines the MCP tool for getting commit details for a tag.
func GetTagCommit(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"getTagCommit",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_GET_TAG_COMMIT_DESCRIPTION)),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get Tag Commit",
			ReadOnlyHint: true,
		}),
		mcp.WithString("projectId",
			mcp.Required(),
			mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
		),
		mcp.WithString("tagName",
			mcp.Required(),
			mcp.Description("The name of the tag."),
		),
	),
	func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse parameters
		projectIDStr, err := requiredParam[string](&request, "projectId")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}
		tagName, err := requiredParam[string](&request, "tagName")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
		}

		// Obtain GitLab client
		glClient, err := getClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitLab client: %w", err)
		}

		// Call GitLab API to get the release (which includes commit info)
		release, resp, err := glClient.Releases.GetRelease(projectIDStr, tagName, gl.WithContext(ctx))

		// Handle API errors
		if err != nil {
			result, apiErr := HandleAPIError(err, resp, fmt.Sprintf("release for tag %q in project %q", tagName, projectIDStr))
			if result != nil {
				return result, nil
			}
			return nil, apiErr
		}

		// Marshal and return success
		data, err := json.Marshal(release)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal release data: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
