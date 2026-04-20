package config

import (
	"fmt"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	"github.com/spf13/cobra"
)

func newDefaultCmd(_ *config.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "default <name>",
		Short: "Set a server as the default",
		Long: `Set a server as the default.

The default server is used when no server is explicitly specified.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgManager, err := config.NewManager("")
			if err != nil {
				return fmt.Errorf("failed to create config manager: %w", err)
			}
			return runDefault(cfgManager, args[0])
		},
	}

	return cmd
}

func runDefault(cfgManager *config.Manager, name string) error {
	// Check if server exists
	if !cfgManager.HasServer(name) {
		return fmt.Errorf("server '%s' not found. Use 'config list' to see available servers", name)
	}

	// Get previous default
	prevDefault, _ := cfgManager.GetDefaultServer()

	// Set new default
	if err := cfgManager.SetDefaultServer(name); err != nil {
		return err
	}

	// Save config
	if err := cfgManager.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Show confirmation
	if prevDefault != nil && prevDefault.Name != name {
		fmt.Printf("Default server changed from '%s' to '%s'.\n", prevDefault.Name, name)
	} else {
		fmt.Printf("Server '%s' is now the default.\n", name)
	}

	return nil
}
