package config

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	"github.com/spf13/cobra"
)

var (
	listJSON bool
)

func newListCmd(_ *config.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured GitLab servers",
		Long: `List all configured GitLab servers.

Shows server name, host, user, read-only status, and default marker.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgManager, err := config.NewManager("")
			if err != nil {
				return fmt.Errorf("failed to create config manager: %w", err)
			}
			return runList(cfgManager, cmd.OutOrStdout())
		},
	}

	cmd.Flags().BoolVar(&listJSON, "json", false, "Output in JSON format")

	return cmd
}

func runList(cfgManager *config.Manager, out io.Writer) error {
	servers := cfgManager.ListServers()

	if len(servers) == 0 {
		fmt.Fprintln(out, "No servers configured.")
		fmt.Fprintln(out, "Use 'gitlab-mcp-server config init' to add your first server.")
		return nil
	}

	if listJSON {
		return listJSONOutput(servers, out)
	}

	return listTableOutput(servers, out)
}

func listTableOutput(servers []*config.ServerConfig, out io.Writer) error {
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tHOST\tUSER\tREAD-ONLY\tDEFAULT")

	for _, s := range servers {
		defaultMarker := ""
		if s.IsDefault {
			defaultMarker = "*"
		}

		user := s.Username
		if user == "" {
			user = "-"
		}

		readOnly := "no"
		if s.ReadOnly {
			readOnly = "yes"
		}

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			s.Name,
			s.Host,
			user,
			readOnly,
			defaultMarker,
		)
	}

	tw.Flush()
	fmt.Fprintln(out, "\n* = default server")
	return nil
}

func listJSONOutput(servers []*config.ServerConfig, out io.Writer) error {
	// Convert to JSON-friendly format
	type JSONServer struct {
		Name      string `json:"name"`
		Host      string `json:"host"`
		User      string `json:"user,omitempty"`
		ReadOnly  bool   `json:"readOnly"`
		IsDefault bool   `json:"isDefault"`
	}

	jsonServers := make([]JSONServer, len(servers))
	for i, s := range servers {
		jsonServers[i] = JSONServer{
			Name:      s.Name,
			Host:      s.Host,
			User:      s.Username,
			ReadOnly:  s.ReadOnly,
			IsDefault: s.IsDefault,
		}
	}

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(jsonServers)
}
