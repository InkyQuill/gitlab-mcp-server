package gitlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ConfigFileName is the name of the local configuration file
const ConfigFileName = ".gmcprc"

// ProjectConfig holds the local project configuration
type ProjectConfig struct {
	ProjectID   string    `json:"projectId"`
	GitLabHost  string    `json:"gitlabHost,omitempty"`
	TokenName   string    `json:"tokenName,omitempty"` // Reference to token in MCP config
	LastUpdated time.Time `json:"lastUpdated"`
}

// FindProjectConfig searches for .gmcprc file in current directory and parent directories
// Similar to how Git searches for .git directory
func FindProjectConfig() (*ProjectConfig, string, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Search upward for .gmcprc file
	dir := cwd
	for {
		configPath := filepath.Join(dir, ConfigFileName)

		if _, err := os.Stat(configPath); err == nil {
			// Found the config file
			config, err := readProjectConfig(configPath)
			if err != nil {
				return nil, "", fmt.Errorf("failed to read config from %s: %w", configPath, err)
			}
			return config, configPath, nil
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory
			return nil, "", nil // No error, just not found
		}
		dir = parent
	}
}

// readProjectConfig reads and parses a .gmcprc file
func readProjectConfig(path string) (*ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var config ProjectConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &config, nil
}

// WriteProjectConfig writes a .gmcprc file to the specified directory
func WriteProjectConfig(dir string, config *ProjectConfig) (string, error) {
	config.LastUpdated = time.Now()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	configPath := filepath.Join(dir, ConfigFileName)

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return configPath, nil
}

// DetectProjectFromGit attempts to detect the project ID from Git remote
func DetectProjectFromGit() (projectID, gitlabHost string, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Search for .git directory
	gitDir := findGitDir(cwd)
	if gitDir == "" {
		return "", "", fmt.Errorf("not a Git repository (or any parent up to mount point)")
	}

	// Read .git/config
	configPath := filepath.Join(gitDir, "config")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read .git/config: %w", err)
	}

	// Parse Git config to find remotes
	projectID, gitlabHost, err = parseGitRemotes(configData)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse Git remotes: %w", err)
	}

	if projectID == "" {
		return "", "", fmt.Errorf("no GitLab remote found in .git/config")
	}

	return projectID, gitlabHost, nil
}

// findGitDir searches for .git directory
func findGitDir(startDir string) string {
	dir := startDir
	for {
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil {
			if info.IsDir() {
				return gitPath
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "" // Reached root
		}
		dir = parent
	}
}

// parseGitRemotes parses .git/config content to extract GitLab remote
func parseGitRemotes(configData []byte) (projectID, gitlabHost string, err error) {
	lines := bytes.Split(configData, []byte{'\n'})

	var url string

	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)

		// Check for remote section
		if bytes.HasPrefix(trimmed, []byte("[remote ")) {
			// Reset URL when entering a new remote section
			url = ""
			continue
		}

		// Check for URL in current remote section
		if bytes.HasPrefix(trimmed, []byte("url = ")) {
			url = string(trimmed[6:])
			url = strings.TrimSpace(url)

			// Try to extract project info from URL
			pid, host, parseErr := parseGitLabURL(url)
			if parseErr != nil {
				// GitHub detection - return error immediately
				return "", "", parseErr
			}
			if pid != "" {
				return pid, host, nil
			}
		}
	}

	return "", "", nil
}

// isGitHubURL checks if the given URL is a GitHub repository URL
func isGitHubURL(url string) bool {
	lowerURL := strings.ToLower(url)
	githubPatterns := []string{
		"github.com",
		"git@github.com:",
		"https://github.com/",
		"http://github.com/",
		"ssh://git@github.com/",
	}

	for _, pattern := range githubPatterns {
		if strings.Contains(lowerURL, pattern) {
			return true
		}
	}
	return false
}

// parseGitLabURL extracts project ID and host from Git remote URL
// Supports both SSH and HTTPS URLs:
// - git@gitlab.com:owner/repo.git
// - https://gitlab.com/owner/repo.git
// - git@gitlab.example.com:owner/repo.git
// - https://gitlab.example.com:owner/repo.git
// Returns error if GitHub URL is detected
func parseGitLabURL(url string) (projectID, host string, err error) {
	// Check for GitHub URLs first - HARD ERROR
	if isGitHubURL(url) {
		return "", "", fmt.Errorf("GitHub repository detected: %s. This MCP server only supports GitLab repositories. Please use github-mcp-server instead", url)
	}

	// Try HTTPS format first
	if strings.HasPrefix(url, "https://") {
		// Remove protocol and .git suffix
		url = strings.TrimPrefix(url, "https://")
		url = strings.TrimSuffix(url, ".git")

		// Split by host and path
		parts := strings.SplitN(url, "/", 2)
		if len(parts) == 2 {
			host = "https://" + parts[0]
			projectID = parts[1]
			return projectID, host, nil
		}
	}

	// Try SSH format: git@host:path/repo.git
	if strings.HasPrefix(url, "git@") {
		// Remove git@ and .git suffix
		url = strings.TrimPrefix(url, "git@")
		url = strings.TrimSuffix(url, ".git")

		// Split by first : to get host and path
		parts := strings.SplitN(url, ":", 2)
		if len(parts) == 2 {
			host = "https://" + parts[0]
			projectID = parts[1]
			return projectID, host, nil
		}
	}

	return "", "", nil
}
