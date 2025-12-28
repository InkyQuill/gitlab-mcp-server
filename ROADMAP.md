# Development Roadmap

This document outlines the planned features and improvements for the GitLab MCP Server.

## Current Status

### Phase 1: Core Infrastructure ✅ Completed

**Implemented Components:**
- Command Logging - MCP JSON-RPC protocol message logging with sensitive data redaction
- i18n / Translations - Customizable tool descriptions via JSON config
- Dynamic Tool Discovery - Lazy-load toolsets on-demand

**Implemented Toolsets (69 tools total):**

- **Projects Toolset** (6 tools): `getProject`, `listProjects`, `getProjectFile`, `listProjectFiles`, `getProjectBranches`, `getProjectCommits`
- **Issues Toolset** (8 tools): `getIssue`, `listIssues`, `getIssueComments`, `getIssueLabels`, `createIssue`, `updateIssue`, `createIssueComment`, `updateIssueComment`
- **Merge Requests Toolset** (7 tools): `getMergeRequest`, `listMergeRequests`, `getMergeRequestComments`, `createMergeRequest`, `updateMergeRequest`, `createMergeRequestComment`, `updateMergeRequestComment`
- **Milestones Toolset** (4 tools): `getMilestone`, `listMilestones`, `createMilestone`, `updateMilestone`
- **Search Toolset** (20 tools): Comprehensive search across projects, issues, MRs, code, commits, milestones, snippets, wiki, notes with global, group-scoped, and project-scoped variants
- **Users Toolset** (12 tools): User information lookup and admin operations (block, unblock, ban, unban, activate, deactivate, approve)
- **Security Toolset** (6 tools): Security scan results (SAST, DAST, dependency scanning, container scanning, secret detection, license compliance)
- **Token Management Toolset** (6 tools): Runtime token CRUD operations and validation
- **Project Configuration Toolset** (4 tools): Auto-detection, `.gmcprc` file management, multi-server support
- **Tags Toolset** (5 tools): `listRepositoryTags`, `getRepositoryTag`, `createRepositoryTag`, `deleteRepositoryTag`, `getTagCommit`
- **Pipeline Jobs Toolset** (7 tools): `listPipelineJobs`, `getPipelineJob`, `getPipelineJobTrace`, `retryPipelineJob`, `playPipelineJob`, `cancelPipeline`, `retryPipeline`

**Quality Metrics:**
- Test Coverage: 86.9% (pkg/gitlab)
- Tool Schema Snapshots: 86
- Test Functions: 174
- Test Cases: 886
- Documentation Files: 11 (comprehensive docs/ folder)

## Upcoming Features

### Phase 2: Repository & Code Management ✅ Completed

#### Tags Management

**Priority:** High
**Status:** Completed

Implemented Tools:
- `listRepositoryTags` - List all tags in a repository (with search and pagination)
- `getRepositoryTag` - Get details of a specific tag
- `createRepositoryTag` - Create a new tag (annotated or lightweight)
- `deleteRepositoryTag` - Delete a tag
- `getTagCommit` - Get release/commit info for a tag

**Implementation Details:**
- Comprehensive error handling for 401, 403, 404, 400, 422, 500 status codes
- Pagination support for listing tags
- Search functionality for filtering tags
- Schema snapshot tests for all 5 tools
- 33 test cases covering success paths, error cases, validation, and edge cases

### Phase 3: CI/CD Pipeline Management ✅ Completed

#### Pipeline Status & Control

**Priority:** High
**Status:** Completed

Implemented Tools:
- `listPipelineJobs` - List jobs for a pipeline
- `getPipelineJob` - Get details of a specific job
- `getPipelineJobTrace` - Get job logs/trace output
- `retryPipelineJob` - Retry a failed job
- `playPipelineJob` - Trigger a manual job
- `cancelPipeline` - Cancel a running pipeline
- `retryPipeline` - Retry all jobs in a pipeline

**Implementation Details:**
- Integer ID validation for pipelineId/jobId parameters
- Raw trace output for job logs (not JSON)
- Write operations use HandleCreateUpdateAPIError for proper error handling
- Schema snapshot tests for all 7 tools
- 40 test cases covering success paths, error cases, validation, and edge cases

#### Pipeline Variables

**Priority:** Medium  
**Status:** Planned

Tools needed:
- `listPipelineVariables` - List CI/CD variables for a project
- `getPipelineVariable` - Get details of a specific variable
- `createPipelineVariable` - Create a new CI/CD variable
- `updatePipelineVariable` - Update an existing variable
- `deletePipelineVariable` - Delete a CI/CD variable

**Use Cases:**
- Automate CI/CD variable management
- Secure credential rotation
- Environment-specific configuration

### Phase 4: Project Labels & Milestones Management

#### Labels Management

**Priority:** Medium  
**Status:** Planned

Tools needed:
- `listProjectLabels` - List all labels in a project
- `getProjectLabel` - Get details of a specific label
- `createProjectLabel` - Create a new label
- `updateProjectLabel` - Update label properties (color, description)
- `deleteProjectLabel` - Delete a label
- `subscribeToLabel` - Subscribe to label notifications

**Use Cases:**
- Standardize labels across projects
- Automated label creation for new projects
- Label cleanup and organization

**GitLab API Endpoints:**
- `GET /projects/:id/labels`
- `GET /projects/:id/labels/:label_id_or_title`
- `POST /projects/:id/labels`
- `PUT /projects/:id/labels/:label_id_or_title`
- `DELETE /projects/:id/labels/:label_id_or_title`

## Infrastructure & Quality Improvements

### Testing & Test Coverage

**Priority:** High
**Status:** Active (86.9% coverage achieved)

Current Status: Comprehensive test suite with high coverage

Planned Improvements:
- Increase coverage to 90%+ across all packages
- Add integration tests for main.go initialization
- Expand edge case testing for admin operations
- Add performance benchmarks for critical paths

### Code Quality & Refactoring

**Priority:** Medium  
**Status:** Planned

Planned Improvements:
1. **Error Handling**: Standardize error messages and add more context
2. **Logging**: Improve log messages with structured logging (JSON format option)
3. **Code Duplication**: Extract common patterns into helper functions
4. **Documentation**: Add GoDoc comments for all exported functions
5. **Type Safety**: Add more type definitions for API responses

**Specific Areas:**
- Consolidate API error handling into shared helper functions
- Extract pagination logic into reusable components
- Standardize tool parameter validation patterns
- Add performance benchmarks for critical paths

### Performance Optimizations

**Priority:** Low  
**Status:** Planned

Planned Improvements:
1. **Caching**: Cache frequently accessed data (project details, user info)
2. **Connection Pooling**: Optimize HTTP client usage for concurrent requests
3. **Lazy Loading**: Already implemented via dynamic tool discovery
4. **Rate Limiting**: Respect GitLab rate limits and implement backoff

**Metrics to Monitor:**
- Average tool execution time
- Memory usage
- API call frequency
- Error rates

### Documentation Improvements

**Priority:** Medium  
**Status:** Planned

Planned Improvements:
1. **Examples**: Add real-world usage examples for each tool
2. **Tutorials**: Create step-by-step guides for common workflows
3. **API Reference**: Auto-generate tool documentation from code
4. **Migration Guides**: Document version changes and breaking changes
5. **Troubleshooting**: Add common issues and solutions

## Implementation Priorities

### High Priority

- Phase 2: Tags Management - High value for users
- Phase 3: CI/CD Pipeline Management - Critical for DevOps workflows
- Testing & Test Coverage - Maintain high coverage

### Medium Priority

- Phase 4: Labels Management - Common project management need
- Code Quality & Refactoring - Maintainability improvements
- Documentation Improvements - Enhanced user experience

### Low Priority

- Performance Optimizations - Scale for larger deployments
- Additional toolsets based on user feedback

### Future Considerations

- Webhook Support - Receive real-time GitLab events
- GraphQL API Support - More efficient queries
- Batch Operations - Reduce API call count
- Advanced Security Features - Enhanced security scanning tools

## Contributing

We welcome contributions! If you'd like to implement any of these features:

1. Check the [CONTRIBUTING.md](CONTRIBUTING.md) guidelines
2. Create an issue for the feature you want to work on
3. Discuss the implementation approach with maintainers
4. Submit a pull request with tests and documentation

**Priority Labels:**
- High Priority - Core functionality, high demand
- Medium Priority - Important but not urgent
- Low Priority - Nice-to-have features

## Questions & Feedback

For questions, suggestions, or feedback:
- Open an issue on GitHub
- Check existing documentation in `/docs` folder
- Review example configurations in installer scripts

**Project Status:** Production-ready, actively maintained
