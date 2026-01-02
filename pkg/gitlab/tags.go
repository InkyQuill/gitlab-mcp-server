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

// Tag defines the consolidated MCP tool for managing GitLab repository tags (get, create, delete, getCommit).
func Tag(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"tag",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_TAG_DESCRIPTION)),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title: "Manage Repository Tag",
		}),
		// Required parameters
		mcp.WithString("action",
			mcp.Description("The action to perform on the tag."),
			mcp.Required(),
			mcp.Enum("get", "create", "delete", "getCommit"),
		),
		mcp.WithString("projectId",
			mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
			mcp.Required(),
		),
		mcp.WithString("tagName",
			mcp.Description("The name of the tag."),
			mcp.Required(),
		),
		// Conditional required parameters
		mcp.WithString("ref",
			mcp.Description("The branch name or commit SHA to create the tag from (required for create)."),
		),
		mcp.WithString("message",
			mcp.Description("The message for the tag annotation (optional, for create)."),
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

		tagName, err := requiredParam[string](&request, "tagName")
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
			tag, resp, err := glClient.Tags.GetTag(projectID, tagName, gl.WithContext(ctx))
			if err != nil {
				result, apiErr := HandleAPIError(err, resp, fmt.Sprintf("tag %q in project %q", tagName, projectID))
				if result != nil {
					return result, nil
				}
				return nil, apiErr
			}
			data, err := json.Marshal(tag)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal tag data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil

		case "create":
			ref, err := requiredParam[string](&request, "ref")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			message, err := OptionalParam[string](&request, "message")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			opts := &gl.CreateTagOptions{
				TagName: &tagName,
				Ref:     &ref,
			}
			if message != "" {
				opts.Message = &message
			}

			tag, resp, err := glClient.Tags.CreateTag(projectID, opts, gl.WithContext(ctx))
			if err != nil {
				result, apiErr := HandleCreateUpdateAPIError(err, resp, fmt.Sprintf("tag %q in project %q", tagName, projectID), "create")
				if result != nil {
					return result, nil
				}
				return nil, apiErr
			}
			data, err := json.Marshal(tag)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal created tag data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil

		case "delete":
			resp, err := glClient.Tags.DeleteTag(projectID, tagName, gl.WithContext(ctx))
			if err != nil {
				result, apiErr := HandleAPIError(err, resp, fmt.Sprintf("tag %q in project %q", tagName, projectID))
				if result != nil {
					return result, nil
				}
				return nil, apiErr
			}
			return mcp.NewToolResultText(fmt.Sprintf(`{"message":"Tag %q successfully deleted from project %q"}`, tagName, projectID)), nil

		case "getCommit":
			release, resp, err := glClient.Releases.GetRelease(projectID, tagName, gl.WithContext(ctx))
			if err != nil {
				result, apiErr := HandleAPIError(err, resp, fmt.Sprintf("release for tag %q in project %q", tagName, projectID))
				if result != nil {
					return result, nil
				}
				return nil, apiErr
			}
			data, err := json.Marshal(release)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal release data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil

		default:
			return mcp.NewToolResultError(fmt.Sprintf("Validation Error: invalid action %q (must be get, create, delete, or getCommit)", action)), nil
		}
	}
}

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
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
