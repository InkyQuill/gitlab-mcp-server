package project

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	pkgConfig "github.com/InkyQuill/gitlab-mcp-server/pkg/config"
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

func TestDetectFromGit_PrefersOriginWithMultipleGitLabRemotes(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	require.NoError(t, os.Mkdir(gitDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "config"), []byte(`[remote "upstream"]
	url = https://gitlab.com/group/upstream.git
[remote "origin"]
	url = https://gitlab.com/group/repo.git
`), 0644))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	projectID, gitlabHost, err := detectFromGit()
	require.NoError(t, err)
	assert.Equal(t, "group/repo", projectID)
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

func TestInitCommand_UsesConfiguredHostToSelectRemote(t *testing.T) {
	resetInitFlags(t)
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	repoDir := filepath.Join(tmpDir, "repo")
	t.Setenv("HOME", homeDir)

	mgr, err := pkgConfig.NewManager("")
	require.NoError(t, err)
	require.NoError(t, mgr.AddServer(&pkgConfig.ServerConfig{
		Name:  "work",
		Host:  "https://gitlab.example.com",
		Token: "test-token",
	}))
	require.NoError(t, mgr.Save())

	gitDir := filepath.Join(repoDir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "config"), []byte(`[remote "origin"]
	url = https://gitlab.com/group/public.git
[remote "upstream"]
	url = git@gitlab.example.com:work/repo.git
`), 0644))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(repoDir))

	var out bytes.Buffer
	cmd := NewCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"init"})
	require.NoError(t, cmd.Execute())

	data, err := os.ReadFile(filepath.Join(repoDir, ".gmcprc"))
	require.NoError(t, err)
	var cfg gitlab.ProjectConfig
	require.NoError(t, json.Unmarshal(data, &cfg))
	assert.Equal(t, "work/repo", cfg.ProjectID)
	assert.Equal(t, "work", cfg.Server)
	assert.Contains(t, out.String(), `Matched server "work" from configured host https://gitlab.example.com.`)
}

func TestInitCommand_ExplicitProjectSurfacesConfigLoadError(t *testing.T) {
	resetInitFlags(t)
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	repoDir := filepath.Join(tmpDir, "repo")
	t.Setenv("HOME", homeDir)

	configDir := filepath.Join(homeDir, pkgConfig.ConfigDir)
	require.NoError(t, os.MkdirAll(configDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, pkgConfig.ConfigFileName), []byte(`{`), 0600))
	require.NoError(t, os.Mkdir(repoDir, 0755))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(repoDir))

	var out bytes.Buffer
	cmd := NewCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"init", "group/repo", "--host", "https://gitlab.example.com"})
	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load GitLab config for server inference")
	assert.NotContains(t, err.Error(), "no --server specified")
}

func TestInitCommand_NormalizesConfiguredHostForExplicitHost(t *testing.T) {
	resetInitFlags(t)
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	repoDir := filepath.Join(tmpDir, "repo")
	t.Setenv("HOME", homeDir)

	mgr, err := pkgConfig.NewManager("")
	require.NoError(t, err)
	require.NoError(t, mgr.AddServer(&pkgConfig.ServerConfig{
		Name:  "work",
		Host:  "https://gitlab.example.com",
		Token: "test-token",
	}))
	require.NoError(t, mgr.Save())
	require.NoError(t, os.Mkdir(repoDir, 0755))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(repoDir))

	var out bytes.Buffer
	cmd := NewCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"init", "group/repo", "--host", "gitlab.example.com"})
	require.NoError(t, cmd.Execute())

	data, err := os.ReadFile(filepath.Join(repoDir, ".gmcprc"))
	require.NoError(t, err)
	var cfg gitlab.ProjectConfig
	require.NoError(t, json.Unmarshal(data, &cfg))
	assert.Equal(t, "group/repo", cfg.ProjectID)
	assert.Equal(t, "work", cfg.Server)
	assert.Contains(t, out.String(), `Matched server "work" from configured host gitlab.example.com.`)
}

func TestDetectCommand_PrintsSelectedRemote(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	require.NoError(t, os.Mkdir(gitDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "config"), []byte(`[remote "upstream"]
	url = https://gitlab.com/group/upstream.git
[remote "origin"]
	url = https://gitlab.com/group/repo.git
`), 0644))
	t.Setenv("GITLAB_TOKEN", "")

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	var out bytes.Buffer
	cmd := NewCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"detect"})
	require.NoError(t, cmd.Execute())

	assert.Contains(t, out.String(), "  Remote:      origin\n")
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

func resetInitFlags(t *testing.T) {
	t.Helper()
	oldServer := initServer
	oldHost := initHost
	initServer = ""
	initHost = ""
	t.Cleanup(func() {
		initServer = oldServer
		initHost = oldHost
	})
}
