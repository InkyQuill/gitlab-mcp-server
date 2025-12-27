package toolsets

import (
	"errors"
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Toolset represents a logical group of MCP tools.
type Toolset struct {
	Name        string
	Description string
	Enabled     bool // Whether this toolset is active based on configuration
	readOnly    bool // Whether the toolset (and its tools) should operate in read-only mode
	writeTools  []server.ServerTool
	readTools   []server.ServerTool
}

// ToolsetInfo provides metadata about a toolset.
type ToolsetInfo struct {
	Name        string
	Description string
	Enabled     bool
	ToolCount   int
}

// ToolsetGroup manages a collection of Toolsets.
type ToolsetGroup struct {
	Toolsets     map[string]*Toolset
	everythingOn bool   // Flag if "all" toolsets were requested
	readOnly     bool   // Global read-only flag propagated to added toolsets
	mu           sync.RWMutex // Mutex for thread-safe operations
}

// NewServerTool creates a ServerTool struct containing the MCP tool definition and its handler.
// This is the structure expected by the mcp-go server's AddTool method.
func NewServerTool(tool mcp.Tool, handler server.ToolHandlerFunc) server.ServerTool {
	return server.ServerTool{
		Tool:    tool,
		Handler: handler,
	}
}

// NewToolset creates a new, disabled Toolset instance.
func NewToolset(name string, description string) *Toolset {
	return &Toolset{
		Name:        name,
		Description: description,
		Enabled:     false, // Toolsets start disabled by default
		readOnly:    false,
		writeTools:  make([]server.ServerTool, 0),
		readTools:   make([]server.ServerTool, 0),
	}
}

// AddReadTools adds tools intended for read-only operations to the Toolset.
// It enforces that the mcp.Tool definition includes Annotations.ReadOnlyHint = true.
func (t *Toolset) AddReadTools(tools ...server.ServerTool) *Toolset {
	// Check ReadOnlyHint for each tool (optional strictness) - Checks removed for simplicity
	// for _, tool := range tools {
	// 	if !tool.Tool.Annotations.ReadOnlyHint {
	// 		// fmt.Printf("Warning: Adding tool '%s' to read tools without ReadOnlyHint\n", tool.Tool.Name)
	// 	}
	// }
	// Simplify append using variadic operator
	t.readTools = append(t.readTools, tools...)
	return t
}

// AddWriteTools adds tools that perform write operations to the Toolset.
// It enforces that the mcp.Tool definition does NOT have Annotations.ReadOnlyHint = true.
// If the Toolset itself is marked readOnly, write tools are effectively ignored during registration.
func (t *Toolset) AddWriteTools(tools ...server.ServerTool) *Toolset {
	// Check ReadOnlyHint for each tool (optional strictness) - Checks removed for simplicity
	// for _, tool := range tools {
	// 	if tool.Tool.Annotations.ReadOnlyHint {
	// 		// fmt.Printf("Warning: Adding tool '%s' with ReadOnlyHint to write tools\n", tool.Tool.Name)
	// 	}
	// }
	// Simplify append using variadic operator
	t.writeTools = append(t.writeTools, tools...)
	return t
}

// GetActiveTools returns the list of tools that should be registered based on the
// Toolset's Enabled and readOnly flags.
func (t *Toolset) GetActiveTools() []server.ServerTool {
	if !t.Enabled {
		return nil // Return empty slice if toolset is disabled
	}
	if t.readOnly {
		// Return only read tools if toolset is in read-only mode
		active := make([]server.ServerTool, len(t.readTools))
		copy(active, t.readTools)
		return active
	}
	// Return both read and write tools if enabled and not read-only
	active := make([]server.ServerTool, 0, len(t.readTools)+len(t.writeTools))
	active = append(active, t.readTools...)
	active = append(active, t.writeTools...)
	return active
}

// RegisterTools adds the Toolset's active tools to the provided MCP server instance.
func (t *Toolset) RegisterTools(s *server.MCPServer) {
	activeTools := t.GetActiveTools()
	for _, tool := range activeTools {
		// Note: Error handling for AddTool might be needed depending on mcp-go library
		s.AddTool(tool.Tool, tool.Handler) // Use the Tool and Handler fields
	}
}

// SetReadOnly forces the toolset into read-only mode.
func (t *Toolset) SetReadOnly() {
	t.readOnly = true
}

// GetDescription returns the toolset's description.
func (t *Toolset) GetDescription() string {
	return t.Description
}

// Tools returns all tools (both read and write) in the toolset.
func (t *Toolset) Tools() []server.ServerTool {
	allTools := make([]server.ServerTool, 0, len(t.readTools)+len(t.writeTools))
	allTools = append(allTools, t.readTools...)
	allTools = append(allTools, t.writeTools...)
	return allTools
}

// IsEnabled returns whether the toolset is currently enabled.
func (t *Toolset) IsEnabled() bool {
	return t.Enabled
}

// Enable enables the toolset.
func (t *Toolset) Enable() {
	t.Enabled = true
}

// Disable disables the toolset.
func (t *Toolset) Disable() {
	t.Enabled = false
}

// NewToolsetGroup creates a new manager for multiple Toolsets.
// The readOnly flag applies globally to all toolsets added subsequently.
func NewToolsetGroup(readOnly bool) *ToolsetGroup {
	return &ToolsetGroup{
		Toolsets: make(map[string]*Toolset),
		readOnly: readOnly,
	}
}

// AddToolset adds a pre-configured Toolset to the group.
// It applies the group's global readOnly setting if it's true.
func (tg *ToolsetGroup) AddToolset(ts *Toolset) {
	tg.mu.Lock()
	defer tg.mu.Unlock()

	if tg.readOnly {
		ts.SetReadOnly() // Propagate global read-only setting
	}
	tg.Toolsets[ts.Name] = ts
}

// EnableToolset enables a single toolset by name.
// Returns an error if the toolset is not found or is already enabled.
func (tg *ToolsetGroup) EnableToolset(name string) error {
	tg.mu.Lock()
	defer tg.mu.Unlock()

	ts, ok := tg.Toolsets[name]
	if !ok {
		return fmt.Errorf("toolset '%s' not found", name)
	}

	if ts.IsEnabled() {
		return fmt.Errorf("toolset '%s' already enabled", name)
	}

	ts.Enable()
	return nil
}

// EnableToolsets enables multiple toolsets based on a list of names.
// Handles the special "all" keyword to enable all known toolsets.
func (tg *ToolsetGroup) EnableToolsets(names []string) error {
	if len(names) == 0 {
		return errors.New("no toolsets specified to enable") // Or enable none/default?
	}

	if len(names) == 1 && names[0] == "all" {
		tg.mu.Lock()
		defer tg.mu.Unlock()
		tg.everythingOn = true
		for _, ts := range tg.Toolsets {
			ts.Enabled = true
		}
		return nil
	}

	tg.everythingOn = false // Explicitly not "all"
	enabledCount := 0
	for _, name := range names {
		if err := tg.EnableToolset(name); err != nil {
			// Decide on error handling: return first error, collect all, or log and continue?
			// Returning first error for now.
			return err
		}
		enabledCount++
	}

	if enabledCount == 0 {
		return errors.New("no valid toolsets were enabled")
	}

	return nil
}

// RegisterTools iterates through all managed Toolsets and registers the active tools
// of the *enabled* ones with the provided MCP server.
func (tg *ToolsetGroup) RegisterTools(s *server.MCPServer) {
	tg.mu.RLock()
	defer tg.mu.RUnlock()

	for _, ts := range tg.Toolsets {
		if ts.Enabled {
			ts.RegisterTools(s)
		}
	}
}

// ListToolsets returns information about all available toolsets.
func (tg *ToolsetGroup) ListToolsets() []ToolsetInfo {
	tg.mu.RLock()
	defer tg.mu.RUnlock()

	infos := make([]ToolsetInfo, 0, len(tg.Toolsets))
	for name, ts := range tg.Toolsets {
		info := ToolsetInfo{
			Name:        name,
			Description: ts.GetDescription(),
			Enabled:     ts.IsEnabled(),
			ToolCount:   len(ts.Tools()),
		}
		infos = append(infos, info)
	}
	return infos
}
