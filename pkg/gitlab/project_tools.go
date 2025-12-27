package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gl "gitlab.com/gitlab-org/api/client-go"
)

// SetCurrentProject sets the current GitLab project for the working directory.
// This creates a .gmcprc file that stores the project ID and optionally the GitLab host.
func SetCurrentProject(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"setCurrentProject",
			mcp.WithDescription("Sets the current GitLab project for this directory by creating a .gmcprc file. The project ID will be automatically used by other tools if not explicitly specified."),
			mcp.WithString("projectId",
				mcp.Required(),
				mcp.Description("The GitLab project ID (e.g., 'owner/repo' or numeric ID)."),
			),
			mcp.WithString("gitlabHost",
				mcp.Description("Optional GitLab host URL (e.g., 'https://gitlab.example.com'). If not specified, uses default or detected from Git remote."),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Parse parameters
			projectID, err := requiredParam[string](&request, "projectId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			gitlabHost, _ := request.Params.Arguments["gitlabHost"].(string)

			// Get current working directory
			cwd, err := os.Getwd()
			if err != nil {
				return nil, fmt.Errorf("failed to get working directory: %w", err)
			}

			// Create project config
			config := &ProjectConfig{
				ProjectID:  projectID,
				GitLabHost: gitlabHost,
			}

			// Write config file
			configPath, err := WriteProjectConfig(cwd, config)
			if err != nil {
				return nil, fmt.Errorf("failed to write project config: %w", err)
			}

			result := map[string]interface{}{
				"success":    true,
				"configPath": configPath,
				"projectId":  projectID,
				"gitlabHost": gitlabHost,
			}

			data, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		}
}

// GetCurrentProject retrieves the current GitLab project from .gmcprc file.
func GetCurrentProject(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"getCurrentProject",
			mcp.WithDescription("Gets the current GitLab project configuration from the .gmcprc file in the current or parent directory."),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			config, configPath, err := FindProjectConfig()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to read project config: %v", err)), nil
			}

			if config == nil {
				return mcp.NewToolResultText(`{
  "found": false,
  "message": "No .gmcprc file found in current or parent directories. Use 'setCurrentProject' to create one."
}`), nil
			}

			result := map[string]interface{}{
				"found":      true,
				"configPath": configPath,
				"projectId":  config.ProjectID,
				"gitlabHost": config.GitLabHost,
			}

			data, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		}
}

// DetectProject attempts to auto-detect the GitLab project from Git remote configuration.
func DetectProject(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"detectProject",
			mcp.WithDescription("Auto-detects the GitLab project from the Git remote URL in the current directory. Useful for quickly setting up the project context."),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID, gitlabHost, err := DetectProjectFromGit()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to detect project: %v", err)), nil
			}

			// Verify the project exists by calling GitLab API
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get GitLab client: %w", err)
			}

			project, _, err := glClient.Projects.GetProject(projectID, nil, gl.WithContext(ctx))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Detected project '%s' but could not verify it exists: %v", projectID, err)), nil
			}

			result := map[string]interface{}{
				"success":      true,
				"projectId":    projectID,
				"gitlabHost":   gitlabHost,
				"projectName":  project.Name,
				"projectPath":  project.PathWithNamespace,
				"message":      fmt.Sprintf("Project detected successfully. Use 'setCurrentProject' with projectId='%s' to save it.", projectID),
			}

			data, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		}
}

// AutoDetectAndSetProject combines detection and setting in one command
func AutoDetectAndSetProject(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"autoDetectAndSetProject",
			mcp.WithDescription("Auto-detects the GitLab project from Git remote and creates a .gmcprc file. This is a convenience command that combines detectProject and setCurrentProject."),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID, gitlabHost, err := DetectProjectFromGit()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to detect project: %v", err)), nil
			}

			// Verify the project exists
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get GitLab client: %w", err)
			}

			project, _, err := glClient.Projects.GetProject(projectID, nil, gl.WithContext(ctx))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Detected project '%s' but could not verify it exists: %v", projectID, err)), nil
			}

			// Get current working directory
			cwd, err := os.Getwd()
			if err != nil {
				return nil, fmt.Errorf("failed to get working directory: %w", err)
			}

			// Create and write project config
			config := &ProjectConfig{
				ProjectID:  projectID,
				GitLabHost: gitlabHost,
			}

			configPath, err := WriteProjectConfig(cwd, config)
			if err != nil {
				return nil, fmt.Errorf("failed to write project config: %w", err)
			}

			result := map[string]interface{}{
				"success":      true,
				"configPath":   configPath,
				"projectId":    projectID,
				"gitlabHost":   gitlabHost,
				"projectName":  project.Name,
				"projectPath":  project.PathWithNamespace,
				"message":      "Project detected and configured successfully!",
			}

			data, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		}
}

// GetProjectIDWithFallback tries to get project ID from parameter, then from .gmcprc, then from Git detection
func GetProjectIDWithFallback(request *mcp.CallToolRequest) (string, error) {
	// First, try to get from explicit parameter
	if projectID, ok := request.Params.Arguments["projectId"].(string); ok && projectID != "" {
		return projectID, nil
	}

	// Second, try to get from .gmcprc file
	config, _, err := FindProjectConfig()
	if err == nil && config != nil && config.ProjectID != "" {
		return config.ProjectID, nil
	}

	// Third, try to auto-detect from Git
	projectID, _, err := DetectProjectFromGit()
	if err == nil && projectID != "" {
		return projectID, nil
	}

	// No project ID found anywhere
	return "", fmt.Errorf("no project ID specified. Either:\n" +
		"1. Provide 'projectId' parameter\n" +
		"2. Create a .gmcprc file using 'setCurrentProject'\n" +
		"3. Use 'autoDetectAndSetProject' to auto-detect from Git remote")
}
