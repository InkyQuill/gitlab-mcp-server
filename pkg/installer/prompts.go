package installer

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

const (
	ServerName = "gitlab-go-mcp"
)

// PromptConfig holds the configuration collected from user prompts
type PromptConfig struct {
	Mode       string // "local" or "docker"
	GitLabHost string
	Token      string
	ReadOnly   bool
}

// PromptUser collects configuration from the user via interactive prompts
func PromptUser() (*PromptConfig, error) {
	config := &PromptConfig{}

	reader := bufio.NewReader(os.Stdin)

	// Prompt for mode
	fmt.Print("Select mode [local/docker] (default: local): ")
	modeInput, _ := reader.ReadString('\n')
	modeInput = strings.TrimSpace(modeInput)
	if modeInput == "" || modeInput == "local" {
		config.Mode = "local"
	} else if modeInput == "docker" {
		config.Mode = "docker"
	} else {
		return nil, fmt.Errorf("invalid mode: %s. Must be 'local' or 'docker'", modeInput)
	}

	// Prompt for GitLab host
	fmt.Print("GitLab host URL (default: https://gitlab.com, press Enter to use default): ")
	hostInput, _ := reader.ReadString('\n')
	hostInput = strings.TrimSpace(hostInput)
	if hostInput == "" {
		config.GitLabHost = "https://gitlab.com"
	} else {
		config.GitLabHost = hostInput
	}

	// Prompt for token (hidden input)
	fmt.Print("GitLab access token: ")
	tokenBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil, fmt.Errorf("failed to read token: %w", err)
	}
	fmt.Println() // New line after hidden input
	config.Token = strings.TrimSpace(string(tokenBytes))
	if config.Token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	// Prompt for read-only mode
	fmt.Print("Enable read-only mode? (y/n, default: n): ")
	readOnlyInput, _ := reader.ReadString('\n')
	readOnlyInput = strings.ToLower(strings.TrimSpace(readOnlyInput))
	config.ReadOnly = readOnlyInput == "y" || readOnlyInput == "yes"

	return config, nil
}

// PromptEnvironments asks user which development environments to configure
func PromptEnvironments() ([]string, error) {
	environments := []string{
		"VS Code",
		"Claude Desktop",
		"Claude Code",
		"Cursor",
	}

	fmt.Println("\nSelect development environments to configure (comma-separated, or 'all'):")
	for i, env := range environments {
		fmt.Printf("  %d. %s\n", i+1, env)
	}
	fmt.Print("Your choice (default: all): ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" || strings.ToLower(input) == "all" {
		return environments, nil
	}

	selected := []string{}
	parts := strings.Split(input, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Try to parse as number
		var idx int
		if _, err := fmt.Sscanf(part, "%d", &idx); err == nil {
			if idx >= 1 && idx <= len(environments) {
				selected = append(selected, environments[idx-1])
			}
		} else {
			// Try to match by name
			found := false
			for _, env := range environments {
				if strings.EqualFold(part, env) {
					selected = append(selected, env)
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("Warning: Unknown environment '%s', skipping\n", part)
			}
		}
	}

	if len(selected) == 0 {
		return environments, nil // Default to all if nothing valid selected
	}

	return selected, nil
}

