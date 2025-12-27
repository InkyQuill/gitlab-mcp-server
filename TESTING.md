# Testing Guide & Status 🧪

This document describes the testing approach, current test coverage, and testing goals for the GitLab MCP Server.

## Current Test Status 📊

**Overall Coverage:** ❌ **No automated tests yet**

| Component | Unit Tests | Integration Tests | Manual Tests | Status |
|-----------|-----------|-------------------|--------------|--------|
| Toolsets (projects, issues, MRs) | ❌ None | ❌ None | ✅ Manual | ⚠️ Needs Tests |
| Token Management | ❌ None | ❌ None | ✅ Manual | ⚠️ Needs Tests |
| Dynamic Tool Discovery | ❌ None | ❌ None | ✅ Manual | ⚠️ Needs Tests |
| Translation System | ❌ None | ❌ None | ✅ Manual | ⚠️ Needs Tests |
| Command Logging | ❌ None | ❌ None | ✅ Manual | ⚠️ Needs Tests |
| API Error Handling | ❌ None | ❌ None | ⚠️ Partial | ⚠️ Needs Tests |

**Priority:** 🔴 **High** - Add automated tests before next major release

---

## Testing Strategy 🎯

### 1. Unit Tests

Unit tests verify individual functions and components in isolation.

**Scope:**
- Parameter validation (required, optional, type checking)
- Error handling and edge cases
- Helper functions (translation, redaction, etc.)
- Toolset enable/disable logic

**Example Test Structure:**
```go
func TestRequiredParam_String(t *testing.T) {
    tests := []struct {
        name    string
        args    mcp.CallToolRequest
        want    string
        wantErr bool
    }{
        {
            name: "valid string parameter",
            args: mcp.CallToolRequest{
                Params: mcp.Params{
                    Arguments: map[string]interface{}{
                        "projectId": "123",
                    },
                },
            },
            want:    "123",
            wantErr: false,
        },
        {
            name: "missing parameter",
            args: mcp.CallToolRequest{
                Params: mcp.Params{Arguments: map[string]interface{}{}},
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := requiredParam[string](&tt.args, "projectId")
            if (err != nil) != tt.wantErr {
                t.Errorf("requiredParam() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("requiredParam() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

**Target Coverage:**
- Core tool functions: >80%
- Helper functions: >90%
- Parameter validation: 100%

---

### 2. Integration Tests

Integration tests verify that components work together correctly, often against a mock GitLab API.

**Scope:**
- MCP server initialization
- Tool registration and execution
- GitLab API interactions
- Error response handling

**Mock Server Setup:**
```go
func setupMockGitlabServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *gitlab.Client) {
    server := httptest.NewServer(handler)
    client, err := gitlab.NewClient(
        "test-token",
        gitlab.WithBaseURL(server.URL),
        gitlab.WithoutRetries(),
    )
    require.NoError(t, err)
    return server, client
}

func TestGetProjectTool_Integration(t *testing.T) {
    mockHandler := func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        assert.Equal(t, "GET", r.Method)
        assert.Contains(t, r.URL.Path, "api/v4/projects")

        // Return mock response
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{
            "id": 123,
            "name": "test-project",
            "path_with_namespace": "group/test-project"
        }`))
    }

    server, client := setupMockGitlabServer(t, mockHandler)
    defer server.Close()

    // Test the tool with mock client
    getClient := func(ctx context.Context) (*gitlab.Client, error) {
        return client, nil
    }

    tool, handler := GetProject(getClient, nil)
    // ... execute tool and verify response
}
```

**Target Coverage:**
- All MCP tools: >70%
- Error scenarios: >60%
- Happy path: 100%

---

### 3. End-to-End Tests

E2E tests verify the complete MCP workflow against a real GitLab instance.

**Scope:**
- MCP protocol message exchange
- Tool discovery and invocation
- Multi-step workflows
- Performance benchmarks

**Test Environment:**
```bash
# Use GitLab's official Docker image for testing
docker run -d --name gitlab-test \
  --publish 8080:80 \
  --publish 2222:22 \
  --env GITLAB_OMNIBUS_CONFIG="external_url 'http://localhost'" \
  gitlab/gitlab-ce:latest

# Run E2E tests
GITLAB_HOST=http://localhost:8080 \
GITLAB_TOKEN=test-token \
go test ./tests/e2e/...
```

**Example E2E Test:**
```go
func TestE2E_ProjectWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }

    // Skip if no GitLab test instance configured
    host := os.Getenv("GITLAB_TEST_HOST")
    token := os.Getenv("GITLAB_TEST_TOKEN")
    if host == "" || token == "" {
        t.Skip("GITLAB_TEST_HOST and GITLAB_TEST_TOKEN required for E2E tests")
    }

    // Create MCP server
    server := setupMCPServer(t, host, token)

    // Test workflow: List projects → Get project details → List files
    t.Run("list_projects", func(t *testing.T) {
        result := callTool(t, server, "listProjects", map[string]interface{}{
            "search": "test",
        })
        assert.Contains(t, result, "test-project")
    })

    // ... more workflow tests
}
```

**Target Coverage:**
- Critical workflows: 100%
- Performance: <2s per tool call

---

## Test Implementation Plan 📋

### Phase 1: Foundation (Week 1-2)

**Setup Testing Infrastructure**
- [ ] Configure `go test` with coverage reporting
- [ ] Set up test dependencies (`testify`, `gomock`)
- [ ] Create mock GitLab server utilities
- [ ] Add Makefile target for tests (`make test`, `make test-coverage`)

**Initial Unit Tests**
- [ ] Test parameter validation helpers
- [ ] Test translation system
- [ ] Test logging with redaction
- [ ] Test toolset enable/disable logic

**Success Criteria:**
- Test infrastructure in place
- CI/CD pipeline runs tests automatically
- Initial coverage report generated

---

### Phase 2: Core Tools (Week 3-4)

**Projects Toolset Tests**
- [ ] Unit tests for `getProject`
- [ ] Unit tests for `listProjects`
- [ ] Unit tests for `getProjectFile`
- [ ] Unit tests for `listProjectFiles`
- [ ] Integration tests with mock GitLab API
- [ ] E2E tests for common project workflows

**Issues Toolset Tests**
- [ ] Unit tests for all issue tools (8 tools)
- [ ] Integration tests with mock GitLab API
- [ ] E2E tests for issue CRUD workflows

**Success Criteria:**
- Projects and Issues toolsets >70% coverage
- All critical paths tested
- Mock GitLab server handles common scenarios

---

### Phase 3: Advanced Features (Week 5-6)

**Merge Requests & Milestones Tests**
- [ ] Unit tests for all MR tools (7 tools)
- [ ] Unit tests for milestone tools (4 tools)
- [ ] Integration tests with mock GitLab API
- [ ] E2E tests for MR workflows (create, review, merge)

**Dynamic Discovery Tests**
- [ ] Unit tests for toolset discovery
- [ ] Unit tests for enable/disable logic
- [ ] Integration tests for dynamic tool loading
- [ ] E2E tests for dynamic discovery workflow

**Success Criteria:**
- All implemented toolsets >70% coverage
- Dynamic discovery fully tested
- E2E tests cover critical user workflows

---

### Phase 4: Quality & Performance (Week 7-8)

**Error Handling Tests**
- [ ] Test all error scenarios (404, 401, 403, 500, etc.)
- [ ] Test rate limiting handling
- [ ] Test network error handling
- [ ] Test timeout handling

**Performance Tests**
- [ ] Benchmark each tool execution time
- [ ] Test concurrent tool execution
- [ ] Profile memory usage
- [ ] Optimize bottlenecks

**Security Tests**
- [ ] Verify sensitive data redaction in logs
- [ ] Test token validation and rotation
- [ ] Test input sanitization
- [ ] Test for injection vulnerabilities

**Success Criteria:**
- Error handling >80% coverage
- All tools execute in <2s
- No memory leaks detected
- Security audit passed

---

## Running Tests 🚀

### Unit Tests
```bash
# Run all unit tests
make test

# Run tests for specific package
go test ./pkg/gitlab -v

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests
```bash
# Run integration tests (requires mock server)
go test ./tests/integration -v

# Run specific integration test
go test ./tests/integration -run TestGetProject -v
```

### E2E Tests
```bash
# Run E2E tests (requires GitLab test instance)
GITLAB_TEST_HOST=http://localhost:8080 \
GITLAB_TEST_TOKEN=test-token \
go test ./tests/e2e -v

# Run quick E2E smoke test
make test-e2e-smoke
```

### All Tests with Coverage
```bash
# Run complete test suite
make test-all

# Generate coverage report
make coverage
```

---

## Test Data 📦

### Mock GitLab Responses

Store common mock responses in `tests/mocks/`:

```
tests/mocks/
├── projects/
│   ├── get_project_success.json
│   ├── list_projects_empty.json
│   └── list_projects_paginated.json
├── issues/
│   ├── get_issue_success.json
│   └── create_issue_success.json
└── errors/
    ├── 404_not_found.json
    ├── 401_unauthorized.json
    └── 500_internal_error.json
```

### Test Fixtures

Use test fixtures for reproducible test data:

```go
// tests/fixtures/projects.go
package fixtures

var TestProject = &gitlab.Project{
    ID:                123,
    Name:              "test-project",
    PathWithNamespace: "group/test-project",
    HTTPURLToRepo:     "http://localhost/group/test-project.git",
}

var TestIssue = &gitlab.Issue{
    IID:         1,
    Title:       "Test Issue",
    Description: "This is a test issue",
    State:       "opened",
}
```

---

## Continuous Integration 🔄

### GitHub Actions Workflow

```yaml
name: Test Suite

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Run unit tests
        run: make test

      - name: Run integration tests
        run: make test-integration

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: coverage.out

      - name: Run E2E tests
        run: make test-e2e
        env:
          GITLAB_TEST_HOST: ${{ secrets.GITLAB_TEST_HOST }}
          GITLAB_TEST_TOKEN: ${{ secrets.GITLAB_TEST_TOKEN }}
```

---

## Coverage Goals 🎯

| Metric | Current | Target | Priority |
|--------|---------|--------|----------|
| Overall Code Coverage | 0% | >70% | 🔴 High |
| Tool Functions Coverage | 0% | >80% | 🔴 High |
| Error Handling Coverage | ~20% | >80% | 🔴 High |
| Critical Paths Coverage | Manual only | 100% | 🔴 High |
| Integration Test Coverage | 0% | >60% | 🟡 Medium |
| E2E Workflow Coverage | 0% | >50% | 🟡 Medium |

**Next Milestone:** Achieve >50% code coverage by end of Phase 2

---

## Contributing Tests 🤝

When adding new features:

1. **Write tests first** (TDD approach preferred)
2. **Test both success and failure scenarios**
3. **Use table-driven tests** for multiple test cases
4. **Mock external dependencies** (GitLab API)
5. **Document edge cases** in comments
6. **Update this file** with test status

**Test Checklist:**
- [ ] Unit tests for new functions
- [ ] Integration tests for API interactions
- [ ] E2E tests for user workflows
- [ ] Error handling tests
- [ ] Update coverage numbers

---

## Questions & Issues ❓

**Common Test Issues:**

1. **"Cannot connect to GitLab"**
   - Solution: Use mock server for unit/integration tests
   - Only use real GitLab for E2E tests

2. **"Test data is inconsistent"**
   - Solution: Use test fixtures and deterministic mock responses

3. **"Tests are slow"**
   - Solution: Use table-driven tests, parallelize with `t.Parallel()`

4. **"How to test authenticated endpoints?"**
   - Solution: Use mock GitLab server, verify token is passed in headers

**For more help:**
- See Go testing best practices: https://golang.org/doc/effective_go.html#testing
- Check `testify` documentation: https://github.com/stretchr/testify

---

**Last Updated:** 2025-12-27
**Next Review:** After Phase 2 completion
