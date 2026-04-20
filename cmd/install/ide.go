// Package install provides the MCP server installation command for IDEs.
package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	pathutil "github.com/InkyQuill/gitlab-mcp-server/pkg/config"
)

// IDEConfig represents the configuration for an IDE.
type IDEConfig struct {
	Name       string
	ConfigPath string
	ConfigType ConfigType
}

// ConfigType represents the type of configuration file.
type ConfigType int

const (
	// ConfigTypeClaudeDesktop is the Claude Desktop config type
	ConfigTypeClaudeDesktop ConfigType = iota
	// ConfigTypeVSCode is the VS Code/Cursor settings type
	ConfigTypeVSCode
)

// ConfigReader reads and parses IDE configuration files.
type ConfigReader struct {
	configPath string
	configType ConfigType
}

// NewConfigReader creates a new ConfigReader for the given IDE.
func NewConfigReader(ide IDE) *ConfigReader {
	var configType ConfigType
	switch ide.Name {
	case IDEClaudeDesktop:
		configType = ConfigTypeClaudeDesktop
	case IDEVSCode, IDECursor:
		configType = ConfigTypeVSCode
	}

	return &ConfigReader{
		configPath: ide.ConfigPath,
		configType: configType,
	}
}

// Read reads the configuration file and returns the parsed config.
// Returns nil if the file doesn't exist (not an error).
func (r *ConfigReader) Read() (map[string]interface{}, error) {
	data, err := os.ReadFile(r.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // File doesn't exist yet
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return config, nil
}

// Write writes the configuration to the file, creating a backup first.
func (r *ConfigReader) Write(cfg map[string]interface{}) error {
	// Validate config path to prevent directory traversal
	if err := pathutil.ValidatePath(r.configPath); err != nil {
		return fmt.Errorf("invalid config path: %w", err)
	}

	// Create directory if needed
	configDir := filepath.Dir(r.configPath)
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create backup if file exists
	if _, err := os.Stat(r.configPath); err == nil {
		backupPath := r.configPath + ".backup"
		if err := backupFile(r.configPath, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Marshal config with indentation
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config
	if err := os.WriteFile(r.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// HasMCPServer checks if the config already has an MCP server with the given name.
func (r *ConfigReader) HasMCPServer(serverName string) (bool, error) {
	config, err := r.Read()
	if err != nil {
		return false, err
	}
	if config == nil {
		return false, nil
	}

	var servers map[string]interface{}
	switch r.configType {
	case ConfigTypeClaudeDesktop:
		val, ok := config["mcpServers"]
		if !ok {
			return false, nil
		}
		servers, ok = val.(map[string]interface{})
		if !ok {
			return false, nil
		}
	case ConfigTypeVSCode:
		val, ok := config["mcp.servers"]
		if !ok {
			return false, nil
		}
		servers, ok = val.(map[string]interface{})
		if !ok {
			return false, nil
		}
	}

	_, exists := servers[serverName]
	return exists, nil
}

// ListMCPServers returns all MCP server names in the config.
func (r *ConfigReader) ListMCPServers() ([]string, error) {
	config, err := r.Read()
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}

	var servers map[string]interface{}
	switch r.configType {
	case ConfigTypeClaudeDesktop:
		val, ok := config["mcpServers"]
		if !ok {
			return nil, nil
		}
		servers, ok = val.(map[string]interface{})
		if !ok {
			return nil, nil
		}
	case ConfigTypeVSCode:
		val, ok := config["mcp.servers"]
		if !ok {
			return nil, nil
		}
		servers, ok = val.(map[string]interface{})
		if !ok {
			return nil, nil
		}
	}

	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	return names, nil
}

// ValidateConfig validates the configuration structure.
func (r *ConfigReader) ValidateConfig(config map[string]interface{}) error {
	// Basic validation: ensure it can be marshaled to JSON
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("config is not valid JSON-serializable: %w", err)
	}

	// Try to unmarshal back to verify it's valid JSON
	var verify map[string]interface{}
	if err := json.Unmarshal(data, &verify); err != nil {
		return fmt.Errorf("config produces invalid JSON: %w", err)
	}

	return nil
}

// GetConfigPath returns the configuration file path.
func (r *ConfigReader) GetConfigPath() string {
	return r.configPath
}

// GetConfigType returns the configuration type.
func (r *ConfigReader) GetConfigType() ConfigType {
	return r.configType
}

// BackupManager handles backup operations for configuration files.
type BackupManager struct {
	configPath string
}

// NewBackupManager creates a new BackupManager for the given config path.
func NewBackupManager(configPath string) *BackupManager {
	return &BackupManager{configPath: configPath}
}

// CreateBackup creates a backup of the configuration file.
// Returns the path to the backup file.
func (b *BackupManager) CreateBackup() (string, error) {
	if _, err := os.Stat(b.configPath); os.IsNotExist(err) {
		return "", nil // No file to backup
	}

	backupPath := b.configPath + ".backup"
	if err := backupFile(b.configPath, backupPath); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	return backupPath, nil
}

// RestoreBackup restores the configuration from a backup.
func (b *BackupManager) RestoreBackup(backupPath string) error {
	// Validate paths to prevent directory traversal
	if err := pathutil.ValidatePath(b.configPath); err != nil {
		return fmt.Errorf("invalid config path: %w", err)
	}
	if err := pathutil.ValidatePath(backupPath); err != nil {
		return fmt.Errorf("invalid backup path: %w", err)
	}

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	if err := os.WriteFile(b.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	return nil
}

// ListBackups returns all backup files for the configuration.
func (b *BackupManager) ListBackups() ([]string, error) {
	configDir := filepath.Dir(b.configPath)
	configBase := filepath.Base(b.configPath)

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	var backups []string
	for _, entry := range entries {
		name := entry.Name()
		// Check if it's a backup of our config file
		if name == configBase+".backup" {
			backups = append(backups, filepath.Join(configDir, name))
		}
	}

	return backups, nil
}

// RemoveBackup removes a backup file.
func (b *BackupManager) RemoveBackup(backupPath string) error {
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("failed to remove backup: %w", err)
	}
	return nil
}

// IDEDetector detects installed IDEs on the system.
type IDEDetector struct {
	homeDir string
}

// NewIDEDetector creates a new IDEDetector.
func NewIDEDetector() (*IDEDetector, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	return &IDEDetector{homeDir: homeDir}, nil
}

// Detect returns all supported IDEs with their installation status.
func (d *IDEDetector) Detect() []IDE {
	return []IDE{
		{
			Name:       IDEClaudeDesktop,
			ConfigPath: getClaudeConfigPath(d.homeDir),
			Enabled:    d.isClaudeInstalled(),
		},
		{
			Name:       IDEVSCode,
			ConfigPath: getVSCodeConfigPath(d.homeDir),
			Enabled:    d.isVSCodeInstalled(),
		},
		{
			Name:       IDECursor,
			ConfigPath: getCursorConfigPath(d.homeDir),
			Enabled:    d.isCursorInstalled(),
		},
	}
}

// isClaudeInstalled checks if Claude Desktop is installed.
func (d *IDEDetector) isClaudeInstalled() bool {
	configPath := getClaudeConfigPath(d.homeDir)
	configDir := filepath.Dir(configPath)
	info, err := os.Stat(configDir)
	return err == nil && info.IsDir()
}

// isVSCodeInstalled checks if VS Code is installed.
func (d *IDEDetector) isVSCodeInstalled() bool {
	configPath := getVSCodeConfigPath(d.homeDir)
	configDir := filepath.Dir(configPath)
	info, err := os.Stat(configDir)
	return err == nil && info.IsDir()
}

// isCursorInstalled checks if Cursor is installed.
func (d *IDEDetector) isCursorInstalled() bool {
	configPath := getCursorConfigPath(d.homeDir)
	configDir := filepath.Dir(configPath)
	info, err := os.Stat(configDir)
	return err == nil && info.IsDir()
}

// GetDetected returns only the IDEs that are detected as installed.
func (d *IDEDetector) GetDetected() []IDE {
	var detected []IDE
	for _, ide := range d.Detect() {
		if ide.Enabled {
			detected = append(detected, ide)
		}
	}
	return detected
}
