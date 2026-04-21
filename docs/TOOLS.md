# Tools reference

This is a catalog of MCP tools registered by the server. Tools are grouped into toolsets; enable a subset via `--toolsets` or the `GITLAB_TOOLSETS` env var. Every tool accepts a `server` argument (optional) to pick which configured GitLab instance to use.

> **Authoritative schemas live in the code.** Parameter names, types, and descriptions are generated from `pkg/gitlab/*.go` and snapshotted in `pkg/gitlab/__toolsnaps__/*.json`. When in doubt, read the snapshot for the tool — it's the exact JSON schema the LLM sees.

## Action-based consolidation

Many related operations share a single tool distinguished by an `action` parameter. This reduces the total number of tools the LLM must keep in context.

| Tool | Actions |
|---|---|
| `search` | `resourceType` = `projects` / `issues` / `merge_requests` / `blobs` / `commits` / `milestones` / `snippet_titles` / `snippet_blobs` / `wiki_blobs` / `notes`; optional `scope` = `global` / `group` / `project`. |
| `issueComment` | `list`, `create`, `update` |
| `mergeRequestComment` | `list`, `create`, `update` |
| `milestone` | `get`, `create`, `update` |
| `tag` | `get`, `create`, `delete`, `getCommit` |
| `pipelineJob` | `list`, `get`, `trace` |
| `pipeline` | `cancel`, `retry` |
| `manageUserState` | `block`, `unblock`, `ban`, `unban`, `activate`, `deactivate`, `approve` |

## Toolsets

### `projects`

Browse projects, repository files, branches, commits.

| Tool | Mode | Notes |
|---|---|---|
| `getProject` | read | Requires `projectId`. |
| `listProjects` | read | Filters: `search`, `owned`, `membership`, `starred`, `visibility`, `orderBy`, `sort`, `page`, `perPage`. |
| `getProjectFile` | read | Needs `projectId`, `filePath`; optional `ref`. |
| `listProjectFiles` | read | Paginated repo tree. |
| `getProjectBranches` | read | |
| `getProjectCommits` | read | Filters by `ref`, `since`, `until`, `path`. |

### `issues`

| Tool | Mode | Notes |
|---|---|---|
| `getIssue` | read | |
| `listIssues` | read | Filters: `state`, `labels`, `assignee`, `author`, `search`, pagination. |
| `getIssueLabels` | read | |
| `createIssue` | write | |
| `updateIssue` | write | |
| `issueComment` | read/write | `action` = list / create / update. |
| `milestone` | read/write | `action` = get / create / update. |
| `listMilestones` | read | |

### `merge_requests`

| Tool | Mode | Notes |
|---|---|---|
| `getMergeRequest` | read | |
| `listMergeRequests` | read | Filters: `state`, `labels`, `milestone`, `author`, `assignee`, `search`, pagination. |
| `createMergeRequest` | write | |
| `updateMergeRequest` | write | Change title, description, labels, assignees, state. |
| `mergeRequestComment` | read/write | `action` = list / create / update. |

### `pipeline_jobs`

| Tool | Mode | Notes |
|---|---|---|
| `pipelineJob` | read | `action` = list / get / trace. |
| `pipeline` | write | `action` = cancel / retry. |
| `retryPipelineJob` | write | Single job. |
| `playPipelineJob` | write | Manually trigger a `manual` job. |

### `search`

| Tool | Mode | Notes |
|---|---|---|
| `search` | read | Unified; see action table above. |

### `users`

| Tool | Mode | Notes |
|---|---|---|
| `getCurrentUser` | read | The token's own user. |
| `getUser` | read | By ID or username. |
| `getUserStatus` | read | |
| `listUsers` | read | |
| `listProjectUsers` | read | Members of a specific project. |
| `manageUserState` | write | Admin action; see action table above. Requires admin token. |

### `tags`

| Tool | Mode | Notes |
|---|---|---|
| `tag` | read/write | `action` = get / create / delete / getCommit. |
| `listRepositoryTags` | read | |

### `security`

Read-only access to GitLab security scan results. Requires the appropriate GitLab tier for each scanner.

| Tool | Notes |
|---|---|
| `getProjectSAST` | Static analysis results. |
| `getProjectDAST` | Dynamic analysis results. |
| `getProjectDependencyScanning` | Dependency vulnerabilities. |
| `getProjectContainerScanning` | Container image scan results. |
| `getProjectSecretDetection` | Secrets in the codebase. |
| `getProjectLicenseCompliance` | License compliance data. |

### `token_management`

Runtime-only token tools. **Persistent changes go through the CLI** (`gitlab-mcp-server config …`).

| Tool | Mode | Notes |
|---|---|---|
| `listTokens` | read | Inspect tokens tracked in the process. |
| `validateToken` | read | Revalidate one or all tokens. |
| `getNotifications` | read | Accumulated warnings (validation failures, expiry, 401s). |
| `updateToken` | write | Replace a token in memory. Not persisted. |
| `removeToken` | write | Drop a token from the runtime store. |
| `clearNotifications` | write | Empty the notification buffer. |

> The source defines an `addToken` tool but it is not registered in the MCP toolset. Add tokens via `gitlab-mcp-server config add` instead.

### `project_config`

| Tool | Mode | Notes |
|---|---|---|
| `getCurrentProject` | read | Return the effective `.gmcprc`. |
| `setCurrentProject` | write | Write `.gmcprc` in the working directory. |

Project detection utilities (`detectProject`, `autoDetectAndSetProject`) are available when the toolset is enabled; see [PROJECT_CONFIG.md](PROJECT_CONFIG.md).

## Dynamic mode

When started with `--dynamic-toolsets`, the server registers only two discovery tools:

- `list_available_toolsets` — names, descriptions, enabled state.
- `enable_toolset` — enable a toolset; its tools become available immediately.

See [DYNAMIC_TOOLS.md](DYNAMIC_TOOLS.md).

## Migrating from v1.x tool names

| v1.x (removed) | v2.x replacement |
|---|---|
| `searchProjects`, `searchIssues`, `searchMergeRequests`, `searchBlobs`, `searchCommits`, `searchMilestones`, … | `search` with `resourceType` |
| `blockUser`, `unblockUser`, `banUser`, `unbanUser`, `activateUser`, `deactivateUser`, `approveUser` | `manageUserState` with `action` |
| `getIssueComments`, `createIssueComment`, `updateIssueComment` | `issueComment` with `action` |
| `getMergeRequestComments`, `createMergeRequestComment`, `updateMergeRequestComment` | `mergeRequestComment` with `action` |
| `getMilestone`, `createMilestone`, `updateMilestone` | `milestone` with `action` |
| `getRepositoryTag`, `createRepositoryTag`, `deleteRepositoryTag`, `getTagCommit` | `tag` with `action` |
| `listPipelineJobs`, `getPipelineJob`, `getPipelineJobTrace` | `pipelineJob` with `action` |
| `retryPipeline`, `cancelPipeline` | `pipeline` with `action` |

## Where to look for exact schemas

- `pkg/gitlab/__toolsnaps__/*.json` — canonical JSON schema for each tool, committed to the repo.
- The tool definition in `pkg/gitlab/*.go` — look at `mcp.NewTool(…)` calls.
- Dump live: `gitlab-mcp-server stdio --export-translations` writes every translation key, which is a proxy for the tool/parameter inventory.
