package project

import (
	"fmt"
	"os"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/gitlab"
	"github.com/spf13/cobra"
)

var (
	initServer string
	initHost   string
)

// newInitCmd creates the init command
func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [projectId]",
		Short: "Initialize .gmcprc in the current directory",
		Long: `Initialize .gmcprc in the current directory.

If projectId is not provided, it will be auto-detected from Git remote.
If multiple servers are configured, you can specify which one to use with --server.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runInit,
	}

	cmd.Flags().StringVar(&initServer, "server", "", "Name of the token/server to use")
	cmd.Flags().StringVar(&initHost, "host", "", "GitLab host URL (e.g., https://gitlab.example.com)")

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	var projectID string
	var gitlabHost string

	// Get projectId from argument or detect from Git
	if len(args) > 0 {
		projectID = args[0]
	} else {
		// Auto-detect from Git remote
		var err error
		projectID, gitlabHost, err = detectFromGit()
		if err != nil {
			return fmt.Errorf("failed to detect project from Git remote: %w\n\n"+
				"Please specify projectId explicitly:\n"+
				"  gitlab-mcp-server project init <projectId>", err)
		}
	}

	// Use provided host flag if set
	if initHost != "" {
		gitlabHost = initHost
	}

	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create config
	config := &gitlab.ProjectConfig{
		ProjectID:  projectID,
		GitLabHost: gitlabHost,
		TokenName:  initServer,
	}

	// Write config
	configPath, err := writeConfig(cwd, config)
	if err != nil {
		return fmt.Errorf("failed to write .gmcprc: %w", err)
	}

	// Success output
	fmt.Fprintf(cmd.OutOrStdout(), "Project configuration created successfully!\n\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  Project ID:  %s\n", config.ProjectID)
	if config.GitLabHost != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  GitLab Host: %s\n", config.GitLabHost)
	}
	if config.TokenName != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  Token Name:  %s\n", config.TokenName)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\n  Config file: %s\n", configPath)

	return nil
}
