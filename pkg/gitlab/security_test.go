package gitlab

import (
	"context"
	"testing"

	"github.com/InkyQuill/gitlab-mcp-server/internal/toolsnaps"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gl "gitlab.com/gitlab-org/api/client-go"
)

// Test fixtures for GraphQL responses

var (
	// Mock SAST findings response
	sastFindingsResponse = map[string]any{
		"project": map[string]any{
			"pipelines": map[string]any{
				"nodes": []map[string]any{
					{
						"securityReportFindings": map[string]any{
							"nodes": []map[string]any{
								{
									"title":    "SQL Injection vulnerability",
									"severity": "High",
									"location": map[string]any{
										"file":      "app/controllers/user.go",
										"startLine": 42,
									},
									"identifiers": []map[string]any{
										{
											"name": "CVE-2023-1234",
											"url":  "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2023-1234",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Mock DAST findings response
	dastFindingsResponse = map[string]any{
		"project": map[string]any{
			"pipelines": map[string]any{
				"nodes": []map[string]any{
					{
						"securityReportFindings": map[string]any{
							"nodes": []map[string]any{
								{
									"title":    "Cross-site scripting (XSS)",
									"severity": "Medium",
									"location": map[string]any{
										"file":      "http://example.com/search",
										"startLine": 0,
									},
									"identifiers": []map[string]any{
										{
											"name": "CWE-79",
											"url":  "https://cwe.mitre.org/data/definitions/79.html",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Mock dependency scanning findings response
	dependencyScanningFindingsResponse = map[string]any{
		"project": map[string]any{
			"pipelines": map[string]any{
				"nodes": []map[string]any{
					{
						"securityReportFindings": map[string]any{
							"nodes": []map[string]any{
								{
									"title":    "Vulnerable dependency: lodash 4.17.15",
									"severity": "Critical",
									"location": map[string]any{
										"file":      "package.json",
										"startLine": 15,
									},
									"identifiers": []map[string]any{
										{
											"name": "CVE-2021-23337",
											"url":  "https://nvd.nist.gov/vuln/detail/CVE-2021-23337",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Mock container scanning findings response
	containerScanningFindingsResponse = map[string]any{
		"project": map[string]any{
			"pipelines": map[string]any{
				"nodes": []map[string]any{
					{
						"securityReportFindings": map[string]any{
							"nodes": []map[string]any{
								{
									"title":    "CVE-2023-1234 in libssl",
									"severity": "High",
									"location": map[string]any{
										"file":      "alpine:3.18",
										"startLine": 0,
									},
									"identifiers": []map[string]any{
										{
											"name": "CVE-2023-1234",
											"url":  "https://nvd.nist.gov/vuln/detail/CVE-2023-1234",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Mock secret detection findings response
	secretDetectionFindingsResponse = map[string]any{
		"project": map[string]any{
			"pipelines": map[string]any{
				"nodes": []map[string]any{
					{
						"securityReportFindings": map[string]any{
							"nodes": []map[string]any{
								{
									"title":    "AWS Access Key detected",
									"severity": "Critical",
									"location": map[string]any{
										"file":      "config/secrets.yml",
										"startLine": 10,
									},
									"identifiers": []map[string]any{
										{
											"name": "AWS Access Key",
											"url":  "https://docs.gitlab.com/ee/user/application_security/secret_detection/index.html",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Mock license compliance response
	licenseComplianceResponse = map[string]any{
		"project": map[string]any{
			"licenses": map[string]any{
				"nodes": []map[string]any{
					{
						"name":           "MIT License",
						"spdxIdentifier": "MIT",
						"url":            "https://opensource.org/licenses/MIT",
					},
					{
						"name":           "Apache License 2.0",
						"spdxIdentifier": "Apache-2.0",
						"url":            "https://opensource.org/licenses/Apache-2.0",
					},
				},
			},
		},
	}

	// Empty findings response
	emptyFindingsResponse = map[string]any{
		"project": map[string]any{
			"pipelines": map[string]any{
				"nodes": []map[string]any{},
			},
		},
	}

	// Empty licenses response
	emptyLicensesResponse = map[string]any{
		"project": map[string]any{
			"licenses": map[string]any{
				"nodes": []map[string]any{},
			},
		},
	}
)

func TestGetProjectSASTHandler(t *testing.T) {
	ctx := context.Background()

	// Create mock HTTP client with matcher for SAST query
	mockHTTPClient := NewMockGraphQLHTTPClient(
		GraphQLMockMatcher{
			Query:    "GetSASTFindings",
			Response: DataResponse(sastFindingsResponse),
		},
	)

	// Create GitLab client with mocked HTTP
	mockClient, err := MockGraphQLClient(mockHTTPClient, "test-token")
	require.NoError(t, err)

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	tool, handler := GetProjectSAST(mockGetClient, nil)

	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      tool.Name,
			Arguments: map[string]any{"pid": "mygroup/myproject"},
		},
	}

	result, err := handler(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, result)

	textContent := getTextResult(t, result)
	assert.Contains(t, textContent.Text, `"title":`)
	assert.Contains(t, textContent.Text, `"severity":`)
	assert.Contains(t, textContent.Text, `"location":`)
}

func TestGetProjectSASTHandler_EmptyResults(t *testing.T) {
	ctx := context.Background()

	// Create mock HTTP client for empty results
	mockHTTPClient := NewMockGraphQLHTTPClient(
		GraphQLMockMatcher{
			Query:    "GetSASTFindings",
			Response: DataResponse(emptyFindingsResponse),
		},
	)

	mockClient, err := MockGraphQLClient(mockHTTPClient, "test-token")
	require.NoError(t, err)

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	tool, handler := GetProjectSAST(mockGetClient, nil)

	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      tool.Name,
			Arguments: map[string]any{"pid": "mygroup/myproject"},
		},
	}

	result, err := handler(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, result)

	textContent := getTextResult(t, result)
	assert.Equal(t, "[]", textContent.Text)
}

func TestGetProjectDASTHandler(t *testing.T) {
	ctx := context.Background()

	mockHTTPClient := NewMockGraphQLHTTPClient(
		GraphQLMockMatcher{
			Query:    "GetDASTFindings",
			Response: DataResponse(dastFindingsResponse),
		},
	)

	mockClient, err := MockGraphQLClient(mockHTTPClient, "test-token")
	require.NoError(t, err)

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	tool, handler := GetProjectDAST(mockGetClient, nil)

	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      tool.Name,
			Arguments: map[string]any{"pid": "mygroup/myproject"},
		},
	}

	result, err := handler(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, result)

	textContent := getTextResult(t, result)
	assert.Contains(t, textContent.Text, `"title":`)
	assert.Contains(t, textContent.Text, `"severity":`)
}

func TestGetProjectDependencyScanningHandler(t *testing.T) {
	ctx := context.Background()

	mockHTTPClient := NewMockGraphQLHTTPClient(
		GraphQLMockMatcher{
			Query:    "GetDependencyScanningFindings",
			Response: DataResponse(dependencyScanningFindingsResponse),
		},
	)

	mockClient, err := MockGraphQLClient(mockHTTPClient, "test-token")
	require.NoError(t, err)

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	tool, handler := GetProjectDependencyScanning(mockGetClient, nil)

	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      tool.Name,
			Arguments: map[string]any{"pid": "mygroup/myproject"},
		},
	}

	result, err := handler(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, result)

	textContent := getTextResult(t, result)
	assert.Contains(t, textContent.Text, `"title":`)
	assert.Contains(t, textContent.Text, `"severity":`)
}

func TestGetProjectContainerScanningHandler(t *testing.T) {
	ctx := context.Background()

	mockHTTPClient := NewMockGraphQLHTTPClient(
		GraphQLMockMatcher{
			Query:    "GetContainerScanningFindings",
			Response: DataResponse(containerScanningFindingsResponse),
		},
	)

	mockClient, err := MockGraphQLClient(mockHTTPClient, "test-token")
	require.NoError(t, err)

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	tool, handler := GetProjectContainerScanning(mockGetClient, nil)

	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      tool.Name,
			Arguments: map[string]any{"pid": "mygroup/myproject"},
		},
	}

	result, err := handler(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, result)

	textContent := getTextResult(t, result)
	assert.Contains(t, textContent.Text, `"title":`)
	assert.Contains(t, textContent.Text, `"severity":`)
}

func TestGetProjectSecretDetectionHandler(t *testing.T) {
	ctx := context.Background()

	mockHTTPClient := NewMockGraphQLHTTPClient(
		GraphQLMockMatcher{
			Query:    "GetSecretDetectionFindings",
			Response: DataResponse(secretDetectionFindingsResponse),
		},
	)

	mockClient, err := MockGraphQLClient(mockHTTPClient, "test-token")
	require.NoError(t, err)

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	tool, handler := GetProjectSecretDetection(mockGetClient, nil)

	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      tool.Name,
			Arguments: map[string]any{"pid": "mygroup/myproject"},
		},
	}

	result, err := handler(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, result)

	textContent := getTextResult(t, result)
	assert.Contains(t, textContent.Text, `"title":`)
	assert.Contains(t, textContent.Text, `"severity":`)
}

func TestGetProjectLicenseComplianceHandler(t *testing.T) {
	ctx := context.Background()

	mockHTTPClient := NewMockGraphQLHTTPClient(
		GraphQLMockMatcher{
			Query:    "GetLicenseCompliance",
			Response: DataResponse(licenseComplianceResponse),
		},
	)

	mockClient, err := MockGraphQLClient(mockHTTPClient, "test-token")
	require.NoError(t, err)

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	tool, handler := GetProjectLicenseCompliance(mockGetClient, nil)

	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      tool.Name,
			Arguments: map[string]any{"pid": "mygroup/myproject"},
		},
	}

	result, err := handler(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, result)

	textContent := getTextResult(t, result)
	assert.Contains(t, textContent.Text, `"name":`)
	assert.Contains(t, textContent.Text, `"spdxIdentifier":`)
	assert.Contains(t, textContent.Text, `"MIT"`)
	assert.Contains(t, textContent.Text, `"Apache-2.0"`)
}

func TestGetProjectLicenseComplianceHandler_EmptyResults(t *testing.T) {
	ctx := context.Background()

	mockHTTPClient := NewMockGraphQLHTTPClient(
		GraphQLMockMatcher{
			Query:    "GetLicenseCompliance",
			Response: DataResponse(emptyLicensesResponse),
		},
	)

	mockClient, err := MockGraphQLClient(mockHTTPClient, "test-token")
	require.NoError(t, err)

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	tool, handler := GetProjectLicenseCompliance(mockGetClient, nil)

	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      tool.Name,
			Arguments: map[string]any{"pid": "mygroup/myproject"},
		},
	}

	result, err := handler(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, result)

	textContent := getTextResult(t, result)
	assert.Equal(t, "[]", textContent.Text)
}

func TestSecurityTools_MissingRequiredParameter(t *testing.T) {
	ctx := context.Background()

	mockHTTPClient := NewMockGraphQLHTTPClient()
	mockClient, err := MockGraphQLClient(mockHTTPClient, "test-token")
	require.NoError(t, err)

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	tests := []struct {
		name            string
		getToolAndFunc  func() (mcp.Tool, server.ToolHandlerFunc)
		arguments       map[string]any
		errorContains   string
	}{
		{
			name:           "getProjectSAST - missing pid",
			getToolAndFunc: func() (mcp.Tool, server.ToolHandlerFunc) { return GetProjectSAST(mockGetClient, nil) },
			arguments:      map[string]any{},
			errorContains:  "Validation Error",
		},
		{
			name:           "getProjectDAST - missing pid",
			getToolAndFunc: func() (mcp.Tool, server.ToolHandlerFunc) { return GetProjectDAST(mockGetClient, nil) },
			arguments:      map[string]any{},
			errorContains:  "Validation Error",
		},
		{
			name:           "getProjectDependencyScanning - missing pid",
			getToolAndFunc: func() (mcp.Tool, server.ToolHandlerFunc) { return GetProjectDependencyScanning(mockGetClient, nil) },
			arguments:      map[string]any{},
			errorContains:  "Validation Error",
		},
		{
			name:           "getProjectContainerScanning - missing pid",
			getToolAndFunc: func() (mcp.Tool, server.ToolHandlerFunc) { return GetProjectContainerScanning(mockGetClient, nil) },
			arguments:      map[string]any{},
			errorContains:  "Validation Error",
		},
		{
			name:           "getProjectSecretDetection - missing pid",
			getToolAndFunc: func() (mcp.Tool, server.ToolHandlerFunc) { return GetProjectSecretDetection(mockGetClient, nil) },
			arguments:      map[string]any{},
			errorContains:  "Validation Error",
		},
		{
			name:           "getProjectLicenseCompliance - missing pid",
			getToolAndFunc: func() (mcp.Tool, server.ToolHandlerFunc) { return GetProjectLicenseCompliance(mockGetClient, nil) },
			arguments:      map[string]any{},
			errorContains:  "Validation Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, handler := tt.getToolAndFunc()

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      tool.Name,
					Arguments: tt.arguments,
				},
			}

			result, err := handler(ctx, request)
			require.NoError(t, err)
			require.NotNil(t, result)

			textContent := getTextResult(t, result)
			assert.Contains(t, textContent.Text, tt.errorContains)
		})
	}
}

func TestSecurityTools_GraphQLErrorResponse(t *testing.T) {
	ctx := context.Background()

	// Create mock HTTP client that returns GraphQL error
	mockHTTPClient := NewMockGraphQLHTTPClient(
		GraphQLMockMatcher{
			Query:    "GetSASTFindings",
			Response: ErrorResponse("Project not found or access denied"),
		},
	)

	mockClient, err := MockGraphQLClient(mockHTTPClient, "test-token")
	require.NoError(t, err)

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	tool, handler := GetProjectSAST(mockGetClient, nil)

	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      tool.Name,
			Arguments: map[string]any{"pid": "nonexistent/project"},
		},
	}

	result, err := handler(ctx, request)
	// Should not return internal error for GraphQL errors
	require.NoError(t, err)
	require.NotNil(t, result)

	textContent := getTextResult(t, result)
	// The error should be in the result text
	assert.NotEmpty(t, textContent.Text)
}

func TestSecurityTools_NoMatchingGraphQLQuery(t *testing.T) {
	ctx := context.Background()

	// Create mock HTTP client with no matchers - will return 404
	mockHTTPClient := NewMockGraphQLHTTPClient()

	mockClient, err := MockGraphQLClient(mockHTTPClient, "test-token")
	require.NoError(t, err)

	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	tool, handler := GetProjectSAST(mockGetClient, nil)

	request := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      tool.Name,
			Arguments: map[string]any{"pid": "mygroup/myproject"},
		},
	}

	result, err := handler(ctx, request)
	// Either result or error should be present (depending on how GraphQL client handles 404)
	// The important thing is it doesn't panic
	if err != nil {
		// Internal error is acceptable for HTTP 404
		assert.Contains(t, err.Error(), "SAST findings")
	} else {
		// If no error, result should contain error message
		require.NotNil(t, result)
		textContent := getTextResult(t, result)
		assert.NotEmpty(t, textContent.Text)
	}
}

// TestSecurityTools_SchemaSnapshots verifies that tool schemas match their snapshots
func TestSecurityTools_SchemaSnapshots(t *testing.T) {
	tools := []struct {
		name string
		tool mcp.Tool
	}{
		{"getProjectSAST", mustMakeTool(GetProjectSAST)},
		{"getProjectDAST", mustMakeTool(GetProjectDAST)},
		{"getProjectDependencyScanning", mustMakeTool(GetProjectDependencyScanning)},
		{"getProjectContainerScanning", mustMakeTool(GetProjectContainerScanning)},
		{"getProjectSecretDetection", mustMakeTool(GetProjectSecretDetection)},
		{"getProjectLicenseCompliance", mustMakeTool(GetProjectLicenseCompliance)},
	}

	for _, tc := range tools {
		t.Run(tc.name, func(t *testing.T) {
			err := toolsnaps.Test(tc.name, tc.tool)
			require.NoError(t, err, "tool schema should match snapshot")
		})
	}
}

// Helper function to create tool from tool definition function
func mustMakeTool(fn func(GetClientFn, map[string]string) (mcp.Tool, server.ToolHandlerFunc)) mcp.Tool {
	tool, _ := fn(nil, nil)
	return tool
}
