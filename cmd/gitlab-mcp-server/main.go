package main

import (
	"context"    // Added for signal handling
	"fmt"        // Added for stdio logging
	"io"         // Added for io.Reader/io.Writer interfaces
	stdlog "log" // Use standard log for initial fatal errors
	"os"
	"os/signal" // Added for signal handling
	"strings"   // Added for toolset parsing
	"syscall"   // Added for signal handling
	"time"      // Added for token validation

	"github.com/InkyQuill/gitlab-mcp-server/pkg/gitlab"       // Reference pkg/gitlab
	iolog "github.com/InkyQuill/gitlab-mcp-server/pkg/log"    // Command logging I/O wrapper
	"github.com/InkyQuill/gitlab-mcp-server/pkg/translations" // i18n support
	"github.com/mark3labs/mcp-go/server"                      // MCP server components
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	gl "gitlab.com/gitlab-org/api/client-go" // Alias for GitLab client library
	// MCP types
)

// Injected by goreleaser
var version = "dev"
var commit = "none"
var date = "unknown"

var (
	rootCmd = &cobra.Command{
		Use:     "gitlab-mcp-server",
		Short:   "GitLab MCP Server",
		Long:    `A GitLab MCP server that provides tools for interacting with GitLab resources via the Model Context Protocol.`,
		Version: fmt.Sprintf("Version: %s\nCommit: %s\nBuild Date: %s", version, commit, date),
	}

	stdioCmd = &cobra.Command{
		Use:   "stdio",
		Short: "Start server communicating via standard input/output",
		Long:  `Starts the GitLab MCP server, listening for JSON-RPC messages on stdin and sending responses to stdout.`,
		Run: func(_ *cobra.Command, _ []string) {
			// --- Subtask 6.2: Initialize Logger ---
			logLevel := viper.GetString("log.level")
			logFile := viper.GetString("log.file")
			logger, err := initLogger(logLevel, logFile)
			if err != nil {
				stdlog.Fatalf("Failed to initialize logger: %v", err) // Use stdlog before logger is ready
			}
			logger.Info("Logger initialized")

			// --- Initialize Translations ---
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

			// --- Subtask 6.2: Initialize Signal Handling ---
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop() // Ensure stop is called to release resources
			logger.Info("Signal handling initialized")

			// --- Subtask 6.3: Main Execution Flow ---
			logger.Info("Starting main execution flow...")

			// Read configuration
			token := viper.GetString("token")
			if token == "" {
				logger.Fatal("Required configuration missing: GITLAB_TOKEN (or --gitlab-token) must be set.")
			}
			host := viper.GetString("host") // Optional, defaults handled by NewClient
			readOnly := viper.GetBool("read-only")

			// Special handling for toolsets slice from env var
			var enabledToolsets []string
			toolsetsStr := viper.GetString("toolsets") // Get as string first
			if toolsetsStr != "" {
				enabledToolsets = strings.Split(toolsetsStr, ",")
			} else {
				// Fallback or default if necessary, viper should handle defaults from flags though
				enabledToolsets = gitlab.DefaultTools
				logger.Infof("No toolsets specified via config/env, using default: %v", enabledToolsets)
			}
			// Alternative using UnmarshalKey (might be cleaner if defaults work correctly)
			// err = viper.UnmarshalKey("toolsets", &enabledToolsets)
			// if err != nil {
			// 	logger.Fatalf("Failed to unmarshal toolsets: %v", err)
			// }
			logger.Infof("Enabled toolsets: %v", enabledToolsets)
			logger.Infof("Read-only mode: %t", readOnly)
			if host != "" {
				logger.Infof("Using custom GitLab host: %s", host)
			}

			// Initialize Token Store
			tokenStore := gitlab.NewTokenStore()
			logger.Info("Token store initialized")

			// Initialize Client Pool
			clientPool := gitlab.NewClientPool(tokenStore, logger)
			logger.Info("Client pool initialized")

			// Initialize client from environment
			if err := clientPool.InitializeFromEnv(ctx, token, host); err != nil {
				logger.Fatalf("Failed to initialize client from environment: %v", err)
			}
			logger.Info("GitLab client initialized and added to pool")

			// Validate token on startup and get metadata
			logger.Info("Validating GitLab token...")
			glClient, serverName, err := clientPool.GetDefaultClient()
			if err != nil {
				logger.Fatalf("Failed to get default client: %v", err)
			}

			tokenMetadata, err := validateTokenOnStartup(ctx, glClient, token)
			if err != nil {
				// Token is invalid or expired - send notification but don't fail
				logger.Warnf("Token validation warning: %v", err)
				// Note: Notifications are sent internally via logger
			} else {
				// Store token metadata
				tokenMetadata.Name = serverName
				tokenMetadata.GitLabHost = host
				if addErr := tokenStore.AddToken(serverName, tokenMetadata); addErr != nil {
					logger.Warnf("Failed to store token metadata: %v", addErr)
				} else {
					logger.Infof("Token validated successfully for user %s (ID: %d) on server '%s'",
						tokenMetadata.Username, tokenMetadata.UserID, serverName)
					// Check if token is expiring soon (less than 30 days)
					if tokenMetadata.ExpiresAt != nil {
						daysUntil := tokenMetadata.DaysUntilExpiry()
						if daysUntil > 0 && daysUntil <= 30 {
							logger.Warnf("Token will expire in %d days. Please create a new token and update it.", daysUntil)
						}
					}
				}
			}

			// Create Client Resolver
			resolver := gitlab.NewClientResolver(clientPool, serverName, logger)
			logger.Infof("Client resolver initialized with default server '%s'", serverName)

			// Check if dynamic toolsets mode is enabled
			dynamicToolsets := viper.GetBool("dynamic-toolsets")

			// Initialize Toolsets, passing the getClient function from resolver, logger, tokenStore and translations
			toolsetGroup, err := gitlab.InitToolsets(enabledToolsets, readOnly, resolver.GetClientFn(), logger, tokenStore, t, dynamicToolsets)
			if err != nil {
				logger.Fatalf("Failed to initialize toolsets: %v", err)
			}
			logger.Info("Toolsets initialized")

			// Create MCP Server
			// Use app name and version
			mcpServer := gitlab.NewServer("gitlab-mcp-server", version)
			logger.Info("MCP server wrapper created")

			// Register toolsets with the server
			if dynamicToolsets {
				// Dynamic mode: only register discovery tools initially
				dynamicManager := gitlab.NewDynamicToolsetManager(toolsetGroup, mcpServer, logger)
				dynamicManager.SetDynamicMode(true)
				dynamicManager.RegisterDiscoveryTools()
				logger.Info("Dynamic toolset discovery tools registered")
			} else {
				// Traditional mode: register all enabled toolsets
				toolsetGroup.RegisterTools(mcpServer)
				logger.Info("Toolsets registered with MCP server")
			}

			// Create Stdio Server
			stdioServer := server.NewStdioServer(mcpServer)
			// Configure logger for the stdio transport layer
			stdioLogger := stdlog.New(logger.Writer(), "[StdioServer] ", 0) // Use logger's writer
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
			case <-ctx.Done(): // Triggered by signal
				logger.Info("Shutdown signal received, context cancelled.")
			case err := <-errC: // Triggered by server.Listen returning an error
				if err != nil && err != context.Canceled {
					logger.Errorf("Server encountered an error: %v", err)
					// We might want os.Exit(1) here depending on desired behavior
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
	rootCmd.PersistentFlags().String("gitlab-token", "", "GitLab Personal Access Token (required)")
	rootCmd.PersistentFlags().String("log-file", "", "Optional: Path to write log output to a file")
	rootCmd.PersistentFlags().String("log-level", "info", "Log level (e.g., debug, info, warn, error)")
	rootCmd.PersistentFlags().Bool("enable-command-logging", false, "Enable logging of all MCP JSON-RPC requests/responses to stderr (WARNING: may contain sensitive data)")
	rootCmd.PersistentFlags().Bool("export-translations", false, "Generate gitlab-mcp-server-config.json with all translation keys and exit")
	rootCmd.PersistentFlags().Bool("dynamic-toolsets", false, "Enable dynamic toolset discovery (toolsets loaded on-demand)")

	// Bind persistent flags to Viper
	// Note the mapping from flag name (kebab-case) to viper key (often snake_case or kept kebab-case) and ENV var (UPPER_SNAKE_CASE)
	_ = viper.BindPFlag("toolsets", rootCmd.PersistentFlags().Lookup("toolsets"))
	_ = viper.BindPFlag("read-only", rootCmd.PersistentFlags().Lookup("read-only"))
	_ = viper.BindPFlag("host", rootCmd.PersistentFlags().Lookup("gitlab-host"))                              // Viper key "host" -> GITLAB_HOST
	_ = viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("gitlab-token"))                            // Viper key "token" -> GITLAB_TOKEN
	_ = viper.BindPFlag("log.file", rootCmd.PersistentFlags().Lookup("log-file"))                             // Viper key "log.file" -> GITLAB_LOG_FILE
	_ = viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))                           // Viper key "log.level" -> GITLAB_LOG_LEVEL
	_ = viper.BindPFlag("enable-command-logging", rootCmd.PersistentFlags().Lookup("enable-command-logging")) // Viper key "enable-command-logging" -> GITLAB_ENABLE_COMMAND_LOGGING
	_ = viper.BindPFlag("export-translations", rootCmd.PersistentFlags().Lookup("export-translations"))       // Viper key "export-translations" -> GITLAB_EXPORT_TRANSLATIONS
	_ = viper.BindPFlag("dynamic-toolsets", rootCmd.PersistentFlags().Lookup("dynamic-toolsets"))             // Viper key "dynamic-toolsets" -> GITLAB_DYNAMIC_TOOLSETS

	// Add subcommands
	rootCmd.AddCommand(stdioCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Set ENV var prefix
	viper.SetEnvPrefix("GITLAB")
	// Read in environment variables that match defined flags/keys
	viper.AutomaticEnv()

	// Optional: Configure reading from a config file
	// viper.SetConfigName("config") // name of config file (without extension)
	// viper.AddConfigPath(".")      // optionally look for config in the working directory
	// viper.AddConfigPath("$HOME/.gitlab-mcp-server") // call multiple times to add search paths
	// If a config file is found, read it in.
	// if err := viper.ReadInConfig(); err == nil {
	//  fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
	// }
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
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file '%s': %w", filePath, err)
		}
		logger.SetOutput(file)
		// Optional: Also log to stderr if logging to file?
		// logger.SetOutput(io.MultiWriter(os.Stderr, file))
	} else {
		logger.SetOutput(os.Stderr)
	}

	// Set Formatter (using TextFormatter for now)
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	return logger, nil
}

// validateTokenOnStartup validates the GitLab token by calling the API
// Returns TokenMetadata with user information if successful
func validateTokenOnStartup(ctx context.Context, client *gl.Client, tokenStr string) (*gitlab.TokenMetadata, error) {
	// Call GitLab API /user to validate token
	user, resp, err := client.Users.CurrentUser(gl.WithContext(ctx))

	if err != nil {
		if resp != nil && resp.StatusCode == 401 {
			return nil, fmt.Errorf("token is invalid or expired (401 Unauthorized)")
		}
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Create token metadata with user info
	metadata := &gitlab.TokenMetadata{
		Token:         tokenStr,
		CreatedAt:     time.Now(), // Approximation - we don't know actual creation date
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
