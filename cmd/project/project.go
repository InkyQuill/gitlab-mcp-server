package project

import (
	"github.com/InkyQuill/gitlab-mcp-server/pkg/gitlab"
	"github.com/spf13/cobra"
)

// NewCmd creates the project command group
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage .gmcprc project configuration",
		Long: `Manage .gmcprc project configuration files.

The .gmcprc file stores the GitLab project ID and optional settings
for the current directory. It is searched for in the current directory
and parent directories, similar to how Git searches for .git directory.`,
	}

	// Add subcommands
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newDetectCmd())
	cmd.AddCommand(newStatusCmd())

	return cmd
}

// readConfig reads the project config from current or parent directories
func readConfig() (*gitlab.ProjectConfig, string, error) {
	return gitlab.FindProjectConfig()
}

// writeConfig writes the project config to the specified directory
func writeConfig(dir string, config *gitlab.ProjectConfig) (string, error) {
	return gitlab.WriteProjectConfig(dir, config)
}

// detectFromGit detects the project from Git remote
func detectFromGit() (projectID, gitlabHost string, err error) {
	return gitlab.DetectProjectFromGit()
}
