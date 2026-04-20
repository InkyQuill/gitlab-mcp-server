// Package install provides the MCP server installation command for IDEs.
package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetClaudeConfigPath tests the Claude Desktop config path generation.
func TestGetClaudeConfigPath(t *testing.T) {
	homeDir := "/home/user"

	// Note: These functions use runtime.GOOS internally, so we test the actual
	// behavior on the current platform. We verify the paths are well-formed.
	t.Run("returns valid path", func(t *testing.T) {
		result := getClaudeConfigPath(homeDir)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "Claude")
		assert.Contains(t, result, "claude_desktop_config.json")
	})

	t.Run("Linux format", func(t *testing.T) {
		// On Linux, expect .config/Claude format
		expected := filepath.Join(homeDir, ".config", "Claude", "claude_desktop_config.json")
		result := getClaudeConfigPath(homeDir)

		// The actual result depends on runtime.GOOS
		if runtime.GOOS == "linux" {
			assert.Equal(t, expected, result)
		}
	})
}

// TestGetVSCodeConfigPath tests the VS Code config path generation.
func TestGetVSCodeConfigPath(t *testing.T) {
	homeDir := "/home/user"

	t.Run("returns valid path", func(t *testing.T) {
		result := getVSCodeConfigPath(homeDir)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "settings.json")
	})

	t.Run("Linux format", func(t *testing.T) {
		expected := filepath.Join(homeDir, ".config", "Code", "User", "settings.json")
		result := getVSCodeConfigPath(homeDir)

		if runtime.GOOS == "linux" {
			assert.Equal(t, expected, result)
		}
	})
}

// TestGetCursorConfigPath tests the Cursor config path generation.
func TestGetCursorConfigPath(t *testing.T) {
	homeDir := "/home/user"

	t.Run("returns valid path", func(t *testing.T) {
		result := getCursorConfigPath(homeDir)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "settings.json")
	})

	t.Run("Linux format", func(t *testing.T) {
		expected := filepath.Join(homeDir, ".config", "Cursor", "User", "settings.json")
		result := getCursorConfigPath(homeDir)

		if runtime.GOOS == "linux" {
			assert.Equal(t, expected, result)
		}
	})
}

// TestMergeClaudeConfig tests merging config for Claude Desktop.
func TestMergeClaudeConfig(t *testing.T) {
	tests := []struct {
		name          string
		existing      map[string]interface{}
		installConfig *InstallConfig
		expected      map[string]interface{}
	}{
		{
			name:          "Empty config",
			existing:      nil,
			installConfig: &InstallConfig{Command: "/path/to/bin", Args: []string{"stdio"}},
			expected: map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"gitlab": map[string]interface{}{
						"command": "/path/to/bin",
						"args":    []interface{}{"stdio"},
					},
				},
			},
		},
		{
			name: "Existing config with other servers",
			existing: map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"other": map[string]interface{}{
						"command": "/other/bin",
						"args":    []interface{}{"stdio"},
					},
				},
			},
			installConfig: &InstallConfig{Command: "/path/to/bin", Args: []string{"stdio"}},
			expected: map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"other": map[string]interface{}{
						"command": "/other/bin",
						"args":    []interface{}{"stdio"},
					},
					"gitlab": map[string]interface{}{
						"command": "/path/to/bin",
						"args":    []interface{}{"stdio"},
					},
				},
			},
		},
		{
			name: "Existing config with gitlab server (overwrites)",
			existing: map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"gitlab": map[string]interface{}{
						"command": "/old/bin",
						"args":    []interface{}{"stdio"},
					},
				},
			},
			installConfig: &InstallConfig{Command: "/path/to/bin", Args: []string{"stdio"}},
			expected: map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"gitlab": map[string]interface{}{
						"command": "/path/to/bin",
						"args":    []interface{}{"stdio"},
					},
				},
			},
		},
		{
			name: "With environment variables",
			existing: map[string]interface{}{
				"mcpServers": map[string]interface{}{},
			},
			installConfig: &InstallConfig{
				Command: "/path/to/bin",
				Args:    []string{"stdio"},
				Env:     map[string]string{"GITLAB_CONFIG_PATH": "/custom/config.json"},
			},
			expected: map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"gitlab": map[string]interface{}{
						"command": "/path/to/bin",
						"args":    []interface{}{"stdio"},
						"env":     map[string]interface{}{"GITLAB_CONFIG_PATH": "/custom/config.json"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mergeClaudeConfig(tt.existing, tt.installConfig)
			require.NoError(t, err)

			// Marshal both to JSON for comparison since the types may differ
			// ([]string vs []interface{}) but produce valid JSON
			expectedJSON, err := json.Marshal(tt.expected)
			require.NoError(t, err)
			resultJSON, err := json.Marshal(result)
			require.NoError(t, err)
			assert.JSONEq(t, string(expectedJSON), string(resultJSON))
		})
	}
}

// TestMergeVSCodeConfig tests merging config for VS Code/Cursor.
func TestMergeVSCodeConfig(t *testing.T) {
	tests := []struct {
		name          string
		existing      map[string]interface{}
		installConfig *InstallConfig
		expected      map[string]interface{}
	}{
		{
			name:          "Empty config",
			existing:      nil,
			installConfig: &InstallConfig{Command: "/path/to/bin", Args: []string{"stdio"}},
			expected: map[string]interface{}{
				"mcp.servers": map[string]interface{}{
					"gitlab": map[string]interface{}{
						"command": "/path/to/bin",
						"args":    []interface{}{"stdio"},
					},
				},
			},
		},
		{
			name: "Existing config with other settings",
			existing: map[string]interface{}{
				"editor.formatOnSave": true,
				"mcp.servers": map[string]interface{}{
					"other": map[string]interface{}{
						"command": "/other/bin",
						"args":    []interface{}{"stdio"},
					},
				},
			},
			installConfig: &InstallConfig{Command: "/path/to/bin", Args: []string{"stdio"}},
			expected: map[string]interface{}{
				"editor.formatOnSave": true,
				"mcp.servers": map[string]interface{}{
					"other": map[string]interface{}{
						"command": "/other/bin",
						"args":    []interface{}{"stdio"},
					},
					"gitlab": map[string]interface{}{
						"command": "/path/to/bin",
						"args":    []interface{}{"stdio"},
					},
				},
			},
		},
		{
			name: "With environment variables",
			existing: map[string]interface{}{
				"mcp.servers": map[string]interface{}{},
			},
			installConfig: &InstallConfig{
				Command: "/path/to/bin",
				Args:    []string{"stdio"},
				Env:     map[string]string{"GITLAB_CONFIG_PATH": "/custom/config.json"},
			},
			expected: map[string]interface{}{
				"mcp.servers": map[string]interface{}{
					"gitlab": map[string]interface{}{
						"command": "/path/to/bin",
						"args":    []interface{}{"stdio"},
						"env":     map[string]interface{}{"GITLAB_CONFIG_PATH": "/custom/config.json"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mergeVSCodeConfig(tt.existing, tt.installConfig)
			require.NoError(t, err)

			// Marshal both to JSON for comparison since the types may differ
			// ([]string vs []interface{}) but produce valid JSON
			expectedJSON, err := json.Marshal(tt.expected)
			require.NoError(t, err)
			resultJSON, err := json.Marshal(result)
			require.NoError(t, err)
			assert.JSONEq(t, string(expectedJSON), string(resultJSON))
		})
	}
}

// TestCreateInstallConfig tests creating install configuration.
func TestCreateInstallConfig(t *testing.T) {
	tests := []struct {
		name       string
		binaryPath string
		configPath string
		expected   *InstallConfig
	}{
		{
			name:       "Without config path",
			binaryPath: "/usr/local/bin/gitlab-mcp-server",
			configPath: "",
			expected: &InstallConfig{
				Command: "/usr/local/bin/gitlab-mcp-server",
				Args:    []string{"stdio"},
				Env:     nil,
			},
		},
		{
			name:       "With config path",
			binaryPath: "/usr/local/bin/gitlab-mcp-server",
			configPath: "/custom/config.json",
			expected: &InstallConfig{
				Command: "/usr/local/bin/gitlab-mcp-server",
				Args:    []string{"stdio"},
				Env:     map[string]string{"GITLAB_CONFIG_PATH": "/custom/config.json"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createInstallConfig(tt.binaryPath, tt.configPath)
			assert.Equal(t, tt.expected.Command, result.Command)
			assert.Equal(t, tt.expected.Args, result.Args)
			assert.Equal(t, tt.expected.Env, result.Env)
		})
	}
}

// TestBackupManager tests the backup manager functionality.
func TestBackupManager(t *testing.T) {
	t.Run("Create and restore backup", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.json")
		backupFile := filepath.Join(tmpDir, "config.json.backup")

		// Create original config
		originalConfig := map[string]interface{}{
			"mcpServers": map[string]interface{}{
				"test": map[string]interface{}{
					"command": "/test/bin",
				},
			},
		}
		data, err := json.MarshalIndent(originalConfig, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(configFile, data, 0644))

		// Create backup
		bm := NewBackupManager(configFile)
		backupPath, err := bm.CreateBackup()
		require.NoError(t, err)
		assert.Equal(t, backupFile, backupPath)

		// Verify backup exists
		assert.FileExists(t, backupPath)

		// Modify original
		modifiedConfig := map[string]interface{}{
			"mcpServers": map[string]interface{}{
				"modified": map[string]interface{}{
					"command": "/modified/bin",
				},
			},
		}
		data, err = json.MarshalIndent(modifiedConfig, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(configFile, data, 0644))

		// Restore backup
		require.NoError(t, bm.RestoreBackup(backupPath))

		// Verify original content restored (compare to originalConfig, not modifiedConfig)
		originalData, err := json.MarshalIndent(originalConfig, "", "  ")
		require.NoError(t, err)
		result, err := os.ReadFile(configFile)
		require.NoError(t, err)
		assert.JSONEq(t, string(originalData), string(result))
	})

	t.Run("List backups", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.json")
		backupFile := filepath.Join(tmpDir, "config.json.backup")

		// Create config and backup
		require.NoError(t, os.WriteFile(configFile, []byte("{}"), 0644))
		require.NoError(t, os.WriteFile(backupFile, []byte("{}"), 0644))

		bm := NewBackupManager(configFile)
		backups, err := bm.ListBackups()
		require.NoError(t, err)
		assert.Len(t, backups, 1)
		assert.Equal(t, backupFile, backups[0])
	})

	t.Run("Remove backup", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.json")
		backupFile := filepath.Join(tmpDir, "config.json.backup")

		// Create config and backup
		require.NoError(t, os.WriteFile(configFile, []byte("{}"), 0644))
		require.NoError(t, os.WriteFile(backupFile, []byte("{}"), 0644))

		bm := NewBackupManager(configFile)
		require.NoError(t, bm.RemoveBackup(backupFile))

		// Verify backup removed
		assert.NoFileExists(t, backupFile)
	})
}

// TestConfigReader tests the config reader functionality.
func TestConfigReader(t *testing.T) {
	t.Run("Read non-existent file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "nonexistent.json")

		cr := NewConfigReader(IDE{
			Name:       IDEClaudeDesktop,
			ConfigPath: configPath,
		})

		config, err := cr.Read()
		require.NoError(t, err)
		assert.Nil(t, config)
	})

	t.Run("Read and write Claude config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "claude_desktop_config.json")

		cr := NewConfigReader(IDE{
			Name:       IDEClaudeDesktop,
			ConfigPath: configPath,
		})

		// Write config
		testConfig := map[string]interface{}{
			"mcpServers": map[string]interface{}{
				"test": map[string]interface{}{
					"command": "/test/bin",
					"args":    []interface{}{"stdio"},
				},
			},
		}
		require.NoError(t, cr.Write(testConfig))

		// Read config
		readConfig, err := cr.Read()
		require.NoError(t, err)
		assert.Equal(t, testConfig, readConfig)
	})

	t.Run("Read and write VS Code config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "settings.json")

		cr := NewConfigReader(IDE{
			Name:       IDEVSCode,
			ConfigPath: configPath,
		})

		// Write config
		testConfig := map[string]interface{}{
			"mcp.servers": map[string]interface{}{
				"test": map[string]interface{}{
					"command": "/test/bin",
					"args":    []interface{}{"stdio"},
				},
			},
		}
		require.NoError(t, cr.Write(testConfig))

		// Read config
		readConfig, err := cr.Read()
		require.NoError(t, err)
		assert.Equal(t, testConfig, readConfig)
	})

	t.Run("HasMCPServer - Claude Desktop", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "claude_desktop_config.json")

		cr := NewConfigReader(IDE{
			Name:       IDEClaudeDesktop,
			ConfigPath: configPath,
		})

		// Initially no servers
		has, err := cr.HasMCPServer("gitlab")
		require.NoError(t, err)
		assert.False(t, has)

		// Write config with servers
		testConfig := map[string]interface{}{
			"mcpServers": map[string]interface{}{
				"gitlab": map[string]interface{}{
					"command": "/gitlab/bin",
				},
				"other": map[string]interface{}{
					"command": "/other/bin",
				},
			},
		}
		require.NoError(t, cr.Write(testConfig))

		// Check for servers
		hasGitlab, err := cr.HasMCPServer("gitlab")
		require.NoError(t, err)
		assert.True(t, hasGitlab)

		hasOther, err := cr.HasMCPServer("other")
		require.NoError(t, err)
		assert.True(t, hasOther)

		hasMissing, err := cr.HasMCPServer("missing")
		require.NoError(t, err)
		assert.False(t, hasMissing)
	})

	t.Run("HasMCPServer - VS Code", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "settings.json")

		cr := NewConfigReader(IDE{
			Name:       IDEVSCode,
			ConfigPath: configPath,
		})

		// Write config with servers
		testConfig := map[string]interface{}{
			"mcp.servers": map[string]interface{}{
				"gitlab": map[string]interface{}{
					"command": "/gitlab/bin",
				},
			},
		}
		require.NoError(t, cr.Write(testConfig))

		// Check for servers
		has, err := cr.HasMCPServer("gitlab")
		require.NoError(t, err)
		assert.True(t, has)
	})

	t.Run("ListMCPServers", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "claude_desktop_config.json")

		cr := NewConfigReader(IDE{
			Name:       IDEClaudeDesktop,
			ConfigPath: configPath,
		})

		// Write config with multiple servers
		testConfig := map[string]interface{}{
			"mcpServers": map[string]interface{}{
				"gitlab":     map[string]interface{}{"command": "/gitlab/bin"},
				"github":     map[string]interface{}{"command": "/github/bin"},
				"filesystem": map[string]interface{}{"command": "/fs/bin"},
			},
		}
		require.NoError(t, cr.Write(testConfig))

		// List servers
		servers, err := cr.ListMCPServers()
		require.NoError(t, err)
		assert.Len(t, servers, 3)
		assert.Contains(t, servers, "gitlab")
		assert.Contains(t, servers, "github")
		assert.Contains(t, servers, "filesystem")
	})

	t.Run("ValidateConfig", func(t *testing.T) {
		cr := NewConfigReader(IDE{
			Name:       IDEClaudeDesktop,
			ConfigPath: "/tmp/test.json",
		})

		// Valid config
		validConfig := map[string]interface{}{
			"mcpServers": map[string]interface{}{
				"test": map[string]interface{}{"command": "/test/bin"},
			},
		}
		assert.NoError(t, cr.ValidateConfig(validConfig))

		// Config with unserializable data (channel)
		invalidConfig := map[string]interface{}{
			"mcpServers": map[string]interface{}{
				"test": make(chan int),
			},
		}
		assert.Error(t, cr.ValidateConfig(invalidConfig))
	})
}

// TestEnsureDir tests directory creation.
func TestEnsureDir(t *testing.T) {
	t.Run("Create new directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		newDir := filepath.Join(tmpDir, "new", "nested", "dir")

		require.NoError(t, ensureDir(newDir))
		assert.DirExists(t, newDir)
	})

	t.Run("Existing directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		require.NoError(t, ensureDir(tmpDir))
		assert.DirExists(t, tmpDir)
	})
}

// TestDetectAndSelectIDE tests IDE detection and selection.
func TestDetectAndSelectIDE(t *testing.T) {
	t.Run("Select Claude Desktop", func(t *testing.T) {
		opts := &InstallOptions{Editor: IDEClaudeDesktop}

		// Create a fake home directory with Claude config
		tmpDir := t.TempDir()
		claudeDir := filepath.Join(tmpDir, ".config", "Claude")
		require.NoError(t, os.MkdirAll(claudeDir, 0755))

		// Note: This test uses the actual home dir from the system,
		// so we can't fully test detection without mocking.
		// The test verifies the selection logic works correctly.
		_, ides, err := detectAndSelectIDE(opts)

		// We expect either success or "not detected" error
		if err != nil {
			assert.Contains(t, err.Error(), "not detected")
		} else {
			assert.NotEmpty(t, ides)
		}
	})

	t.Run("Select all", func(t *testing.T) {
		opts := &InstallOptions{Editor: IDEAll}

		id, ides, err := detectAndSelectIDE(opts)

		// May succeed if any IDEs are detected, or fail if none
		if err == nil {
			assert.Equal(t, "all IDEs", id.Name)
			assert.NotEmpty(t, ides)
		} else {
			assert.Contains(t, err.Error(), "no supported IDEs detected")
		}
	})

	t.Run("Invalid editor", func(t *testing.T) {
		opts := &InstallOptions{Editor: "invalid"}

		_, _, err := detectAndSelectIDE(opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown editor")
	})
}

// TestInstallForIDE tests the installation logic for a single IDE.
func TestInstallForIDE(t *testing.T) {
	t.Run("Install to new config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "claude_desktop_config.json")

		ide := IDE{
			Name:       IDEClaudeDesktop,
			ConfigPath: configPath,
		}

		installConfig := &InstallConfig{
			Command: "/usr/local/bin/gitlab-mcp-server",
			Args:    []string{"stdio"},
		}

		require.NoError(t, installForIDE(ide, installConfig, false))

		// Verify config was created
		data, err := os.ReadFile(configPath)
		require.NoError(t, err)

		var config map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &config))

		mcpServers, ok := config["mcpServers"].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, mcpServers, "gitlab")
	})

	t.Run("Install to existing config - preserves other servers", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "claude_desktop_config.json")

		// Create existing config with other servers
		existingConfig := map[string]interface{}{
			"mcpServers": map[string]interface{}{
				"other": map[string]interface{}{
					"command": "/other/bin",
					"args":    []interface{}{"stdio"},
				},
			},
		}
		data, err := json.MarshalIndent(existingConfig, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(configPath, data, 0644))

		ide := IDE{
			Name:       IDEClaudeDesktop,
			ConfigPath: configPath,
		}

		installConfig := &InstallConfig{
			Command: "/usr/local/bin/gitlab-mcp-server",
			Args:    []string{"stdio"},
		}

		require.NoError(t, installForIDE(ide, installConfig, false))

		// Verify both servers exist
		data, err = os.ReadFile(configPath)
		require.NoError(t, err)

		var config map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &config))

		mcpServers, ok := config["mcpServers"].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, mcpServers, "gitlab")
		assert.Contains(t, mcpServers, "other")
	})

	t.Run("Dry run", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "claude_desktop_config.json")

		ide := IDE{
			Name:       IDEClaudeDesktop,
			ConfigPath: configPath,
		}

		installConfig := &InstallConfig{
			Command: "/usr/local/bin/gitlab-mcp-server",
			Args:    []string{"stdio"},
		}

		require.NoError(t, installForIDE(ide, installConfig, true))

		// Verify config was NOT created
		_, err := os.Stat(configPath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("VS Code config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "settings.json")

		ide := IDE{
			Name:       IDEVSCode,
			ConfigPath: configPath,
		}

		installConfig := &InstallConfig{
			Command: "/usr/local/bin/gitlab-mcp-server",
			Args:    []string{"stdio"},
		}

		require.NoError(t, installForIDE(ide, installConfig, false))

		// Verify config was created
		data, err := os.ReadFile(configPath)
		require.NoError(t, err)

		var config map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &config))

		mcpServers, ok := config["mcp.servers"].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, mcpServers, "gitlab")
	})
}

// TestFormatIDEName tests the IDE name formatting.
func TestFormatIDEName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{IDEClaudeDesktop, "Claude Desktop"},
		{IDEVSCode, "Visual Studio Code"},
		{IDECursor, "Cursor"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatIDEName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCreateInstallConfig_DoesNotEmitTokenEnv is a regression test pinning
// that the installer never emits GITLAB_TOKEN/GITLAB_HOST env vars in
// generated IDE entries (v2.1 single-MCP-entry model).
func TestCreateInstallConfig_DoesNotEmitTokenEnv(t *testing.T) {
	cfg := createInstallConfig("/usr/local/bin/gitlab-mcp-server", "")
	if cfg.Env != nil {
		_, hasTok := cfg.Env["GITLAB_TOKEN"]
		_, hasHost := cfg.Env["GITLAB_HOST"]
		assert.Falsef(t, hasTok, "GITLAB_TOKEN must not be emitted: %v", cfg.Env)
		assert.Falsef(t, hasHost, "GITLAB_HOST must not be emitted: %v", cfg.Env)
	}
}

// TestCreateInstallConfig_EmitsConfigPathWhenGiven verifies that a custom
// GITLAB_CONFIG_PATH is properly emitted when provided.
func TestCreateInstallConfig_EmitsConfigPathWhenGiven(t *testing.T) {
	cfg := createInstallConfig("/usr/local/bin/gitlab-mcp-server", "/tmp/x.json")
	assert.Equal(t, "/tmp/x.json", cfg.Env["GITLAB_CONFIG_PATH"])
}
