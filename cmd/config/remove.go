package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	"github.com/spf13/cobra"
)

var (
	removeForce bool
)

func newRemoveCmd(_ *config.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a GitLab server configuration",
		Long: `Remove a GitLab server configuration.

The default server cannot be removed unless it's the only server.
Set another server as default first using 'config default'.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgManager, err := config.NewManager("")
			if err != nil {
				return fmt.Errorf("failed to create config manager: %w", err)
			}
			return runRemove(cfgManager, args[0])
		},
	}

	cmd.Flags().BoolVar(&removeForce, "force", false, "Skip confirmation prompt")

	return cmd
}

func runRemove(cfgManager *config.Manager, name string) error {
	// Check if server exists
	if !cfgManager.HasServer(name) {
		return fmt.Errorf("server '%s' not found. Use 'config list' to see available servers", name)
	}

	// Get server info
	server, err := cfgManager.GetServer(name)
	if err != nil {
		return err
	}

	// Check if it's the default server
	if server.IsDefault && cfgManager.ServerCount() > 1 {
		return fmt.Errorf("cannot remove default server '%s'. Set another server as default first using 'config default <name>'", name)
	}

	// Confirm removal
	if !removeForce {
		fmt.Printf("Are you sure you want to remove server '%s'? ", name)
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("Operation cancelled.")
			return nil
		}
	}

	// Remove server
	if err := cfgManager.RemoveServer(name); err != nil {
		return err
	}

	// Save config
	if err := cfgManager.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Server '%s' removed successfully.\n", name)
	return nil
}
