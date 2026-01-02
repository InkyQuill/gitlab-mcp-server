# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2025-12-29

### Breaking Changes

This release introduces a major consolidation of tools, reducing the total tool count from ~70 to ~35. This change optimizes the context window usage for AI models and makes the API more intuitive by using an `action` parameter pattern.

#### Search Tools (19 tools → 1 tool)

**Removed Tools:**
- `searchProjects`
- `searchIssues`
- `searchMergeRequests`
- `searchBlobs`
- `searchCommits`
- `searchMilestones`
- `searchSnippetTitles`
- `searchSnippetBlobs`
- `searchWikiBlobs`
- `searchNotes`
- `searchBlobsByGroup`
- `searchIssuesByGroup`
- `searchMergeRequestsByGroup`
- `searchMilestonesByGroup`
- `searchProjectsByGroup`
- `searchBlobsByProject`
- `searchCommitsByProject`
- `searchIssuesByProject`
- `searchMergeRequestsByProject`

**New Tool:**
- `search` - Unified search tool with `resourceType` parameter and optional `scope` parameter (global/group/project)

**Migration:**
```json
// Old: searchIssues
{"tool": "searchIssues", "search": "bug"}

// New: search
{
  "tool": "search",
  "resourceType": "issues",
  "search": "bug",
  "scope": "global"
}

// Old: searchIssuesByProject
{
  "tool": "searchIssuesByProject",
  "pid": "123",
  "search": "bug"
}

// New: search
{
  "tool": "search",
  "resourceType": "issues",
  "search": "bug",
  "scope": "project",
  "pid": "123"
}
```

#### User Management Tools (7 tools → 1 tool)

**Removed Tools:**
- `blockUser`
- `unblockUser`
- `banUser`
- `unbanUser`
- `activateUser`
- `deactivateUser`
- `approveUser`

**New Tool:**
- `manageUserState` - Single tool with `action` parameter (block/unblock/ban/unban/activate/deactivate/approve)

**Migration:**
```json
// Old: blockUser
{"tool": "blockUser", "userId": 123}

// New: manageUserState
{
  "tool": "manageUserState",
  "action": "block",
  "userId": 123
}
```

#### Issue Comment Tools (3 tools → 1 tool)

**Removed Tools:**
- `getIssueComments`
- `createIssueComment`
- `updateIssueComment`

**New Tool:**
- `issueComment` - Single tool with `action` parameter (list/create/update)

**Migration:**
```json
// Old: getIssueComments
{
  "tool": "getIssueComments",
  "projectId": "123",
  "issueIid": 456
}

// New: issueComment
{
  "tool": "issueComment",
  "action": "list",
  "projectId": "123",
  "issueIid": 456
}

// Old: createIssueComment
{
  "tool": "createIssueComment",
  "projectId": "123",
  "issueIid": 456,
  "body": "New comment"
}

// New: issueComment
{
  "tool": "issueComment",
  "action": "create",
  "projectId": "123",
  "issueIid": 456,
  "body": "New comment"
}
```

#### Merge Request Comment Tools (3 tools → 1 tool)

**Removed Tools:**
- `getMergeRequestComments`
- `createMergeRequestComment`
- `updateMergeRequestComment`

**New Tool:**
- `mergeRequestComment` - Single tool with `action` parameter (list/create/update)

**Migration:**
```json
// Old: getMergeRequestComments
{
  "tool": "getMergeRequestComments",
  "projectId": "123",
  "mergeRequestIid": 789
}

// New: mergeRequestComment
{
  "tool": "mergeRequestComment",
  "action": "list",
  "projectId": "123",
  "mergeRequestIid": 789
}
```

#### Milestone Tools (3 tools → 1 tool + 1 unchanged)

**Removed Tools:**
- `getMilestone`
- `createMilestone`
- `updateMilestone`

**New Tools:**
- `milestone` - Consolidated tool with `action` parameter (get/create/update)
- `listMilestones` - Unchanged (still separate tool)

**Migration:**
```json
// Old: getMilestone
{
  "tool": "getMilestone",
  "projectId": "123",
  "milestoneId": 456
}

// New: milestone
{
  "tool": "milestone",
  "action": "get",
  "projectId": "123",
  "milestoneId": 456
}
```

#### Tag Management Tools (4 tools → 1 tool + 1 unchanged)

**Removed Tools:**
- `getRepositoryTag`
- `createRepositoryTag`
- `deleteRepositoryTag`
- `getTagCommit`

**New Tools:**
- `tag` - Consolidated tool with `action` parameter (get/create/delete/getCommit)
- `listRepositoryTags` - Unchanged (still separate tool)

**Migration:**
```json
// Old: getRepositoryTag
{
  "tool": "getRepositoryTag",
  "projectId": "123",
  "tagName": "v1.0.0"
}

// New: tag
{
  "tool": "tag",
  "action": "get",
  "projectId": "123",
  "tagName": "v1.0.0"
}

// Old: createRepositoryTag
{
  "tool": "createRepositoryTag",
  "projectId": "123",
  "tagName": "v1.0.0",
  "ref": "main"
}

// New: tag
{
  "tool": "tag",
  "action": "create",
  "projectId": "123",
  "tagName": "v1.0.0",
  "ref": "main"
}
```

#### Pipeline Job Tools (5 tools → 2 tools + 2 unchanged)

**Removed Tools:**
- `listPipelineJobs`
- `getPipelineJob`
- `getPipelineJobTrace`
- `retryPipeline`
- `cancelPipeline`

**New Tools:**
- `pipelineJob` - Consolidated tool with `action` parameter (list/get/trace)
- `pipeline` - Consolidated tool with `action` parameter (cancel/retry)
- `retryPipelineJob` - Unchanged (still separate tool)
- `playPipelineJob` - Unchanged (still separate tool)

**Migration:**
```json
// Old: listPipelineJobs
{
  "tool": "listPipelineJobs",
  "projectId": "123",
  "pipelineId": 456
}

// New: pipelineJob
{
  "tool": "pipelineJob",
  "action": "list",
  "projectId": "123",
  "pipelineId": 456
}

// Old: retryPipeline
{
  "tool": "retryPipeline",
  "projectId": "123",
  "pipelineId": 456
}

// New: pipeline
{
  "tool": "pipeline",
  "action": "retry",
  "projectId": "123",
  "pipelineId": 456
}
```

### Added

- **Action Parameter Pattern**: Introduced a unified `action` parameter pattern for consolidated tools, making the API more consistent and reducing the number of tools
- **Unified Search Tool**: Single search tool with `resourceType` and `scope` parameters replaces 19 previous search tools
- **Improved Tool Descriptions**: Updated all tool descriptions to better reflect the new action-based pattern

### Changed

- **Tool Count**: Reduced from ~70 tools to ~35 tools (~50% reduction)
- **Context Window Optimization**: Consolidated tools significantly reduce the token footprint for tool descriptions in AI model context windows
- **API Consistency**: All consolidated tools follow the same action parameter pattern, making the API more predictable

### Migration Guide Summary

| Category | Old Count | New Count | Reduction |
|----------|-----------|-----------|-----------|
| Search | 19 tools | 1 tool | 18 tools |
| User Management | 7 tools | 1 tool | 6 tools |
| Issue Comments | 3 tools | 1 tool | 2 tools |
| MR Comments | 3 tools | 1 tool | 2 tools |
| Milestones | 3 tools | 1 tool (+ list) | 2 tools |
| Tags | 4 tools | 1 tool (+ list) | 3 tools |
| Pipeline Jobs | 5 tools | 2 tools (+ 2 unchanged) | 3 tools |
| **Total** | **~70 tools** | **~35 tools** | **~35 tools (50%)** |

For detailed migration examples and usage, see the [Tools Reference](docs/TOOLS.md).

## [1.x.x] - Previous Releases

See git history for changes prior to v2.0.0.
