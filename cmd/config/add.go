package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	"github.com/spf13/cobra"
	gl "gitlab.com/gitlab-org/api/client-go"
)

var (
	addHost        string
	addToken       string
	addTokenRef    string
	addBackend     string
	addReadOnly    bool
	addEncryptFlag bool
)

func newAddCmd(_ *config.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new GitLab server configuration",
		Long: `Add a new GitLab server configuration.

Token can be supplied three ways:
  --token <raw>             DEPRECATED: exposes the secret in shell history.
  --token-ref <ref>         A ref like keyring://work or op://Work/gitlab/token.
                            The ref must resolve in a configured backend.
  (none of the above)       Prompt on TTY with no echo.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			registry, err := buildBackendRegistry(addBackend)
			if err != nil {
				return err
			}
			cfgManager, err := config.NewManagerWithRegistry("", registry)
			if err != nil {
				return fmt.Errorf("failed to create config manager: %w", err)
			}
			return runAdd(cmd.Context(), cfgManager, registry, args[0])
		},
	}

	cmd.Flags().StringVar(&addHost, "host", "", "GitLab host URL (required)")
	cmd.Flags().StringVar(&addToken, "token", "", "GitLab personal access token (DEPRECATED: use --token-ref or interactive)")
	cmd.Flags().StringVar(&addTokenRef, "token-ref", "", "Ref to a secret in a configured backend (e.g. keyring://work, op://Work/gitlab/token)")
	cmd.Flags().StringVar(&addBackend, "backend", "keyring", "Backend to store newly-entered secret (keyring|file); ignored when --token-ref is set")
	cmd.Flags().BoolVar(&addReadOnly, "read-only", false, "Configure server in read-only mode")
	cmd.Flags().BoolVar(&addEncryptFlag, "encrypt", false, "(Legacy) encrypt token using system keyring")
	_ = cmd.MarkFlagRequired("host")
	return cmd
}

func buildBackendRegistry(_ string) (*config.BackendRegistry, error) {
	r := config.NewBackendRegistry()
	if err := r.Register(config.NewKeyringBackend("gitlab-mcp-server")); err != nil {
		return nil, err
	}
	// EncryptedFileBackend is registered lazily — only if the user chooses it.
	// External backends are wired in from config file at server startup, not here.
	return r, nil
}

func runAdd(ctx context.Context, cfgManager *config.Manager, registry *config.BackendRegistry, name string) error {
	if cfgManager.HasServer(name) {
		return fmt.Errorf("server '%s' already exists. Use 'config list' to see existing servers", name)
	}
	host := normalizeHost(addHost)

	var secret string
	var tokenRef string
	switch {
	case addToken != "" && addTokenRef != "":
		return fmt.Errorf("--token and --token-ref are mutually exclusive")
	case addTokenRef != "":
		s, err := registry.Resolve(ctx, addTokenRef)
		if err != nil {
			return fmt.Errorf("resolve --token-ref: %w", err)
		}
		secret = s
		tokenRef = addTokenRef
	case addToken != "":
		fmt.Fprintln(os.Stderr, "DEPRECATION: --token exposes the secret in shell history and process listings. "+
			"Use --token-ref or interactive mode. --token will be removed in v3.0.")
		secret = addToken
		ref, err := registry.Store(ctx, addBackend, name, secret)
		if err != nil {
			return fmt.Errorf("store secret in %q backend: %w", addBackend, err)
		}
		tokenRef = ref
	default:
		s, err := promptSecret(fmt.Sprintf("GitLab access token for %q: ", name))
		if err != nil {
			return err
		}
		if s == "" {
			return fmt.Errorf("no token entered")
		}
		secret = s
		ref, err := registry.Store(ctx, addBackend, name, secret)
		if err != nil {
			return fmt.Errorf("store secret in %q backend: %w", addBackend, err)
		}
		tokenRef = ref
	}

	fmt.Printf("Validating token for %s...\n", name)
	client, err := createGitLabClient(host, secret)
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
	fmt.Printf("Token validated for user %s (ID: %d)\n", user.Username, user.ID)

	cfg := &config.ServerConfig{
		Name:     name,
		Host:     host,
		TokenRef: tokenRef,
		ReadOnly: addReadOnly,
		UserID:   user.ID,
		Username: user.Username,
	}
	if err := cfgManager.AddServer(cfg); err != nil {
		return fmt.Errorf("failed to add server: %w", err)
	}
	if err := cfgManager.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Printf("\nServer '%s' added.\n", name)
	fmt.Printf("  Host:      %s\n", host)
	fmt.Printf("  User:      %s\n", user.Username)
	fmt.Printf("  TokenRef:  %s\n", tokenRef)
	fmt.Printf("  Read-only: %v\n", addReadOnly)
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
