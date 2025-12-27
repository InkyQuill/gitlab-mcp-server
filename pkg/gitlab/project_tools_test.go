package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	gl "gitlab.com/gitlab-org/api/client-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/InkyQuill/gitlab-mcp-server/internal/toolsnaps"
)

func TestSetCurrentProjectHandler(t *testing.T) {
	// Tool schema snapshot test
	tool, _ := SetCurrentProject(nil)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	mockClient, _, ctrl := setupMockClient(t)
	defer ctrl.Finish()

	// Mock getClient function
	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	tool, handler := SetCurrentProject(mockGetClient)

	tests := []struct {
		name               string
		inputArgs          map[string]any
		setupDir           func() string
		mockSetup          func()
		expectedResult     map[string]interface{}
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - Set project with host",
			inputArgs: map[string]any{
				"projectId":  "group/project",
				"gitlabHost": "https://gitlab.example.com",
			},
			setupDir: func() string {
				return t.TempDir()
			},
			mockSetup: func() {
				// No API calls needed for SetCurrentProject
			},
			expectedResult: map[string]interface{}{
				"success":    true,
				"projectId":  "group/project",
				"gitlabHost": "https://gitlab.example.com",
			},
		},
		{
			name: "Success - Set project without host",
			inputArgs: map[string]any{
				"projectId": "group/project",
			},
			setupDir: func() string {
				return t.TempDir()
			},
			mockSetup: func() {},
			expectedResult: map[string]interface{}{
				"success":    true,
				"projectId":  "group/project",
				"gitlabHost": "",
			},
		},
		{
			name:               "Error - Missing projectId",
			inputArgs:          map[string]any{},
			setupDir:           func() string { return t.TempDir() },
			mockSetup:          func() {},
			expectedResult:     nil,
			expectResultError:  true,
			errorContains:      "Validation Error: missing required parameter: projectId",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()

			// Create temporary directory and change to it
			tmpDir := tc.setupDir()
			oldWd, err := os.Getwd()
			require.NoError(t, err)
			defer os.Chdir(oldWd)

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			// Create request
			req := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      tool.Name,
					Arguments: tc.inputArgs,
				},
			}

			// Execute handler
			result, err := handler(context.Background(), req)

			if tc.expectHandlerError {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				textContent := getTextResult(t, result)

				if tc.expectResultError {
					assert.Contains(t, textContent.Text, tc.errorContains)
				} else {
					// Parse JSON result
					var resultMap map[string]interface{}
					err = json.Unmarshal([]byte(textContent.Text), &resultMap)
					require.NoError(t, err)

					// Verify expected fields
					for key, expectedVal := range tc.expectedResult {
						assert.Equal(t, expectedVal, resultMap[key], "Field %s mismatch", key)
					}
					assert.Contains(t, resultMap, "configPath")

					// Verify .gmcprc file was created
					configPath := resultMap["configPath"].(string)
					assert.FileExists(t, configPath)
				}
			}
		})
	}
}

func TestGetCurrentProjectHandler(t *testing.T) {
	// Tool schema snapshot test
	tool, _ := GetCurrentProject(nil)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	mockClient, _, ctrl := setupMockClient(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	tool, handler := GetCurrentProject(mockGetClient)

	tests := []struct {
		name               string
		setupDir           func() string
		expectedResult     map[string]interface{}
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - Found .gmcprc file",
			setupDir: func() string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ".gmcprc")
				configContent := `{
  "projectId": "group/project",
  "gitlabHost": "https://gitlab.example.com"
}`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
				return tmpDir
			},
			expectedResult: map[string]interface{}{
				"found":      true,
				"projectId":  "group/project",
				"gitlabHost": "https://gitlab.example.com",
			},
		},
		{
			name: "Success - No .gmcprc file found",
			setupDir: func() string {
				return t.TempDir()
			},
			expectedResult: map[string]interface{}{
				"found": false,
			},
		},
		{
			name: "Success - .gmcprc in parent directory",
			setupDir: func() string {
				tmpDir := t.TempDir()
				// Create .gmcprc in parent
				configPath := filepath.Join(tmpDir, ".gmcprc")
				configContent := `{"projectId": "parent/project"}`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)

				// Create subdirectory
				subDir := filepath.Join(tmpDir, "subdir")
				err = os.Mkdir(subDir, 0755)
				require.NoError(t, err)
				return subDir
			},
			expectedResult: map[string]interface{}{
				"found":      true,
				"projectId":  "parent/project",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary directory and change to it
			tmpDir := tc.setupDir()
			oldWd, err := os.Getwd()
			require.NoError(t, err)
			defer os.Chdir(oldWd)

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			// Create request
			req := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name: tool.Name,
				},
			}

			// Execute handler
			result, err := handler(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, result)

			textContent := getTextResult(t, result)

			if tc.expectResultError {
				assert.Contains(t, textContent.Text, tc.errorContains)
			} else {
				// Parse JSON result
				var resultMap map[string]interface{}
				err = json.Unmarshal([]byte(textContent.Text), &resultMap)
				require.NoError(t, err)

				// Verify expected fields
				for key, expectedVal := range tc.expectedResult {
					assert.Equal(t, expectedVal, resultMap[key], "Field %s mismatch", key)
				}

				if resultMap["found"].(bool) {
					assert.Contains(t, resultMap, "configPath")
				}
			}
		})
	}
}

func TestGetProjectIDWithFallback(t *testing.T) {
	tests := []struct {
		name          string
		setupDir      func() string
		requestArgs   map[string]any
		expectedID    string
		expectError   bool
		errorContains string
	}{
		{
			name: "Success - Project ID from parameter",
			setupDir: func() string {
				return t.TempDir()
			},
			requestArgs: map[string]any{
				"projectId": "param/project",
			},
			expectedID: "param/project",
		},
		{
			name: "Success - Project ID from .gmcprc",
			setupDir: func() string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ".gmcprc")
				configContent := `{"projectId": "config/project"}`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
				return tmpDir
			},
			requestArgs: map[string]any{},
			expectedID:  "config/project",
		},
		{
			name: "Success - Parameter takes priority over config",
			setupDir: func() string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ".gmcprc")
				configContent := `{"projectId": "config/project"}`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
				return tmpDir
			},
			requestArgs: map[string]any{
				"projectId": "param/project",
			},
			expectedID: "param/project",
		},
		{
			name: "Error - No project ID found anywhere",
			setupDir: func() string {
				return t.TempDir()
			},
			requestArgs:   map[string]any{},
			expectError:   true,
			errorContains: "no project ID specified",
		},
		{
			name: "Success - Empty parameter string ignored, uses config",
			setupDir: func() string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ".gmcprc")
				configContent := `{"projectId": "config/project"}`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
				return tmpDir
			},
			requestArgs: map[string]any{
				"projectId": "",
			},
			expectedID: "config/project",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary directory and change to it
			tmpDir := tc.setupDir()
			oldWd, err := os.Getwd()
			require.NoError(t, err)
			defer os.Chdir(oldWd)

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			// Create request
			request := &mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Arguments: tc.requestArgs,
				},
			}

			// Execute function
			projectID, err := GetProjectIDWithFallback(request)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedID, projectID)
			}
		})
	}
}

func TestDetectProjectHandler(t *testing.T) {
	// Tool schema snapshot test
	tool, _ := DetectProject(nil)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	mockClient, mockProjects, ctrl := setupMockClient(t)
	defer ctrl.Finish()

	// Mock getClient function
	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	tool, handler := DetectProject(mockGetClient)

	tests := []struct {
		name               string
		setupDir           func() string
		mockSetup          func()
		expectResultError  bool
		expectHandlerError bool
		errorContains      string
	}{
		{
			name: "Error - Not a git repository",
			setupDir: func() string {
				return t.TempDir()
			},
			mockSetup:         func() {},
			expectResultError: true,
			errorContains:     "Failed to detect project",
		},
		{
			name: "Error - Client initialization error",
			setupDir: func() string {
				// Create a .git directory to make it a git repo
				tmpDir := t.TempDir()
				gitDir := filepath.Join(tmpDir, ".git")
				err := os.Mkdir(gitDir, 0755)
				require.NoError(t, err)
				return tmpDir
			},
			mockSetup: func() {
				// We can't easily mock getClient returning error here
				// because getClient is a function, not a mockable interface
			},
		},
		{
			name: "Success - Detect project from Git remote",
			setupDir: func() string {
				tmpDir := t.TempDir()
				gitDir := filepath.Join(tmpDir, ".git")
				err := os.MkdirAll(gitDir, 0755)
				require.NoError(t, err)

				// Create a minimal git config with a remote
				gitConfig := filepath.Join(gitDir, "config")
				configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
	logallrefupdates = true
[remote "origin"]
	url = git@gitlab.com:group/project.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`
				err = os.WriteFile(gitConfig, []byte(configContent), 0644)
				require.NoError(t, err)

				// Create HEAD file
				headFile := filepath.Join(gitDir, "HEAD")
				err = os.WriteFile(headFile, []byte("ref: refs/heads/main\n"), 0644)
				require.NoError(t, err)

				return tmpDir
			},
			mockSetup: func() {
				expectedProject := &gl.Project{
					ID:                123,
					Name:              "project",
					PathWithNamespace: "group/project",
				}
				mockProjects.EXPECT().
					GetProject("group/project", gomock.Any(), gomock.Any()).
					Return(expectedProject, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectResultError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()

			// Create temporary directory and change to it
			tmpDir := tc.setupDir()
			oldWd, err := os.Getwd()
			require.NoError(t, err)
			defer os.Chdir(oldWd)

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			// Create request
			req := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name: tool.Name,
				},
			}

			// Execute handler
			result, err := handler(context.Background(), req)

			if tc.expectHandlerError {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				textContent := getTextResult(t, result)

				if tc.expectResultError {
					assert.Contains(t, textContent.Text, tc.errorContains)
				}
			}
		})
	}
}

func TestAutoDetectAndSetProjectHandler(t *testing.T) {
	// Tool schema snapshot test
	tool, _ := AutoDetectAndSetProject(nil)
	require.NoError(t, toolsnaps.Test(tool.Name, tool), "tool schema should match snapshot")

	mockClient, mockProjects, ctrl := setupMockClient(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	tool, handler := AutoDetectAndSetProject(mockGetClient)

	t.Run("Error - Not a git repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(oldWd)

		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		req := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name: tool.Name,
			},
		}

		result, err := handler(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, result)

		textContent := getTextResult(t, result)
		assert.Contains(t, textContent.Text, "Failed to detect project")
	})

	t.Run("Success - Auto-detect and create .gmcprc", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(oldWd)

		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Create a minimal git repo with remote
		gitDir := filepath.Join(tmpDir, ".git")
		err = os.MkdirAll(gitDir, 0755)
		require.NoError(t, err)

		gitConfig := filepath.Join(gitDir, "config")
		configContent := `[core]
	repositoryformatversion = 0
	filemode = true
[remote "origin"]
	url = https://gitlab.com/group/project.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`
		err = os.WriteFile(gitConfig, []byte(configContent), 0644)
		require.NoError(t, err)

		headFile := filepath.Join(gitDir, "HEAD")
		err = os.WriteFile(headFile, []byte("ref: refs/heads/main\n"), 0644)
		require.NoError(t, err)

		// Setup mock expectation
		expectedProject := &gl.Project{
			ID:                123,
			Name:              "project",
			PathWithNamespace: "group/project",
		}
		mockProjects.EXPECT().
			GetProject("group/project", gomock.Any(), gomock.Any()).
			Return(expectedProject, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)

		req := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name: tool.Name,
			},
		}

		result, err := handler(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, result)

		textContent := getTextResult(t, result)

		// Verify the result contains expected fields
		assert.Contains(t, textContent.Text, "success")
		assert.Contains(t, textContent.Text, "projectId")
		assert.Contains(t, textContent.Text, "group/project")
		assert.Contains(t, textContent.Text, "configPath")

		// Verify .gmcprc file was created
		configPath := filepath.Join(tmpDir, ".gmcprc")
		assert.FileExists(t, configPath)

		// Verify config file content
		configData, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Contains(t, string(configData), "group/project")
	})
}

// Integration test for the full project config workflow
func TestProjectConfigWorkflow(t *testing.T) {
	mockClient, _, ctrl := setupMockClient(t)
	defer ctrl.Finish()

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	// Create temporary directory
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	t.Run("Set then Get current project", func(t *testing.T) {
		// Step 1: Set current project
		setTool, setHandler := SetCurrentProject(mockGetClient)
		setReq := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name: setTool.Name,
				Arguments: map[string]any{
					"projectId":  "mygroup/myproject",
					"gitlabHost": "https://gitlab.example.com",
				},
			},
		}

		setResult, err := setHandler(context.Background(), setReq)
		require.NoError(t, err)
		require.NotNil(t, setResult)

		setTextContent := getTextResult(t, setResult)
		var setResultMap map[string]interface{}
		err = json.Unmarshal([]byte(setTextContent.Text), &setResultMap)
		require.NoError(t, err)
		assert.True(t, setResultMap["success"].(bool))

		configPath := setResultMap["configPath"].(string)
		assert.FileExists(t, configPath)

		// Step 2: Get current project
		getTool, getHandler := GetCurrentProject(mockGetClient)
		getReq := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name: getTool.Name,
			},
		}

		getResult, err := getHandler(context.Background(), getReq)
		require.NoError(t, err)
		require.NotNil(t, getResult)

		getTextContent := getTextResult(t, getResult)
		var getResultMap map[string]interface{}
		err = json.Unmarshal([]byte(getTextContent.Text), &getResultMap)
		require.NoError(t, err)

		assert.True(t, getResultMap["found"].(bool))
		assert.Equal(t, "mygroup/myproject", getResultMap["projectId"])
		assert.Equal(t, "https://gitlab.example.com", getResultMap["gitlabHost"])
	})
}
