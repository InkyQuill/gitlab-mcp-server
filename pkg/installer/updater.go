package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// UpdateConfig updates MCP configuration files for different development environments
func UpdateConfig(env string, paths *ConfigPaths, config ServerConfig) error {
	switch env {
	case "VS Code":
		return updateVSCodeConfig(paths, config)
	case "Claude Desktop":
		return updateClaudeConfig(paths.ClaudeDesktop, config)
	case "Claude Code":
		return updateClaudeCodeConfig(paths.ClaudeCode, config)
	case "Cursor":
		return updateClaudeConfig(paths.Cursor, config)
	default:
		return fmt.Errorf("unknown environment: %s", env)
	}
}

// updateVSCodeConfig updates VS Code configuration
func updateVSCodeConfig(paths *ConfigPaths, config ServerConfig) error {
	// Try workspace config first (check if .vscode directory exists or can be created)
	workspaceDir := filepath.Dir(paths.VSCodeWorkspace)
	if err := os.MkdirAll(workspaceDir, 0755); err == nil {
		// Try to use workspace config
		if err := updateVSCodeWorkspaceConfig(paths.VSCodeWorkspace, config); err == nil {
			return nil
		}
	}

	// Fall back to user settings
	return updateVSCodeUserConfig(paths.VSCodeUserSettings, config)
}

// updateVSCodeWorkspaceConfig updates .vscode/mcp.json
func updateVSCodeWorkspaceConfig(path string, config ServerConfig) error {
	var mcpConfig map[string]interface{}

	// Read existing config if it exists
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &mcpConfig); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		mcpConfig = make(map[string]interface{})
	}

	// Get or create servers map
	var servers map[string]interface{}
	if serversVal, ok := mcpConfig["servers"]; ok {
		if serversMap, ok := serversVal.(map[string]interface{}); ok {
			servers = serversMap
		} else {
			servers = make(map[string]interface{})
		}
	} else {
		servers = make(map[string]interface{})
	}

	// Convert config to map for JSON serialization
	configMap := make(map[string]interface{})
	configMap["command"] = config.Command
	if len(config.Args) > 0 {
		configMap["args"] = config.Args
	}
	if len(config.Env) > 0 {
		configMap["env"] = config.Env
	}

	// Update or add server config
	servers[ServerName] = configMap
	mcpConfig["servers"] = servers

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write updated config
	return writeJSONFile(path, mcpConfig)
}

// updateVSCodeUserConfig updates VS Code user settings.json
func updateVSCodeUserConfig(path string, config ServerConfig) error {
	var vscodeConfig VSCodeConfig

	// Read existing config if it exists
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &vscodeConfig); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	}

	if vscodeConfig.MCP.Servers == nil {
		vscodeConfig.MCP.Servers = make(map[string]ServerConfig)
	}

	// Update or add server config
	vscodeConfig.MCP.Servers[ServerName] = config

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write updated config
	return writeJSONFile(path, vscodeConfig)
}

// updateClaudeConfig updates Claude Desktop or Cursor configuration
func updateClaudeConfig(path string, config ServerConfig) error {
	var claudeConfig ClaudeConfig

	// Read existing config if it exists
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &claudeConfig); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	}

	if claudeConfig.MCPServers == nil {
		claudeConfig.MCPServers = make(map[string]ServerConfig)
	}

	// Update or add server config
	claudeConfig.MCPServers[ServerName] = config

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write updated config
	return writeJSONFile(path, claudeConfig)
}

// updateClaudeCodeConfig updates Claude Code configuration
func updateClaudeCodeConfig(path string, config ServerConfig) error {
	var claudeConfig ClaudeConfig

	// Read existing config if it exists
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &claudeConfig); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	}

	if claudeConfig.MCPServers == nil {
		claudeConfig.MCPServers = make(map[string]ServerConfig)
	}

	// Add type field for Claude Code
	config.Type = "stdio"

	// Update or add server config
	claudeConfig.MCPServers[ServerName] = config

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write updated config
	return writeJSONFile(path, claudeConfig)
}

// writeJSONFile writes JSON data to a file with backup
func writeJSONFile(path string, data interface{}) error {
	// Create backup if file exists
	if _, err := os.Stat(path); err == nil {
		backupPath := path + ".bak"
		if backupData, err := os.ReadFile(path); err == nil {
			if err := os.WriteFile(backupPath, backupData, 0644); err != nil {
				return fmt.Errorf("failed to create backup: %w", err)
			}
		}
	}

	// Marshal with indentation
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		// Try to restore backup on error
		if backupPath := path + ".bak"; backupPath != "" {
			if backupData, err := os.ReadFile(backupPath); err == nil {
				os.WriteFile(path, backupData, 0644)
			}
		}
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

