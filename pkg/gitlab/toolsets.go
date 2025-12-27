package gitlab

import (
	"context" // Added for GetClientFn
	// Import necessary packages, including your toolsets package
	"github.com/InkyQuill/gitlab-mcp-server/pkg/toolsets" // Adjust path if needed
	gl "gitlab.com/gitlab-org/api/client-go"                   // Import the GitLab client library
	log "github.com/sirupsen/logrus"                           // Import logger
	// "github.com/InkyQuill/gitlab-mcp-server/pkg/translations" // Removed for now
)

// GetClientFn defines the function signature for retrieving an initialized GitLab client.
// This allows decoupling toolset initialization from direct client creation.
type GetClientFn func(context.Context) (*gl.Client, error)

// DefaultTools defines the list of toolsets enabled by default.
var DefaultTools = []string{"all"}

// InitToolsets initializes the ToolsetGroup with GitLab-specific toolsets.
// It accepts a function to retrieve the GitLab client.
func InitToolsets(
	enabledToolsets []string,
	readOnly bool,
	getClient GetClientFn,   // Restore parameter name
	logger *log.Logger,      // Logger for notifications
	tokenStore *TokenStore,  // Token store for token management
	translations map[string]string, // Translation map for i18n
	dynamicMode bool, // Enable dynamic toolset discovery mode
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

	// 3. Add Tools to Toolsets (Actual tool implementation TBD in separate tasks)
	//    Tool definition functions will need to accept GetClientFn or call it.
	//    Example (placeholder):
	//    getProjectTool := toolsets.NewServerTool(GetProject(getClient, t))

	// --- Add tools to tokenManagementTS (Token management) ---
	tokenManagementTS.AddReadTools(
		toolsets.NewServerTool(ListTokens(tokenStore)),
		toolsets.NewServerTool(ValidateToken(nil, logger, tokenStore)),
		toolsets.NewServerTool(GetNotificationsTool(logger)),
	)
	tokenManagementTS.AddWriteTools(
		toolsets.NewServerTool(AddToken(nil, logger, tokenStore)),
		toolsets.NewServerTool(UpdateToken(nil, logger, tokenStore)),
		toolsets.NewServerTool(RemoveToken(tokenStore)),
		toolsets.NewServerTool(ClearNotificationsTool(logger)),
	)

	// --- Add tools to projectConfigTS (Project configuration management) ---
	projectConfigTS.AddReadTools(
		toolsets.NewServerTool(GetCurrentProject(getClient)),
		toolsets.NewServerTool(DetectProject(getClient)),
	)
	projectConfigTS.AddWriteTools(
		toolsets.NewServerTool(SetCurrentProject(getClient)),
		toolsets.NewServerTool(AutoDetectAndSetProject(getClient)),
	)

	// --- Add tools to projectsTS (Task 7 & 12) ---
	projectsTS.AddReadTools(
		toolsets.NewServerTool(GetProject(getClient, translations)),
		toolsets.NewServerTool(ListProjects(getClient, translations)),
		toolsets.NewServerTool(GetProjectFile(getClient, translations)),
		toolsets.NewServerTool(ListProjectFiles(getClient, translations)),
		toolsets.NewServerTool(GetProjectBranches(getClient, translations)),
		toolsets.NewServerTool(GetProjectCommits(getClient, translations)),
	)
	// projectsTS.AddWriteTools(...)

	// --- Add tools to issuesTS (Task 8 & 13) ---
	issuesTS.AddReadTools(
		toolsets.NewServerTool(GetIssue(getClient, translations)),
		toolsets.NewServerTool(ListIssues(getClient, translations)),
		toolsets.NewServerTool(GetIssueComments(getClient, translations)),
		toolsets.NewServerTool(GetIssueLabels(getClient, translations)),
		// Milestones read tools (milestones are related to issues)
		toolsets.NewServerTool(GetMilestone(getClient, translations)),
		toolsets.NewServerTool(ListMilestones(getClient, translations)),
	)
	issuesTS.AddWriteTools(
		toolsets.NewServerTool(CreateIssue(getClient, translations)),
		toolsets.NewServerTool(UpdateIssue(getClient, translations)),
		toolsets.NewServerTool(CreateIssueComment(getClient, translations)),
		toolsets.NewServerTool(UpdateIssueComment(getClient, translations)),
		// Milestones write tools
		toolsets.NewServerTool(CreateMilestone(getClient, translations)),
		toolsets.NewServerTool(UpdateMilestone(getClient, translations)),
	)

	// --- Add tools to mergeRequestsTS (Task 9 & 14) ---
	mergeRequestsTS.AddReadTools(
		toolsets.NewServerTool(GetMergeRequest(getClient, translations)),
		toolsets.NewServerTool(ListMergeRequests(getClient, translations)),
		toolsets.NewServerTool(GetMergeRequestComments(getClient, translations)),
	)
	mergeRequestsTS.AddWriteTools(
		toolsets.NewServerTool(CreateMergeRequest(getClient, translations)),
		toolsets.NewServerTool(UpdateMergeRequest(getClient, translations)),
		toolsets.NewServerTool(CreateMergeRequestComment(getClient, translations)),
		toolsets.NewServerTool(UpdateMergeRequestComment(getClient, translations)),
	)

	// --- Add tools to securityTS (Part of future tasks?) ---
	// securityTS.AddReadTools(...) // Likely read-only

	// --- Add tools to usersTS (Task 10) ---
	// usersTS.AddReadTools(...) // Likely read-only

	// --- Add tools to searchTS (Task 10/11) ---
	// searchTS.AddReadTools(...) // Likely read-only

	// 4. Add defined Toolsets to the Group
	tg.AddToolset(tokenManagementTS)
	tg.AddToolset(projectConfigTS)
	tg.AddToolset(projectsTS)
	tg.AddToolset(issuesTS)
	tg.AddToolset(mergeRequestsTS)
	tg.AddToolset(securityTS)
	tg.AddToolset(usersTS)
	tg.AddToolset(searchTS)

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
