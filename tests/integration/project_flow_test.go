package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/gitlab"
)

func TestProjectConfigFlow(t *testing.T) {
	t.Run("create and find project config", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		// Change to temp directory
		os.Chdir(tmpDir)

		// Create .gmcprc in tmpDir
		pc := &gitlab.ProjectConfig{
			ProjectID:  "owner/repo",
			GitLabHost: "https://gitlab.com",
			TokenName:  "test",
		}

		configPath, err := gitlab.WriteProjectConfig(tmpDir, pc)
		if err != nil {
			t.Fatalf("WriteProjectConfig() error = %v", err)
		}

		// Verify file was created at expected location
		expectedPath := filepath.Join(tmpDir, gitlab.ConfigFileName)
		if configPath != expectedPath {
			t.Errorf("config path = %s, want %s", configPath, expectedPath)
		}

		// Check that file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Fatalf("config file not created at %s", configPath)
		}

		// Find config from current directory
		cfg, foundPath, err := gitlab.FindProjectConfig()
		if err != nil {
			t.Fatalf("FindProjectConfig() error = %v", err)
		}

		if cfg == nil {
			t.Fatal("config not found")
		}

		if foundPath != configPath {
			t.Errorf("found path = %s, want %s", foundPath, configPath)
		}

		// Verify content
		if cfg.ProjectID != "owner/repo" {
			t.Errorf("expected ProjectID 'owner/repo', got '%s'", cfg.ProjectID)
		}

		if cfg.GitLabHost != "https://gitlab.com" {
			t.Errorf("expected GitLabHost 'https://gitlab.com', got '%s'", cfg.GitLabHost)
		}

		if cfg.TokenName != "test" {
			t.Errorf("expected TokenName 'test', got '%s'", cfg.TokenName)
		}
	})

	t.Run("search in parent directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		// Create .gmcprc in tmpDir (parent)
		pc := &gitlab.ProjectConfig{
			ProjectID: "parent/repo",
		}
		_, err := gitlab.WriteProjectConfig(tmpDir, pc)
		if err != nil {
			t.Fatalf("WriteProjectConfig() error = %v", err)
		}

		// Create nested subdirectories
		subDir := filepath.Join(tmpDir, "level1", "level2", "level3")
		err = os.MkdirAll(subDir, 0755)
		if err != nil {
			t.Fatalf("failed to create subdirectories: %v", err)
		}

		// Change to nested directory
		os.Chdir(subDir)

		// Find config from nested directory
		cfg, foundPath, err := gitlab.FindProjectConfig()
		if err != nil {
			t.Fatalf("FindProjectConfig() error = %v", err)
		}

		if cfg == nil {
			t.Fatal("config not found in parent directory")
		}

		// Verify it found the parent config
		expectedPath := filepath.Join(tmpDir, gitlab.ConfigFileName)
		if foundPath != expectedPath {
			t.Errorf("found path = %s, want %s", foundPath, expectedPath)
		}

		if cfg.ProjectID != "parent/repo" {
			t.Errorf("expected ProjectID 'parent/repo', got '%s'", cfg.ProjectID)
		}
	})

	t.Run("config not found returns nil without error", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		os.Chdir(tmpDir)

		// Don't create any config file
		cfg, foundPath, err := gitlab.FindProjectConfig()
		if err != nil {
			t.Fatalf("FindProjectConfig() unexpected error = %v", err)
		}

		if cfg != nil {
			t.Error("expected nil config when not found")
		}

		if foundPath != "" {
			t.Errorf("expected empty path, got '%s'", foundPath)
		}
	})

	t.Run("nearest config takes precedence", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		// Create parent config
		parentPC := &gitlab.ProjectConfig{
			ProjectID: "parent/repo",
		}
		gitlab.WriteProjectConfig(tmpDir, parentPC)

		// Create subdirectory with its own config
		subDir := filepath.Join(tmpDir, "subdir")
		os.Mkdir(subDir, 0755)

		subPC := &gitlab.ProjectConfig{
			ProjectID: "child/repo",
		}
		gitlab.WriteProjectConfig(subDir, subPC)

		// Change to subdirectory
		os.Chdir(subDir)

		// Should find child config, not parent
		cfg, foundPath, err := gitlab.FindProjectConfig()
		if err != nil {
			t.Fatalf("FindProjectConfig() error = %v", err)
		}

		if cfg == nil {
			t.Fatal("config not found")
		}

		if cfg.ProjectID != "child/repo" {
			t.Errorf("expected ProjectID 'child/repo', got '%s'", cfg.ProjectID)
		}

		// Verify it's the child config path
		expectedPath := filepath.Join(subDir, gitlab.ConfigFileName)
		if foundPath != expectedPath {
			t.Errorf("found path = %s, want %s", foundPath, expectedPath)
		}
	})

	t.Run("update existing config", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		os.Chdir(tmpDir)

		// Create initial config
		pc := &gitlab.ProjectConfig{
			ProjectID:  "original/repo",
			GitLabHost: "https://gitlab.com",
			TokenName:  "old-token",
		}

		gitlab.WriteProjectConfig(tmpDir, pc)

		// Update the config
		updatedPC := &gitlab.ProjectConfig{
			ProjectID:  "updated/repo",
			GitLabHost: "https://gitlab.example.com",
			TokenName:  "new-token",
		}

		_, err := gitlab.WriteProjectConfig(tmpDir, updatedPC)
		if err != nil {
			t.Fatalf("WriteProjectConfig() update error = %v", err)
		}

		// Read back and verify
		cfg, _, err := gitlab.FindProjectConfig()
		if err != nil {
			t.Fatalf("FindProjectConfig() error = %v", err)
		}
		if cfg.ProjectID != "updated/repo" {
			t.Errorf("ProjectID = %s, want 'updated/repo'", cfg.ProjectID)
		}
	})

	t.Run("malformed config returns error", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Write invalid JSON
		configPath := filepath.Join(tmpDir, gitlab.ConfigFileName)
		err := os.WriteFile(configPath, []byte("{invalid json"), 0644)
		if err != nil {
			t.Fatalf("failed to write invalid config: %v", err)
		}

		// Try to find it - it should find the file but fail to parse
		originalWd, _ := os.Getwd()
		os.Chdir(tmpDir)
		_, _, err = gitlab.FindProjectConfig()
		os.Chdir(originalWd)

		if err == nil {
			t.Error("expected error for malformed config")
		}
	})
}
