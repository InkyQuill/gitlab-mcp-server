package config

import (
	"github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	"github.com/spf13/cobra"
)

// NewCmd creates the config command group
func NewCmd(cfgManager *config.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage GitLab server configurations",
		Long: `Manage GitLab server configurations.

Commands for adding, listing, removing, and validating GitLab server configurations.
Configuration is stored in ~/.gitlab-mcp-server/gitlab-mcp-server-config.json.`,
	}

	// Add subcommands
	cmd.AddCommand(
		newInitCmd(cfgManager),
		newAddCmd(cfgManager),
		newListCmd(cfgManager),
		newRemoveCmd(cfgManager),
		newDefaultCmd(cfgManager),
		newValidateCmd(cfgManager),
	)

	return cmd
}
