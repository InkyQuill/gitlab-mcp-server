package integration

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/InkyQuill/gitlab-mcp-server/cmd/install"
)

func TestInstallFlow(t *testing.T) {
	t.Run("backup and restore config", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create existing config
		configDir := filepath.Join(tmpDir, "claude")
		err := os.MkdirAll(configDir, 0750)
		if err != nil {
			t.Fatalf("failed to create config dir: %v", err)
		}

		configPath := filepath.Join(configDir, "claude_desktop_config.json")
		existingConfig := map[string]interface{}{
			"mcpServers": map[string]interface{}{
				"existing": map[string]interface{}{
					"command": "/existing/path",
					"args":    []string{"serve"},
				},
			},
		}

		data, err := json.MarshalIndent(existingConfig, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal config: %v", err)
		}

		err = os.WriteFile(configPath, data, 0600)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Create backup using the backup helper
		backupPath := configPath + ".backup"
		err = backupFile(configPath, backupPath)
		if err != nil {
			t.Fatalf("backupFile() error = %v", err)
		}

		// Verify backup exists
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			t.Fatalf("backup not created at %s", backupPath)
		}

		// Modify the original config
		modifiedConfig := map[string]interface{}{
			"mcpServers": map[string]interface{}{
				"existing": map[string]interface{}{
					"command": "/existing/path",
					"args":    []string{"serve"},
				},
				"new": map[string]interface{}{
					"command": "/new/path",
					"args":    []string{"serve"},
				},
			},
		}

		data, _ = json.MarshalIndent(modifiedConfig, "", "  ")
		os.WriteFile(configPath, data, 0600)

		// Restore from backup
		err = restoreBackup(backupPath, configPath)
		if err != nil {
			t.Fatalf("restoreBackup() error = %v", err)
		}

		// Verify restored content matches original
		restored, _ := os.ReadFile(configPath)
		var restoredConfig map[string]interface{}
		err = json.Unmarshal(restored, &restoredConfig)
		if err != nil {
			t.Fatalf("failed to parse restored config: %v", err)
		}

		servers := restoredConfig["mcpServers"].(map[string]interface{})
		if _, ok := servers["existing"]; !ok {
			t.Error("existing server not restored")
		}

		if _, ok := servers["new"]; ok {
			t.Error("new server should not exist after restore")
		}
	})

	t.Run("merge claude config without existing", func(t *testing.T) {
		// No existing config
		installConfig := &install.InstallConfig{
			Command: "/path/to/gitlab-mcp-server",
			Args:    []string{"stdio"},
		}

		// Merge with nil existing config
		merged, err := mergeClaudeConfigHelper(nil, installConfig)
		if err != nil {
			t.Fatalf("mergeClaudeConfigHelper() error = %v", err)
		}

		// Verify structure
		servers, ok := merged["mcpServers"]
		if !ok {
			t.Fatal("mcpServers not in merged config")
		}

		serversMap, ok := servers.(map[string]interface{})
		if !ok {
			t.Fatal("mcpServers is not a map")
		}

		gitlabServer, ok := serversMap["gitlab"]
		if !ok {
			t.Fatal("gitlab server not in mcpServers")
		}

		gitlabMap, ok := gitlabServer.(map[string]interface{})
		if !ok {
			t.Fatal("gitlab server is not a map")
		}

		if gitlabMap["command"] != "/path/to/gitlab-mcp-server" {
			t.Errorf("command = %v, want '/path/to/gitlab-mcp-server'", gitlabMap["command"])
		}
	})

	t.Run("merge claude config with existing", func(t *testing.T) {
		installConfig := &install.InstallConfig{
			Command: "/path/to/gitlab-mcp-server",
			Args:    []string{"stdio"},
		}

		// Existing config with other servers
		existing := map[string]interface{}{
			"mcpServers": map[string]interface{}{
				"other-server": map[string]interface{}{
					"command": "/other/path",
					"args":    []string{"serve"},
				},
			},
		}

		merged, err := mergeClaudeConfigHelper(existing, installConfig)
		if err != nil {
			t.Fatalf("mergeClaudeConfigHelper() error = %v", err)
		}

		servers := merged["mcpServers"].(map[string]interface{})

		// Should have both servers
		if _, ok := servers["other-server"]; !ok {
			t.Error("other-server should be preserved")
		}

		if _, ok := servers["gitlab"]; !ok {
			t.Error("gitlab server should be added")
		}
	})

	t.Run("merge claude config with existing gitlab server", func(t *testing.T) {
		installConfig := &install.InstallConfig{
			Command: "/new/path/to/gitlab-mcp-server",
			Args:    []string{"stdio"},
		}

		// Existing config with gitlab server
		existing := map[string]interface{}{
			"mcpServers": map[string]interface{}{
				"gitlab": map[string]interface{}{
					"command": "/old/path",
					"args":    []string{"serve"},
				},
			},
		}

		merged, err := mergeClaudeConfigHelper(existing, installConfig)
		if err != nil {
			t.Fatalf("mergeClaudeConfigHelper() error = %v", err)
		}

		servers := merged["mcpServers"].(map[string]interface{})
		gitlabServer := servers["gitlab"].(map[string]interface{})

		// Should be overwritten with new config
		if gitlabServer["command"] != "/new/path/to/gitlab-mcp-server" {
			t.Errorf("command = %v, want '/new/path/to/gitlab-mcp-server'", gitlabServer["command"])
		}

		// Check args - could be []string or []interface{}
		args := gitlabServer["args"]
		argsSlice, ok := args.([]string)
		if !ok {
			// Try as []interface{}
			argsInterface, ok := args.([]interface{})
			if !ok {
				t.Fatalf("args is neither []string nor []interface{}: %T", args)
			}
			if len(argsInterface) != 1 || argsInterface[0] != "stdio" {
				t.Errorf("args = %v, want ['stdio']", argsInterface)
			}
		} else {
			if len(argsSlice) != 1 || argsSlice[0] != "stdio" {
				t.Errorf("args = %v, want ['stdio']", argsSlice)
			}
		}
	})

	t.Run("merge vscode config", func(t *testing.T) {
		installConfig := &install.InstallConfig{
			Command: "/path/to/gitlab-mcp-server",
			Args:    []string{"stdio"},
		}

		// VS Code uses different structure
		existing := map[string]interface{}{
			"mcp.servers": map[string]interface{}{
				"other": map[string]interface{}{
					"command": "/other/path",
				},
			},
		}

		merged, err := mergeVSCodeConfigHelper(existing, installConfig)
		if err != nil {
			t.Fatalf("mergeVSCodeConfigHelper() error = %v", err)
		}

		servers := merged["mcp.servers"].(map[string]interface{})

		if _, ok := servers["other"]; !ok {
			t.Error("other server should be preserved")
		}

		if _, ok := servers["gitlab"]; !ok {
			t.Error("gitlab server should be added")
		}
	})

	t.Run("install config with environment variables", func(t *testing.T) {
		tmpDir := t.TempDir()

		installConfig := &install.InstallConfig{
			Command: "/path/to/gitlab-mcp-server",
			Args:    []string{"stdio"},
			Env: map[string]string{
				"CUSTOM_CONFIG_PATH": tmpDir,
			},
		}

		// Merge into config
		merged, err := mergeClaudeConfigHelper(nil, installConfig)
		if err != nil {
			t.Fatalf("mergeClaudeConfigHelper() error = %v", err)
		}

		servers := merged["mcpServers"].(map[string]interface{})
		gitlabServer := servers["gitlab"].(map[string]interface{})

		// Check env is present
		env, ok := gitlabServer["env"]
		if !ok {
			t.Fatal("env not present in server config")
		}

		envMap, ok := env.(map[string]string)
		if !ok {
			t.Fatal("env is not a map[string]string")
		}

		if envMap["CUSTOM_CONFIG_PATH"] != tmpDir {
			t.Errorf("env[CUSTOM_CONFIG_PATH] = %s, want %s", envMap["CUSTOM_CONFIG_PATH"], tmpDir)
		}
	})
}

// Helper functions for testing
// These mirror the internal functions from cmd/install

func backupFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func restoreBackup(backupPath, configPath string) error {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

func mergeClaudeConfigHelper(existing map[string]interface{}, installConfig *install.InstallConfig) (map[string]interface{}, error) {
	if existing == nil {
		existing = make(map[string]interface{})
	}

	var mcpServers map[string]interface{}
	if val, ok := existing["mcpServers"]; ok {
		if servers, ok := val.(map[string]interface{}); ok {
			mcpServers = servers
		} else {
			return nil, errors.New("existing mcpServers is not a valid object")
		}
	} else {
		mcpServers = make(map[string]interface{})
	}

	gitlabServer := map[string]interface{}{
		"command": installConfig.Command,
		"args":    installConfig.Args,
	}
	if len(installConfig.Env) > 0 {
		gitlabServer["env"] = installConfig.Env
	}

	mcpServers["gitlab"] = gitlabServer
	existing["mcpServers"] = mcpServers
	return existing, nil
}

func mergeVSCodeConfigHelper(existing map[string]interface{}, installConfig *install.InstallConfig) (map[string]interface{}, error) {
	if existing == nil {
		existing = make(map[string]interface{})
	}

	var mcpServers map[string]interface{}
	if val, ok := existing["mcp.servers"]; ok {
		if servers, ok := val.(map[string]interface{}); ok {
			mcpServers = servers
		} else {
			return nil, errors.New("existing mcp.servers is not a valid object")
		}
	} else {
		mcpServers = make(map[string]interface{})
	}

	serverConfig := map[string]interface{}{
		"command": installConfig.Command,
		"args":    installConfig.Args,
	}
	if len(installConfig.Env) > 0 {
		serverConfig["env"] = installConfig.Env
	}

	mcpServers["gitlab"] = serverConfig
	existing["mcp.servers"] = mcpServers
	return existing, nil
}

func TestBackupFile(t *testing.T) {
	t.Run("backup existing file", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create source file
		srcPath := filepath.Join(tmpDir, "source.txt")
		dstPath := filepath.Join(tmpDir, "backup.txt")

		content := []byte("test content")
		err := os.WriteFile(srcPath, content, 0644)
		if err != nil {
			t.Fatalf("failed to create source file: %v", err)
		}

		// Backup
		err = backupFile(srcPath, dstPath)
		if err != nil {
			t.Fatalf("backupFile() error = %v", err)
		}

		// Verify backup exists
		backupContent, err := os.ReadFile(dstPath)
		if err != nil {
			t.Fatalf("failed to read backup: %v", err)
		}

		if string(backupContent) != "test content" {
			t.Errorf("backup content = %s, want 'test content'", string(backupContent))
		}
	})

	t.Run("backup non-existent file returns error", func(t *testing.T) {
		tmpDir := t.TempDir()

		srcPath := filepath.Join(tmpDir, "nonexistent.txt")
		dstPath := filepath.Join(tmpDir, "backup.txt")

		err := backupFile(srcPath, dstPath)
		if err == nil {
			t.Error("expected error for non-existent source file")
		}
	})
}

func TestRestoreBackup(t *testing.T) {
	t.Run("restore from backup", func(t *testing.T) {
		tmpDir := t.TempDir()

		backupPath := filepath.Join(tmpDir, "backup.txt")
		configPath := filepath.Join(tmpDir, "config.txt")

		// Create backup
		backupContent := []byte("original content")
		err := os.WriteFile(backupPath, backupContent, 0644)
		if err != nil {
			t.Fatalf("failed to create backup: %v", err)
		}

		// Create different config file
		configContent := []byte("modified content")
		err = os.WriteFile(configPath, configContent, 0644)
		if err != nil {
			t.Fatalf("failed to create config: %v", err)
		}

		// Restore
		err = restoreBackup(backupPath, configPath)
		if err != nil {
			t.Fatalf("restoreBackup() error = %v", err)
		}

		// Verify restore
		result, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("failed to read restored file: %v", err)
		}

		if string(result) != "original content" {
			t.Errorf("restored content = %s, want 'original content'", string(result))
		}
	})
}
