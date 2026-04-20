package project

import (
	"fmt"
	"os"
	"strings"

	pkgConfig "github.com/InkyQuill/gitlab-mcp-server/pkg/config"
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

	if len(args) > 0 {
		projectID = args[0]
	} else {
		var err error
		projectID, gitlabHost, err = detectFromGit()
		if err != nil {
			return fmt.Errorf("failed to detect project from Git remote: %w\n\n"+
				"Please specify projectId explicitly:\n"+
				"  gitlab-mcp-server project init <projectId>", err)
		}
	}
	if initHost != "" {
		gitlabHost = initHost
	}

	// Server is effectively required in v2.1 for clean forward-compat with v3.0.
	// If not provided, try to match exactly one configured server by host.
	serverName := initServer
	if serverName == "" {
		mgr, err := pkgConfig.NewManager("")
		if err == nil && mgr.ServerCount() > 0 && gitlabHost != "" {
			matches := []string{}
			for _, s := range mgr.ListServers() {
				if strings.EqualFold(strings.TrimSuffix(s.Host, "/"),
					strings.TrimSuffix(gitlabHost, "/")) {
					matches = append(matches, s.Name)
				}
			}
			if len(matches) == 1 {
				serverName = matches[0]
				fmt.Fprintf(cmd.OutOrStdout(), "Matched server %q from configured host %s.\n", serverName, gitlabHost)
			} else if len(matches) > 1 {
				return fmt.Errorf("multiple configured servers match host %s: %v — re-run with --server <name>",
					gitlabHost, matches)
			}
		}
	}
	if serverName == "" {
		return fmt.Errorf("no --server specified and could not infer one. " +
			"Run 'gitlab-mcp-server config list' to see configured servers and pass --server <name>")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// v3 .gmcprc shape: no gitlabHost, no tokenName.
	cfg := &gitlab.ProjectConfig{
		ProjectID: projectID,
		Server:    serverName,
	}
	configPath, err := writeConfig(cwd, cfg)
	if err != nil {
		return fmt.Errorf("failed to write .gmcprc: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Project configuration created.\n\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  Project ID: %s\n", cfg.ProjectID)
	fmt.Fprintf(cmd.OutOrStdout(), "  Server:     %s\n", cfg.Server)
	fmt.Fprintf(cmd.OutOrStdout(), "\n  Config file: %s\n", configPath)
	return nil
}
