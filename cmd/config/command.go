// Package config provides CLI commands for managing GitLab MCP configuration.
package config

import (
	"github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	"github.com/spf13/cobra"
)

var (
	interactiveFlag bool
)

// Cmd is the root config command exported for use in main.go.
var Cmd = &cobra.Command{
	Use:   "config",
	Short: "Manage GitLab MCP server configuration",
	Long: `Manage the GitLab MCP server global configuration file.

This configuration file is stored at ~/.gitlab-mcp-server/gitlab-mcp-server-config.json
and allows you to configure multiple GitLab servers with their access tokens.

Run 'config -i' or 'config --interactive' for an interactive TUI experience.`,
	RunE: runConfigCmd,
}

func init() {
	// Create config manager - if it fails, we still add subcommands
	// that will handle the error when run
	cfgManager, _ := config.NewManager("")

	Cmd.Flags().BoolVarP(&interactiveFlag, "interactive", "i", false,
		"Run in interactive TUI mode")

	Cmd.AddCommand(
		newInitCmd(cfgManager),
		newAddCmd(cfgManager),
		newListCmd(cfgManager),
		newRemoveCmd(cfgManager),
		newDefaultCmd(cfgManager),
		newValidateCmd(cfgManager),
		newBackendsCmd(),
	)
}

// runConfigCmd handles the root config command
func runConfigCmd(cmd *cobra.Command, args []string) error {
	// Get a fresh config manager
	cfgManager, err := config.NewManager("")
	if err != nil {
		return err
	}

	// If interactive flag is set or no subcommand provided, run TUI
	if interactiveFlag || len(args) == 0 {
		return RunTUI(cfgManager)
	}

	// If no subcommand and not interactive, show help
	return cmd.Help()
}
