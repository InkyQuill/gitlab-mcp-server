package gitlab

import (
	"context" // Added for GetClientFn
	"fmt"

	// Import necessary packages, including your toolsets package
	"github.com/InkyQuill/gitlab-mcp-server/pkg/toolsets" // Adjust path if needed
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"         // Import logger
	gl "gitlab.com/gitlab-org/api/client-go" // Import the GitLab client library
	// "github.com/InkyQuill/gitlab-mcp-server/pkg/translations" // Removed for now
)

// GetClientFn defines the function signature for retrieving an initialized GitLab client.
// This allows decoupling toolset initialization from direct client creation.
type GetClientFn func(context.Context) (*gl.Client, error)

type ServerPolicy struct {
	Name     string
	ReadOnly bool
}

type ServerPolicyFn func(context.Context) (ServerPolicy, error)

func newGitLabReadTool(tool mcp.Tool, handler server.ToolHandlerFunc) server.ServerTool {
	tool, handler = WithServerSelection(tool, handler)
	return toolsets.NewServerTool(tool, handler)
}

func newGitLabWriteTool(policy ServerPolicyFn, tool mcp.Tool, handler server.ToolHandlerFunc) server.ServerTool {
	if handler == nil || policy == nil {
		tool, handler = WithServerSelection(tool, handler)
		return toolsets.NewServerTool(tool, handler)
	}
	guardedHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		p, err := policy(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to resolve server policy: %v", err)), nil
		}
		if p.ReadOnly {
			serverName := p.Name
			if serverName == "" {
				serverName = "selected"
			}
			return mcp.NewToolResultError(fmt.Sprintf("server %q is configured read-only; write tool %q is blocked", serverName, tool.Name)), nil
		}
		return handler(ctx, request)
	}
	tool, guardedHandler = WithServerSelection(tool, guardedHandler)
	return toolsets.NewServerTool(tool, guardedHandler)
}

// DefaultTools defines the list of toolsets enabled by default.
var DefaultTools = []string{"all"}

// InitToolsets initializes the ToolsetGroup with GitLab-specific toolsets.
// It accepts a function to retrieve the GitLab client.
func InitToolsets(
	enabledToolsets []string,
	readOnly bool,
	getClient GetClientFn, // Restore parameter name
	logger *log.Logger, // Logger for notifications
	tokenStore *TokenStore, // Token store for token management
	translations map[string]string, // Translation map for i18n
	dynamicMode bool, // Enable dynamic toolset discovery mode
	serverPolicy ServerPolicyFn,
) (*toolsets.ToolsetGroup, error) {

	// 1. Create the ToolsetGroup
	tg := toolsets.NewToolsetGroup(readOnly)

	// Log dynamic mode status
	if dynamicMode {
		logger.Info("Dynamic toolset mode enabled - toolsets will be loaded on-demand")
	}

	// 2. Define Toolsets (as per PDR section 5.3)
	tokenManagementTS := toolsets.NewToolset("token_management", "Tools for managing GitLab tokens and authentication.")
	projectConfigTS := toolsets.NewToolset("project_config", "Tools for managing GitLab project configuration and auto-detection.")
	projectsTS := toolsets.NewToolset("projects", "Tools for interacting with GitLab projects, repositories, branches, commits, tags.")
	issuesTS := toolsets.NewToolset("issues", "Tools for CRUD operations on GitLab issues, comments, labels.")
	mergeRequestsTS := toolsets.NewToolset("merge_requests", "Tools for CRUD operations on GitLab merge requests, comments, approvals, diffs.")
	securityTS := toolsets.NewToolset("security", "Tools for accessing GitLab security scan results (SAST, DAST, etc.).")
	usersTS := toolsets.NewToolset("users", "Tools for looking up GitLab user information.")
	searchTS := toolsets.NewToolset("search", "Tools for utilizing GitLab's scoped search capabilities.")
	tagsTS := toolsets.NewToolset("tags", "Tools for managing GitLab repository tags and releases.")
	pipelineJobsTS := toolsets.NewToolset("pipeline_jobs", "Tools for monitoring and controlling GitLab CI/CD pipeline jobs.")

	// 3. Add Tools to Toolsets (Actual tool implementation TBD in separate tasks)
	//    Tool definition functions will need to accept GetClientFn or call it.
	//    Example (placeholder):
	//    getProjectTool := toolsets.NewServerTool(GetProject(getClient, t))
	newWriteTool := func(tool mcp.Tool, handler server.ToolHandlerFunc) server.ServerTool {
		return newGitLabWriteTool(serverPolicy, tool, handler)
	}

	// --- Add tools to tokenManagementTS (Token management) ---
	tokenManagementTS.AddReadTools(
		toolsets.NewServerTool(ListTokens(tokenStore)),
		toolsets.NewServerTool(ValidateToken(nil, logger, tokenStore)),
		toolsets.NewServerTool(GetNotificationsTool(logger)),
	)
	tokenManagementTS.AddWriteTools(
		toolsets.NewServerTool(UpdateToken(nil, logger, tokenStore)),
		toolsets.NewServerTool(RemoveToken(tokenStore)),
		toolsets.NewServerTool(ClearNotificationsTool(logger)),
	)

	// --- Add tools to projectConfigTS (Project configuration management) ---
	projectConfigTS.AddReadTools(
		toolsets.NewServerTool(GetCurrentProject(getClient, tokenStore)),
	)
	projectConfigTS.AddWriteTools(
		toolsets.NewServerTool(SetCurrentProject(getClient, tokenStore)),
	)

	// --- Add tools to projectsTS (Task 7 & 12) ---
	projectsTS.AddReadTools(
		newGitLabReadTool(GetProject(getClient, translations)),
		newGitLabReadTool(ListProjects(getClient, translations)),
		newGitLabReadTool(GetProjectFile(getClient, translations)),
		newGitLabReadTool(ListProjectFiles(getClient, translations)),
		newGitLabReadTool(GetProjectBranches(getClient, translations)),
		newGitLabReadTool(GetProjectCommits(getClient, translations)),
	)
	// projectsTS.AddWriteTools(...)

	// --- Add tools to issuesTS (Task 8 & 13) ---
	issuesTS.AddReadTools(
		newGitLabReadTool(GetIssue(getClient, translations)),
		newGitLabReadTool(ListIssues(getClient, translations)),
		newGitLabReadTool(GetIssueLabels(getClient, translations)),
		// Milestones list tool
		newGitLabReadTool(ListMilestones(getClient, translations)),
	)
	issuesTS.AddWriteTools(
		newWriteTool(CreateIssue(getClient, translations)),
		newWriteTool(UpdateIssue(getClient, translations)),
		newWriteTool(IssueComment(getClient, translations)),
		// Milestones write tools
		newWriteTool(Milestone(getClient, translations)),
	)

	// --- Add tools to mergeRequestsTS (Task 9 & 14) ---
	mergeRequestsTS.AddReadTools(
		newGitLabReadTool(GetMergeRequest(getClient, translations)),
		newGitLabReadTool(ListMergeRequests(getClient, translations)),
	)
	mergeRequestsTS.AddWriteTools(
		newWriteTool(CreateMergeRequest(getClient, translations)),
		newWriteTool(UpdateMergeRequest(getClient, translations)),
		newWriteTool(MergeRequestComment(getClient, translations)),
	)

	// --- Add tools to securityTS (Security scanning reports) ---
	securityTS.AddReadTools(
		newGitLabReadTool(GetProjectSAST(getClient, translations)),
		newGitLabReadTool(GetProjectDAST(getClient, translations)),
		newGitLabReadTool(GetProjectDependencyScanning(getClient, translations)),
		newGitLabReadTool(GetProjectContainerScanning(getClient, translations)),
		newGitLabReadTool(GetProjectSecretDetection(getClient, translations)),
		newGitLabReadTool(GetProjectLicenseCompliance(getClient, translations)),
	)

	// --- Add tools to usersTS (User management) ---
	usersTS.AddReadTools(
		newGitLabReadTool(GetCurrentUser(getClient, translations)),
		newGitLabReadTool(GetUser(getClient, translations)),
		newGitLabReadTool(GetUserStatus(getClient, translations)),
		newGitLabReadTool(ListUsers(getClient, translations)),
		newGitLabReadTool(ListProjectUsers(getClient, translations)),
	)
	usersTS.AddWriteTools(
		newWriteTool(ManageUserState(getClient, translations)),
	)

	// --- Add tools to searchTS (Search capabilities) ---
	searchTS.AddReadTools(
		newGitLabReadTool(Search(getClient, translations)),
	)

	// --- Add tools to tagsTS (Tags Management) ---
	tagsTS.AddReadTools(
		newGitLabReadTool(ListRepositoryTags(getClient, translations)),
	)
	tagsTS.AddWriteTools(
		newWriteTool(Tag(getClient, translations)),
	)

	// --- Add tools to pipelineJobsTS (CI/CD Pipeline Jobs) ---
	pipelineJobsTS.AddReadTools(
		newGitLabReadTool(PipelineJob(getClient, translations)),
	)
	pipelineJobsTS.AddWriteTools(
		newWriteTool(Pipeline(getClient, translations)),
		newWriteTool(RetryPipelineJob(getClient, translations)),
		newWriteTool(PlayPipelineJob(getClient, translations)),
	)

	// 4. Add defined Toolsets to the Group
	tg.AddToolset(tokenManagementTS)
	tg.AddToolset(projectConfigTS)
	tg.AddToolset(projectsTS)
	tg.AddToolset(issuesTS)
	tg.AddToolset(mergeRequestsTS)
	tg.AddToolset(securityTS)
	tg.AddToolset(usersTS)
	tg.AddToolset(searchTS)
	tg.AddToolset(tagsTS)
	tg.AddToolset(pipelineJobsTS)

	// 5. Enable Toolsets based on configuration
	// In dynamic mode, toolsets are enabled on-demand, so we skip this step
	if !dynamicMode {
		err := tg.EnableToolsets(enabledToolsets)
		if err != nil {
			// Consider logging the error here in a real implementation
			return nil, err // Return error if enabling failed (e.g., unknown toolset name)
		}
	}

	// 6. Return the configured group
	return tg, nil
}
