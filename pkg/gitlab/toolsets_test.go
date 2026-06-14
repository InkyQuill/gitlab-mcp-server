package gitlab

import (
	"context"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Import the actual gitlab client library
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// Reintroduce mockGetClientFn for testing InitToolsets with the GetClientFn signature
func mockGetClientFn(_ context.Context) (*gitlab.Client, error) {
	// Return a nil client, as InitToolsets and tool definitions don't use it yet
	return nil, nil
}

// Mock TranslationHelperFunc (placeholder) - Kept commented out as it's not used
// func mockTranslationHelper(key string, defaultVal string, args ...interface{}) string {
// 	return defaultVal // Simple passthrough for now
// }

func TestInitToolsets(t *testing.T) {
	// Define the expected toolset names based on the implementation
	// All 10 toolsets defined in InitToolsets
	expectedToolsetNames := []string{
		"token_management",
		"project_config",
		"projects",
		"issues",
		"merge_requests",
		"security",
		"users",
		"search",
		"tags",
		"pipeline_jobs",
	}

	tests := []struct {
		name            string
		enabledToolsets []string
		readOnly        bool
		expectError     bool
		errContains     string
		expectEnabled   []string // Which toolsets should end up enabled
		// Removed checks for unexported fields (everythingOn, groupReadOnly)
		// These should be tested within the toolsets package itself.
	}{
		{
			name:            "Enable specific toolsets, not read-only",
			enabledToolsets: []string{"projects", "issues"},
			readOnly:        false,
			expectError:     false,
			expectEnabled:   []string{"projects", "issues"},
		},
		{
			name:            "Enable all toolsets, not read-only",
			enabledToolsets: []string{"all"},
			readOnly:        false,
			expectError:     false,
			expectEnabled:   expectedToolsetNames, // All defined toolsets
		},
		{
			name:            "Enable specific toolsets, read-only group",
			enabledToolsets: []string{"users", "search"},
			readOnly:        true,
			expectError:     false,
			expectEnabled:   []string{"users", "search"},
		},
		{
			name:            "Enable all toolsets, read-only group",
			enabledToolsets: []string{"all"},
			readOnly:        true,
			expectError:     false,
			expectEnabled:   expectedToolsetNames,
		},
		{
			name:            "Enable non-existent toolset",
			enabledToolsets: []string{"projects", "invalid-toolset"},
			readOnly:        false,
			expectError:     true,
			errContains:     "toolset 'invalid-toolset' not found",
			expectEnabled:   []string{"projects"}, // projects should still be enabled before error
		},
		{
			name:            "Enable empty list",
			enabledToolsets: []string{},
			readOnly:        false,
			expectError:     true,
			errContains:     "no toolsets specified",
			expectEnabled:   []string{}, // None should be enabled
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Call InitToolsets using the mock function
			// Parameters: enabledToolsets, readOnly, getClient, logger, tokenStore, translations, dynamicMode
			tg, err := InitToolsets(tc.enabledToolsets, tc.readOnly, mockGetClientFn, nil, nil, nil, false, nil)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				// We might get a partially configured tg even on error
				if tg == nil {
					return // Nothing more to check if tg is nil
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, tg, "ToolsetGroup should not be nil on success")
			}

			// Removed checks for tg.readOnly and tg.everythingOn as they are unexported

			// Verify all expected toolsets exist in the returned group
			assert.Len(t, tg.Toolsets, len(expectedToolsetNames), "Should contain all defined toolsets")
			for _, name := range expectedToolsetNames {
				assert.Contains(t, tg.Toolsets, name, "Expected toolset %s to be in the group", name)
			}

			// Verify enabled status
			enabledMap := make(map[string]bool)
			for _, name := range tc.expectEnabled {
				enabledMap[name] = true
			}

			for name, ts := range tg.Toolsets {
				expectedEnabled := enabledMap[name]
				assert.Equal(t, expectedEnabled, ts.Enabled, "Enabled status mismatch for toolset: %s", name)
				// Removed check for ts.readOnly as it's unexported and tested in toolsets package
			}
		})
	}
}

func TestInitToolsets_AddsServerParameter(t *testing.T) {
	tg, err := InitToolsets([]string{"projects"}, false, mockGetClientFn, nil, NewTokenStore(), nil, false, nil)
	require.NoError(t, err)

	tools := tg.Toolsets["projects"].Tools()
	require.NotEmpty(t, tools)

	for _, tool := range tools {
		assert.Contains(t, tool.Tool.InputSchema.Properties, "server", "tool %s should accept server", tool.Tool.Name)
	}
}

func TestInitToolsets_DoesNotOverwriteProjectConfigServerParameter(t *testing.T) {
	tg, err := InitToolsets([]string{"project_config"}, false, mockGetClientFn, nil, NewTokenStore(), nil, false, nil)
	require.NoError(t, err)

	var setCurrentProject server.ServerTool
	for _, tool := range tg.Toolsets["project_config"].Tools() {
		if tool.Tool.Name == "setCurrentProject" {
			setCurrentProject = tool
			break
		}
	}
	require.NotEmpty(t, setCurrentProject.Tool.Name)

	serverProperty := setCurrentProject.Tool.InputSchema.Properties["server"]
	assert.Contains(t, fmt.Sprint(serverProperty), "auto-detected from Git remote")
	assert.NotContains(t, fmt.Sprint(serverProperty), "Overrides .gmcprc/default routing")
}

func TestInitToolsets_WriteGuardRejectsReadOnlyServer(t *testing.T) {
	policy := func(context.Context) (ServerPolicy, error) {
		return ServerPolicy{Name: "mirror", ReadOnly: true}, nil
	}

	tg, err := InitToolsets([]string{"issues"}, false, mockGetClientFn, nil, NewTokenStore(), nil, false, policy)
	require.NoError(t, err)

	var createIssue server.ServerTool
	for _, tool := range tg.Toolsets["issues"].Tools() {
		if tool.Tool.Name == "createIssue" {
			createIssue = tool
			break
		}
	}
	require.NotNil(t, createIssue.Handler)

	result, err := createIssue.Handler(context.Background(), mcp.CallToolRequest{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "read-only")
}

func TestInitToolsets_WriteGuardUsesRequestedServer(t *testing.T) {
	var gotServer string
	policy := func(ctx context.Context) (ServerPolicy, error) {
		gotServer, _ = RequestedServerFromContext(ctx)
		return ServerPolicy{Name: gotServer, ReadOnly: true}, nil
	}

	tg, err := InitToolsets([]string{"issues"}, false, mockGetClientFn, nil, NewTokenStore(), nil, false, policy)
	require.NoError(t, err)

	var createIssue server.ServerTool
	for _, tool := range tg.Toolsets["issues"].Tools() {
		if tool.Tool.Name == "createIssue" {
			createIssue = tool
			break
		}
	}
	require.NotNil(t, createIssue.Handler)

	result, err := createIssue.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{"server": "mirror"},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.IsError)
	assert.Equal(t, "mirror", gotServer)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, `server "mirror"`)
}
