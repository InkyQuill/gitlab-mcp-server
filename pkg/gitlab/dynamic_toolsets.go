package gitlab

import (
	"context"
	"fmt"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/toolsets"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
)

// DynamicToolsetManager handles runtime toolset registration and discovery.
type DynamicToolsetManager struct {
	toolsetGroup *toolsets.ToolsetGroup
	mcpServer    *server.MCPServer
	logger       *log.Logger
	dynamicMode  bool
}

// NewDynamicToolsetManager creates a new DynamicToolsetManager instance.
func NewDynamicToolsetManager(tg *toolsets.ToolsetGroup, mcpServer *server.MCPServer, logger *log.Logger) *DynamicToolsetManager {
	return &DynamicToolsetManager{
		toolsetGroup: tg,
		mcpServer:    mcpServer,
		logger:       logger,
		dynamicMode:  false,
	}
}

// SetDynamicMode enables or disables dynamic toolset discovery mode.
func (dtm *DynamicToolsetManager) SetDynamicMode(enabled bool) {
	dtm.dynamicMode = enabled
	if enabled {
		dtm.logger.Info("Dynamic toolset discovery enabled - only discovery tools will be available initially")
	}
}

// RegisterDiscoveryTools registers MCP tools for listing and enabling toolsets dynamically.
func (dtm *DynamicToolsetManager) RegisterDiscoveryTools() {
	// Tool 1: List available toolsets
	listToolsets := mcp.NewTool("list_available_toolsets",
		mcp.WithDescription("Lists all available GitLab MCP toolsets that can be enabled."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "List Available Toolsets",
			ReadOnlyHint: true,
		}),
	)
	dtm.mcpServer.AddTool(listToolsets, dtm.handleListToolsets)

	// Tool 2: Enable a toolset
	enableToolset := mcp.NewTool("enable_toolset",
		mcp.WithDescription("Enables a specific GitLab MCP toolset, making its tools available."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title: "Enable Toolset",
		}),
		mcp.WithString("toolset",
			mcp.Required(),
			mcp.Description("Name of the toolset to enable (e.g., 'projects', 'issues', 'merge_requests')"),
		),
	)
	dtm.mcpServer.AddTool(enableToolset, dtm.handleEnableToolset)

	dtm.logger.Info("Dynamic toolset discovery tools registered")
}

// handleListToolsets returns all available toolsets with their status.
func (dtm *DynamicToolsetManager) handleListToolsets(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	infos := dtm.toolsetGroup.ListToolsets()

	result := fmt.Sprintf("Available Toolsets (%d):\n", len(infos))
	for _, info := range infos {
		status := "disabled"
		if info.Enabled {
			status = "enabled"
		}
		result += fmt.Sprintf("- %s: %s [%d tools] (%s)\n", info.Name, info.Description, info.ToolCount, status)
	}

	return mcp.NewToolResultText(result), nil
}

// handleEnableToolset enables a specific toolset and registers its tools.
func (dtm *DynamicToolsetManager) handleEnableToolset(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Parse toolset name from arguments
	toolsetArg, ok := request.Params.Arguments["toolset"]
	if !ok {
		return mcp.NewToolResultError("Missing required parameter: toolset"), nil
	}

	toolsetName, ok := toolsetArg.(string)
	if !ok {
		return mcp.NewToolResultError("Parameter 'toolset' must be a string"), nil
	}

	dtm.logger.Infof("Enabling toolset: %s", toolsetName)

	// Enable the toolset
	if err := dtm.toolsetGroup.EnableToolset(toolsetName); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to enable toolset '%s': %v", toolsetName, err)), nil
	}

	// Re-register tools with MCP server
	if err := dtm.registerToolsetTools(toolsetName); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to register tools for '%s': %v", toolsetName, err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully enabled toolset '%s'. Tools are now available.", toolsetName)), nil
}

// registerToolsetTools registers all tools from a specific toolset with the MCP server.
func (dtm *DynamicToolsetManager) registerToolsetTools(toolsetName string) error {
	// Get toolset info to verify it exists
	infos := dtm.toolsetGroup.ListToolsets()
	var found bool
	for _, info := range infos {
		if info.Name == toolsetName {
			found = true
			dtm.logger.Infof("Registering %d tools from toolset '%s'", info.ToolCount, toolsetName)
			break
		}
	}

	if !found {
		return fmt.Errorf("toolset '%s' not found", toolsetName)
	}

	// Note: The actual tool registration happens through the toolsetGroup
	// We need to re-register all enabled toolsets to include the newly enabled one
	dtm.toolsetGroup.RegisterTools(dtm.mcpServer)

	dtm.logger.Infof("Successfully registered tools for toolset '%s'", toolsetName)
	return nil
}
