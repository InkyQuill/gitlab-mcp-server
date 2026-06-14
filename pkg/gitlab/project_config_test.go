package gitlab

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadProjectConfig(t *testing.T) {
	tests := []struct {
		name        string
		setupFile   func(t *testing.T) string
		expectError bool
		validate    func(t *testing.T, config *ProjectConfig)
	}{
		{
			name: "Success - Valid config file",
			setupFile: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ".gmcprc")
				content := `{
  "projectId": "group/project",
  "gitlabHost": "https://gitlab.example.com",
  "tokenName": "work-token"
}`
				err := os.WriteFile(configPath, []byte(content), 0644)
				require.NoError(t, err)
				return configPath
			},
			expectError: false,
			validate: func(t *testing.T, config *ProjectConfig) {
				assert.Equal(t, "group/project", config.ProjectID)
				assert.Equal(t, "https://gitlab.example.com", config.GitLabHost)
				assert.Equal(t, "work-token", config.TokenName)
			},
		},
		{
			name: "Success - Minimal config",
			setupFile: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ".gmcprc")
				content := `{"projectId": "myproject"}`
				err := os.WriteFile(configPath, []byte(content), 0644)
				require.NoError(t, err)
				return configPath
			},
			expectError: false,
			validate: func(t *testing.T, config *ProjectConfig) {
				assert.Equal(t, "myproject", config.ProjectID)
				assert.Empty(t, config.GitLabHost)
				assert.Empty(t, config.TokenName)
			},
		},
		{
			name: "Error - File does not exist",
			setupFile: func(t *testing.T) string {
				return "/nonexistent/file.json"
			},
			expectError: true,
		},
		{
			name: "Error - Invalid JSON",
			setupFile: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ".gmcprc")
				content := `{invalid json content`
				err := os.WriteFile(configPath, []byte(content), 0644)
				require.NoError(t, err)
				return configPath
			},
			expectError: true,
		},
		{
			name: "Error - Empty file",
			setupFile: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ".gmcprc")
				err := os.WriteFile(configPath, []byte{}, 0644)
				require.NoError(t, err)
				return configPath
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configPath := tc.setupFile(t)

			config, err := readProjectConfig(configPath)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, config)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				if tc.validate != nil {
					tc.validate(t, config)
				}
			}
		})
	}
}

func TestWriteProjectConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *ProjectConfig
		expectError bool
		validate    func(t *testing.T, configPath string, config *ProjectConfig)
	}{
		{
			name: "Success - Full config",
			config: &ProjectConfig{
				ProjectID:  "group/project",
				GitLabHost: "https://gitlab.example.com",
				TokenName:  "my-token",
			},
			expectError: false,
			validate: func(t *testing.T, configPath string, config *ProjectConfig) {
				// Verify file exists
				assert.FileExists(t, configPath)

				// Read and verify content
				data, err := os.ReadFile(configPath)
				require.NoError(t, err)

				var readConfig ProjectConfig
				err = json.Unmarshal(data, &readConfig)
				require.NoError(t, err)

				assert.Equal(t, config.ProjectID, readConfig.ProjectID)
				assert.Equal(t, config.GitLabHost, readConfig.GitLabHost)
				assert.Equal(t, config.TokenName, readConfig.TokenName)
				assert.False(t, readConfig.LastUpdated.IsZero())
			},
		},
		{
			name: "Success - Minimal config",
			config: &ProjectConfig{
				ProjectID: "simple-project",
			},
			expectError: false,
			validate: func(t *testing.T, configPath string, config *ProjectConfig) {
				data, err := os.ReadFile(configPath)
				require.NoError(t, err)

				var readConfig ProjectConfig
				err = json.Unmarshal(data, &readConfig)
				require.NoError(t, err)

				assert.Equal(t, "simple-project", readConfig.ProjectID)
				assert.False(t, readConfig.LastUpdated.IsZero())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			configPath, err := WriteProjectConfig(tmpDir, tc.config)

			if tc.expectError {
				assert.Error(t, err)
				assert.Empty(t, configPath)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, configPath)
				if tc.validate != nil {
					tc.validate(t, configPath, tc.config)
				}
			}
		})
	}
}

func TestFindProjectConfig(t *testing.T) {
	tests := []struct {
		name        string
		setupDir    func(t *testing.T) string
		expectError bool
		validate    func(t *testing.T, config *ProjectConfig, path string)
	}{
		{
			name: "Success - Config in current directory",
			setupDir: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ".gmcprc")
				content := `{"projectId": "current/project"}`
				err := os.WriteFile(configPath, []byte(content), 0644)
				require.NoError(t, err)
				return tmpDir
			},
			expectError: false,
			validate: func(t *testing.T, config *ProjectConfig, path string) {
				assert.Equal(t, "current/project", config.ProjectID)
				assert.Contains(t, path, ".gmcprc")
			},
		},
		{
			name: "Success - Config in parent directory",
			setupDir: func(t *testing.T) string {
				tmpDir := t.TempDir()
				// Create config in parent (tmpDir)
				configPath := filepath.Join(tmpDir, ".gmcprc")
				content := `{"projectId": "parent/project"}`
				err := os.WriteFile(configPath, []byte(content), 0644)
				require.NoError(t, err)

				// Create subdirectory
				subDir := filepath.Join(tmpDir, "subdir")
				err = os.Mkdir(subDir, 0755)
				require.NoError(t, err)

				return subDir
			},
			expectError: false,
			validate: func(t *testing.T, config *ProjectConfig, path string) {
				assert.Equal(t, "parent/project", config.ProjectID)
			},
		},
		{
			name: "Success - Config in grandparent directory",
			setupDir: func(t *testing.T) string {
				tmpDir := t.TempDir()
				// Create config in grandparent (tmpDir)
				configPath := filepath.Join(tmpDir, ".gmcprc")
				content := `{"projectId": "grandparent/project"}`
				err := os.WriteFile(configPath, []byte(content), 0644)
				require.NoError(t, err)

				// Create nested subdirectories
				subDir := filepath.Join(tmpDir, "level1", "level2", "level3")
				err = os.MkdirAll(subDir, 0755)
				require.NoError(t, err)

				return subDir
			},
			expectError: false,
			validate: func(t *testing.T, config *ProjectConfig, path string) {
				assert.Equal(t, "grandparent/project", config.ProjectID)
			},
		},
		{
			name: "Success - No config found",
			setupDir: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return tmpDir
			},
			expectError: false,
			validate: func(t *testing.T, config *ProjectConfig, path string) {
				assert.Nil(t, config)
				assert.Empty(t, path)
			},
		},
		{
			name: "Error - Invalid JSON in config",
			setupDir: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ".gmcprc")
				content := `{invalid json`
				err := os.WriteFile(configPath, []byte(content), 0644)
				require.NoError(t, err)
				return tmpDir
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testDir := tc.setupDir(t)

			// Change to test directory
			originalWd, err := os.Getwd()
			require.NoError(t, err)
			defer os.Chdir(originalWd)

			err = os.Chdir(testDir)
			require.NoError(t, err)

			// Run FindProjectConfig
			config, path, err := FindProjectConfig()

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.validate != nil {
					tc.validate(t, config, path)
				}
			}
		})
	}
}

func TestFindGitDir(t *testing.T) {
	tests := []struct {
		name     string
		setupDir func(t *testing.T) string
		expected string
	}{
		{
			name: "Found - .git in current directory",
			setupDir: func(t *testing.T) string {
				tmpDir := t.TempDir()
				gitDir := filepath.Join(tmpDir, ".git")
				err := os.Mkdir(gitDir, 0755)
				require.NoError(t, err)
				return tmpDir
			},
			expected: ".git",
		},
		{
			name: "Found - .git in parent directory",
			setupDir: func(t *testing.T) string {
				tmpDir := t.TempDir()
				gitDir := filepath.Join(tmpDir, ".git")
				err := os.Mkdir(gitDir, 0755)
				require.NoError(t, err)

				subDir := filepath.Join(tmpDir, "subdir")
				err = os.Mkdir(subDir, 0755)
				require.NoError(t, err)
				return subDir
			},
			expected: ".git", // Returns path to .git
		},
		{
			name: "Not Found - No .git directory",
			setupDir: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return tmpDir
			},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testDir := tc.setupDir(t)

			result := findGitDir(testDir)

			if tc.expected == "" {
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result)
				assert.Contains(t, result, ".git")
			}
		})
	}
}

func TestIsGitHubURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "GitHub HTTPS URL",
			url:      "https://github.com/owner/repo.git",
			expected: true,
		},
		{
			name:     "GitHub HTTP URL",
			url:      "http://github.com/owner/repo.git",
			expected: true,
		},
		{
			name:     "GitHub SSH URL",
			url:      "git@github.com:owner/repo.git",
			expected: true,
		},
		{
			name:     "GitHub SSH with protocol",
			url:      "ssh://git@github.com/owner/repo.git",
			expected: true,
		},
		{
			name:     "GitLab HTTPS URL",
			url:      "https://gitlab.com/owner/repo.git",
			expected: false,
		},
		{
			name:     "GitLab SSH URL",
			url:      "git@gitlab.com:owner/repo.git",
			expected: false,
		},
		{
			name:     "GitLab self-hosted",
			url:      "https://gitlab.example.com/owner/repo.git",
			expected: false,
		},
		{
			name:     "Mixed case GitHub",
			url:      "https://GITHUB.COM/owner/repo.git",
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isGitHubURL(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseGitLabURL(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		expectError   bool
		errorContains string
		expectedID    string
		expectedHost  string
	}{
		{
			name:         "Success - GitLab HTTPS gitlab.com",
			url:          "https://gitlab.com/owner/repo.git",
			expectError:  false,
			expectedID:   "owner/repo",
			expectedHost: "https://gitlab.com",
		},
		{
			name:         "Success - GitLab HTTPS self-hosted",
			url:          "https://gitlab.example.com/group/project.git",
			expectError:  false,
			expectedID:   "group/project",
			expectedHost: "https://gitlab.example.com",
		},
		{
			name:         "Success - GitLab SSH gitlab.com",
			url:          "git@gitlab.com:owner/repo.git",
			expectError:  false,
			expectedID:   "owner/repo",
			expectedHost: "https://gitlab.com",
		},
		{
			name:         "Success - GitLab SSH self-hosted",
			url:          "git@gitlab.example.com:group/project.git",
			expectError:  false,
			expectedID:   "group/project",
			expectedHost: "https://gitlab.example.com",
		},
		{
			name:         "Success - HTTPS without .git",
			url:          "https://gitlab.com/owner/repo",
			expectError:  false,
			expectedID:   "owner/repo",
			expectedHost: "https://gitlab.com",
		},
		{
			name:         "Success - SSH without .git",
			url:          "git@gitlab.com:owner/repo",
			expectError:  false,
			expectedID:   "owner/repo",
			expectedHost: "https://gitlab.com",
		},
		{
			name:          "Error - GitHub HTTPS URL",
			url:           "https://github.com/owner/repo.git",
			expectError:   true,
			errorContains: "GitHub repository detected",
		},
		{
			name:          "Error - GitHub SSH URL",
			url:           "git@github.com:owner/repo.git",
			expectError:   true,
			errorContains: "GitHub repository detected",
		},
		{
			name:         "Unknown format - HTTP (not HTTPS)",
			url:          "http://example.com/repo.git",
			expectError:  false,
			expectedID:   "",
			expectedHost: "",
		},
		{
			name:         "Non-GitLab HTTPS - Gets parsed",
			url:          "https://bitbucket.org/owner/repo.git",
			expectError:  false,
			expectedID:   "owner/repo",
			expectedHost: "https://bitbucket.org",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			projectID, host, err := parseGitLabURL(tc.url)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedID, projectID)
				assert.Equal(t, tc.expectedHost, host)
			}
		})
	}
}

func TestParseGitRemoteCandidates(t *testing.T) {
	configData := []byte(`[remote "upstream"]
	url = https://gitlab.com/group/upstream.git
[remote "origin"]
	url = git@gitlab.example.com:team/repo.git
[remote "mirror"]
	url = /srv/git/local.git
`)

	candidates, err := ParseGitRemoteCandidates(configData)
	require.NoError(t, err)
	require.Len(t, candidates, 2)

	assert.Equal(t, GitRemoteCandidate{
		RemoteName: "upstream",
		URL:        "https://gitlab.com/group/upstream.git",
		ProjectID:  "group/upstream",
		Host:       "https://gitlab.com",
	}, candidates[0])
	assert.Equal(t, GitRemoteCandidate{
		RemoteName: "origin",
		URL:        "git@gitlab.example.com:team/repo.git",
		ProjectID:  "team/repo",
		Host:       "https://gitlab.example.com",
	}, candidates[1])
}

func TestParseGitRemoteCandidates_GitHubHardError(t *testing.T) {
	configData := []byte(`[remote "origin"]
	url = https://github.com/owner/repo.git
`)

	candidates, err := ParseGitRemoteCandidates(configData)
	require.Error(t, err)
	assert.Nil(t, candidates)
	assert.Contains(t, err.Error(), "GitHub repository detected")
}

func TestParseGitRemoteCandidates_IgnoresURLsInNonRemoteSections(t *testing.T) {
	configData := []byte(`[remote "origin"]
	url = https://gitlab.com/group/repo.git
[submodule "x"]
	url = https://github.com/owner/submodule.git
`)

	candidates, err := ParseGitRemoteCandidates(configData)
	require.NoError(t, err)
	require.Len(t, candidates, 1)

	assert.Equal(t, GitRemoteCandidate{
		RemoteName: "origin",
		URL:        "https://gitlab.com/group/repo.git",
		ProjectID:  "group/repo",
		Host:       "https://gitlab.com",
	}, candidates[0])
}

func TestParseGitRemoteCandidates_URLAssignmentSpacing(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{
			name: "no spaces",
			line: "url=https://gitlab.com/group/repo.git",
		},
		{
			name: "extra spaces",
			line: "url   =   https://gitlab.com/group/repo.git",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configData := []byte("[remote \"origin\"]\n" + tc.line + "\n")

			candidates, err := ParseGitRemoteCandidates(configData)
			require.NoError(t, err)
			require.Len(t, candidates, 1)

			assert.Equal(t, GitRemoteCandidate{
				RemoteName: "origin",
				URL:        "https://gitlab.com/group/repo.git",
				ProjectID:  "group/repo",
				Host:       "https://gitlab.com",
			}, candidates[0])
		})
	}
}

func TestSelectGitRemoteCandidate_MatchesAllowedHost(t *testing.T) {
	candidates := []GitRemoteCandidate{
		{RemoteName: "origin", ProjectID: "oss/repo", Host: "https://gitlab.com"},
		{RemoteName: "work", ProjectID: "team/repo", Host: "https://gitlab.example.com"},
	}

	selected, err := SelectGitRemoteCandidate(candidates, []string{"https://gitlab.example.com/"})
	require.NoError(t, err)
	assert.Equal(t, "work", selected.RemoteName)
	assert.Equal(t, "team/repo", selected.ProjectID)
}

func TestSelectGitRemoteCandidate_PrefersOriginWhenNoHostFilter(t *testing.T) {
	candidates := []GitRemoteCandidate{
		{RemoteName: "upstream", ProjectID: "group/upstream", Host: "https://gitlab.com"},
		{RemoteName: "origin", ProjectID: "group/repo", Host: "https://gitlab.com"},
	}

	selected, err := SelectGitRemoteCandidate(candidates, nil)
	require.NoError(t, err)
	assert.Equal(t, "origin", selected.RemoteName)
	assert.Equal(t, "group/repo", selected.ProjectID)
}

func TestSelectGitRemoteCandidate_AmbiguousWhenMultipleRemain(t *testing.T) {
	candidates := []GitRemoteCandidate{
		{RemoteName: "work", ProjectID: "team/api", Host: "https://gitlab.example.com"},
		{RemoteName: "mirror", ProjectID: "team/api-mirror", Host: "https://gitlab.example.com"},
	}

	_, err := SelectGitRemoteCandidate(candidates, []string{"https://gitlab.example.com"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous GitLab remotes")
	assert.Contains(t, err.Error(), "work")
	assert.Contains(t, err.Error(), "mirror")
}

func TestSelectGitRemoteCandidate_MatchesAllowedHostIgnoringScheme(t *testing.T) {
	candidates := []GitRemoteCandidate{
		{RemoteName: "origin", ProjectID: "team/repo", Host: "https://gitlab.local"},
	}

	selected, err := SelectGitRemoteCandidate(candidates, []string{"http://gitlab.local"})
	require.NoError(t, err)
	assert.Equal(t, "origin", selected.RemoteName)
	assert.Equal(t, "team/repo", selected.ProjectID)
}

func TestNormalizeGitLabHost(t *testing.T) {
	assert.Equal(t, "gitlab.example.com", NormalizeGitLabHost("https://GitLab.Example.com/"))
	assert.Equal(t, "gitlab.example.com", NormalizeGitLabHost("gitlab.example.com"))
	assert.Equal(t, "gitlab.example.com:8080", NormalizeGitLabHost("  http://GitLab.Example.com:8080/  "))
}

func TestParseGitRemotes(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		expectError   bool
		errorContains string
		expectedID    string
		expectedHost  string
	}{
		{
			name: "Success - GitLab HTTPS origin",
			configContent: `[remote "origin"]
	url = https://gitlab.com/owner/repo.git
`,
			expectError:  false,
			expectedID:   "owner/repo",
			expectedHost: "https://gitlab.com",
		},
		{
			name: "Success - GitLab SSH origin",
			configContent: `[remote "origin"]
	url = git@gitlab.com:owner/repo.git
`,
			expectError:  false,
			expectedID:   "owner/repo",
			expectedHost: "https://gitlab.com",
		},
		{
			name: "Success - Multiple GitLab remotes, origin selected",
			configContent: `[remote "origin"]
	url = git@gitlab.com:owner/repo.git
[remote "upstream"]
	url = https://gitlab.com/upstream/repo.git
`,
			expectError:  false,
			expectedID:   "owner/repo",
			expectedHost: "https://gitlab.com",
		},
		{
			name: "Success - Self-hosted GitLab",
			configContent: `[remote "origin"]
	url = https://gitlab.example.com/group/project.git
`,
			expectError:  false,
			expectedID:   "group/project",
			expectedHost: "https://gitlab.example.com",
		},
		{
			name: "Error - GitHub remote",
			configContent: `[remote "origin"]
	url = https://github.com/owner/repo.git
`,
			expectError:   true,
			errorContains: "GitHub repository detected",
		},
		{
			name: "Error - GitHub remote after GitLab remote",
			configContent: `[remote "origin"]
	url = git@gitlab.com:owner/repo.git
[remote "upstream"]
	url = https://github.com/upstream/repo.git
`,
			expectError:   true,
			errorContains: "GitHub repository detected",
		},
		{
			name: "Success - Non-GitLab remote (Bitbucket) gets parsed",
			configContent: `[remote "origin"]
	url = https://bitbucket.org/owner/repo.git
`,
			expectError:  false,
			expectedID:   "owner/repo",
			expectedHost: "https://bitbucket.org",
		},
		{
			name:          "Success - Empty config",
			configContent: ``,
			expectError:   false,
			expectedID:    "",
			expectedHost:  "",
		},
		{
			name: "Success - Config without remotes",
			configContent: `[core]
	bare = false
`,
			expectError:  false,
			expectedID:   "",
			expectedHost: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configData := []byte(tc.configContent)

			projectID, host, err := parseGitRemotes(configData)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedID, projectID)
				assert.Equal(t, tc.expectedHost, host)
			}
		})
	}
}

func TestParseGitRemotes_UsesCandidateSelection(t *testing.T) {
	configData := []byte(`[remote "upstream"]
	url = https://gitlab.com/group/upstream.git
[remote "origin"]
	url = https://gitlab.com/group/repo.git
`)

	projectID, host, err := parseGitRemotes(configData)
	require.NoError(t, err)
	assert.Equal(t, "group/repo", projectID)
	assert.Equal(t, "https://gitlab.com", host)
}

func TestDetectProjectFromGit(t *testing.T) {
	tests := []struct {
		name          string
		setupGitRepo  func(t *testing.T) string
		expectError   bool
		errorContains string
		expectedID    string
		expectedHost  string
	}{
		{
			name: "Success - GitLab HTTPS remote",
			setupGitRepo: func(t *testing.T) string {
				tmpDir := t.TempDir()
				gitDir := filepath.Join(tmpDir, ".git")
				err := os.Mkdir(gitDir, 0755)
				require.NoError(t, err)

				configContent := `[remote "origin"]
	url = https://gitlab.com/owner/repo.git
`
				configPath := filepath.Join(gitDir, "config")
				err = os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)

				return tmpDir
			},
			expectError:  false,
			expectedID:   "owner/repo",
			expectedHost: "https://gitlab.com",
		},
		{
			name: "Success - GitLab SSH remote",
			setupGitRepo: func(t *testing.T) string {
				tmpDir := t.TempDir()
				gitDir := filepath.Join(tmpDir, ".git")
				err := os.Mkdir(gitDir, 0755)
				require.NoError(t, err)

				configContent := `[remote "origin"]
	url = git@gitlab.com:group/project.git
`
				configPath := filepath.Join(gitDir, "config")
				err = os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)

				return tmpDir
			},
			expectError:  false,
			expectedID:   "group/project",
			expectedHost: "https://gitlab.com",
		},
		{
			name: "Success - Self-hosted GitLab",
			setupGitRepo: func(t *testing.T) string {
				tmpDir := t.TempDir()
				gitDir := filepath.Join(tmpDir, ".git")
				err := os.Mkdir(gitDir, 0755)
				require.NoError(t, err)

				configContent := `[remote "origin"]
	url = git@gitlab.example.com:team/repo.git
`
				configPath := filepath.Join(gitDir, "config")
				err = os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)

				return tmpDir
			},
			expectError:  false,
			expectedID:   "team/repo",
			expectedHost: "https://gitlab.example.com",
		},
		{
			name: "Error - Not a git repository",
			setupGitRepo: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return tmpDir
			},
			expectError:   true,
			errorContains: "not a Git repository",
		},
		{
			name: "Error - GitHub remote",
			setupGitRepo: func(t *testing.T) string {
				tmpDir := t.TempDir()
				gitDir := filepath.Join(tmpDir, ".git")
				err := os.Mkdir(gitDir, 0755)
				require.NoError(t, err)

				configContent := `[remote "origin"]
	url = https://github.com/owner/repo.git
`
				configPath := filepath.Join(gitDir, "config")
				err = os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)

				return tmpDir
			},
			expectError:   true,
			errorContains: "GitHub repository detected",
		},
		{
			name: "Error - No GitLab remote (unknown format)",
			setupGitRepo: func(t *testing.T) string {
				tmpDir := t.TempDir()
				gitDir := filepath.Join(tmpDir, ".git")
				err := os.Mkdir(gitDir, 0755)
				require.NoError(t, err)

				configContent := `[remote "origin"]
	url = /path/to/local/repo.git
`
				configPath := filepath.Join(gitDir, "config")
				err = os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)

				return tmpDir
			},
			expectError:   true,
			errorContains: "no GitLab remote found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testDir := tc.setupGitRepo(t)

			// Change to test directory
			originalWd, err := os.Getwd()
			require.NoError(t, err)
			defer os.Chdir(originalWd)

			err = os.Chdir(testDir)
			require.NoError(t, err)

			// Run DetectProjectFromGit
			projectID, gitlabHost, err := DetectProjectFromGit()

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedID, projectID)
				assert.Equal(t, tc.expectedHost, gitlabHost)
			}
		})
	}
}

func TestDetectProjectCandidateFromGit_GitFile(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "worktree")
	actualGitDir := filepath.Join(tmpDir, "actual.git")
	require.NoError(t, os.Mkdir(repoDir, 0755))
	require.NoError(t, os.Mkdir(actualGitDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, ".git"), []byte("gitdir: ../actual.git\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(actualGitDir, "config"), []byte(`[remote "origin"]
	url = https://gitlab.example.com/group/repo.git
`), 0644))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(repoDir))

	candidate, err := DetectProjectCandidateFromGit([]string{"gitlab.example.com"})

	require.NoError(t, err)
	assert.Equal(t, "origin", candidate.RemoteName)
	assert.Equal(t, "group/repo", candidate.ProjectID)
	assert.Equal(t, "https://gitlab.example.com", candidate.Host)
}

func TestDetectProjectCandidateFromGit_Ambiguous(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	require.NoError(t, os.Mkdir(gitDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "config"), []byte(`[remote "work"]
	url = https://gitlab.example.com/team/api.git
[remote "mirror"]
	url = https://gitlab.example.com/team/api-mirror.git
`), 0644))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	_, err = DetectProjectCandidateFromGit([]string{"https://gitlab.example.com"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous GitLab remotes")
}

func TestReadProjectConfig_PromotesTokenNameToServer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gmcprc")
	body := `{"projectId":"g/p","tokenName":"work"}`
	require.NoError(t, os.WriteFile(path, []byte(body), 0600))

	cfg, err := readProjectConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "g/p", cfg.ProjectID)
	assert.Equal(t, "work", cfg.Server, "tokenName should auto-promote to server")
}

func TestReadProjectConfig_KeepsServerWhenBothSet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gmcprc")
	body := `{"projectId":"g/p","server":"work","tokenName":"other"}`
	require.NoError(t, os.WriteFile(path, []byte(body), 0600))

	cfg, err := readProjectConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "work", cfg.Server, "explicit server wins over deprecated tokenName")
}

func TestProjectConfig_LastUpdated(t *testing.T) {
	tmpDir := t.TempDir()

	config := &ProjectConfig{
		ProjectID:  "test/project",
		GitLabHost: "https://gitlab.com",
		TokenName:  "test-token",
	}

	// Write config
	configPath, err := WriteProjectConfig(tmpDir, config)
	require.NoError(t, err)

	// Read it back
	readConfig, err := readProjectConfig(configPath)
	require.NoError(t, err)

	// Verify LastUpdated was set
	assert.False(t, readConfig.LastUpdated.IsZero())
	assert.WithinDuration(t, time.Now(), readConfig.LastUpdated, 2*time.Second)
}
