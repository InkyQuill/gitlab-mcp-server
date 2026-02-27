package main

import (
	"context"
	"fmt"
	"io"
	stdlog "log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	cmdConfig "github.com/InkyQuill/gitlab-mcp-server/cmd/config"
	cmdInstall "github.com/InkyQuill/gitlab-mcp-server/cmd/install"
	cmdProject "github.com/InkyQuill/gitlab-mcp-server/cmd/project"
	"github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	"github.com/InkyQuill/gitlab-mcp-server/pkg/gitlab"
	iolog "github.com/InkyQuill/gitlab-mcp-server/pkg/log"
	"github.com/InkyQuill/gitlab-mcp-server/pkg/translations"
	"github.com/awnumar/memguard"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	gl "gitlab.com/gitlab-org/api/client-go"
)

// Injected by goreleaser
var version = "dev"
var commit = "none"
var date = "unknown"

var (
	rootCmd = &cobra.Command{
		Use:   "gitlab-mcp-server",
		Short: "GitLab MCP Server",
		Long: `A GitLab MCP server that provides tools for interacting with GitLab resources
via the Model Context Protocol.

This server supports multiple GitLab instances and can be configured via:
- Global config file (~/.gitlab-mcp-server/gitlab-mcp-server-config.json)
- Environment variables (GITLAB_TOKEN, GITLAB_HOST)
- Project-specific .gmcprc files`,
		Version: fmt.Sprintf("Version: %s\nCommit: %s\nBuild Date: %s", version, commit, date),
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("GitLab MCP Server v%s\n", version)
			if commit != "none" {
				fmt.Printf("Commit: %s\n", commit)
			}
			if date != "unknown" {
				fmt.Printf("Build Date: %s\n", date)
			}
		},
	}

	stdioCmd = &cobra.Command{
		Use:   "stdio",
		Short: "Start server communicating via standard input/output",
		Long:  `Starts the GitLab MCP server, listening for JSON-RPC messages on stdin and sending responses to stdout.`,
		Run: func(_ *cobra.Command, _ []string) {
			// Check if secure memory is enabled
			useSecureMemory := viper.GetBool("use-secure-memory")
			if useSecureMemory {
				// Initialize memguard for secure memory handling
				memguard.CatchInterrupt()
				defer memguard.Purge()
			}

			// Initialize Logger
			logLevel := viper.GetString("log.level")
			logFile := viper.GetString("log.file")
			logger, err := initLogger(logLevel, logFile)
			if err != nil {
				stdlog.Fatalf("Failed to initialize logger: %v", err)
			}
			logger.Info("Logger initialized")

			if useSecureMemory {
				logger.Info("Secure memory (memguard) enabled - tokens will be stored in encrypted memory")
			}

			// Initialize Translations
			t, dumpTranslations := translations.TranslationHelper(logger)
			defer func() {
				if viper.GetBool("export-translations") {
					logger.Info("Exporting translations...")
					dumpTranslations()
				}
			}()

			// Handle export-translations flag
			if viper.GetBool("export-translations") {
				logger.Info("Exporting translations and exiting...")
				dumpTranslations()
				return
			}

			// Initialize Signal Handling
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			logger.Info("Signal handling initialized")

			logger.Info("Starting main execution flow...")

			// Try to load global config first
			cfgManager, err := config.NewManager("")
			if err != nil {
				logger.Warnf("Failed to create config manager: %v", err)
			}

			hasConfigServers := false
			if cfgManager != nil && cfgManager.ServerCount() > 0 {
				hasConfigServers = true
				logger.Infof("Loaded global config with %d server(s)", cfgManager.ServerCount())
			}

			// Get configuration from flags/env (for backward compatibility)
			token := viper.GetString("token")
			host := viper.GetString("host")
			readOnly := viper.GetBool("read-only")

			// Initialize Token Store with secure memory option
			var tokenStore *gitlab.TokenStore
			if useSecureMemory {
				tokenStore = gitlab.NewTokenStoreWithConfig(gitlab.TokenStoreConfig{
					UseSecureMemory: true,
					Logger:          logger,
				})
			} else {
				tokenStore = gitlab.NewTokenStore()
			}
			logger.Info("Token store initialized")

			// Initialize Client Pool
			clientPool := gitlab.NewClientPool(tokenStore, logger)
			logger.Info("Client pool initialized")

			// Priority 1: Use global config if available and has servers
			defaultServer := "default"
			if hasConfigServers {
				servers := cfgManager.ListServers()
				for _, serverCfg := range servers {
					if err := clientPool.AddServerFromConfig(ctx, serverCfg); err != nil {
						logger.Warnf("Failed to initialize client '%s': %v", serverCfg.Name, err)
					} else {
						logger.Infof("Added client '%s' from config", serverCfg.Name)
						if serverCfg.IsDefault {
							defaultServer = serverCfg.Name
						}
					}
				}
			}

			// Priority 2: Fallback to environment variables (backward compatibility)
			if token != "" {
				logger.Info("Using environment variables for client initialization")
				if err := clientPool.InitializeFromEnv(ctx, token, host); err != nil {
					logger.Fatalf("Failed to initialize client from environment: %v", err)
				}
				logger.Info("GitLab client initialized from environment")
			}

			// Check if we have any clients
			clientList := clientPool.ListClients()
			if len(clientList) == 0 {
				logger.Fatal("No clients configured. Please either:")
				logger.Fatal("  1. Run 'gitlab-mcp-server config add' to add servers")
				logger.Fatal("  2. Set GITLAB_TOKEN environment variable")
			}

			// Get toolsets
			var enabledToolsets []string
			toolsetsStr := viper.GetString("toolsets")
			if toolsetsStr != "" {
				enabledToolsets = strings.Split(toolsetsStr, ",")
			} else {
				enabledToolsets = gitlab.DefaultTools
				logger.Infof("No toolsets specified via config/env, using default: %v", enabledToolsets)
			}

			logger.Infof("Enabled toolsets: %v", enabledToolsets)
			logger.Infof("Read-only mode: %t", readOnly)

			// Validate default token on startup
			logger.Info("Validating GitLab token...")
			glClient, serverName, err := clientPool.GetDefaultClient()
			if err != nil {
				logger.Warnf("Failed to get default client: %v", err)
			} else {
				// Use the actual server name from the pool
				if serverName == "" || serverName == "default" {
					serverName = defaultServer
				}

				// Try to get token for validation
				var tokenToValidate string
				if hasConfigServers {
					if serverCfg, err := cfgManager.GetServer(serverName); err == nil {
						tokenToValidate = serverCfg.Token
					}
				}
				if tokenToValidate == "" {
					tokenToValidate = token
				}

				if tokenToValidate != "" {
					tokenMetadata, err := validateTokenOnStartup(ctx, glClient, tokenToValidate)
					if err != nil {
						logger.Warnf("Token validation warning: %v", err)
					} else {
						tokenMetadata.Name = serverName
						if host != "" {
							tokenMetadata.GitLabHost = host
						}
						if addErr := tokenStore.AddToken(serverName, tokenMetadata); addErr != nil {
							logger.Warnf("Failed to store token metadata: %v", addErr)
						} else {
							logger.Infof("Token validated successfully for user %s (ID: %d) on server '%s'",
								tokenMetadata.Username, tokenMetadata.UserID, serverName)
							if tokenMetadata.ExpiresAt != nil {
								daysUntil := tokenMetadata.DaysUntilExpiry()
								if daysUntil > 0 && daysUntil <= 30 {
									logger.Warnf("Token will expire in %d days. Please create a new token and update it.", daysUntil)
								}
							}
						}
					}
				}
			}

			// Create Client Resolver
			resolver := gitlab.NewClientResolver(clientPool, defaultServer, logger)
			logger.Infof("Client resolver initialized with default server '%s'", defaultServer)

			// Check if dynamic toolsets mode is enabled
			dynamicToolsets := viper.GetBool("dynamic-toolsets")

			// Initialize Toolsets
			toolsetGroup, err := gitlab.InitToolsets(enabledToolsets, readOnly, resolver.GetClientFn(), logger, tokenStore, t, dynamicToolsets)
			if err != nil {
				logger.Fatalf("Failed to initialize toolsets: %v", err)
			}
			logger.Info("Toolsets initialized")

			// Create MCP Server
			mcpServer := gitlab.NewServer("gitlab-mcp-server", version)
			logger.Info("MCP server wrapper created")

			// Register toolsets with the server
			if dynamicToolsets {
				dynamicManager := gitlab.NewDynamicToolsetManager(toolsetGroup, mcpServer, logger)
				dynamicManager.SetDynamicMode(true)
				dynamicManager.RegisterDiscoveryTools()
				logger.Info("Dynamic toolset discovery tools registered")
			} else {
				toolsetGroup.RegisterTools(mcpServer)
				logger.Info("Toolsets registered with MCP server")
			}

			// Create Stdio Server
			stdioServer := server.NewStdioServer(mcpServer)
			stdioLogger := stdlog.New(logger.Writer(), "[StdioServer] ", 0)
			stdioServer.SetErrorLogger(stdioLogger)
			logger.Info("Stdio server transport created")

			// Start Listening in a goroutine
			errC := make(chan error, 1)
			go func() {
				logger.Info("Starting to listen on stdio...")
				in, out := io.Reader(os.Stdin), io.Writer(os.Stdout)
				if viper.GetBool("enable-command-logging") {
					logger.Warn("Command logging enabled - sensitive data will be redacted but not guaranteed")
					loggedIO := iolog.NewIOLogger(in, out, logger)
					in, out = loggedIO, loggedIO
				}
				errC <- stdioServer.Listen(ctx, in, out)
			}()

			// Announce readiness on stderr
			fmt.Fprintf(os.Stderr, "GitLab MCP Server running on stdio (Version: %s, Commit: %s)\n", version, commit)
			logger.Info("Server running, waiting for requests or signals...")

			// Wait for shutdown signal or server error
			select {
			case <-ctx.Done():
				logger.Info("Shutdown signal received, context cancelled.")
			case err := <-errC:
				if err != nil && err != context.Canceled {
					logger.Errorf("Server encountered an error: %v", err)
				} else {
					logger.Info("Server listener stopped gracefully.")
				}
			}

			logger.Info("Server shutting down.")
		},
	}
)

func init() {
	cobra.OnInitialize(initConfig)

	// Set version template
	rootCmd.SetVersionTemplate("{{.Short}}\n{{.Version}}\n")

	// Define persistent flags for the root command (and inherited by subcommands)
	rootCmd.PersistentFlags().StringSlice("toolsets", gitlab.DefaultTools, "Comma-separated list of toolsets to enable (e.g., 'projects,issues' or 'all')")
	rootCmd.PersistentFlags().Bool("read-only", false, "Restrict the server to read-only operations")
	rootCmd.PersistentFlags().String("gitlab-host", "", "Optional: Specify the GitLab hostname for self-managed instances (e.g., gitlab.example.com)")
	rootCmd.PersistentFlags().String("gitlab-token", "", "GitLab Personal Access Token (required if not using config)")
	rootCmd.PersistentFlags().String("log-file", "", "Optional: Path to write log output to a file")
	rootCmd.PersistentFlags().String("log-level", "info", "Log level (e.g., debug, info, warn, error)")
	rootCmd.PersistentFlags().Bool("enable-command-logging", false, "Enable logging of all MCP JSON-RPC requests/responses to stderr (WARNING: may contain sensitive data)")
	rootCmd.PersistentFlags().Bool("export-translations", false, "Generate gitlab-mcp-server-config.json with all translation keys and exit")
	rootCmd.PersistentFlags().Bool("dynamic-toolsets", false, "Enable dynamic toolset discovery (toolsets loaded on-demand)")
	rootCmd.PersistentFlags().Bool("use-secure-memory", false, "Use secure memory (memguard) for storing tokens in encrypted memory to prevent swapping to disk")

	// Bind persistent flags to Viper
	_ = viper.BindPFlag("toolsets", rootCmd.PersistentFlags().Lookup("toolsets"))
	_ = viper.BindPFlag("read-only", rootCmd.PersistentFlags().Lookup("read-only"))
	_ = viper.BindPFlag("host", rootCmd.PersistentFlags().Lookup("gitlab-host"))
	_ = viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("gitlab-token"))
	_ = viper.BindPFlag("log.file", rootCmd.PersistentFlags().Lookup("log-file"))
	_ = viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
	_ = viper.BindPFlag("enable-command-logging", rootCmd.PersistentFlags().Lookup("enable-command-logging"))
	_ = viper.BindPFlag("export-translations", rootCmd.PersistentFlags().Lookup("export-translations"))
	_ = viper.BindPFlag("dynamic-toolsets", rootCmd.PersistentFlags().Lookup("dynamic-toolsets"))
	_ = viper.BindPFlag("use-secure-memory", rootCmd.PersistentFlags().Lookup("use-secure-memory"))

	// Add subcommands
	rootCmd.AddCommand(
		stdioCmd,
		cmdConfig.Cmd,
		cmdProject.Cmd,
		cmdInstall.Cmd,
		versionCmd,
	)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Set ENV var prefix
	viper.SetEnvPrefix("GITLAB")
	// Read in environment variables that match defined flags/keys
	viper.AutomaticEnv()
}

// initLogger sets up the logrus logger based on configuration.
func initLogger(level string, filePath string) (*log.Logger, error) {
	logger := log.New()

	// Set Log Level
	lvl, err := log.ParseLevel(level)
	if err != nil {
		logger.Warnf("Invalid log level '%s', defaulting to 'info': %v", level, err)
		lvl = log.InfoLevel
	}
	logger.SetLevel(lvl)

	// Set Output
	if filePath != "" {
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file '%s': %w", filePath, err)
		}
		logger.SetOutput(file)
	} else {
		logger.SetOutput(os.Stderr)
	}

	// Set Formatter
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	return logger, nil
}

// validateTokenOnStartup validates the GitLab token by calling the API
// Returns TokenMetadata with user information if successful
func validateTokenOnStartup(ctx context.Context, client *gl.Client, tokenStr string) (*gitlab.TokenMetadata, error) {
	user, resp, err := client.Users.CurrentUser(gl.WithContext(ctx))

	if err != nil {
		if resp != nil && resp.StatusCode == 401 {
			return nil, fmt.Errorf("token is invalid or expired (401 Unauthorized)")
		}
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	metadata := &gitlab.TokenMetadata{
		Token:         tokenStr,
		CreatedAt:     time.Now(),
		LastValidated: time.Now(),
		UserID:        user.ID,
		Username:      user.Username,
		IsExpiredFlag: false,
	}

	return metadata, nil
}

func main() {
	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}
