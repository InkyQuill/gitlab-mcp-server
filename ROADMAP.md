# GitLab MCP Server - Development Roadmap 🗺️

This document outlines the planned features and improvements for the GitLab MCP Server.

## Current Status ✅

### Completed Features

**Phase 1: Core Infrastructure (Completed ✅)**
- ✅ Command Logging - MCP JSON-RPC protocol message logging with sensitive data redaction
- ✅ i18n / Translations - Customizable tool descriptions via JSON config
- ✅ Dynamic Tool Discovery - Lazy-load toolsets on-demand

**Implemented Toolsets (29 tools across 3 major toolsets)**
- ✅ **Projects Toolset** (6 tools): `getProject`, `listProjects`, `getProjectFile`, `listProjectFiles`, `getProjectBranches`, `getProjectCommits`
- ✅ **Issues Toolset** (8 tools): `getIssue`, `listIssues`, `getIssueComments`, `getIssueLabels`, `createIssue`, `updateIssue`, `createIssueComment`, `updateIssueComment`
- ✅ **Merge Requests Toolset** (7 tools): `getMergeRequest`, `listMergeRequests`, `getMergeRequestComments`, `createMergeRequest`, `updateMergeRequest`, `createMergeRequestComment`, `updateMergeRequestComment`
- ✅ **Milestones** (4 tools): `getMilestone`, `listMilestones`, `createMilestone`, `updateMilestone`
- ✅ **Token Management** (6 tools): Runtime token CRUD operations
- ✅ **Project Configuration** (4 tools): Auto-detection, `.gmcprc` file management

---

## Upcoming Features 🔜

### Phase 2: Repository & Code Management

#### Tags Management 🔖
**Priority: High**
**Estimated Effort: 1-2 days**

Tools needed:
- `listRepositoryTags` - List all tags in a repository
- `getRepositoryTag` - Get details of a specific tag
- `createRepositoryTag` - Create a new tag (annotated or lightweight)
- `deleteRepositoryTag` - Delete a tag
- `getTagCommit` - Get commit details for a tag

**Use Cases:**
- Release management automation
- Version tagging workflows
- Cleaning up old tags

**GitLab API Endpoints:**
- `GET /projects/:id/repository/tags`
- `GET /projects/:id/repository/tags/:tag_name`
- `POST /projects/:id/repository/tags`
- `DELETE /projects/:id/repository/tags/:tag_name`

---

### Phase 3: CI/CD Pipeline Management

#### Pipeline Status & Control 🚀
**Priority: High**
**Estimated Effort: 2-3 days**

Tools needed:
- `listPipelineJobs` - List jobs for a pipeline
- `getPipelineJob` - Get details of a specific job (including logs)
- `getPipelineJobTrace` - Get job logs/trace output
- `retryPipelineJob` - Retry a failed job
- `playPipelineJob` - Trigger a manual job
- `cancelPipeline` - Cancel a running pipeline
- `retryPipeline` - Retry all jobs in a pipeline

**Use Cases:**
- Monitor CI/CD status from AI assistants
- Debug failed pipelines by retrieving logs
- Automate pipeline retries and cleanups

**GitLab API Endpoints:**
- `GET /projects/:id/pipelines/:pipeline_id/jobs`
- `GET /projects/:id/jobs/:job_id`
- `GET /projects/:id/jobs/:job_id/trace`
- `POST /projects/:id/jobs/:job_id/retry`
- `POST /projects/:id/jobs/:job_id/play`
- `POST /projects/:id/pipelines/:pipeline_id/cancel`
- `POST /projects/:id/pipelines/:pipeline_id/retry`

#### Pipeline Variables 🔐
**Priority: Medium**
**Estimated Effort: 1-2 days**

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

---

### Phase 4: Project Labels & Milestones Management

#### Labels Management 🏷️
**Priority: Medium**
**Estimated Effort: 1-2 days**

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

---

### Phase 5: Search & Discovery

#### Scoped Search 🔍
**Priority: High**
**Estimated Effort: 2-3 days**

Tools needed:
- `searchProjects` - Search for projects across GitLab
- `searchIssues` - Search issues globally or within a project
- `searchMergeRequests` - Search merge requests globally or within a project
- `searchCode` - Search code across repositories (blobs)
- `searchCommits` - Search commits in a project
- `searchWiki` - Search wiki content

**Use Cases:**
- Find relevant projects/issues quickly
- Code search across multiple repositories
- Research and analysis workflows

**GitLab API Endpoints:**
- `GET /search?scope=projects`
- `GET /search?scope=issues`
- `GET /search?scope=merge_requests`
- `GET /search?scope=blobs`
- `GET /projects/:id/search?scope=commits`

---

### Phase 6: Users & Teams

#### User Management 👥
**Priority: Medium**
**Estimated Effort: 1-2 days**

Tools needed:
- `getCurrentUser` - Get current authenticated user details
- `getUser` - Get details of a specific user
- `listProjectUsers` - List users with access to a project
- `listGroupMembers` - List members of a group
- `listProjectMembers` - List members of a project

**Use Cases:**
- Identify team members for assignments
- Check user permissions
- Team audit and reporting

**GitLab API Endpoints:**
- `GET /user`
- `GET /users/:id`
- `GET /projects/:id/users`
- `GET /groups/:id/members`
- `GET /projects/:id/members/all`

---

### Phase 7: Security & Compliance

#### Security Scans 🛡️
**Priority: Low**
**Estimated Effort: 2-3 days**

Tools needed:
- `listPipelineArtifacts` - List artifacts from a pipeline
- `getSecurityReport` - Get security scan results (SAST, DAST, dependency scanning)
- `listProjectVulnerabilities` - List known vulnerabilities
- `getVulnerabilityDetails` - Get details of a specific vulnerability

**Use Cases:**
- Monitor security posture
- Automated vulnerability reporting
- Compliance checking

**Note:** Many security features require GitLab Premium/Ultimate.

---

## Infrastructure & Quality Improvements 🔧

### Testing & Test Coverage 🧪
**Priority: High**
**Estimated Effort: 3-5 days**

Current Status: ❌ No automated tests

Planned Improvements:
- **Unit Tests**: Test individual tool functions, parameter parsing, API error handling
- **Integration Tests**: Test against GitLab API mock server or test instance
- **End-to-End Tests**: Test MCP protocol interactions
- **Test Coverage**: Aim for >80% code coverage

**Test Framework:**
- Use `testing` package + `testify` for assertions
- Mock GitLab API responses using `httptest`
- Test table-driven patterns for parameter validation

---

### Code Quality & Refactoring 🎨
**Priority: Medium**
**Estimated Effort: 2-3 days**

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

---

### Performance Optimizations ⚡
**Priority: Low**
**Estimated Effort: 2-3 days**

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

---

### Documentation Improvements 📚
**Priority: Medium**
**Estimated Effort: 1-2 days**

Planned Improvements:
1. **Examples**: Add real-world usage examples for each tool
2. **Tutorials**: Create step-by-step guides for common workflows
3. **API Reference**: Auto-generate tool documentation from code
4. **Migration Guides**: Document version changes and breaking changes
5. **Troubleshooting**: Add common issues and solutions

---

## Implementation Priorities 📋

### Immediate (Next 1-2 weeks)
1. ✅ Phase 1: Core Infrastructure - **COMPLETED**
2. 🔜 **Phase 5: Search & Discovery** - High value for users
3. 🔜 **Phase 2: Tags Management** - Common workflow need

### Short-term (Next 1-2 months)
4. **Phase 3: CI/CD Pipeline Management** - Critical for DevOps workflows
5. **Testing & Test Coverage** - Essential for stability
6. **Phase 4: Labels Management** - Common project management need

### Long-term (3-6 months)
7. **Phase 6: Users & Teams** - Nice-to-have for collaboration
8. **Code Quality & Refactoring** - Maintainability
9. **Performance Optimizations** - Scale for larger deployments

### Future Considerations
- **Phase 7: Security & Compliance** - If there's demand
- **Webhook Support** - Receive real-time GitLab events
- **GraphQL API Support** - More efficient queries
- **Batch Operations** - Reduce API call count

---

## Contributing 🤝

We welcome contributions! If you'd like to implement any of these features:

1. Check the [CONTRIBUTING.md](CONTRIBUTING.md) guidelines
2. Create an issue for the feature you want to work on
3. Discuss the implementation approach with maintainers
4. Submit a pull request with tests and documentation

**Priority Labels:**
- 🚀 High Priority - Core functionality, high demand
- 🔧 Medium Priority - Important but not urgent
- 💡 Low Priority - Nice-to-have features

---

## Questions & Feedback 💬

For questions, suggestions, or feedback:
- Open an issue on GitHub
- Check existing documentation in `/docs` folder
- Review example configurations in installer scripts

**Last Updated:** 2025-12-27
**Project Status:** Active Development ✨
