package gitlab

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/InkyQuill/gitlab-mcp-server/internal/toolsnaps"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
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

func requireToolText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	require.NotNil(t, result)
	require.NotEmpty(t, result.Content)
	textContent, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok, "expected first tool result content to be mcp.TextContent")
	return textContent.Text
}

func TestInitToolsets(t *testing.T) {
	allowWritesPolicy := func(context.Context) (ServerPolicy, error) {
		return ServerPolicy{Name: "default"}, nil
	}

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
			tg, err := InitToolsets(tc.enabledToolsets, tc.readOnly, mockGetClientFn, nil, nil, nil, false, allowWritesPolicy)

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

	serverProperty, ok := setCurrentProject.Tool.InputSchema.Properties["server"]
	require.True(t, ok)
	serverSchema, ok := serverProperty.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", serverSchema["type"])
	description, ok := serverSchema["description"].(string)
	require.True(t, ok)
	assert.Contains(t, description, "auto-detected from Git remote")
	assert.NotContains(t, description, "Overrides .gmcprc/default routing")
}

func TestInitToolsets_RegisteredToolSchemaSnapshots(t *testing.T) {
	allowWritesPolicy := func(context.Context) (ServerPolicy, error) {
		return ServerPolicy{Name: "default"}, nil
	}
	tg, err := InitToolsets([]string{"all"}, false, mockGetClientFn, nil, NewTokenStore(), nil, false, allowWritesPolicy)
	require.NoError(t, err)

	routedToolsets := map[string]bool{
		"projects":         true,
		"issues":           true,
		"merge_requests":   true,
		"security":         true,
		"users":            true,
		"search":           true,
		"tags":             true,
		"pipeline_jobs":    true,
		"project_config":   false,
		"token_management": false,
	}

	for toolsetName, toolset := range tg.Toolsets {
		expectsRouting, knownToolset := routedToolsets[toolsetName]
		require.True(t, knownToolset, "registered snapshot test must classify routing behavior for toolset %s", toolsetName)

		for _, serverTool := range toolset.Tools() {
			tool := serverTool.Tool
			require.NoError(t, toolsnaps.Test("registered_"+tool.Name, tool), "registered tool schema should match snapshot")

			serverProperty, hasServer := tool.InputSchema.Properties["server"]
			if expectsRouting {
				require.True(t, hasServer, "registered GitLab API tool %s should accept routing server", tool.Name)
				serverSchema, ok := serverProperty.(map[string]any)
				require.True(t, ok, "server schema for %s should be an object", tool.Name)
				assert.Equal(t, "string", serverSchema["type"])
				assert.Equal(t, "Configured GitLab server name. Overrides .gmcprc/default routing for this call.", serverSchema["description"])
				continue
			}

			if !hasServer {
				continue
			}
			serverSchema, ok := serverProperty.(map[string]any)
			require.True(t, ok, "server schema for %s should be an object", tool.Name)
			description, _ := serverSchema["description"].(string)
			assert.NotContains(t, description, "Overrides .gmcprc/default routing", "tool %s should preserve non-routing server semantics", tool.Name)
		}
	}
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
	assert.Contains(t, requireToolText(t, result), "read-only")
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
	assert.Contains(t, requireToolText(t, result), `server "mirror"`)
}

func TestInitToolsets_WriteGuardUsesResolverSelectedProjectConfigServer(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	pool := NewClientPool(NewTokenStore(), logger)
	require.NoError(t, pool.AddClientWithInfo(ClientInfo{Name: "default"}, &gitlab.Client{}))
	require.NoError(t, pool.AddClientWithInfo(ClientInfo{Name: "mirror", ReadOnly: true}, &gitlab.Client{}))

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gmcprc")
	require.NoError(t, os.WriteFile(configPath, []byte(`{"projectId":"g/p","server":"mirror"}`), 0600))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	resolver := NewClientResolver(pool, "default", logger)
	policy := func(ctx context.Context) (ServerPolicy, error) {
		_, name, err := resolver.Resolve(ctx)
		if err != nil {
			return ServerPolicy{}, err
		}
		info, err := pool.GetClientInfo(name)
		if err != nil {
			return ServerPolicy{Name: name}, nil
		}
		return ServerPolicy{Name: name, ReadOnly: info.ReadOnly}, nil
	}

	tg, err := InitToolsets([]string{"issues"}, false, resolver.GetClientFn(), logger, NewTokenStore(), nil, false, policy)
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
	resultText := requireToolText(t, result)
	assert.Contains(t, resultText, `server "mirror"`)
	assert.Contains(t, resultText, "read-only")
}

func TestGitLabWriteToolUsesPolicySelectedServer(t *testing.T) {
	policy := func(context.Context) (ServerPolicy, error) {
		return ServerPolicy{Name: "mirror", ReadOnly: false}, nil
	}

	var gotServer string
	tool := newGitLabWriteTool(policy, mcp.NewTool("writeThing"), func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		gotServer, _ = RequestedServerFromContext(ctx)
		return mcp.NewToolResultText("ok"), nil
	})

	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.Equal(t, "mirror", gotServer)
	assert.Equal(t, "ok", requireToolText(t, result))
}

func TestGitLabWriteToolPolicyErrorSkipsHandler(t *testing.T) {
	policyErr := errors.New("metadata missing")
	policy := func(context.Context) (ServerPolicy, error) {
		return ServerPolicy{}, policyErr
	}

	handlerCalled := false
	tool := newGitLabWriteTool(policy, mcp.NewTool("writeThing"), func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		handlerCalled = true
		return mcp.NewToolResultText("unexpected"), nil
	})

	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.IsError)
	assert.False(t, handlerCalled)
	assert.Contains(t, requireToolText(t, result), "failed to resolve server policy")
	assert.Contains(t, requireToolText(t, result), policyErr.Error())
}

func TestGitLabWriteToolNilPolicySkipsHandler(t *testing.T) {
	handlerCalled := false
	tool := newGitLabWriteTool(nil, mcp.NewTool("writeThing"), func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		handlerCalled = true
		return mcp.NewToolResultText("unexpected"), nil
	})

	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.IsError)
	assert.False(t, handlerCalled)
	assert.Contains(t, requireToolText(t, result), `server policy is required for write tool "writeThing"`)
}

func TestGitLabWriteToolNilHandlerPassesThroughSchema(t *testing.T) {
	policyCalled := false
	policy := func(context.Context) (ServerPolicy, error) {
		policyCalled = true
		return ServerPolicy{Name: "default"}, nil
	}

	tool := newGitLabWriteTool(policy, mcp.NewTool("writeThing"), nil)

	assert.Nil(t, tool.Handler)
	assert.False(t, policyCalled)
	assert.Contains(t, tool.Tool.InputSchema.Properties, "server")
}
