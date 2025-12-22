package installer

import (
	"os"
	"path/filepath"
	"runtime"
)

// ConfigPaths holds paths to configuration files for different development environments
type ConfigPaths struct {
	VSCodeUserSettings string
	VSCodeWorkspace    string
	ClaudeDesktop      string
	ClaudeCode         string
	Cursor             string
}

// GetConfigPaths returns platform-specific paths to MCP configuration files
func GetConfigPaths() *ConfigPaths {
	homeDir := getHomeDir()
	paths := &ConfigPaths{}

	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		userProfile := os.Getenv("USERPROFILE")
		
		paths.VSCodeUserSettings = filepath.Join(appData, "Code", "User", "settings.json")
		paths.ClaudeDesktop = filepath.Join(appData, "Claude", "claude_desktop_config.json")
		paths.ClaudeCode = filepath.Join(userProfile, ".claude.json")
		paths.Cursor = filepath.Join(appData, "Cursor", "mcp.json")
	case "darwin": // macOS
		paths.VSCodeUserSettings = filepath.Join(homeDir, "Library", "Application Support", "Code", "User", "settings.json")
		paths.ClaudeDesktop = filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")
		paths.ClaudeCode = filepath.Join(homeDir, ".claude.json")
		paths.Cursor = filepath.Join(homeDir, ".cursor", "mcp.json")
	default: // Linux and others
		paths.VSCodeUserSettings = filepath.Join(homeDir, ".config", "Code", "User", "settings.json")
		paths.ClaudeDesktop = filepath.Join(homeDir, ".config", "Claude", "claude_desktop_config.json")
		paths.ClaudeCode = filepath.Join(homeDir, ".claude.json")
		paths.Cursor = filepath.Join(homeDir, ".cursor", "mcp.json")
	}

	// VSCode workspace config is relative to current directory
	paths.VSCodeWorkspace = filepath.Join(".vscode", "mcp.json")

	return paths
}

// getHomeDir returns the user's home directory
func getHomeDir() string {
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE") // Windows fallback
	}
	return home
}

// GetProjectRoot attempts to find the project root directory
// by looking for go.mod file in current directory and parent directories
func GetProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory
			break
		}
		dir = parent
	}

	// If not found, return current directory
	return os.Getwd()
}

