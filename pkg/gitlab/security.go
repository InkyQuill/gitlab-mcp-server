package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gl "gitlab.com/gitlab-org/api/client-go"
	"github.com/InkyQuill/gitlab-mcp-server/pkg/translations"
)

// GraphQL query templates as constants

const (
	// graphqlQuerySAST retrieves SAST (Static Application Security Testing) findings
	graphqlQuerySAST = `
query GetSASTFindings($fullPath: ID!) {
	project(fullPath: $fullPath) {
		pipelines(last: 1) {
			nodes {
				securityReportFindings(reportTypes: [SAST]) {
					nodes {
						title
						severity
						location {
							file
							startLine
						}
						identifiers {
							name
							url
						}
					}
				}
			}
		}
	}
}
`

	// graphqlQueryDAST retrieves DAST (Dynamic Application Security Testing) findings
	graphqlQueryDAST = `
query GetDASTFindings($fullPath: ID!) {
	project(fullPath: $fullPath) {
		pipelines(last: 1) {
			nodes {
				securityReportFindings(reportTypes: [DAST]) {
					nodes {
						title
						severity
						location {
							file
							startLine
						}
						identifiers {
							name
							url
						}
					}
				}
			}
		}
	}
}
`

	// graphqlQueryDependencyScanning retrieves dependency scanning findings
	graphqlQueryDependencyScanning = `
query GetDependencyScanningFindings($fullPath: ID!) {
	project(fullPath: $fullPath) {
		pipelines(last: 1) {
			nodes {
				securityReportFindings(reportTypes: [DEPENDENCY_SCANNING]) {
					nodes {
						title
						severity
						location {
							file
							startLine
						}
						identifiers {
							name
							url
						}
					}
				}
			}
		}
	}
}
`

	// graphqlQueryContainerScanning retrieves container scanning findings
	graphqlQueryContainerScanning = `
query GetContainerScanningFindings($fullPath: ID!) {
	project(fullPath: $fullPath) {
		pipelines(last: 1) {
			nodes {
				securityReportFindings(reportTypes: [CONTAINER_SCANNING]) {
					nodes {
						title
						severity
						location {
							file
							startLine
						}
						identifiers {
							name
							url
						}
					}
				}
			}
		}
	}
}
`

	// graphqlQuerySecretDetection retrieves secret detection findings
	graphqlQuerySecretDetection = `
query GetSecretDetectionFindings($fullPath: ID!) {
	project(fullPath: $fullPath) {
		pipelines(last: 1) {
			nodes {
				securityReportFindings(reportTypes: [SECRET_DETECTION]) {
					nodes {
						title
						severity
						location {
							file
							startLine
						}
						identifiers {
							name
							url
						}
					}
				}
			}
		}
	}
}
`

	// graphqlQueryLicenseCompliance retrieves license information
	graphqlQueryLicenseCompliance = `
query GetLicenseCompliance($fullPath: ID!) {
	project(fullPath: $fullPath) {
		licenses {
			nodes {
				name
				spdxIdentifier
				url
			}
		}
	}
}
`
)

// GraphQL response structures

// SecurityFinding represents a single security vulnerability finding
type SecurityFinding struct {
	Title     string     `json:"title"`
	Severity  string     `json:"severity"`
	Location  *Location  `json:"location,omitempty"`
	Identifiers []Identifier `json:"identifiers,omitempty"`
}

// Location represents the location of a finding
type Location struct {
	File      string `json:"file,omitempty"`
	StartLine int    `json:"startLine,omitempty"`
}

// Identifier represents an identifier for a vulnerability (e.g., CVE)
type Identifier struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// PipelineSecurityReport represents security findings from a pipeline
type PipelineSecurityReport struct {
	Nodes []PipelineNode `json:"nodes"`
}

// PipelineNode represents a single pipeline
type PipelineNode struct {
	SecurityReportFindings SecurityReportFindings `json:"securityReportFindings"`
}

// SecurityReportFindings contains security findings
type SecurityReportFindings struct {
	Nodes []SecurityFinding `json:"nodes"`
}

// ProjectSecurityResponse represents the top-level GraphQL response for security queries
type ProjectSecurityResponse struct {
	Project Project `json:"project"`
}

// Project represents a GitLab project in GraphQL responses
type Project struct {
	Pipelines Pipelines `json:"pipelines"`
}

// Pipelines represents pipeline nodes
type Pipelines struct {
	Nodes []PipelineNode `json:"nodes"`
}

// LicenseComplianceResponse represents the GraphQL response for license compliance
type LicenseComplianceResponse struct {
	Project ProjectLicenses `json:"project"`
}

// ProjectLicenses contains license nodes
type ProjectLicenses struct {
	Licenses Licenses `json:"licenses"`
}

// Licenses contains license nodes
type Licenses struct {
	Nodes []License `json:"nodes"`
}

// License represents a software license
type License struct {
	Name           string `json:"name"`
	SpdxIdentifier string `json:"spdxIdentifier"`
	URL            string `json:"url,omitempty"`
}

// GetProjectSAST defines the MCP tool for retrieving SAST findings
func GetProjectSAST(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"getProjectSAST",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_GET_PROJECT_SAST_DESCRIPTION)),
		mcp.WithString("pid",
			mcp.Description("The ID or URL-encoded path of the project"),
			mcp.Required(),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get GitLab Project SAST Findings",
			ReadOnlyHint: true,
		}),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return getSecurityFindings(ctx, req, getClient, "SAST", graphqlQuerySAST)
	}
}

// GetProjectDAST defines the MCP tool for retrieving DAST findings
func GetProjectDAST(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"getProjectDAST",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_GET_PROJECT_DAST_DESCRIPTION)),
		mcp.WithString("pid",
			mcp.Description("The ID or URL-encoded path of the project"),
			mcp.Required(),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get GitLab Project DAST Findings",
			ReadOnlyHint: true,
		}),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return getSecurityFindings(ctx, req, getClient, "DAST", graphqlQueryDAST)
	}
}

// GetProjectDependencyScanning defines the MCP tool for retrieving dependency scanning findings
func GetProjectDependencyScanning(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"getProjectDependencyScanning",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_GET_PROJECT_DEPENDENCY_SCANNING_DESCRIPTION)),
		mcp.WithString("pid",
			mcp.Description("The ID or URL-encoded path of the project"),
			mcp.Required(),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get GitLab Project Dependency Scanning Findings",
			ReadOnlyHint: true,
		}),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return getSecurityFindings(ctx, req, getClient, "Dependency Scanning", graphqlQueryDependencyScanning)
	}
}

// GetProjectContainerScanning defines the MCP tool for retrieving container scanning findings
func GetProjectContainerScanning(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"getProjectContainerScanning",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_GET_PROJECT_CONTAINER_SCANNING_DESCRIPTION)),
		mcp.WithString("pid",
			mcp.Description("The ID or URL-encoded path of the project"),
			mcp.Required(),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get GitLab Project Container Scanning Findings",
			ReadOnlyHint: true,
		}),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return getSecurityFindings(ctx, req, getClient, "Container Scanning", graphqlQueryContainerScanning)
	}
}

// GetProjectSecretDetection defines the MCP tool for retrieving secret detection findings
func GetProjectSecretDetection(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"getProjectSecretDetection",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_GET_PROJECT_SECRET_DETECTION_DESCRIPTION)),
		mcp.WithString("pid",
			mcp.Description("The ID or URL-encoded path of the project"),
			mcp.Required(),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get GitLab Project Secret Detection Findings",
			ReadOnlyHint: true,
		}),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return getSecurityFindings(ctx, req, getClient, "Secret Detection", graphqlQuerySecretDetection)
	}
}

// GetProjectLicenseCompliance defines the MCP tool for retrieving license compliance information
func GetProjectLicenseCompliance(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
		"getProjectLicenseCompliance",
		mcp.WithDescription(translations.Translate(t, translations.TOOL_GET_PROJECT_LICENSE_COMPLIANCE_DESCRIPTION)),
		mcp.WithString("pid",
			mcp.Description("The ID or URL-encoded path of the project"),
			mcp.Required(),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:        "Get GitLab Project License Compliance",
			ReadOnlyHint: true,
		}),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return getLicenseCompliance(ctx, req, getClient)
	}
}

// getSecurityFindings is a helper function for security report tools
func getSecurityFindings(ctx context.Context, req mcp.CallToolRequest, getClient GetClientFn, reportType string, query string) (*mcp.CallToolResult, error) {
	client, err := getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	pid, err := requiredParam[string](&req, "pid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
	}

	// Execute GraphQL query
	// Note: Using simple string replacement for fullPath variable
	queryWithPID := strings.Replace(query, "$fullPath", fmt.Sprintf(`"%s"`, pid), 1)

	graphqlReq := gl.GraphQLQuery{
		Query: queryWithPID,
	}

	var responseData ProjectSecurityResponse
	resp, err := client.GraphQL.Do(ctx, graphqlReq, &responseData)
	if err != nil {
		result, apiErr := HandleGraphQLError(err, resp, fmt.Sprintf("%s findings", reportType))
		if result != nil {
			return result, nil
		}
		return nil, apiErr
	}

	// Extract findings from response
	var findings []SecurityFinding
	if len(responseData.Project.Pipelines.Nodes) > 0 {
		findings = responseData.Project.Pipelines.Nodes[0].SecurityReportFindings.Nodes
	}

	if len(findings) == 0 {
		return mcp.NewToolResultText("[]"), nil
	}

	jsonData, err := json.Marshal(findings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal %s findings: %w", reportType, err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

// getLicenseCompliance retrieves license compliance information
func getLicenseCompliance(ctx context.Context, req mcp.CallToolRequest, getClient GetClientFn) (*mcp.CallToolResult, error) {
	client, err := getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	pid, err := requiredParam[string](&req, "pid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
	}

	// Execute GraphQL query
	queryWithPID := strings.Replace(graphqlQueryLicenseCompliance, "$fullPath", fmt.Sprintf(`"%s"`, pid), 1)

	graphqlReq := gl.GraphQLQuery{
		Query: queryWithPID,
	}

	var responseData LicenseComplianceResponse
	resp, err := client.GraphQL.Do(ctx, graphqlReq, &responseData)
	if err != nil {
		result, apiErr := HandleGraphQLError(err, resp, "license compliance")
		if result != nil {
			return result, nil
		}
		return nil, apiErr
	}

	licenses := responseData.Project.Licenses.Nodes

	if len(licenses) == 0 {
		return mcp.NewToolResultText("[]"), nil
	}

	jsonData, err := json.Marshal(licenses)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal license compliance data: %w", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}
