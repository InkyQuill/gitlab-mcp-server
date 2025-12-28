# Tools Reference

Complete reference for all available tools in the GitLab MCP Server.

## Projects Toolset

### getProject

Retrieves details of a specific GitLab project.

**Parameters:**
- `projectId` (required, string): Project ID (e.g., "owner/repo" or numeric ID)

**Returns:** Project details including ID, name, path, description, visibility, and more.

**Example:**
```json
{
  "name": "getProject",
  "arguments": {
    "projectId": "mygroup/myproject"
  }
}
```

### listProjects

Returns a list of GitLab projects with optional filtering.

**Parameters:**
- `search` (optional, string): Search projects by name
- `owned` (optional, boolean): Limit to owned projects
- `membership` (optional, boolean): Limit to projects user is a member of
- `starred` (optional, boolean): Limit to starred projects
- `visibility` (optional, string): Filter by visibility (public, internal, private)
- `orderBy` (optional, string): Order by field (id, name, path, created_at, updated_at, last_activity_at)
- `sort` (optional, string): Sort order (asc, desc)
- `page` (optional, number): Page number
- `perPage` (optional, number): Items per page

**Returns:** List of projects matching the criteria.

### getProjectFile

Retrieves the content of a specific file within a GitLab project repository.

**Parameters:**
- `projectId` (required, string): Project ID
- `filePath` (required, string): Path to file within repository
- `ref` (optional, string): Branch, tag, or commit SHA (defaults to default branch)

**Returns:** File content and metadata.

### listProjectFiles

Retrieves a list of files and directories within a specific path in a GitLab project repository.

**Parameters:**
- `projectId` (required, string): Project ID
- `path` (optional, string): Path inside repository (defaults to root)
- `ref` (optional, string): Branch, tag, or commit SHA
- `recursive` (optional, boolean): List files recursively
- `page` (optional, number): Page number
- `perPage` (optional, number): Items per page

**Returns:** List of files and directories.

### getProjectBranches

Retrieves a list of repository branches from a project.

**Parameters:**
- `projectId` (required, string): Project ID
- `search` (optional, string): Search branches by name
- `page` (optional, number): Page number
- `perPage` (optional, number): Items per page

**Returns:** List of branches.

### getProjectCommits

Retrieves a list of repository commits in a project.

**Parameters:**
- `projectId` (required, string): Project ID
- `ref` (optional, string): Branch, tag, or commit SHA
- `path` (optional, string): File path to retrieve commits for
- `since` (optional, string): Only commits after this date (ISO 8601)
- `until` (optional, string): Only commits before this date (ISO 8601)
- `withStats` (optional, boolean): Include commit stats
- `page` (optional, number): Page number
- `perPage` (optional, number): Items per page

**Returns:** List of commits.

## Issues Toolset

### getIssue

Retrieves details for a specific GitLab issue.

**Parameters:**
- `projectId` (required, string): Project ID
- `issueIid` (required, number): Issue IID (internal ID)

**Returns:** Issue details including title, description, state, labels, assignees, and more.

### listIssues

Retrieves a list of issues in a GitLab project with pagination and filtering.

**Parameters:**
- `projectId` (required, string): Project ID
- `state` (optional, string): Return issues with state (opened, closed, all)
- `labels` (optional, string): Comma-separated list of label names
- `milestone` (optional, string): Milestone title
- `scope` (optional, string): Return issues for scope (created_by_me, assigned_to_me, all)
- `search` (optional, string): Search issues against title and description
- `orderBy` (optional, string): Order by field (created_at, updated_at, priority)
- `sort` (optional, string): Sort order (asc, desc)
- `page` (optional, number): Page number
- `perPage` (optional, number): Items per page

**Returns:** List of issues.

### getIssueComments

Retrieves comments or notes from a specific issue.

**Parameters:**
- `projectId` (required, string): Project ID
- `issueIid` (required, number): Issue IID
- `page` (optional, number): Page number
- `perPage` (optional, number): Items per page

**Returns:** List of issue comments.

### getIssueLabels

Retrieves the labels associated with a specific issue.

**Parameters:**
- `projectId` (required, string): Project ID
- `issueIid` (required, number): Issue IID

**Returns:** List of labels.

### createIssue

Creates a new issue in a GitLab project.

**Parameters:**
- `projectId` (required, string): Project ID
- `title` (required, string): Issue title
- `description` (optional, string): Issue description
- `labels` (optional, string): Comma-separated list of label names
- `assigneeIds` (optional, string): Comma-separated list of user IDs
- `milestoneId` (optional, number): Milestone ID
- `dueDate` (optional, string): Due date (ISO 8601 format: YYYY-MM-DD)

**Returns:** Created issue details.

### updateIssue

Updates an existing issue in a GitLab project.

**Parameters:**
- `projectId` (required, string): Project ID
- `issueIid` (required, number): Issue IID
- `title` (optional, string): New title
- `description` (optional, string): New description
- `stateEvent` (optional, string): State event (close, reopen)
- `labels` (optional, string): Comma-separated list of label names
- `assigneeIds` (optional, string): Comma-separated list of user IDs
- `milestoneId` (optional, number): Milestone ID
- `dueDate` (optional, string): Due date (ISO 8601 format: YYYY-MM-DD)

**Returns:** Updated issue details.

### createIssueComment

Creates a comment (note) on a specific issue.

**Parameters:**
- `projectId` (required, string): Project ID
- `issueIid` (required, number): Issue IID
- `body` (required, string): Comment content

**Returns:** Created comment details.

### updateIssueComment

Updates an existing comment (note) on a specific issue.

**Parameters:**
- `projectId` (required, string): Project ID
- `issueIid` (required, number): Issue IID
- `noteId` (required, number): Note (comment) ID
- `body` (required, string): Updated comment content

**Returns:** Updated comment details.

## Merge Requests Toolset

### getMergeRequest

Retrieves details for a specific GitLab merge request.

**Parameters:**
- `projectId` (required, string): Project ID
- `mergeRequestIid` (required, number): Merge request IID

**Returns:** Merge request details.

### listMergeRequests

Lists merge requests for a GitLab project with filtering and pagination options.

**Parameters:**
- `projectId` (required, string): Project ID
- `state` (optional, string): Return MRs with state (opened, closed, locked, merged, all)
- `labels` (optional, string): Comma-separated list of label names
- `milestone` (optional, string): Milestone title
- `scope` (optional, string): Return MRs for scope (created_by_me, assigned_to_me, all)
- `search` (optional, string): Search MRs matching query in title or description
- `orderBy` (optional, string): Order by field (created_at, updated_at, title)
- `sort` (optional, string): Sort order (asc, desc)
- `page` (optional, number): Page number
- `perPage` (optional, number): Items per page

**Returns:** List of merge requests.

### getMergeRequestComments

Retrieves comments or notes from a specific merge request.

**Parameters:**
- `projectId` (required, string): Project ID
- `mergeRequestIid` (required, number): Merge request IID
- `page` (optional, number): Page number
- `perPage` (optional, number): Items per page

**Returns:** List of merge request comments.

### createMergeRequest

Creates a new merge request in a GitLab project.

**Parameters:**
- `projectId` (required, string): Project ID
- `sourceBranch` (required, string): Source branch name
- `targetBranch` (required, string): Target branch name
- `title` (required, string): Merge request title
- `description` (optional, string): Merge request description
- `labels` (optional, string): Comma-separated list of label names
- `assigneeIds` (optional, string): Comma-separated list of user IDs
- `milestoneId` (optional, number): Milestone ID
- `removeSourceBranch` (optional, boolean): Remove source branch after merge
- `squash` (optional, boolean): Squash commits into single commit

**Returns:** Created merge request details.

### updateMergeRequest

Updates an existing merge request in a GitLab project.

**Parameters:**
- `projectId` (required, string): Project ID
- `mergeRequestIid` (required, number): Merge request IID
- `title` (optional, string): New title
- `description` (optional, string): New description
- `stateEvent` (optional, string): State event (close, reopen, merge)
- `targetBranch` (optional, string): New target branch
- `labels` (optional, string): Comma-separated list of label names
- `assigneeIds` (optional, string): Comma-separated list of user IDs
- `milestoneId` (optional, number): Milestone ID
- `removeSourceBranch` (optional, boolean): Remove source branch after merge
- `squash` (optional, boolean): Squash commits into single commit

**Returns:** Updated merge request details.

### createMergeRequestComment

Creates a comment (note) on a specific merge request.

**Parameters:**
- `projectId` (required, string): Project ID
- `mergeRequestIid` (required, number): Merge request IID
- `body` (required, string): Comment content

**Returns:** Created comment details.

### updateMergeRequestComment

Updates an existing comment (note) on a specific merge request.

**Parameters:**
- `projectId` (required, string): Project ID
- `mergeRequestIid` (required, number): Merge request IID
- `noteId` (required, number): Note (comment) ID
- `body` (required, string): Updated comment content

**Returns:** Updated comment details.

## Milestones

### getMilestone

Retrieves details for a specific GitLab milestone.

**Parameters:**
- `projectId` (required, string): Project ID
- `milestoneId` (required, number): Milestone ID

**Returns:** Milestone details.

### listMilestones

Retrieves a list of milestones in a GitLab project with pagination and filtering.

**Parameters:**
- `projectId` (required, string): Project ID
- `state` (optional, string): Return milestones with state (active, closed, all)
- `search` (optional, string): Search milestones against title and description
- `page` (optional, number): Page number
- `perPage` (optional, number): Items per page

**Returns:** List of milestones.

### createMilestone

Creates a new milestone in a GitLab project.

**Parameters:**
- `projectId` (required, string): Project ID
- `title` (required, string): Milestone title
- `description` (optional, string): Milestone description
- `dueDate` (optional, string): Due date (ISO 8601 format: YYYY-MM-DD)
- `startDate` (optional, string): Start date (ISO 8601 format: YYYY-MM-DD)

**Returns:** Created milestone details.

### updateMilestone

Updates an existing milestone in a GitLab project.

**Parameters:**
- `projectId` (required, string): Project ID
- `milestoneId` (required, number): Milestone ID
- `title` (optional, string): New title
- `description` (optional, string): New description
- `dueDate` (optional, string): Due date (ISO 8601 format: YYYY-MM-DD)
- `startDate` (optional, string): Start date (ISO 8601 format: YYYY-MM-DD)
- `stateEvent` (optional, string): State event (activate, close)

**Returns:** Updated milestone details.

## Search

The search toolset provides scoped search capabilities across GitLab. Available tools include:

- `searchProjects` - Search for projects
- `searchIssues` - Search issues (global or project-scoped)
- `searchMergeRequests` - Search merge requests (global or project-scoped)
- `searchBlobs` - Search code across repositories
- `searchCommits` - Search commits in a project
- `searchMilestones` - Search milestones (global or project-scoped)
- Plus group/project-scoped variants

Each search tool supports:
- `scope` (required, string): Search scope (projects, issues, merge_requests, blobs, commits, milestones)
- `search` (required, string): Search query
- `projectId` (optional, string): Project ID for project-scoped search
- `groupId` (optional, string): Group ID for group-scoped search

## Users

### getCurrentUser

Retrieves the current authenticated user's details.

**Parameters:** None

**Returns:** Current user details.

### getUser

Retrieves details for a specific user.

**Parameters:**
- `userId` (required, number): User ID

**Returns:** User details.

### getUserStatus

Retrieves the status of a specific user.

**Parameters:**
- `userId` (required, number): User ID

**Returns:** User status.

### listUsers

Lists users with optional filtering.

**Parameters:**
- `search` (optional, string): Search users
- `page` (optional, number): Page number
- `perPage` (optional, number): Items per page

**Returns:** List of users.

### listProjectUsers

Lists users with access to a project.

**Parameters:**
- `projectId` (required, string): Project ID
- `search` (optional, string): Search users
- `page` (optional, number): Page number
- `perPage` (optional, number): Items per page

**Returns:** List of project users.

### Admin Operations

Admin-only operations (require admin permissions):

- `blockUser` - Block a user
- `unblockUser` - Unblock a user
- `banUser` - Ban a user
- `unbanUser` - Unban a user
- `activateUser` - Activate a user
- `deactivateUser` - Deactivate a user
- `approveUser` - Approve a user

Each takes `userId` (required, number) as parameter.

## Security

Security scanning tools (may require GitLab Premium/Ultimate):

- `getProjectSAST` - Get SAST scan results
- `getProjectDAST` - Get DAST scan results
- `getProjectDependencyScanning` - Get dependency scanning results
- `getProjectContainerScanning` - Get container scanning results
- `getProjectSecretDetection` - Get secret detection results
- `getProjectLicenseCompliance` - Get license compliance data

Each requires:
- `projectId` (required, string): Project ID

## Token Management

### listTokens

Lists all configured tokens with their validation status and metadata.

**Parameters:** None

**Returns:** List of tokens with validation status.

### validateToken

Manually validates a GitLab token by calling the GitLab API.

**Parameters:**
- `name` (optional, string): Token name to validate. If not provided, validates all tokens.

**Returns:** Validation result.

### addToken

Adds a new GitLab token configuration to the runtime token store.

**Parameters:**
- `name` (required, string): Token/server name (e.g., 'work', 'personal')
- `token` (required, string): GitLab Personal Access Token
- `gitlabHost` (optional, string): GitLab host URL (default: https://gitlab.com)

**Returns:** Token addition result.

**Note:** Tokens added via `addToken` are only stored in runtime memory and will be lost when the server restarts.

### updateToken

Updates an existing GitLab token. Validates the new token before updating.

**Parameters:**
- `name` (required, string): Token name to update
- `token` (optional, string): New token value (if not provided, only revalidates existing token)

**Returns:** Update result.

### removeToken

Removes a token from the runtime token store.

**Parameters:**
- `name` (required, string): Token name to remove

**Returns:** Removal result.

**Note:** This only removes the token from runtime memory. It does not update your MCP configuration files.

### getNotifications

Returns recent notifications about token issues, validation results, and other important messages.

**Parameters:** None

**Returns:** List of notifications.

### clearNotifications

Clears all stored notifications.

**Parameters:** None

**Returns:** Clear result.

## Project Configuration

### getCurrentProject

Gets the current GitLab project configuration from the `.gmcprc` file.

**Parameters:** None

**Returns:** Current project configuration.

### setCurrentProject

Sets the current GitLab project for this directory by creating a `.gmcprc` file.

**Parameters:**
- `projectId` (required, string): GitLab project ID (e.g., 'owner/repo' or numeric ID)
- `gitlabHost` (optional, string): GitLab host URL (e.g., 'https://gitlab.example.com')

**Returns:** Configuration result.

### detectProject

Auto-detects the GitLab project from the Git remote URL in the current directory.

**Parameters:** None

**Returns:** Detected project information.

### autoDetectAndSetProject

Auto-detects the GitLab project from Git remote and creates a `.gmcprc` file. Combines detection and setting in one step.

**Parameters:** None

**Returns:** Configuration result.

## Tags Toolset

### listRepositoryTags

Lists all tags in a GitLab repository with optional filtering and pagination.

**Parameters:**
- `projectId` (required, string): Project ID
- `search` (optional, string): Return tags matching the search criteria
- `page` (optional, number): Page number
- `perPage` (optional, number): Items per page

**Returns:** List of repository tags with commit information.

### getRepositoryTag

Retrieves details of a specific tag in a repository.

**Parameters:**
- `projectId` (required, string): Project ID
- `tagName` (required, string): The name of the tag

**Returns:** Tag details including name, commit, message, and release information.

### createRepositoryTag

Creates a new tag in the repository (annotated or lightweight).

**Parameters:**
- `projectId` (required, string): Project ID
- `tagName` (required, string): The name of the tag
- `ref` (required, string): The branch name or commit SHA to create the tag from
- `message` (optional, string): The message for the tag annotation (creates annotated tag)

**Returns:** Created tag details.

### deleteRepositoryTag

Deletes a tag from the repository.

**Parameters:**
- `projectId` (required, string): Project ID
- `tagName` (required, string): The name of the tag to delete

**Returns:** Success message.

### getTagCommit

Retrieves release/commit information for a specific tag.

**Parameters:**
- `projectId` (required, string): Project ID
- `tagName` (required, string): The name of the tag

**Returns:** Release information including commit details, description, and assets.

## Pipeline Jobs Toolset

### listPipelineJobs

Lists all jobs in a GitLab CI/CD pipeline.

**Parameters:**
- `projectId` (required, string): Project ID
- `pipelineId` (required, number): The ID of the pipeline
- `page` (optional, number): Page number
- `perPage` (optional, number): Items per page

**Returns:** List of pipeline jobs with status, stage, and execution details.

### getPipelineJob

Retrieves details of a specific pipeline job.

**Parameters:**
- `projectId` (required, string): Project ID
- `jobId` (required, number): The ID of the job

**Returns:** Job details including name, status, stage, duration, and artifacts.

### getPipelineJobTrace

Retrieves the log trace output from a pipeline job.

**Parameters:**
- `projectId` (required, string): Project ID
- `jobId` (required, number): The ID of the job

**Returns:** Raw log output from the job execution.

### retryPipelineJob

Retries a failed or canceled job.

**Parameters:**
- `projectId` (required, string): Project ID
- `jobId` (required, number): The ID of the job to retry

**Returns:** Details of the newly created job.

### playPipelineJob

Triggers a manual job to start (e.g., a deployment job that requires manual approval).

**Parameters:**
- `projectId` (required, string): Project ID
- `jobId` (required, number): The ID of the job to play

**Returns:** Details of the triggered job.

### cancelPipeline

Cancels a running pipeline.

**Parameters:**
- `projectId` (required, string): Project ID
- `pipelineId` (required, number): The ID of the pipeline to cancel

**Returns:** Updated pipeline details with canceled status.

### retryPipeline

Retries all failed jobs in a pipeline.

**Parameters:**
- `projectId` (required, string): Project ID
- `pipelineId` (required, number): The ID of the pipeline to retry

**Returns:** Details of the newly created pipeline.

## Error Handling

All tools return consistent error responses:

- **401 Unauthorized**: Token is invalid or expired
- **403 Forbidden**: Insufficient permissions
- **404 Not Found**: Resource not found
- **422 Unprocessable Entity**: Validation error
- **500 Internal Server Error**: Server error

Error responses include:
- Error code
- Error message
- Additional context when available

## Rate Limiting

GitLab API has rate limits. The server respects these limits and will return appropriate errors when limits are exceeded. Consider implementing retry logic in your applications.

## Related Documentation

- [Installation Guide](INSTALLATION.md)
- [Editor Setup](EDITOR_SETUP.md)
- [Token Management](TOKEN_MANAGEMENT.md)
- [Project Configuration](PROJECT_CONFIG.md)
- [Multi-Server Setup](MULTI_SERVER_SETUP.md)

