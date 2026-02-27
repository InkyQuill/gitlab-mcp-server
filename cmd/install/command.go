package install

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dryRun     bool
	binaryPath string
	configPath string
)

var Cmd = &cobra.Command{
	Use:   "install [editor]",
	Short: "Install GitLab MCP server for your IDE",
	Long: `Configure GitLab MCP server for your IDE.

This command will:
- Detect installed IDEs (VS Code, Cursor, Claude Desktop)
- Add the GitLab MCP server to the IDE's configuration
- Create backups before modifying configuration files
- Preserve existing MCP server configurations

Supported editors:
  - claude   Claude Desktop (default)
  - vscode   Visual Studio Code
  - cursor   Cursor Editor
  - all      All detected editors`,
	Example: `  gitlab-mcp-server install
  gitlab-mcp-server install claude
  gitlab-mcp-server install vscode --dry-run
  gitlab-mcp-server install all`,
	RunE: runInstall,
}

func init() {
	Cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without applying")
	Cmd.Flags().StringVar(&binaryPath, "binary-path", "", "Path to the MCP server binary (auto-detected if not specified)")
	Cmd.Flags().StringVar(&configPath, "config-path", "", "Custom config file path (overrides default)")

	// Bind to viper for consistency
	_ = viper.BindPFlag("install.dry-run", Cmd.Flags().Lookup("dry-run"))
	_ = viper.BindPFlag("install.binary-path", Cmd.Flags().Lookup("binary-path"))
	_ = viper.BindPFlag("install.config-path", Cmd.Flags().Lookup("config-path"))

	Cmd.AddCommand(statusCmd, pathCmd, uninstallCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	// Build install options
	opts := &InstallOptions{
		DryRun:     dryRun,
		BinaryPath: binaryPath,
		ConfigPath: configPath,
	}

	// Get editor from argument or flag
	if len(args) > 0 {
		opts.Editor = args[0]
	} else if viper.GetString("install.editor") != "" {
		opts.Editor = viper.GetString("install.editor")
	} else {
		opts.Editor = IDEClaudeDesktop // Default to Claude Desktop
	}

	return RunInstall(opts)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show installation status",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for various IDE configurations
		fmt.Println("Checking installation status...")

		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		// Check VS Code
		checkVSCode(homeDir)

		// Check Cursor
		checkCursor(homeDir)

		// Check Claude Desktop
		checkClaudeDesktop(homeDir)

		// Check Claude Code
		checkClaudeCode(homeDir)

		return nil
	},
}

var pathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show the installer script path",
	RunE: func(cmd *cobra.Command, args []string) error {
		installerPath, err := findInstallerScript()
		if err != nil {
			return fmt.Errorf("failed to find installer script: %w", err)
		}

		fmt.Println(installerPath)
		return nil
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove GitLab MCP server configuration from all IDEs",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("To uninstall GitLab MCP server:")
		fmt.Println()
		fmt.Println("1. Run the installer and use the 'Remove server' option")
		fmt.Println("2. Or manually edit your IDE configuration files:")
		fmt.Println()
		fmt.Println("   VS Code: ~/.config/Code/User/settings.json (mcp.servers section)")
		fmt.Println("   Cursor:   ~/.cursor/mcp.json")
		fmt.Println("   Claude:   ~/.config/Claude/claude_desktop_config.json (mcpServers section)")
		fmt.Println()
		fmt.Println("Then delete the 'gitlab-mcp-server' entry.")

		return nil
	},
}

func findInstallerScript() (string, error) {
	// Get the executable path
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}

	// Navigate to the project root
	execDir := filepath.Dir(execPath)

	// Try to find scripts/install.js
	// Check relative to executable
	installerPath := filepath.Join(execDir, "..", "scripts", "install.js")
	if _, err := os.Stat(installerPath); err == nil {
		return installerPath, nil
	}

	// Check if we're in development (binary in bin/)
	installerPath = filepath.Join(execDir, "..", "scripts", "install.js")
	if _, err := os.Stat(installerPath); err == nil {
		return installerPath, nil
	}

	// Try current working directory (for development)
	cwd, err := os.Getwd()
	if err == nil {
		installerPath = filepath.Join(cwd, "scripts", "install.js")
		if _, err := os.Stat(installerPath); err == nil {
			return installerPath, nil
		}
	}

	return "", fmt.Errorf("installer script not found (tried: %s)", installerPath)
}

func checkVSCode(homeDir string) {
	var configPath string
	switch runtime.GOOS {
	case "windows":
		configPath = filepath.Join(os.Getenv("APPDATA"), "Code", "User", "settings.json")
	case "darwin":
		configPath = filepath.Join(homeDir, "Library", "Application Support", "Code", "User", "settings.json")
	default: // linux and others
		configPath = filepath.Join(homeDir, ".config", "Code", "User", "settings.json")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("  VS Code: not installed")
		return
	}

	// Try to read and check for mcp.servers
	// For now, just check if file exists
	fmt.Println("  VS Code: installed (config file exists)")
}

func checkCursor(homeDir string) {
	var configPath string
	switch runtime.GOOS {
	case "windows":
		configPath = filepath.Join(os.Getenv("APPDATA"), "Cursor", "mcp.json")
	case "darwin":
		configPath = filepath.Join(homeDir, ".cursor", "mcp.json")
	default: // linux and others
		configPath = filepath.Join(homeDir, ".cursor", "mcp.json")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("  Cursor: not installed")
		return
	}

	fmt.Println("  Cursor: installed (config file exists)")
}

func checkClaudeDesktop(homeDir string) {
	var configPath string
	switch runtime.GOOS {
	case "windows":
		configPath = filepath.Join(os.Getenv("APPDATA"), "Claude", "claude_desktop_config.json")
	case "darwin":
		configPath = filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	default: // linux and others
		configPath = filepath.Join(homeDir, ".config", "Claude", "claude_desktop_config.json")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("  Claude Desktop: not installed")
		return
	}

	fmt.Println("  Claude Desktop: installed (config file exists)")
}

func checkClaudeCode(homeDir string) {
	configPath := filepath.Join(homeDir, ".claude.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("  Claude Code: not installed")
		return
	}

	fmt.Println("  Claude Code: installed (config file exists)")
}
