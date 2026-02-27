package config

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	"github.com/spf13/cobra"
	gl "gitlab.com/gitlab-org/api/client-go"
)

var (
	initName           string
	initHost           string
	initToken          string
	initReadOnly       bool
	initNonInteractive bool
	initEncryptTokens  bool
)

func newInitCmd(_ *config.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize GitLab server configuration",
		Long: `Initialize a new GitLab server configuration.

Interactive mode (default): Prompts for all required information.
Non-interactive mode: Requires --name, --host, and --token flags.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgManager, err := config.NewManagerWithEncryption("", initEncryptTokens)
			if err != nil {
				return fmt.Errorf("failed to create config manager: %w", err)
			}
			return runInit(cmd.Context(), cfgManager)
		},
	}

	cmd.Flags().StringVar(&initName, "name", "", "Server name (e.g., work, personal)")
	cmd.Flags().StringVar(&initHost, "host", "", "GitLab host URL (e.g., https://gitlab.com)")
	cmd.Flags().StringVar(&initToken, "token", "", "GitLab personal access token")
	cmd.Flags().BoolVar(&initReadOnly, "read-only", false, "Configure server in read-only mode")
	cmd.Flags().BoolVar(&initNonInteractive, "non-interactive", false, "Run without prompts")
	cmd.Flags().BoolVar(&initEncryptTokens, "encrypt-tokens", false, "Encrypt tokens using system keyring")

	return cmd
}

func runInit(ctx context.Context, cfgManager *config.Manager) error {
	// Check if config already has servers
	if cfgManager.ServerCount() > 0 {
		fmt.Printf("Configuration already exists with %d server(s).\n", cfgManager.ServerCount())
		fmt.Println("Use 'gitlab-mcp-server config add' to add more servers.")
		fmt.Println("Use 'gitlab-mcp-server config list' to see existing servers.")
		return nil
	}

	var name, host, token string
	var readOnly bool

	if initNonInteractive {
		// Non-interactive mode - use flags
		name = initName
		host = initHost
		token = initToken
		readOnly = initReadOnly

		if name == "" || host == "" || token == "" {
			return fmt.Errorf("non-interactive mode requires --name, --host, and --token flags")
		}
	} else {
		// Interactive mode - prompt for values
		reader := bufio.NewReader(os.Stdin)

		// Prompt for name
		var err error
		name, err = promptWithDefault(reader, "Server name", "default")
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		// Prompt for host
		hostInput, err := promptWithDefault(reader, "GitLab host URL", "https://gitlab.com")
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		host = normalizeHost(hostInput)

		// Prompt for token
		token, err = promptRequired(reader, "GitLab personal access token")
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		// Prompt for read-only mode
		roInput, err := promptWithDefault(reader, "Read-only mode", "false")
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		readOnly, _ = strconv.ParseBool(roInput)

		// Prompt for token encryption
		encryptInput, err := promptWithDefault(reader, "Encrypt tokens in system keyring", "false")
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		initEncryptTokens, _ = strconv.ParseBool(encryptInput)
	}

	// Validate token by creating a client
	fmt.Println("\nValidating token...")
	client, err := createGitLabClient(host, token)
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
		Token:    token,
		ReadOnly: readOnly,
		// First server is automatically default
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

	fmt.Printf("\nConfiguration saved successfully!\n")
	fmt.Printf("  Name: %s\n", serverCfg.Name)
	fmt.Printf("  Host: %s\n", serverCfg.Host)
	fmt.Printf("  User: %s\n", user.Username)
	fmt.Printf("  Read-only: %t\n", serverCfg.ReadOnly)
	if initEncryptTokens {
		fmt.Printf("  Token encryption: enabled\n")
	}
	fmt.Printf("\nConfig file: %s\n", cfgManager.FilePath)

	// Ask if user wants to add more servers
	if !initNonInteractive {
		fmt.Print("\nWould you like to add another server? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "y" || response == "yes" {
			fmt.Println("\nUse 'gitlab-mcp-server config add' to add more servers.")
		}
	}

	return nil
}

// promptWithDefault prompts the user with a default value
func promptWithDefault(reader *bufio.Reader, prompt, defaultValue string) (string, error) {
	fmt.Printf("%s [%s]: ", prompt, defaultValue)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue, nil
	}
	return input, nil
}

// promptRequired prompts the user for a required value
func promptRequired(reader *bufio.Reader, prompt string) (string, error) {
	for {
		fmt.Printf("%s: ", prompt)
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		input = strings.TrimSpace(input)
		if input != "" {
			return input, nil
		}
		fmt.Println("This field is required. Please enter a value.")
	}
}
