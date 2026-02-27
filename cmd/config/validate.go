package config

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	"github.com/spf13/cobra"
)

func newValidateCmd(_ *config.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [name]",
		Short: "Validate GitLab server configuration(s)",
		Long: `Validate GitLab server configuration(s).

If no server name is provided, validates all configured servers.
Validates token by calling GitLab API and updates user information.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgManager, err := config.NewManager("")
			if err != nil {
				return fmt.Errorf("failed to create config manager: %w", err)
			}
			return runValidate(cmd.Context(), cfgManager, cmd.OutOrStdout(), args)
		},
	}

	return cmd
}

func runValidate(ctx context.Context, cfgManager *config.Manager, out io.Writer, args []string) error {
	if len(args) == 0 {
		// Validate all servers
		return validateAll(ctx, cfgManager, out)
	}

	// Validate specific server
	return validateOne(ctx, cfgManager, out, args[0])
}

func validateOne(ctx context.Context, cfgManager *config.Manager, out io.Writer, name string) error {
	// Check if server exists
	if !cfgManager.HasServer(name) {
		return fmt.Errorf("server '%s' not found. Use 'config list' to see available servers", name)
	}

	fmt.Fprintf(out, "Validating server '%s'...\n", name)

	serverCfg, err := cfgManager.ValidateServer(ctx, name)
	if err != nil {
		fmt.Fprintf(out, "Validation failed: %v\n", err)
		return err
	}

	fmt.Fprintf(out, "Validation successful!\n")
	fmt.Fprintf(out, "  Host: %s\n", serverCfg.Host)
	fmt.Fprintf(out, "  User: %s (ID: %d)\n", serverCfg.Username, serverCfg.UserID)
	return nil
}

func validateAll(ctx context.Context, cfgManager *config.Manager, out io.Writer) error {
	servers := cfgManager.ListServers()

	if len(servers) == 0 {
		fmt.Fprintln(out, "No servers configured.")
		return nil
	}

	fmt.Fprintf(out, "Validating %d server(s)...\n\n", len(servers))

	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SERVER\tHOST\tUSER\tSTATUS")

	hasFailure := false

	for _, s := range servers {
		serverCfg, err := cfgManager.ValidateServer(ctx, s.Name)
		status := "OK"
		user := "-"

		if err != nil {
			status = "FAILED"
			user = "-"
			hasFailure = true
		} else if serverCfg != nil {
			user = serverCfg.Username
		}

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", s.Name, s.Host, user, status)
	}

	tw.Flush()

	if hasFailure {
		return fmt.Errorf("some servers failed validation")
	}

	fmt.Fprintln(out, "\nAll servers validated successfully!")
	return nil
}
