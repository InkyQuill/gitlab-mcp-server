package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/gitlab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()

	assert.Equal(t, "project", cmd.Use)
	assert.Equal(t, "Manage .gmcprc project configuration", cmd.Short)
	assert.Equal(t, 3, len(cmd.Commands()))

	// Check subcommands exist
	subcommands := cmd.Commands()
	commandNames := make([]string, len(subcommands))
	for i, subcmd := range subcommands {
		commandNames[i] = subcmd.Name()
	}

	assert.Contains(t, commandNames, "init")
	assert.Contains(t, commandNames, "detect")
	assert.Contains(t, commandNames, "status")
}

func TestWriteConfig(t *testing.T) {
	tmpDir := t.TempDir()

	config := &gitlab.ProjectConfig{
		ProjectID:  "test/project",
		GitLabHost: "https://gitlab.example.com",
		TokenName:  "test-token",
	}

	configPath, err := writeConfig(tmpDir, config)
	require.NoError(t, err)

	expectedPath := filepath.Join(tmpDir, ".gmcprc")
	assert.Equal(t, expectedPath, configPath)

	// Verify file exists
	_, err = os.Stat(configPath)
	require.NoError(t, err)
}

func TestDetectFromGit(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	err := os.Mkdir(gitDir, 0755)
	require.NoError(t, err)

	// Create git config with GitLab remote
	configContent := `[remote "origin"]
	url = https://gitlab.com/owner/repo.git
`
	configPath := filepath.Join(gitDir, "config")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Change to test directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Run detect
	projectID, gitlabHost, err := detectFromGit()
	require.NoError(t, err)
	assert.Equal(t, "owner/repo", projectID)
	assert.Equal(t, "https://gitlab.com", gitlabHost)
}

func TestDetectFromGit_SelfHosted(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	err := os.Mkdir(gitDir, 0755)
	require.NoError(t, err)

	// Create git config with self-hosted GitLab remote
	configContent := `[remote "origin"]
	url = git@gitlab.example.com:group/project.git
`
	configPath := filepath.Join(gitDir, "config")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Change to test directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Run detect
	projectID, gitlabHost, err := detectFromGit()
	require.NoError(t, err)
	assert.Equal(t, "group/project", projectID)
	assert.Equal(t, "https://gitlab.example.com", gitlabHost)
}

func TestDetectFromGit_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to non-git directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Run detect - should fail
	_, _, err = detectFromGit()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a Git repository")
}

func TestDetectFromGit_GitHubRemote(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	err := os.Mkdir(gitDir, 0755)
	require.NoError(t, err)

	// Create git config with GitHub remote
	configContent := `[remote "origin"]
	url = https://github.com/owner/repo.git
`
	configPath := filepath.Join(gitDir, "config")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Change to test directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Run detect - should fail with GitHub error
	_, _, err = detectFromGit()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub repository detected")
}

func TestReadConfig_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to empty directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Run read - should return nil without error
	config, path, err := readConfig()
	assert.NoError(t, err)
	assert.Nil(t, config)
	assert.Empty(t, path)
}

func TestReadConfig_Found(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config file
	configPath := filepath.Join(tmpDir, ".gmcprc")
	content := `{
  "projectId": "test/project",
  "gitlabHost": "https://gitlab.example.com",
  "tokenName": "work-token"
}`
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	// Change to test directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Run read
	config, path, err := readConfig()
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Contains(t, path, ".gmcprc")
	assert.Equal(t, "test/project", config.ProjectID)
	assert.Equal(t, "https://gitlab.example.com", config.GitLabHost)
	assert.Equal(t, "work-token", config.TokenName)
}

func TestReadConfig_FoundInParent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config in parent
	configPath := filepath.Join(tmpDir, ".gmcprc")
	content := `{"projectId": "parent/project"}`
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "subdir", "nested")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	// Change to subdirectory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	err = os.Chdir(subDir)
	require.NoError(t, err)

	// Run read - should find parent config
	config, path, err := readConfig()
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "parent/project", config.ProjectID)
	assert.Contains(t, path, ".gmcprc")
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			maxLen:   10,
			expected: "(none)",
		},
		{
			name:     "Short string",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "Exact length",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "Too long",
			input:    "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "Way too long",
			input:    "this is a very long string that needs truncation",
			maxLen:   20,
			expected: "this is a very lo...",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := truncateString(tc.input, tc.maxLen)
			assert.Equal(t, tc.expected, result)
		})
	}
}
