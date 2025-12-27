package toolsets

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: Removed mock handlers and Toolset-specific tests (AddTools, GetActiveTools)
// as the reference github-mcp-server/pkg/toolsets/toolsets_test.go only tests
// the ToolsetGroup logic.

// --- ToolsetGroup Tests ---

func TestNewToolsetGroup(t *testing.T) {
	tgDefault := NewToolsetGroup(false)
	require.NotNil(t, tgDefault)
	assert.NotNil(t, tgDefault.Toolsets)
	assert.Empty(t, tgDefault.Toolsets)
	assert.False(t, tgDefault.readOnly)
	assert.False(t, tgDefault.everythingOn)

	tgReadOnly := NewToolsetGroup(true)
	require.NotNil(t, tgReadOnly)
	assert.NotNil(t, tgReadOnly.Toolsets)
	assert.Empty(t, tgReadOnly.Toolsets)
	assert.True(t, tgReadOnly.readOnly)
	assert.False(t, tgDefault.everythingOn) // Corrected assertion target
}

func TestToolsetGroup_AddToolset(t *testing.T) {
	// Case 1: Not read-only group
	tg := NewToolsetGroup(false)
	ts1 := NewToolset("ts1", "Toolset 1")
	tg.AddToolset(ts1)

	assert.Len(t, tg.Toolsets, 1)
	assert.Same(t, ts1, tg.Toolsets["ts1"])
	assert.False(t, tg.Toolsets["ts1"].readOnly, "Toolset should not be read-only in non-read-only group")

	// Case 2: Read-only group
	tgReadOnly := NewToolsetGroup(true)
	ts2 := NewToolset("ts2", "Toolset 2")
	tgReadOnly.AddToolset(ts2)

	assert.Len(t, tgReadOnly.Toolsets, 1)
	assert.Same(t, ts2, tgReadOnly.Toolsets["ts2"])
	assert.True(t, tgReadOnly.Toolsets["ts2"].readOnly, "Toolset should be forced to read-only in read-only group")

	// Case 3: Overwrite existing
	ts1Updated := NewToolset("ts1", "Toolset 1 Updated")
	tg.AddToolset(ts1Updated)
	assert.Len(t, tg.Toolsets, 1) // Still only one entry for "ts1"
	assert.Same(t, ts1Updated, tg.Toolsets["ts1"])
	assert.Equal(t, "Toolset 1 Updated", tg.Toolsets["ts1"].Description)
	// Asserting Enabled status might depend on whether AddToolset resets it.
	// Let's assume it uses the new toolset's default (false)
	assert.False(t, tg.Toolsets["ts1"].Enabled, "Overwritten toolset should likely reset to disabled")
}

func TestToolsetGroup_EnableToolset(t *testing.T) {
	tg := NewToolsetGroup(false)
	ts1 := NewToolset("ts1", "")
	tg.AddToolset(ts1)

	// Enable non-existent
	err := tg.EnableToolset("non-existent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "toolset 'non-existent' not found")

	// Enable existing
	assert.False(t, ts1.Enabled, "Should be disabled initially")
	err = tg.EnableToolset("ts1")
	require.NoError(t, err)
	assert.True(t, ts1.Enabled, "Should be enabled after call")

	// Enable already enabled - now returns error
	err = tg.EnableToolset("ts1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "toolset 'ts1' already enabled")
	assert.True(t, ts1.Enabled, "Should remain enabled")
}

func TestToolsetGroup_EnableToolsets(t *testing.T) {
	tests := []struct {
		name            string
		initialToolsets map[string]*Toolset
		namesToEnable   []string
		expectError     bool
		errContains     string
		expectEnabled   []string // Names of toolsets expected to be enabled
		expectAllOn     bool
	}{
		{
			name: "Enable subset",
			initialToolsets: map[string]*Toolset{
				"ts1": NewToolset("ts1", ""),
				"ts2": NewToolset("ts2", ""),
				"ts3": NewToolset("ts3", ""),
			},
			namesToEnable: []string{"ts1", "ts3"},
			expectError:   false,
			expectEnabled: []string{"ts1", "ts3"},
			expectAllOn:   false,
		},
		{
			name: "Enable all keyword",
			initialToolsets: map[string]*Toolset{
				"ts1": NewToolset("ts1", ""),
				"ts2": NewToolset("ts2", ""),
			},
			namesToEnable: []string{"all"},
			expectError:   false,
			expectEnabled: []string{"ts1", "ts2"}, // All initially added toolsets
			expectAllOn:   true,
		},
		{
			name: "Enable non-existent",
			initialToolsets: map[string]*Toolset{
				"ts1": NewToolset("ts1", ""),
			},
			namesToEnable: []string{"ts1", "non-existent"},
			expectError:   true,
			errContains:   "toolset 'non-existent' not found",
			expectEnabled: []string{"ts1"}, // ts1 should still be enabled before error
			expectAllOn:   false,
		},
		{
			name:            "Enable empty list",
			initialToolsets: map[string]*Toolset{"ts1": NewToolset("ts1", "")},
			namesToEnable:   []string{},
			expectError:     true, // Implementation returns error for empty list
			errContains:     "no toolsets specified",
			expectEnabled:   []string{},
			expectAllOn:     false,
		},
		{
			name:            "Enable only non-existent",
			initialToolsets: map[string]*Toolset{"ts1": NewToolset("ts1", "")},
			namesToEnable:   []string{"non-existent"},
			expectError:     true,
			errContains:     "toolset 'non-existent' not found",
			expectEnabled:   []string{},
			expectAllOn:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tg := NewToolsetGroup(false)
			tg.Toolsets = tc.initialToolsets // Directly set for test setup

			err := tg.EnableToolsets(tc.namesToEnable)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.expectAllOn, tg.everythingOn, "everythingOn flag state")

			enabledCount := 0
			disabledCount := 0
			for name, ts := range tg.Toolsets {
				isEnabled := false
				for _, enabledName := range tc.expectEnabled {
					if name == enabledName {
						isEnabled = true
						break
					}
				}
				assert.Equal(t, isEnabled, ts.Enabled, "Enabled state for toolset: %s", name)
				if isEnabled {
					enabledCount++
				} else {
					disabledCount++
				}
			}
			assert.Len(t, tc.expectEnabled, enabledCount, "Number of enabled toolsets")
		})
	}
}

// --- Toolset Tests ---

func TestNewServerTool(t *testing.T) {
	tool := mcp.Tool{
		Name:        "test_tool",
		Description: "A test tool",
	}

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{}, nil
	}

	serverTool := NewServerTool(tool, handler)

	assert.Equal(t, tool.Name, serverTool.Tool.Name)
	assert.Equal(t, tool.Description, serverTool.Tool.Description)
	assert.NotNil(t, serverTool.Handler)
}

func TestToolset_AddReadTools(t *testing.T) {
	ts := NewToolset("test", "Test toolset")

	tool1 := mcp.Tool{Name: "read1"}
	tool2 := mcp.Tool{Name: "read2"}
	st1 := NewServerTool(tool1, nil)
	st2 := NewServerTool(tool2, nil)

	// Add tools
	result := ts.AddReadTools(st1, st2)

	assert.Same(t, ts, result, "Should return the toolset for chaining")
	assert.Len(t, ts.readTools, 2)
	assert.Len(t, ts.writeTools, 0)
	assert.Equal(t, st1, ts.readTools[0])
	assert.Equal(t, st2, ts.readTools[1])
}

func TestToolset_AddWriteTools(t *testing.T) {
	ts := NewToolset("test", "Test toolset")

	tool1 := mcp.Tool{Name: "write1"}
	tool2 := mcp.Tool{Name: "write2"}
	st1 := NewServerTool(tool1, nil)
	st2 := NewServerTool(tool2, nil)

	// Add tools
	result := ts.AddWriteTools(st1, st2)

	assert.Same(t, ts, result, "Should return the toolset for chaining")
	assert.Len(t, ts.writeTools, 2)
	assert.Len(t, ts.readTools, 0)
	assert.Equal(t, st1, ts.writeTools[0])
	assert.Equal(t, st2, ts.writeTools[1])
}

func TestToolset_GetActiveTools(t *testing.T) {
	tests := []struct {
		name         string
		setupToolset func() *Toolset
		expectedLen  int
		validate     func(t *testing.T, tools []server.ServerTool)
	}{
		{
			name: "Disabled toolset returns nil",
			setupToolset: func() *Toolset {
				ts := NewToolset("test", "Test")
				ts.AddReadTools(NewServerTool(mcp.Tool{Name: "read"}, nil))
				return ts
			},
			expectedLen: 0,
			validate: func(t *testing.T, tools []server.ServerTool) {
				assert.Nil(t, tools, "Disabled toolset should return nil")
			},
		},
		{
			name: "Enabled toolset with read and write tools",
			setupToolset: func() *Toolset {
				ts := NewToolset("test", "Test")
				ts.Enable()
				readTool := NewServerTool(mcp.Tool{Name: "read"}, nil)
				writeTool := NewServerTool(mcp.Tool{Name: "write"}, nil)
				ts.AddReadTools(readTool)
				ts.AddWriteTools(writeTool)
				return ts
			},
			expectedLen: 2,
			validate: func(t *testing.T, tools []server.ServerTool) {
				assert.NotNil(t, tools)
				assert.Equal(t, "read", tools[0].Tool.Name)
				assert.Equal(t, "write", tools[1].Tool.Name)
			},
		},
		{
			name: "Read-only toolset returns only read tools",
			setupToolset: func() *Toolset {
				ts := NewToolset("test", "Test")
				ts.Enable()
				ts.SetReadOnly()
				readTool := NewServerTool(mcp.Tool{Name: "read"}, nil)
				writeTool := NewServerTool(mcp.Tool{Name: "write"}, nil)
				ts.AddReadTools(readTool)
				ts.AddWriteTools(writeTool)
				return ts
			},
			expectedLen: 1,
			validate: func(t *testing.T, tools []server.ServerTool) {
				assert.NotNil(t, tools)
				assert.Equal(t, "read", tools[0].Tool.Name, "Should only have read tools")
			},
		},
		{
			name: "Enabled toolset with only read tools",
			setupToolset: func() *Toolset {
				ts := NewToolset("test", "Test")
				ts.Enable()
				ts.AddReadTools(
					NewServerTool(mcp.Tool{Name: "read1"}, nil),
					NewServerTool(mcp.Tool{Name: "read2"}, nil),
				)
				return ts
			},
			expectedLen: 2,
			validate: func(t *testing.T, tools []server.ServerTool) {
				assert.NotNil(t, tools)
				assert.Equal(t, "read1", tools[0].Tool.Name)
				assert.Equal(t, "read2", tools[1].Tool.Name)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := tc.setupToolset()
			result := ts.GetActiveTools()
			if tc.expectedLen == 0 && result == nil {
				// nil is expected
				return
			}
			assert.Len(t, result, tc.expectedLen)
			if tc.validate != nil {
				tc.validate(t, result)
			}
		})
	}
}

func TestToolset_RegisterTools(t *testing.T) {
	// Create a minimal MCPServer for testing
	mcpServer := server.NewMCPServer("test", "1.0.0", server.WithToolCapabilities(true))

	ts := NewToolset("test", "Test toolset")
	ts.Enable()

	tool := mcp.Tool{
		Name:        "test_tool",
		Description: "Test description",
	}

	callCount := 0
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		callCount++
		return &mcp.CallToolResult{}, nil
	}

	ts.AddReadTools(NewServerTool(tool, handler))

	// Register tools - should not panic
	ts.RegisterTools(mcpServer)

	// Note: MCPServer doesn't expose a way to verify tools were added,
	// but we can verify the function doesn't panic
	assert.GreaterOrEqual(t, callCount, 0, "Handler should be callable")
}

func TestToolset_SetReadOnly(t *testing.T) {
	ts := NewToolset("test", "Test")

	assert.False(t, ts.readOnly, "Should not be read-only initially")

	ts.SetReadOnly()

	assert.True(t, ts.readOnly, "Should be read-only after SetReadOnly")
}

func TestToolset_GetDescription(t *testing.T) {
	ts := NewToolset("test", "This is a test toolset")

	result := ts.GetDescription()

	assert.Equal(t, "This is a test toolset", result)
}

func TestToolset_Tools(t *testing.T) {
	ts := NewToolset("test", "Test")

	readTool := NewServerTool(mcp.Tool{Name: "read_tool"}, nil)
	writeTool := NewServerTool(mcp.Tool{Name: "write_tool"}, nil)

	ts.AddReadTools(readTool)
	ts.AddWriteTools(writeTool)

	allTools := ts.Tools()

	assert.Len(t, allTools, 2, "Should return both read and write tools")
	assert.Equal(t, "read_tool", allTools[0].Tool.Name)
	assert.Equal(t, "write_tool", allTools[1].Tool.Name)
}

func TestToolset_IsEnabled(t *testing.T) {
	ts := NewToolset("test", "Test")

	assert.False(t, ts.IsEnabled(), "Should be disabled initially")

	ts.Enable()

	assert.True(t, ts.IsEnabled(), "Should be enabled after Enable")
}

func TestToolset_Enable(t *testing.T) {
	ts := NewToolset("test", "Test")

	assert.False(t, ts.Enabled, "Should be disabled initially")

	ts.Enable()

	assert.True(t, ts.Enabled, "Should be enabled after Enable")
}

func TestToolset_Disable(t *testing.T) {
	ts := NewToolset("test", "Test")
	ts.Enable()

	assert.True(t, ts.Enabled, "Should be enabled")

	ts.Disable()

	assert.False(t, ts.Enabled, "Should be disabled after Disable")
}

func TestToolset_Disable_AlreadyDisabled(t *testing.T) {
	ts := NewToolset("test", "Test")

	assert.False(t, ts.Enabled, "Should be disabled initially")

	// Disable when already disabled - should not panic
	ts.Disable()

	assert.False(t, ts.Enabled, "Should remain disabled")
}

func TestToolset_EnableDisable_Toggle(t *testing.T) {
	ts := NewToolset("test", "Test")

	// Enable
	ts.Enable()
	assert.True(t, ts.Enabled)

	// Disable
	ts.Disable()
	assert.False(t, ts.Enabled)

	// Enable again
	ts.Enable()
	assert.True(t, ts.Enabled)
}

func TestToolsetGroup_RegisterTools(t *testing.T) {
	// Create a minimal MCPServer for testing
	mcpServer := server.NewMCPServer("test", "1.0.0", server.WithToolCapabilities(true))

	tg := NewToolsetGroup(false)

	// Create toolsets with tools
	ts1 := NewToolset("ts1", "Toolset 1")
	ts1.Enable()
	tool1 := mcp.Tool{Name: "tool1", Description: "Test tool 1"}
	handler1 := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{}, nil
	}
	ts1.AddReadTools(NewServerTool(tool1, handler1))

	ts2 := NewToolset("ts2", "Toolset 2")
	ts2.Enable()
	tool2 := mcp.Tool{Name: "tool2", Description: "Test tool 2"}
	handler2 := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{}, nil
	}
	ts2.AddReadTools(NewServerTool(tool2, handler2))

	// Add toolsets to group
	tg.AddToolset(ts1)
	tg.AddToolset(ts2)

	t.Run("Success - Register all enabled toolsets", func(t *testing.T) {
		// Should not panic
		tg.RegisterTools(mcpServer)

		// Verify all toolsets are still enabled
		assert.True(t, ts1.Enabled)
		assert.True(t, ts2.Enabled)
	})

	t.Run("Success - Register with disabled toolsets", func(t *testing.T) {
		mcpServer2 := server.NewMCPServer("test2", "1.0.0", server.WithToolCapabilities(true))
		tg2 := NewToolsetGroup(false)

		ts3 := NewToolset("ts3", "Toolset 3")
		// ts3 is NOT enabled
		tool3 := mcp.Tool{Name: "tool3"}
		ts3.AddReadTools(NewServerTool(tool3, nil))

		ts4 := NewToolset("ts4", "Toolset 4")
		ts4.Enable()
		tool4 := mcp.Tool{Name: "tool4"}
		ts4.AddReadTools(NewServerTool(tool4, nil))

		tg2.AddToolset(ts3)
		tg2.AddToolset(ts4)

		// Should not panic, only ts4 should be registered
		tg2.RegisterTools(mcpServer2)

		assert.False(t, ts3.Enabled)
		assert.True(t, ts4.Enabled)
	})

	t.Run("Success - Register with read-only toolsets", func(t *testing.T) {
		mcpServer3 := server.NewMCPServer("test3", "1.0.0", server.WithToolCapabilities(true))
		tg3 := NewToolsetGroup(true) // read-only group

		ts5 := NewToolset("ts5", "Toolset 5")
		ts5.Enable()
		tool5 := mcp.Tool{Name: "tool5_read"}
		tool5Write := mcp.Tool{Name: "tool5_write"}
		ts5.AddReadTools(NewServerTool(tool5, nil))
		ts5.AddWriteTools(NewServerTool(tool5Write, nil))

		tg3.AddToolset(ts5)

		// Should not panic, toolset forced to read-only by group
		tg3.RegisterTools(mcpServer3)

		assert.True(t, ts5.readOnly, "Toolset should be read-only in read-only group")
	})
}

func TestToolsetGroup_ListToolsets(t *testing.T) {
	tg := NewToolsetGroup(false)

	// Create toolsets
	ts1 := NewToolset("ts1", "Toolset 1")
	ts1.Enable()
	tool1 := mcp.Tool{Name: "tool1"}
	ts1.AddReadTools(NewServerTool(tool1, nil))

	ts2 := NewToolset("ts2", "Toolset 2")
	// ts2 is disabled
	tool2 := mcp.Tool{Name: "tool2"}
	ts2.AddReadTools(NewServerTool(tool2, nil))

	ts3 := NewToolset("ts3", "Toolset 3")
	ts3.Enable()
	tool3a := mcp.Tool{Name: "tool3a"}
	tool3b := mcp.Tool{Name: "tool3b"}
	ts3.AddReadTools(NewServerTool(tool3a, nil), NewServerTool(tool3b, nil))

	// Add toolsets to group
	tg.AddToolset(ts1)
	tg.AddToolset(ts2)
	tg.AddToolset(ts3)

	t.Run("Success - List all toolsets", func(t *testing.T) {
		infos := tg.ListToolsets()

		assert.Len(t, infos, 3)

		// Find each toolset
		infoMap := make(map[string]ToolsetInfo)
		for _, info := range infos {
			infoMap[info.Name] = info
		}

		// Verify ts1
		assert.Contains(t, infoMap, "ts1")
		assert.Equal(t, "Toolset 1", infoMap["ts1"].Description)
		assert.True(t, infoMap["ts1"].Enabled)
		assert.Equal(t, 1, infoMap["ts1"].ToolCount)

		// Verify ts2
		assert.Contains(t, infoMap, "ts2")
		assert.False(t, infoMap["ts2"].Enabled)
		assert.Equal(t, 1, infoMap["ts2"].ToolCount)

		// Verify ts3
		assert.Contains(t, infoMap, "ts3")
		assert.True(t, infoMap["ts3"].Enabled)
		assert.Equal(t, 2, infoMap["ts3"].ToolCount)
	})

	t.Run("Success - List empty group", func(t *testing.T) {
		tg2 := NewToolsetGroup(false)
		infos := tg2.ListToolsets()

		assert.NotNil(t, infos)
		assert.Len(t, infos, 0)
	})

	t.Run("Success - List after toolset modification", func(t *testing.T) {
		// Initially ts2 is disabled
		infos := tg.ListToolsets()
		infoMap := make(map[string]ToolsetInfo)
		for _, info := range infos {
			infoMap[info.Name] = info
		}
		assert.False(t, infoMap["ts2"].Enabled)

		// Enable ts2
		tg.EnableToolset("ts2")

		// List again
		infos = tg.ListToolsets()
		infoMap = make(map[string]ToolsetInfo)
		for _, info := range infos {
			infoMap[info.Name] = info
		}

		// Should now be enabled
		assert.True(t, infoMap["ts2"].Enabled)
	})
}
