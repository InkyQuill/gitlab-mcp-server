package installer

// ServerConfig represents the MCP server configuration
type ServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Type    string            `json:"type,omitempty"` // For Claude Code
}

// VSCodeConfig represents VS Code MCP configuration structure
type VSCodeConfig struct {
	MCP struct {
		Servers map[string]ServerConfig `json:"servers"`
	} `json:"mcp"`
}

// ClaudeConfig represents Claude Desktop/Code/Cursor MCP configuration structure
type ClaudeConfig struct {
	MCPServers map[string]ServerConfig `json:"mcpServers"`
}

// CreateServerConfig creates a ServerConfig from BinaryConfig
func CreateServerConfig(bc *BinaryConfig, gitlabHost string, readOnly bool, token string) ServerConfig {
	config := ServerConfig{
		Command: bc.Command,
		Args:    make([]string, len(bc.Args)),
		Env:     make(map[string]string),
	}

	copy(config.Args, bc.Args)

	// Copy environment variables from binary config
	for k, v := range bc.Env {
		config.Env[k] = v
	}

	// Add GitLab-specific environment variables
	// For Docker mode, token is passed via env in the config
	// For local mode, token is also in env
	config.Env["GITLAB_TOKEN"] = token
	if gitlabHost != "" {
		config.Env["GITLAB_HOST"] = gitlabHost
	}
	// If gitlabHost is empty, GITLAB_HOST is not set
	// The server will use gitlab.com as default

	if readOnly {
		config.Env["GITLAB_READ_ONLY"] = "true"
	}

	return config
}

