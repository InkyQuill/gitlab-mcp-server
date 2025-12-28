# GitLab MCP Server

[![Tests](https://github.com/InkyQuill/gitlab-mcp-server/actions/workflows/test.yml/badge.svg)](https://github.com/InkyQuill/gitlab-mcp-server/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/InkyQuill/gitlab-mcp-server/branch/main/graph/badge.svg)](https://codecov.io/gh/InkyQuill/gitlab-mcp-server)
[![Go Report Card](https://goreportcard.com/badge/github.com/InkyQuill/gitlab-mcp-server)](https://goreportcard.com/report/github.com/InkyQuill/gitlab-mcp-server)
[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Maintained](https://img.shields.io/badge/maintained-yes-green.svg)](https://github.com/InkyQuill/gitlab-mcp-server/graphs/commit-activity)

The GitLab MCP Server is a [Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction) server that provides seamless integration with GitLab APIs, enabling advanced automation and interaction capabilities for developers and AI tools within the GitLab ecosystem.

> **Note:** This is a fork of the original [gitlab-mcp-server](https://github.com/LuisCusihuaman/gitlab-mcp-server) project, substantially rewritten with improved testing, better error handling, and additional features.

## Overview

The GitLab MCP Server bridges the gap between AI development tools and GitLab's extensive API, providing a standardized interface for managing projects, issues, merge requests, and other GitLab resources. It supports both GitLab.com and self-managed GitLab instances, with features like multi-server configuration, dynamic tool discovery, and comprehensive token management.

## Features

- **Comprehensive GitLab API Integration**: Access projects, issues, merge requests, milestones, and more
- **Multi-Server Support**: Configure multiple GitLab instances (work, personal, etc.) simultaneously
- **Dynamic Tool Discovery**: Load toolsets on-demand to reduce initial context size
- **Token Management**: Automatic validation, expiration tracking, and runtime token management
- **Project Auto-Detection**: Automatically detect GitLab projects from Git remotes
- **Read-Only Mode**: Restrict operations to read-only for enhanced security
- **Self-Hosted Support**: Works with GitLab.com and self-managed instances
- **Internationalization**: Customize tool descriptions and translations
- **Comprehensive Testing**: 88.9% code coverage with extensive test suite

## Quick Start

### Prerequisites

1. **GitLab Access Token**: Create a [Personal Access Token](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html), [Project Access Token](https://docs.gitlab.com/ee/user/project/settings/project_access_tokens.html), or [Group Access Token](https://docs.gitlab.com/ee/user/group/settings/group_access_tokens.html)
2. **Docker (optional)**: For containerized deployment
3. **Go 1.23+ (optional)**: For building from source

### Installation

The easiest way to get started is using the automated installer:

```bash
git clone <repository-url>
cd gitlab-mcp-server
make setup
make install-mcp
```

Or use the setup script:

```bash
./setup.sh  # Linux/macOS
setup.bat   # Windows
```

The installer will guide you through configuration and automatically set up your development environment.

For detailed installation instructions, see [Installation Guide](docs/INSTALLATION.md).

## Documentation

### Getting Started

- [Installation Guide](docs/INSTALLATION.md) - Detailed installation and setup instructions
- [Editor Setup](docs/EDITOR_SETUP.md) - Configure VS Code, Claude Desktop, Claude Code, and Cursor
- [Quick Start Guide](docs/QUICK_START.md) - Get up and running in minutes

### Core Features

- [Tools Reference](docs/TOOLS.md) - Complete reference for all available tools
- [Token Management](docs/TOKEN_MANAGEMENT.md) - Managing access tokens and authentication
- [Multi-Server Setup](docs/MULTI_SERVER_SETUP.md) - Configure multiple GitLab instances
- [Project Configuration](docs/PROJECT_CONFIG.md) - Using `.gmcprc` files for project-specific settings

### Advanced Topics

- [Dynamic Tool Discovery](docs/DYNAMIC_TOOLS.md) - On-demand toolset loading
- [Internationalization](docs/I18N.md) - Customizing tool descriptions and translations
- [Self-Hosted GitLab](docs/SELF_HOSTED.md) - Connecting to self-managed instances
- [GitHub Detection](docs/GITHUB_DETECTION.md) - Why GitHub isn't supported and error handling

### Development

- [Contributing Guide](CONTRIBUTING.md) - How to contribute to the project
- [Testing Guide](TESTING.md) - Testing practices and coverage
- [Roadmap](ROADMAP.md) - Planned features and improvements

## Available Tools

The server provides tools organized into logical toolsets:

### Projects Toolset
- `getProject` - Get project details
- `listProjects` - List projects with filtering
- `getProjectFile` - Retrieve file contents
- `listProjectFiles` - List repository files
- `getProjectBranches` - List repository branches
- `getProjectCommits` - List project commits

### Issues Toolset
- `getIssue` - Get issue details
- `listIssues` - List issues with filters
- `getIssueComments` - Get issue comments
- `getIssueLabels` - Get issue labels
- `createIssue` - Create a new issue
- `updateIssue` - Update an issue
- `createIssueComment` - Add a comment to an issue
- `updateIssueComment` - Update an issue comment

### Merge Requests Toolset
- `getMergeRequest` - Get merge request details
- `listMergeRequests` - List merge requests
- `getMergeRequestComments` - Get MR comments
- `createMergeRequest` - Create a merge request
- `updateMergeRequest` - Update a merge request
- `createMergeRequestComment` - Add a comment to an MR
- `updateMergeRequestComment` - Update an MR comment

### Milestones
- `getMilestone` - Get milestone details
- `listMilestones` - List project milestones
- `createMilestone` - Create a milestone
- `updateMilestone` - Update a milestone

### Search
- `searchProjects` - Search for projects
- `searchIssues` - Search issues
- `searchMergeRequests` - Search merge requests
- `searchBlobs` - Search code
- `searchCommits` - Search commits
- `searchMilestones` - Search milestones
- Plus group/project-scoped variants

### Users
- `getCurrentUser` - Get authenticated user
- `getUser` - Get user details
- `getUserStatus` - Get user status
- `listUsers` - List users
- `listProjectUsers` - List project members
- Admin operations: `blockUser`, `unblockUser`, `banUser`, `unbanUser`, `activateUser`, `deactivateUser`, `approveUser`

### Security
- `getProjectSAST` - Get SAST scan results
- `getProjectDAST` - Get DAST scan results
- `getProjectDependencyScanning` - Get dependency scanning results
- `getProjectContainerScanning` - Get container scanning results
- `getProjectSecretDetection` - Get secret detection results
- `getProjectLicenseCompliance` - Get license compliance data

### Token Management
- `listTokens` - List configured tokens
- `validateToken` - Validate a token
- `addToken` - Add a new token
- `updateToken` - Update a token
- `removeToken` - Remove a token
- `getNotifications` - Get token notifications
- `clearNotifications` - Clear notifications

### Project Configuration
- `getCurrentProject` - Get current project config
- `setCurrentProject` - Set project configuration
- `detectProject` - Auto-detect from Git remotes
- `autoDetectAndSetProject` - Detect and set in one step

For complete tool documentation, see [Tools Reference](docs/TOOLS.md).

## Configuration

### Environment Variables

- `GITLAB_TOKEN` - GitLab access token (required)
- `GITLAB_HOST` - GitLab host URL (default: `https://gitlab.com`)
- `GITLAB_TOOLSETS` - Comma-separated list of toolsets to enable (default: all)
- `GITLAB_READ_ONLY` - Enable read-only mode (default: false)
- `GITLAB_DYNAMIC_TOOLSETS` - Enable dynamic tool discovery (default: false)
- `GITLAB_ENABLE_COMMAND_LOGGING` - Enable JSON-RPC logging (default: false)

### Toolsets

Control which toolsets are available using the `--toolsets` flag or `GITLAB_TOOLSETS` environment variable:

```bash
./gitlab-mcp-server stdio --toolsets issues,merge_requests,projects
```

Available toolsets: `projects`, `issues`, `merge_requests`, `search`, `users`, `security`, `token_management`, `project_config`.

### Read-Only Mode

Restrict operations to read-only:

```bash
./gitlab-mcp-server stdio --read-only
```

### Dynamic Tool Discovery

Enable on-demand toolset loading:

```bash
./gitlab-mcp-server stdio --dynamic-toolsets
```

This starts with only discovery tools, allowing you to enable toolsets as needed.

## Building from Source

### Prerequisites

- Go 1.23 or later
- Make (for Makefile targets)

### Build Commands

```bash
# Install dependencies
make setup

# Build binary
make build

# Run tests
make test

# Build Docker image
make docker-build

# Clean build artifacts
make clean
```

The binary will be created at `bin/gitlab-mcp-server`.

## Testing

This project maintains comprehensive test coverage. See the [codecov badge](https://codecov.io/gh/InkyQuill/gitlab-mcp-server) above for current coverage metrics.

```bash
# Run all tests
make test

# Run with coverage
go test -coverprofile=coverage.out ./pkg/...
go tool cover -html=coverage.out -o coverage.html

# Run with race detection
go test -race ./pkg/... ./internal/...
```

See [Testing Guide](TESTING.md) for detailed testing practices.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup

1. Fork and clone the repository
2. Install prerequisites: Go 1.23+, golangci-lint
3. Run tests: `go test -v ./...`
4. Run linter: `golangci-lint run`
5. Create a branch and make your changes
6. Submit a pull request

## License

This project is released under the [MIT License](LICENSE).

## Support

- **Issues**: [GitHub Issues](https://github.com/InkyQuill/gitlab-mcp-server/issues)
- **Documentation**: See the [docs](docs/) directory
- **Model Context Protocol**: [Official MCP Documentation](https://modelcontextprotocol.io/)

## Acknowledgments

This project is a fork of [LuisCusihuaman/gitlab-mcp-server](https://github.com/LuisCusihuaman/gitlab-mcp-server), substantially rewritten with improved testing, error handling, and additional features.
