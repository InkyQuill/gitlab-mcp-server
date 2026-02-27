package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	"github.com/spf13/cobra"
	gl "gitlab.com/gitlab-org/api/client-go"
)

var (
	addHost        string
	addToken       string
	addReadOnly    bool
	addEncryptFlag bool // Flag to override encryption setting
)

func newAddCmd(_ *config.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new GitLab server configuration",
		Long: `Add a new GitLab server configuration.

The server will be validated by calling GitLab API before being added.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Detect if encryption is enabled by checking if keyring has a master key
			// We use NewManagerWithEncryption to preserve existing encryption settings
			cfgManager, err := config.NewManagerWithEncryption("", addEncryptFlag)
			if err != nil {
				return fmt.Errorf("failed to create config manager: %w", err)
			}
			return runAdd(cmd.Context(), cfgManager, args[0])
		},
	}

	cmd.Flags().StringVar(&addHost, "host", "", "GitLab host URL (required)")
	cmd.Flags().StringVar(&addToken, "token", "", "GitLab personal access token (required)")
	cmd.Flags().BoolVar(&addReadOnly, "read-only", false, "Configure server in read-only mode")
	cmd.Flags().BoolVar(&addEncryptFlag, "encrypt", false, "Encrypt token using system keyring (enables encryption if not already enabled)")

	_ = cmd.MarkFlagRequired("host")
	_ = cmd.MarkFlagRequired("token")

	return cmd
}

func runAdd(ctx context.Context, cfgManager *config.Manager, name string) error {
	// Check if server already exists
	if cfgManager.HasServer(name) {
		return fmt.Errorf("server '%s' already exists. Use 'config list' to see existing servers", name)
	}

	// Normalize host
	host := normalizeHost(addHost)

	// Validate token
	fmt.Printf("Validating token for %s...\n", name)
	client, err := createGitLabClient(host, addToken)
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}

	user, resp, err := client.Users.CurrentUser(gl.WithContext(ctx))
	if err != nil {
		if resp != nil && resp.StatusCode == 401 {
			return fmt.Errorf("token validation failed: invalid or expired token (401)")
		}
		return fmt.Errorf("token validation failed: %w", err)
	}

	fmt.Printf("Token validated successfully for user %s (ID: %d)\n", user.Username, user.ID)

	// Create server config
	serverCfg := &config.ServerConfig{
		Name:     name,
		Host:     host,
		Token:    addToken,
		ReadOnly: addReadOnly,
		UserID:   user.ID,
		Username: user.Username,
	}

	// Add to config
	if err := cfgManager.AddServer(serverCfg); err != nil {
		return fmt.Errorf("failed to add server: %w", err)
	}

	// Save config
	if err := cfgManager.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Check if this is now the default
	isDefault := ""
	if serverCfg.IsDefault {
		isDefault = " (default)"
	}

	fmt.Printf("\nServer '%s'%s added successfully!\n", name, isDefault)
	fmt.Printf("  Host: %s\n", serverCfg.Host)
	fmt.Printf("  User: %s\n", user.Username)
	fmt.Printf("  Read-only: %t\n", serverCfg.ReadOnly)

	return nil
}

// normalizeHost ensures the host URL has a proper format
func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}
	return host
}

// createGitLabClient creates a GitLab client with the given host and token
func createGitLabClient(host, token string) (*gl.Client, error) {
	opts := []gl.ClientOptionFunc{}
	if host != "" && host != "https://gitlab.com" {
		opts = append(opts, gl.WithBaseURL(host))
	}
	return gl.NewClient(token, opts...)
}
