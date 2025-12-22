package main

import (
	"fmt"
	"os"

	"github.com/LuisCusihuaman/gitlab-mcp-server/pkg/installer"
)

func main() {
	fmt.Println("=== GitLab MCP Server Installer ===")
	fmt.Println()

	// Get project root
	projectRoot, err := installer.GetProjectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get project root: %v\n", err)
		os.Exit(1)
	}

	// Prompt user for configuration
	promptConfig, err := installer.PromptUser()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get binary configuration
	binaryConfig, err := installer.GetBinaryConfig(promptConfig.Mode, projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Add read-only environment variable if needed
	binaryConfig.AddReadOnlyEnv(promptConfig.ReadOnly)

	// Create server configuration
	serverConfig := installer.CreateServerConfig(
		binaryConfig,
		promptConfig.GitLabHost,
		promptConfig.ReadOnly,
		promptConfig.Token,
	)

	// Prompt for environments
	environments, err := installer.PromptEnvironments()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get configuration paths
	paths := installer.GetConfigPaths()

	// Update configurations for selected environments
	successCount := 0
	for _, env := range environments {
		fmt.Printf("\nConfiguring %s...\n", env)
		if err := installer.UpdateConfig(env, paths, serverConfig); err != nil {
			fmt.Fprintf(os.Stderr, "  Error configuring %s: %v\n", env, err)
		} else {
			fmt.Printf("  ✓ %s configured successfully\n", env)
			successCount++
		}
	}

	fmt.Println()
	if successCount > 0 {
		fmt.Printf("Successfully configured %d environment(s)!\n", successCount)
		fmt.Println("\nNext steps:")
		fmt.Println("1. Restart your development environment(s)")
		fmt.Println("2. The MCP server will be available as 'gitlab-go-mcp'")
		if promptConfig.Mode == "local" {
			fmt.Println("3. Make sure the binary exists at:", binaryConfig.LocalPath)
		} else {
			fmt.Println("3. Make sure Docker image exists: docker build -t gitlab-mcp-server:latest .")
		}
	} else {
		fmt.Println("No environments were configured successfully.")
		os.Exit(1)
	}
}

