# Testing Guide

This document describes the testing approach, current test coverage, and testing practices for the GitLab MCP Server.

## Current Test Status

**Overall Coverage:** 88.9%

| Component | Coverage | Status |
|-----------|----------|--------|
| pkg/gitlab | 88.7% | Good |
| pkg/toolsets | 98.6% | Excellent |
| pkg/log | 100.0% | Complete |
| pkg/translations | High | Good |

The project maintains comprehensive test coverage with unit tests, integration tests, and extensive use of GitLab's official mock framework.

## Testing Strategy

### Unit Tests

Unit tests verify individual functions and components in isolation.

**Scope:**
- Parameter validation (required, optional, type checking)
- Error handling and edge cases
- Helper functions (translation, redaction, etc.)
- Toolset enable/disable logic
- Token management operations

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

### Integration Tests

Integration tests verify that components work together correctly, often against a mock GitLab API.

**Scope:**
- MCP server initialization
- Tool registration and execution
- GitLab API interactions
- Error response handling
- Multi-server scenarios

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
```

**Target Coverage:**
- All MCP tools: >70%
- Error scenarios: >60%
- Happy path: 100%

### Test Framework

The project uses:
- Go's standard `testing` package
- GitLab's official mock framework for API mocking
- Table-driven tests for parameter validation
- Test fixtures for reproducible test data

## Running Tests

### All Tests

```bash
# Run all tests
make test

# Run with verbose output
go test -v ./...

# Run specific package
go test ./pkg/gitlab -v
```

### Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./pkg/...

# View coverage in browser
go tool cover -html=coverage.out -o coverage.html

# Coverage by package
go test -cover ./pkg/...
```

### Race Detection

```bash
# Run tests with race detector
go test -race ./pkg/... ./internal/...
```

### Specific Test Functions

```bash
# Run specific test
go test -run TestGetProject ./pkg/gitlab

# Run tests matching pattern
go test -run "TestGet.*" ./pkg/gitlab
```

## Writing Tests

### Best Practices

1. **Use table-driven tests** for multiple test cases:
   ```go
   func TestFunction(t *testing.T) {
       tests := []struct {
           name    string
           input   string
           want    string
           wantErr bool
       }{
           {"valid input", "test", "test", false},
           {"empty input", "", "", true},
       }
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               // test implementation
           })
       }
   }
   ```

2. **Test both success and failure cases**:
   - Valid inputs
   - Invalid inputs
   - Edge cases
   - Error conditions

3. **Mock external dependencies**:
   - Use GitLab's mock framework
   - Mock HTTP responses
   - Avoid real API calls in unit tests

4. **Keep tests focused**:
   - One assertion per test when possible
   - Clear test names describing what is tested
   - Isolate test data

5. **Use test helpers**:
   - Extract common setup code
   - Create reusable test utilities
   - Keep tests DRY

### Test Organization

Tests are organized alongside the code they test:

```
pkg/gitlab/
├── issues.go
├── issues_test.go
├── merge_requests.go
├── merge_requests_test.go
└── ...
```

### Example Test

```go
func TestGetProject(t *testing.T) {
    tests := []struct {
        name      string
        projectID string
        setupMock func(*gitlab.Client)
        wantErr   bool
    }{
        {
            name:      "valid project",
            projectID: "123",
            setupMock: func(client *gitlab.Client) {
                // Setup mock response
            },
            wantErr: false,
        },
        {
            name:      "project not found",
            projectID: "999",
            setupMock: func(client *gitlab.Client) {
                // Setup 404 response
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## CI/CD Testing

GitHub Actions automatically runs tests on all pull requests and pushes to main/develop branches.

The CI pipeline:
- Runs the full test suite with race detection
- Enforces a minimum coverage threshold of 85%
- Uploads coverage reports to Codecov
- Fails if coverage drops below the threshold

### Local CI Simulation

To simulate CI locally:

```bash
# Run all checks
make test
golangci-lint run

# Check coverage meets threshold
go test -cover ./pkg/... | grep coverage
```

## Test Data

### Mock Responses

Common mock responses are stored in test files or generated programmatically:

```go
var mockProject = &gitlab.Project{
    ID:                123,
    Name:              "test-project",
    PathWithNamespace: "group/test-project",
    HTTPURLToRepo:     "http://localhost/group/test-project.git",
}
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
}
```

## Coverage Goals

| Metric | Current | Target | Status |
|--------|---------|--------|--------|
| Overall Code Coverage | 88.9% | >85% | Met |
| Tool Functions Coverage | >80% | >80% | Met |
| Error Handling Coverage | >80% | >80% | Met |
| Integration Test Coverage | >60% | >60% | Met |

## Contributing Tests

When adding new features:

1. **Write tests first** (TDD approach preferred)
2. **Test both success and failure scenarios**
3. **Use table-driven tests** for multiple test cases
4. **Mock external dependencies** (GitLab API)
5. **Document edge cases** in comments
6. **Maintain or improve coverage**

### Test Checklist

- [ ] Unit tests for new functions
- [ ] Integration tests for API interactions
- [ ] Error handling tests
- [ ] Edge case coverage
- [ ] Update coverage numbers if applicable

## Troubleshooting

### Common Issues

**Problem:** Tests fail with "cannot connect to GitLab"

**Solution:** Use mock server for unit/integration tests. Only use real GitLab for E2E tests.

**Problem:** Test data is inconsistent

**Solution:** Use test fixtures and deterministic mock responses.

**Problem:** Tests are slow

**Solution:** Use table-driven tests, parallelize with `t.Parallel()`, and avoid real network calls.

**Problem:** How to test authenticated endpoints?

**Solution:** Use mock GitLab server, verify token is passed in headers.

### Getting Help

- See Go testing best practices: https://golang.org/doc/effective_go.html#testing
- Check GitLab Go SDK documentation for mocking
- Review existing tests in the codebase for examples

## Related Documentation

- [Contributing Guide](CONTRIBUTING.md) - General contribution guidelines
- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [GitLab Go SDK](https://pkg.go.dev/gitlab.com/gitlab-org/go-gitlab)
