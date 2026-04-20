package project

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newStatusCmd creates the status command
func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show .gmcprc configuration status",
		Long: `Show the current .gmcprc configuration status.

Displays the path to the configuration file (if found) and its contents.
The configuration file is searched for in the current directory and
all parent directories, similar to how Git searches for .git directory.`,
		Args: cobra.NoArgs,
		RunE: runStatus,
	}

	return cmd
}

func runStatus(cmd *cobra.Command, args []string) error {
	config, configPath, err := readConfig()

	if err != nil {
		return fmt.Errorf("error reading .gmcprc: %w", err)
	}

	if config == nil {
		// No config found
		fmt.Fprintf(cmd.OutOrStdout(), "No .gmcprc configuration found.\n\n")
		fmt.Fprintf(cmd.OutOrStdout(), "The .gmcprc file stores the GitLab project ID for this directory.\n\n")
		fmt.Fprintf(cmd.OutOrStdout(), "To create one, run:\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  gitlab-mcp-server project init\n\n")
		fmt.Fprintf(cmd.OutOrStdout(), "Or detect from Git remote:\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  gitlab-mcp-server project detect\n")
		return nil
	}

	// Config found - display details
	fmt.Fprintf(cmd.OutOrStdout(), "Project configuration found!\n\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  Config file: %s\n\n", configPath)

	fmt.Fprintf(cmd.OutOrStdout(), "Configuration:\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  Project ID:  %s\n", config.ProjectID)

	if config.GitLabHost != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  GitLab Host: %s\n", config.GitLabHost)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  GitLab Host: (default: https://gitlab.com)\n")
	}

	if config.TokenName != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  Token Name:  %s\n", config.TokenName)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  Token Name:  (using default token)\n")
	}

	if !config.LastUpdated.IsZero() {
		fmt.Fprintf(cmd.OutOrStdout(), "  Last Updated: %s\n", config.LastUpdated.Format("2006-01-02 15:04:05"))
	}

	return nil
}
