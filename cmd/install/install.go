// Package install provides the MCP server installation command for IDEs.
package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	pathutil "github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// IDEClaudeDesktop is the identifier for Claude Desktop
	IDEClaudeDesktop = "claude"
	// IDEVSCode is the identifier for VS Code
	IDEVSCode = "vscode"
	// IDECursor is the identifier for Cursor
	IDECursor = "cursor"
	// IDEAll is the identifier for all IDEs
	IDEAll = "all"
)

// IDE represents an IDE that can be configured with the MCP server.
type IDE struct {
	Name       string
	ConfigPath string
	Enabled    bool
}

// InstallConfig holds the configuration for the MCP server entry.
type InstallConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// InstallOptions holds the options for the install command.
type InstallOptions struct {
	Editor     string
	DryRun     bool
	BinaryPath string
	ConfigPath string
}

// NewInstallCommand creates the install command.
func NewInstallCommand() *cobra.Command {
	opts := &InstallOptions{}

	cmd := &cobra.Command{
		Use:   "install [editor]",
		Short: "Install GitLab MCP server in IDE configuration",
		Long: `Install the GitLab MCP server in your IDE's MCP configuration.

Supported editors:
  - claude   Claude Desktop (default)
  - vscode   Visual Studio Code
  - cursor   Cursor Editor
  - all      All detected editors

The command will:
  1. Detect the executable path of the MCP server
  2. Find or create the IDE's configuration file
  3. Add the GitLab MCP server entry (preserving existing servers)
  4. Create a backup before modifying`,
		Example: `  gitlab-mcp-server install claude
  gitlab-mcp-server install vscode --dry-run
  gitlab-mcp-server install all --binary-path /usr/local/bin/gitlab-mcp-server`,
		Args: cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Get editor from argument or flag
			if len(args) > 0 {
				opts.Editor = args[0]
			} else if viper.GetString("install.editor") != "" {
				opts.Editor = viper.GetString("install.editor")
			} else {
				opts.Editor = IDEClaudeDesktop // Default to Claude Desktop
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.DryRun = viper.GetBool("install.dry-run")
			opts.BinaryPath = viper.GetString("install.binary-path")
			opts.ConfigPath = viper.GetString("install.config-path")

			return RunInstall(opts)
		},
	}

	// Flags
	cmd.Flags().String("editor", "", "IDE to install to (claude, vscode, cursor, all)")
	cmd.Flags().Bool("dry-run", false, "Preview changes without applying")
	cmd.Flags().String("binary-path", "", "Path to the MCP server binary (auto-detected if not specified)")
	cmd.Flags().String("config-path", "", "Custom config file path (overrides default)")

	// Bind to viper for consistency
	_ = viper.BindPFlag("install.editor", cmd.Flags().Lookup("editor"))
	_ = viper.BindPFlag("install.dry-run", cmd.Flags().Lookup("dry-run"))
	_ = viper.BindPFlag("install.binary-path", cmd.Flags().Lookup("binary-path"))
	_ = viper.BindPFlag("install.config-path", cmd.Flags().Lookup("config-path"))

	return cmd
}

// RunInstall executes the install command.
func RunInstall(opts *InstallOptions) error {
	// Detect or validate binary path
	binaryPath, err := getSelfExecutablePath(opts.BinaryPath)
	if err != nil {
		return fmt.Errorf("failed to determine binary path: %w", err)
	}

	// Detect IDEs
	id, ides, err := detectAndSelectIDE(opts)
	if err != nil {
		return fmt.Errorf("IDE detection failed: %w", err)
	}

	if opts.DryRun {
		fmt.Printf("Dry run mode - no changes will be made\n\n")
	}

	fmt.Printf("Installing GitLab MCP server for %s\n", id.Name)
	fmt.Printf("Binary path: %s\n\n", binaryPath)

	// Create install config
	installConfig := createInstallConfig(binaryPath, opts.ConfigPath)

	// Process each IDE
	var errors []error
	for _, ide := range ides {
		if err := installForIDE(ide, installConfig, opts.DryRun); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", ide.Name, err))
		}
	}

	// Report results
	if len(errors) > 0 {
		fmt.Println("\nErrors encountered:")
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
		return fmt.Errorf("installation completed with %d error(s)", len(errors))
	}

	// Success message
	fmt.Println("\nInstallation complete!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add your GitLab token: gitlab-mcp-server config init")
	fmt.Println("  2. Configure your project: gitlab-mcp-server project init")
	fmt.Println("  3. Restart your IDE")

	return nil
}

// detectAndSelectIDE detects available IDEs and returns the selected one(s).
func detectAndSelectIDE(opts *InstallOptions) (*IDE, []IDE, error) {
	allIDEs := getAllIDEs()

	// Filter based on selection
	var selected []IDE
	switch opts.Editor {
	case IDEAll:
		for _, ide := range allIDEs {
			if ide.Enabled {
				selected = append(selected, ide)
			}
		}
		if len(selected) == 0 {
			return nil, nil, fmt.Errorf("no supported IDEs detected")
		}
		return &IDE{Name: "all IDEs", Enabled: true}, selected, nil

	case IDEClaudeDesktop, IDEVSCode, IDECursor:
		for _, ide := range allIDEs {
			if ide.Name == opts.Editor {
				if !ide.Enabled {
					return nil, nil, fmt.Errorf("%s not detected on this system", ide.Name)
				}
				return &ide, []IDE{ide}, nil
			}
		}
		return nil, nil, fmt.Errorf("unknown editor: %s", opts.Editor)

	default:
		return nil, nil, fmt.Errorf("unknown editor: %s (use: claude, vscode, cursor, or all)", opts.Editor)
	}
}

// getAllIDEs returns all supported IDEs with their detection status.
func getAllIDEs() []IDE {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "~"
	}

	id := []IDE{
		{
			Name:       IDEClaudeDesktop,
			ConfigPath: getClaudeConfigPath(homeDir),
			Enabled:    claudeDesktopInstalled(homeDir),
		},
		{
			Name:       IDEVSCode,
			ConfigPath: getVSCodeConfigPath(homeDir),
			Enabled:    vsCodeInstalled(homeDir),
		},
		{
			Name:       IDECursor,
			ConfigPath: getCursorConfigPath(homeDir),
			Enabled:    cursorInstalled(homeDir),
		},
	}

	return id
}

// createInstallConfig creates the MCP server configuration entry.
func createInstallConfig(binaryPath, configPath string) *InstallConfig {
	cfg := &InstallConfig{
		Command: binaryPath,
		Args:    []string{"stdio"},
	}

	// Add environment variable for config path if specified
	if configPath != "" {
		cfg.Env = map[string]string{
			"GITLAB_CONFIG_PATH": configPath,
		}
	}

	return cfg
}

// getSelfExecutablePath returns the path to the current executable.
func getSelfExecutablePath(overridePath string) (string, error) {
	if overridePath != "" {
		// Validate the path exists
		if _, err := os.Stat(overridePath); err != nil {
			return "", fmt.Errorf("binary path does not exist: %s", overridePath)
		}
		return filepath.Abs(overridePath)
	}

	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks
	realPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		// Fallback to the original path if symlink resolution fails
		return execPath, nil
	}

	return realPath, nil
}

// getClaudeConfigPath returns the Claude Desktop config path for the current OS.
func getClaudeConfigPath(homeDir string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			return filepath.Join(homeDir, "AppData", "Roaming", "Claude", "claude_desktop_config.json")
		}
		return filepath.Join(appdata, "Claude", "claude_desktop_config.json")
	default: // linux, freebsd, etc.
		return filepath.Join(homeDir, ".config", "Claude", "claude_desktop_config.json")
	}
}

// getVSCodeConfigPath returns the VS Code settings path.
func getVSCodeConfigPath(homeDir string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "Code", "User", "settings.json")
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			return filepath.Join(homeDir, "AppData", "Roaming", "Code", "User", "settings.json")
		}
		return filepath.Join(appdata, "Code", "User", "settings.json")
	default:
		return filepath.Join(homeDir, ".config", "Code", "User", "settings.json")
	}
}

// getCursorConfigPath returns the Cursor settings path.
func getCursorConfigPath(homeDir string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "Cursor", "User", "settings.json")
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			return filepath.Join(homeDir, "AppData", "Roaming", "Cursor", "User", "settings.json")
		}
		return filepath.Join(appdata, "Cursor", "User", "settings.json")
	default:
		return filepath.Join(homeDir, ".config", "Cursor", "User", "settings.json")
	}
}

// claudeDesktopInstalled checks if Claude Desktop is installed.
func claudeDesktopInstalled(homeDir string) bool {
	configPath := getClaudeConfigPath(homeDir)
	// Check if config directory exists (config file may not exist yet)
	configDir := filepath.Dir(configPath)
	if info, err := os.Stat(configDir); err == nil && info.IsDir() {
		return true
	}
	return false
}

// vsCodeInstalled checks if VS Code is installed.
func vsCodeInstalled(homeDir string) bool {
	configPath := getVSCodeConfigPath(homeDir)
	configDir := filepath.Dir(configPath)
	if info, err := os.Stat(configDir); err == nil && info.IsDir() {
		return true
	}
	return false
}

// cursorInstalled checks if Cursor is installed.
func cursorInstalled(homeDir string) bool {
	configPath := getCursorConfigPath(homeDir)
	configDir := filepath.Dir(configPath)
	if info, err := os.Stat(configDir); err == nil && info.IsDir() {
		return true
	}
	return false
}

// installForIDE installs the MCP server for a specific IDE.
func installForIDE(ide IDE, installConfig *InstallConfig, dryRun bool) error {
	fmt.Printf("Configuring %s...\n", ide.Name)
	fmt.Printf("  Config path: %s\n", ide.ConfigPath)

	// Validate config path to prevent directory traversal
	if err := pathutil.ValidatePath(ide.ConfigPath); err != nil {
		return fmt.Errorf("invalid config path: %w", err)
	}

	// Check if we can write to the config location
	configDir := filepath.Dir(ide.ConfigPath)
	if !dryRun {
		if err := ensureDir(configDir); err != nil {
			return fmt.Errorf("cannot access config directory: %w", err)
		}
	}

	// Read existing config
	var existingConfig map[string]interface{}
	if data, err := os.ReadFile(ide.ConfigPath); err == nil {
		if err := json.Unmarshal(data, &existingConfig); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Merge config based on IDE type
	var newConfig map[string]interface{}
	var err error

	switch ide.Name {
	case IDEClaudeDesktop:
		newConfig, err = mergeClaudeConfig(existingConfig, installConfig)
	case IDEVSCode, IDECursor:
		newConfig, err = mergeVSCodeConfig(existingConfig, installConfig)
	default:
		return fmt.Errorf("unsupported IDE: %s", ide.Name)
	}

	if err != nil {
		return err
	}

	// Validate JSON
	data, err := json.MarshalIndent(newConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if dryRun {
		fmt.Printf("  Would write:\n%s\n", string(data))
		return nil
	}

	// Create backup
	if existingConfig != nil {
		backupPath := ide.ConfigPath + ".backup"
		if err := backupFile(ide.ConfigPath, backupPath); err != nil {
			fmt.Printf("  Warning: failed to create backup: %v\n", err)
		} else {
			fmt.Printf("  Backup created: %s\n", backupPath)
		}
	}

	// Write new config
	if err := os.WriteFile(ide.ConfigPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("  Successfully configured %s\n", ide.Name)
	return nil
}

// mergeClaudeConfig merges the GitLab MCP config into Claude Desktop config.
func mergeClaudeConfig(existing map[string]interface{}, installConfig *InstallConfig) (map[string]interface{}, error) {
	if existing == nil {
		existing = make(map[string]interface{})
	}

	// Get or create mcpServers section
	var mcpServers map[string]interface{}
	if val, ok := existing["mcpServers"]; ok {
		if servers, ok := val.(map[string]interface{}); ok {
			mcpServers = servers
		} else {
			return nil, fmt.Errorf("existing mcpServers is not a valid object")
		}
	} else {
		mcpServers = make(map[string]interface{})
	}

	// Add gitlab server (this will overwrite existing "gitlab" entry)
	mcpServers["gitlab"] = map[string]interface{}{
		"command": installConfig.Command,
		"args":    installConfig.Args,
	}
	if len(installConfig.Env) > 0 {
		if entry, ok := mcpServers["gitlab"].(map[string]interface{}); ok {
			entry["env"] = installConfig.Env
		}
	}

	existing["mcpServers"] = mcpServers
	return existing, nil
}

// mergeVSCodeConfig merges the GitLab MCP config into VS Code/Cursor settings.
func mergeVSCodeConfig(existing map[string]interface{}, installConfig *InstallConfig) (map[string]interface{}, error) {
	if existing == nil {
		existing = make(map[string]interface{})
	}

	// Get or create mcp.servers section
	var mcpServers map[string]interface{}
	if val, ok := existing["mcp.servers"]; ok {
		if servers, ok := val.(map[string]interface{}); ok {
			mcpServers = servers
		} else {
			return nil, fmt.Errorf("existing mcp.servers is not a valid object")
		}
	} else {
		mcpServers = make(map[string]interface{})
	}

	// Add gitlab server
	serverConfig := map[string]interface{}{
		"command": installConfig.Command,
		"args":    installConfig.Args,
	}
	if len(installConfig.Env) > 0 {
		serverConfig["env"] = installConfig.Env
	}

	mcpServers["gitlab"] = serverConfig
	existing["mcp.servers"] = mcpServers
	return existing, nil
}

// ensureDir ensures a directory exists, creating it if necessary.
func ensureDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0750)
	}
	return nil
}

// backupFile creates a backup of a file.
func backupFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0600)
}
