package gitlab

import (
	"context"
	"testing"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/toolsets"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDynamicToolsetManager(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	// Create a toolset group
	tg := toolsets.NewToolsetGroup(false)

	// Create a minimal MCP server
	mcpServer := server.NewMCPServer("test-server", "1.0.0",
		server.WithToolCapabilities(true),
	)

	dtm := NewDynamicToolsetManager(tg, mcpServer, logger)

	assert.NotNil(t, dtm)
	assert.Same(t, tg, dtm.toolsetGroup)
	assert.Same(t, mcpServer, dtm.mcpServer)
	assert.Same(t, logger, dtm.logger)
	assert.False(t, dtm.dynamicMode, "Dynamic mode should be disabled by default")
}

func TestDynamicToolsetManager_SetDynamicMode(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	tg := toolsets.NewToolsetGroup(false)
	mcpServer := server.NewMCPServer("test-server", "1.0.0",
		server.WithToolCapabilities(true),
	)

	dtm := NewDynamicToolsetManager(tg, mcpServer, logger)

	// Test enabling dynamic mode
	dtm.SetDynamicMode(true)
	assert.True(t, dtm.dynamicMode)

	// Test disabling dynamic mode
	dtm.SetDynamicMode(false)
	assert.False(t, dtm.dynamicMode)
}

func TestDynamicToolsetManager_RegisterDiscoveryTools(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	tg := toolsets.NewToolsetGroup(false)
	mcpServer := server.NewMCPServer("test-server", "1.0.0",
		server.WithToolCapabilities(true),
	)

	dtm := NewDynamicToolsetManager(tg, mcpServer, logger)

	// Register discovery tools
	dtm.RegisterDiscoveryTools()

	// Verify tools were registered
	// Note: MCPServer doesn't expose a way to list registered tools,
	// but we can verify the function doesn't panic
	assert.True(t, true)
}

func TestDynamicToolsetManager_handleListToolsets(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	tests := []struct {
		name                string
		setupToolsets       func(*toolsets.ToolsetGroup)
		expectedInResult    []string
		notExpectedInResult []string
	}{
		{
			name: "Empty toolset group",
			setupToolsets: func(tg *toolsets.ToolsetGroup) {
				// No toolsets added
			},
			expectedInResult:    []string{"Available Toolsets (0):"},
			notExpectedInResult: []string{},
		},
		{
			name: "Multiple toolsets, all disabled",
			setupToolsets: func(tg *toolsets.ToolsetGroup) {
				ts1 := toolsets.NewToolset("projects", "Project tools")
				ts2 := toolsets.NewToolset("issues", "Issue tools")
				tg.AddToolset(ts1)
				tg.AddToolset(ts2)
			},
			expectedInResult:    []string{"Available Toolsets (2):", "projects: Project tools", "issues: Issue tools", "(disabled)"},
			notExpectedInResult: []string{"(enabled)"},
		},
		{
			name: "Mixed enabled/disabled toolsets",
			setupToolsets: func(tg *toolsets.ToolsetGroup) {
				ts1 := toolsets.NewToolset("projects", "Project tools")
				ts2 := toolsets.NewToolset("issues", "Issue tools")
				ts3 := toolsets.NewToolset("merge_requests", "MR tools")

				tg.AddToolset(ts1)
				tg.AddToolset(ts2)
				tg.AddToolset(ts3)

				// Enable issues toolset
				tg.EnableToolset("issues")
			},
			expectedInResult:    []string{"Available Toolsets (3):", "projects: Project tools", "issues: Issue tools", "merge_requests: MR tools", "(disabled)", "(enabled)"},
			notExpectedInResult: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tg := toolsets.NewToolsetGroup(false)
			tc.setupToolsets(tg)

			mcpServer := server.NewMCPServer("test-server", "1.0.0",
				server.WithToolCapabilities(true),
			)
			dtm := NewDynamicToolsetManager(tg, mcpServer, logger)

			// Call handleListToolsets
			result, err := dtm.handleListToolsets(context.Background(), mcp.CallToolRequest{})

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Content, 1, "Should return one content block")

			// Extract text content
			textContent, ok := result.Content[0].(mcp.TextContent)
			require.True(t, ok, "Result should be TextContent")

			// Verify expected strings are in result
			for _, expected := range tc.expectedInResult {
				assert.Contains(t, textContent.Text, expected)
			}

			// Verify unexpected strings are NOT in result
			for _, notExpected := range tc.notExpectedInResult {
				assert.NotContains(t, textContent.Text, notExpected)
			}
		})
	}
}

func TestDynamicToolsetManager_handleEnableToolset(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	tests := []struct {
		name                string
		setupToolsets       func(*toolsets.ToolsetGroup)
		requestArgs         map[string]interface{}
		expectError         bool
		expectedInResult    string
		notExpectedInResult string
	}{
		{
			name: "Missing toolset parameter",
			setupToolsets: func(tg *toolsets.ToolsetGroup) {
				ts := toolsets.NewToolset("projects", "Project tools")
				tg.AddToolset(ts)
			},
			requestArgs:         map[string]interface{}{},
			expectError:         false,
			expectedInResult:    "Missing required parameter: toolset",
			notExpectedInResult: "Successfully enabled",
		},
		{
			name: "Invalid toolset parameter type",
			setupToolsets: func(tg *toolsets.ToolsetGroup) {
				ts := toolsets.NewToolset("projects", "Project tools")
				tg.AddToolset(ts)
			},
			requestArgs:         map[string]interface{}{"toolset": 123},
			expectError:         false,
			expectedInResult:    "Parameter 'toolset' must be a string",
			notExpectedInResult: "Successfully enabled",
		},
		{
			name: "Non-existent toolset",
			setupToolsets: func(tg *toolsets.ToolsetGroup) {
				ts := toolsets.NewToolset("projects", "Project tools")
				tg.AddToolset(ts)
			},
			requestArgs:         map[string]interface{}{"toolset": "nonexistent"},
			expectError:         false,
			expectedInResult:    "Failed to enable toolset 'nonexistent'",
			notExpectedInResult: "Successfully enabled",
		},
		{
			name: "Successfully enable toolset",
			setupToolsets: func(tg *toolsets.ToolsetGroup) {
				ts := toolsets.NewToolset("projects", "Project tools")
				tg.AddToolset(ts)
			},
			requestArgs:         map[string]interface{}{"toolset": "projects"},
			expectError:         false,
			expectedInResult:    "Successfully enabled toolset 'projects'. Tools are now available.",
			notExpectedInResult: "Failed",
		},
		{
			name: "Enable already enabled toolset",
			setupToolsets: func(tg *toolsets.ToolsetGroup) {
				ts := toolsets.NewToolset("projects", "Project tools")
				tg.AddToolset(ts)
				tg.EnableToolset("projects")
			},
			requestArgs:         map[string]interface{}{"toolset": "projects"},
			expectError:         false,
			expectedInResult:    "Failed to enable toolset 'projects'",
			notExpectedInResult: "Successfully enabled",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tg := toolsets.NewToolsetGroup(false)
			tc.setupToolsets(tg)

			mcpServer := server.NewMCPServer("test-server", "1.0.0",
				server.WithToolCapabilities(true),
			)
			dtm := NewDynamicToolsetManager(tg, mcpServer, logger)

			// Create request
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      "enable_toolset",
					Arguments: tc.requestArgs,
				},
			}

			// Call handleEnableToolset
			result, err := dtm.handleEnableToolset(context.Background(), request)

			if tc.expectError {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1, "Should return one content block")

				// Extract text content
				textContent, ok := result.Content[0].(mcp.TextContent)
				require.True(t, ok, "Result should be TextContent")

				assert.Contains(t, textContent.Text, tc.expectedInResult)
				if tc.notExpectedInResult != "" {
					assert.NotContains(t, textContent.Text, tc.notExpectedInResult)
				}
			}
		})
	}
}

func TestDynamicToolsetManager_registerToolsetTools(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	tests := []struct {
		name          string
		setupToolsets func(*toolsets.ToolsetGroup)
		toolsetName   string
		expectError   bool
		errorContains string
	}{
		{
			name: "Successfully register tools for existing toolset",
			setupToolsets: func(tg *toolsets.ToolsetGroup) {
				ts := toolsets.NewToolset("projects", "Project tools")
				tg.AddToolset(ts)
			},
			toolsetName: "projects",
			expectError: false,
		},
		{
			name: "Fail to register tools for non-existent toolset",
			setupToolsets: func(tg *toolsets.ToolsetGroup) {
				ts := toolsets.NewToolset("projects", "Project tools")
				tg.AddToolset(ts)
			},
			toolsetName:   "nonexistent",
			expectError:   true,
			errorContains: "toolset 'nonexistent' not found",
		},
		{
			name: "Register tools for multiple toolsets",
			setupToolsets: func(tg *toolsets.ToolsetGroup) {
				ts1 := toolsets.NewToolset("projects", "Project tools")
				ts2 := toolsets.NewToolset("issues", "Issue tools")
				tg.AddToolset(ts1)
				tg.AddToolset(ts2)
			},
			toolsetName: "projects",
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tg := toolsets.NewToolsetGroup(false)
			tc.setupToolsets(tg)

			mcpServer := server.NewMCPServer("test-server", "1.0.0",
				server.WithToolCapabilities(true),
			)
			dtm := NewDynamicToolsetManager(tg, mcpServer, logger)

			// Call registerToolsetTools
			err := dtm.registerToolsetTools(tc.toolsetName)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDynamicToolsetManager_Integration_EnableThenList(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	tg := toolsets.NewToolsetGroup(false)
	mcpServer := server.NewMCPServer("test-server", "1.0.0",
		server.WithToolCapabilities(true),
	)
	dtm := NewDynamicToolsetManager(tg, mcpServer, logger)

	// Add toolsets
	ts1 := toolsets.NewToolset("projects", "Project tools")
	ts2 := toolsets.NewToolset("issues", "Issue tools")
	tg.AddToolset(ts1)
	tg.AddToolset(ts2)

	// List toolsets - should show both as disabled
	listResult1, err := dtm.handleListToolsets(context.Background(), mcp.CallToolRequest{})
	require.NoError(t, err)
	textContent1 := listResult1.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent1.Text, "projects: Project tools")
	assert.Contains(t, textContent1.Text, "(disabled)")

	// Enable projects toolset
	enableRequest := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      "enable_toolset",
			Arguments: map[string]interface{}{"toolset": "projects"},
		},
	}
	enableResult, err := dtm.handleEnableToolset(context.Background(), enableRequest)
	require.NoError(t, err)
	textContent2 := enableResult.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent2.Text, "Successfully enabled toolset 'projects'")

	// List toolsets again - projects should be enabled
	listResult2, err := dtm.handleListToolsets(context.Background(), mcp.CallToolRequest{})
	require.NoError(t, err)
	textContent3 := listResult2.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent3.Text, "projects: Project tools")
	assert.Contains(t, textContent3.Text, "(enabled)")
	assert.Contains(t, textContent3.Text, "issues: Issue tools")
	assert.Contains(t, textContent3.Text, "(disabled)")
}
