# AGENTS.md - Agent Coding Guidelines

This document provides guidelines for AI agents operating in this repository.

## Project Overview

GitLab MCP Server is a Go-based Model Context Protocol server using `mark3labs/mcp-go` and `gitlab.com/gitlab-org/api/client-go`.

## Build Commands

```bash
# Setup dependencies, build, run all tests
make setup && make build && make test

# Run a specific test function
go test -run TestGetProject ./pkg/gitlab

# Run tests with verbose output and race detection
go test -race -v ./...

# Generate coverage report
go test -coverprofile=coverage.out ./pkg/...
go tool cover -html=coverage.out -o coverage.html

# Validate tool schema snapshots
make test-toolsnaps

# Run the linter
golangci-lint run
```

## Code Style Guidelines

### General Conventions
- **Go Version**: 1.24.2 (minimum 1.23)
- **Formatting**: Use `go fmt` or IDE auto-formatting
- **Package Names**: short, lowercase, no underscores (e.g., `gitlab`, `config`)

### Imports (in this order with blank lines between groups)
1. Standard library
2. External/third-party packages
3. Internal packages (prefixed with `github.com/InkyQuill/gitlab-mcp-server`)

```go
import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/InkyQuill/gitlab-mcp-server/pkg/translations"
    "github.com/mark3labs/mcp-go/mcp"
    gl "gitlab.com/gitlab-org/api/client-go" // GitLab client alias
)
```

### Naming Conventions
- **Variables/Functions**: `camelCase` (e.g., `getClient`, `projectID`)
- **Types/Interfaces**: `PascalCase` (e.g., `GetClientFn`, `ToolHandlerFunc`)
- **Files**: `snake_case.go` (e.g., `projects.go`, `projects_test.go`)
- **Test Files**: `*_test.go` co-located with the code they test

### Error Handling
- Use `fmt.Errorf` with `%w` for wrapping errors
- Return errors explicitly; never ignore with `_`
- Handle API errors using `HandleAPIError` helper
- Check errors immediately after calls

```go
project, resp, err := glClient.Projects.GetProject(projectIDStr, nil, gl.WithContext(ctx))
if err != nil {
    result, apiErr := HandleAPIError(err, resp, fmt.Sprintf("project %q", projectIDStr))
    if result != nil {
        return result, nil
    }
    return nil, apiErr
}
```

### Tool Function Pattern
```go
func GetProject(getClient GetClientFn, t map[string]string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
    return mcp.NewTool(
            "getProject",
            mcp.WithDescription(translations.Translate(t, translations.TOOL_GET_PROJECT_DESCRIPTION)),
            mcp.WithString("projectId", mcp.Required(), mcp.Description("...")),
        ),
        func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            // Handler implementation
        },
    }
}
```

### Testing Conventions
- Use table-driven tests with `github.com/stretchr/testify/assert` and `require`
- Use `go.uber.org/mock/gomock` for mocking GitLab API calls
- Use tool schema snapshots via `internal/toolsnaps`

### Type Annotations
- Prefer explicit types for function parameters
- Use generics for helpers (e.g., `requiredParam[T any]`)
- Use `context.Context` as the first parameter

### Linting
Enabled linters: govet, errcheck, staticcheck, ineffassign, unused, misspell, bodyclose, gocritic, gosec, nakedret, contextcheck, goconst, nilerr, noctx, unconvert, unparam, whitespace, prealloc, durationcheck, errorlint, reassign, thelper, tparallel

Run `golangci-lint run` before committing.

## Project Structure

```
cmd/           # Entry points (main server, config CLI)
pkg/
  ├── gitlab/    # Core GitLab tools and MCP server
  ├── config/    # Configuration handling
  ├── log/       # Logging utilities
  ├── toolsets/  # Toolset definitions
  └── translations/ # i18n support
internal/      # Internal packages
tests/         # Integration tests
```

## Common Patterns

### Parameter Handling
- `requiredParam[T](request, name)` - Get required parameter
- `OptionalParam[T](request, name)` - Get optional parameter

### GitLab Client
```go
glClient, err := getClient(ctx)
if err != nil {
    return nil, fmt.Errorf("failed to get GitLab client: %w", err)
}
```

### MCP Results
- Success: `mcp.NewToolResultText(string(data))`
- Error: `mcp.NewToolResultError(fmt.Sprintf("..."))`

## CI/CD
- Full test suite with race detection
- Minimum 85% code coverage enforcement
- Linting checks on all PRs

---

Note: No Cursor rules (.cursor/rules/, .cursorrules) or Copilot rules (.github/copilot-instructions.md) found in this repository.
