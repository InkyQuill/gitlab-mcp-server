package installer

import (
	"fmt"
	"os"
	"path/filepath"
)

// BinaryConfig holds configuration for the MCP server binary
type BinaryConfig struct {
	Mode          string // "local" or "docker"
	LocalPath     string // Absolute path to local binary
	DockerImage   string // Docker image name
	Command       string // Command to run
	Args          []string
	Env           map[string]string
}

// GetBinaryConfig determines the binary configuration based on mode
func GetBinaryConfig(mode string, projectRoot string) (*BinaryConfig, error) {
	config := &BinaryConfig{
		Mode: mode,
		Env:  make(map[string]string),
	}

	switch mode {
	case "docker":
		config.DockerImage = "gitlab-mcp-server:latest"
		config.Command = "docker"
		config.Args = []string{
			"run",
			"-i",
			"--rm",
			"-e", "GITLAB_TOKEN",
			"-e", "GITLAB_HOST",
		}
		// GITLAB_READ_ONLY will be added conditionally
		config.Args = append(config.Args, config.DockerImage)
		return config, nil

	case "local":
		fallthrough
	default:
		// Use local binary
		localPath := filepath.Join(projectRoot, "bin", "gitlab-mcp-server")
		
		// Check if binary exists
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("binary not found at %s. Please run 'make build' first", localPath)
		}

		// Convert to absolute path
		absPath, err := filepath.Abs(localPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path: %w", err)
		}

		config.LocalPath = absPath
		config.Command = absPath
		config.Args = []string{"stdio"}
		return config, nil
	}
}

// AddReadOnlyEnv adds GITLAB_READ_ONLY environment variable if read-only mode is enabled
func (bc *BinaryConfig) AddReadOnlyEnv(readOnly bool) {
	if readOnly {
		if bc.Mode == "docker" {
			// Insert before the image name
			args := make([]string, 0, len(bc.Args)+2)
			args = append(args, bc.Args[:len(bc.Args)-1]...)
			args = append(args, "-e", "GITLAB_READ_ONLY")
			args = append(args, bc.Args[len(bc.Args)-1])
			bc.Args = args
		}
		bc.Env["GITLAB_READ_ONLY"] = "true"
	}
}

