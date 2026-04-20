package project

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	gl "gitlab.com/gitlab-org/api/client-go"
)

// newDetectCmd creates the detect command
func newDetectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect GitLab project from Git remote",
		Long: `Detect GitLab project information from the Git remote URL.

This command analyzes the .git/config file to find GitLab remotes
and displays project information including ID, name, and host.`,
		Args: cobra.NoArgs,
		RunE: runDetect,
	}

	return cmd
}

func runDetect(cmd *cobra.Command, args []string) error {
	// Detect from Git
	projectID, gitlabHost, err := detectFromGit()
	if err != nil {
		return fmt.Errorf("failed to detect project: %w", err)
	}

	// Output detected information
	fmt.Fprintf(cmd.OutOrStdout(), "Detected GitLab project:\n\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  Project ID:  %s\n", projectID)
	fmt.Fprintf(cmd.OutOrStdout(), "  GitLab Host: %s\n", gitlabHost)

	// Try to get more info from GitLab API
	// First, get a client - try to get default or any available client
	// Since we don't have access to client pool from here, we'll try to create a basic client
	// from environment if available
	token := os.Getenv("GITLAB_TOKEN")
	if token != "" {
		// Create client for verification
		host := gitlabHost
		if host == "" {
			host = "https://gitlab.com"
		}

		client, err := gl.NewClient(token, gl.WithBaseURL(host))
		if err == nil {
			ctx := context.Background()
			project, _, err := client.Projects.GetProject(projectID, nil, gl.WithContext(ctx))
			if err == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "\nProject Details:\n")
				fmt.Fprintf(cmd.OutOrStdout(), "  Name:        %s\n", project.Name)
				fmt.Fprintf(cmd.OutOrStdout(), "  Description: %s\n", truncateString(project.Description, 60))
				if project.PathWithNamespace != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  Namespace:   %s\n", project.PathWithNamespace)
				}
				if !project.CreatedAt.IsZero() {
					fmt.Fprintf(cmd.OutOrStdout(), "  Created:     %s\n", project.CreatedAt.Format("2006-01-02"))
				}
			}
			// If we can't get project details, that's okay - just show detected info
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nTo create .gmcprc, run:\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  gitlab-mcp-server project init\n")
	if gitlabHost != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "\nOr with specific host:\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  gitlab-mcp-server project init --host %s\n", gitlabHost)
	}

	return nil
}

// truncateString truncates a string to max length
func truncateString(s string, maxLen int) string {
	if s == "" {
		return "(none)"
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
