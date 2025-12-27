# GitLab MCP Server 🦊

The GitLab MCP Server is a [Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction)
server that provides seamless integration with GitLab APIs, enabling advanced
automation and interaction capabilities for developers and AI tools within the GitLab ecosystem.

## Use Cases ✨

- Automating GitLab workflows and processes (e.g., managing issues, merge requests).
- Extracting and analyzing data from GitLab projects and groups.
- Building AI-powered tools and applications that interact with GitLab.

## Prerequisites ⚙️

1. **GitLab Access Token:** You need a GitLab Access Token to authenticate with the API. You can create:
   - A [Personal Access Token (PAT)](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html)
   - A [Project Access Token](https://docs.gitlab.com/ee/user/project/settings/project_access_tokens.html)
   - A [Group Access Token](https://docs.gitlab.com/ee/user/group/settings/group_access_tokens.html)
   
   The required [scopes](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html#available-scopes) depend on the tools you intend to use (e.g., `api` scope grants broad access, or select more granular scopes like `read_repository`, `write_repository`, `read_api`). Grant only the permissions you feel comfortable giving your AI tools.

2. **Docker (optional):** To run the server in a container, you need [Docker](https://www.docker.com/) installed and running. Alternatively, you can build from source (see below).

3. **Go (for building from source):** If building from source, you need [Go](https://go.dev/) installed (version 1.23 or later).

## Installation 🚀

### Quick Setup (Recommended)

The easiest way to set up the GitLab MCP Server is using the automated installer:

**Linux:**
```bash
git clone <repository-url>
cd gitlab-mcp-server
make setup
make install-mcp
```

**macOS/Linux (using setup script):**
```bash
git clone <repository-url>
cd gitlab-mcp-server
./setup.sh
```

**Windows:**
```cmd
git clone <repository-url>
cd gitlab-mcp-server
setup.bat
```

The installer will:
- Check prerequisites (Go version, dependencies)
- Build the server binary
- Prompt you for:
  - Deployment mode: local binary (default) or Docker
  - GitLab host URL (defaults to `https://gitlab.com`)
  - GitLab access token (entered securely, not displayed)
  - Read-only mode preference
  - Development environments to configure (VS Code, Claude Desktop, Claude Code, Cursor)
- Automatically configure your selected development environments
- Create backup copies (`.bak` files) of existing configuration files before modifying them

The server will be registered as `gitlab-go-mcp` in your MCP configurations.

**Note:** The installer intelligently merges with existing configurations, preserving your other MCP server settings.

### Building the Server

#### Using Makefile (Linux)

The project includes a Makefile with the following targets:

- `make setup` - Install prerequisites and dependencies
- `make build` - Build the binary to `bin/gitlab-mcp-server`
- `make clean` - Clean build artifacts
- `make test` - Run tests
- `make docker-build` - Build Docker image
- `make install-mcp` - Build installer and run MCP configuration
- `make help` - Show all available targets

**Example:**
```bash
make setup    # Install prerequisites
make build     # Build the server
make install-mcp  # Configure MCP servers
```

#### Docker Build

To build the Docker image:

```bash
docker build -t gitlab-mcp-server:latest .
```

Or using Makefile:
```bash
make docker-build
```

#### Standalone Build

To build the binary from source:

```bash
git clone <repository-url>
cd gitlab-mcp-server
go build -o bin/gitlab-mcp-server ./cmd/gitlab-mcp-server
```

Or using Makefile:
```bash
make build
```

The binary will be created as `bin/gitlab-mcp-server`.

### Editor Setup

<details>
<summary><strong>VS Code / VS Code Agent Mode</strong></summary>

#### Using Docker

Add the following JSON block to your User Settings (JSON) file (`Preferences: Open User Settings (JSON)` or `Ctrl+Shift+P` / `Cmd+Shift+P`):

```json
{
  "mcp": {
    "inputs": [
      {
        "type": "promptString",
        "id": "gitlab_token",
        "description": "GitLab Access Token (PAT, Project, or Group)",
        "password": true
      },
      {
        "type": "promptString",
        "id": "gitlab_host",
        "description": "GitLab Host (e.g., gitlab.com or self-managed URL, leave empty for gitlab.com)",
        "password": false
      }
    ],
    "servers": {
      "gitlab-go-mcp": {
        "command": "docker",
        "args": [
          "run",
          "-i",
          "--rm",
          "-e", "GITLAB_TOKEN",
          "-e", "GITLAB_HOST",
          "gitlab-mcp-server:latest"
        ],
        "env": {
          "GITLAB_TOKEN": "${input:gitlab_token}",
          "GITLAB_HOST": "${input:gitlab_host}"
        }
      }
    }
  }
}
```

You can also add a similar configuration (without the top-level `mcp` key) to a `.vscode/mcp.json` file in your workspace to share the setup with your team.

#### Using Standalone Binary

For the standalone binary, update your VS Code User Settings (JSON):

```json
{
  "mcp": {
    "servers": {
      "gitlab-go-mcp": {
        "command": "/path/to/gitlab-mcp-server",
        "args": ["stdio"],
        "env": {
          "GITLAB_TOKEN": "<YOUR_TOKEN>",
          "GITLAB_HOST": "<YOUR_GITLAB_URL_OR_EMPTY>"
        }
      }
    }
  }
}
```

Or use `.vscode/mcp.json` in your workspace:

```json
{
  "servers": {
    "gitlab-go-mcp": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "<YOUR_TOKEN>",
        "GITLAB_HOST": "<YOUR_GITLAB_URL_OR_EMPTY>"
      }
    }
  }
}
```

More about using MCP server tools in VS Code's [agent mode documentation](https://code.visualstudio.com/docs/copilot/chat/mcp-servers).

</details>

<details>
<summary><strong>Claude Desktop</strong></summary>

The configuration file location varies by operating system:

- **Linux:** `~/.config/Claude/claude_desktop_config.json`
- **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

If the file doesn't exist, create it with the following structure.

#### Using Docker

```json
{
  "mcpServers": {
    "gitlab-go-mcp": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "-e", "GITLAB_TOKEN",
        "-e", "GITLAB_HOST",
        "gitlab-mcp-server:latest"
      ],
      "env": {
        "GITLAB_TOKEN": "<YOUR_TOKEN>",
        "GITLAB_HOST": "<YOUR_GITLAB_URL_OR_EMPTY>"
      }
    }
  }
}
```

#### Using Standalone Binary

```json
{
  "mcpServers": {
    "gitlab-go-mcp": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "<YOUR_TOKEN>",
        "GITLAB_HOST": "<YOUR_GITLAB_URL_OR_EMPTY>"
      }
    }
  }
}
```

**Note:** After editing the configuration file, restart Claude Desktop for the changes to take effect.

</details>

<details>
<summary><strong>Claude Code</strong></summary>

Claude Code uses a command-line interface to manage MCP servers. The configuration file location varies by operating system:

- **macOS/Linux:** `~/.claude.json`
- **Windows:** `%USERPROFILE%\.claude.json`

#### Using Command Line (Recommended)

**Using Docker:**

```bash
claude mcp add gitlab-go-mcp -s user -e GITLAB_TOKEN=<YOUR_TOKEN> -e GITLAB_HOST=<YOUR_GITLAB_URL_OR_EMPTY> -- docker run -i --rm -e GITLAB_TOKEN -e GITLAB_HOST gitlab-mcp-server:latest
```

**Using Standalone Binary:**

```bash
claude mcp add gitlab-go-mcp -s user -e GITLAB_TOKEN=<YOUR_TOKEN> -e GITLAB_HOST=<YOUR_GITLAB_URL_OR_EMPTY> -- /path/to/gitlab-mcp-server stdio
```

**Verify the installation:**

```bash
claude mcp list
```

#### Using Direct Configuration File Editing

Alternatively, you can directly edit the configuration file:

**Using Docker:**

```json
{
  "mcpServers": {
    "gitlab": {
      "type": "stdio",
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "-e", "GITLAB_TOKEN",
        "-e", "GITLAB_HOST",
        "gitlab-mcp-server:latest"
      ],
      "env": {
        "GITLAB_TOKEN": "<YOUR_TOKEN>",
        "GITLAB_HOST": "<YOUR_GITLAB_URL_OR_EMPTY>"
      }
    }
  }
}
```

**Using Standalone Binary:**

```json
{
  "mcpServers": {
    "gitlab-go-mcp": {
      "type": "stdio",
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "<YOUR_TOKEN>",
        "GITLAB_HOST": "<YOUR_GITLAB_URL_OR_EMPTY>"
      }
    }
  }
}
```

**Scope Options:**

- `-s user` (or omit for default): User-level scope, available in all projects
- `-s project`: Project-level scope, creates `.mcp.json` in the project root for team sharing
- No flag: Local scope, available only in the current directory

**Note:** After editing the configuration file or using the command line, restart Claude Code for the changes to take effect.

</details>

<details>
<summary><strong>Cursor</strong></summary>

Create or edit the file `~/.cursor/mcp.json` with the following configuration.

#### Using Docker

```json
{
  "mcpServers": {
    "gitlab-go-mcp": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "-e", "GITLAB_TOKEN",
        "-e", "GITLAB_HOST",
        "gitlab-mcp-server:latest"
      ],
      "env": {
        "GITLAB_TOKEN": "<YOUR_TOKEN>",
        "GITLAB_HOST": "<YOUR_GITLAB_URL_OR_EMPTY>"
      }
    }
  }
}
```

#### Using Standalone Binary

```json
{
  "mcpServers": {
    "gitlab-go-mcp": {
      "command": "/home/inky/Development/gitlab-mcp-server/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "<YOUR_TOKEN>",
        "GITLAB_HOST": "<YOUR_GITLAB_URL_OR_EMPTY>"
      }
    }
  }
}
```

Replace `/path/to/gitlab-mcp-server` with the actual path to your binary (typically `bin/gitlab-mcp-server` relative to the project root).

**Note:** The automated installer (`make install-mcp` or `./setup.sh`) will automatically configure all supported environments with the correct paths and settings.

</details>

## Available Toolsets 🛠️

The GitLab MCP Server supports enabling or disabling specific groups of functionalities (toolsets) via the `--toolsets` flag or the `GITLAB_TOOLSETS` environment variable. This allows fine-grained control over the GitLab API capabilities exposed to your AI tools. Enabling only necessary toolsets can improve LLM tool selection and reduce context size.

### Implemented Toolsets

The following toolsets are fully implemented and available:

| Toolset | Status | Description | Available Tools |
|---------|--------|-------------|----------------|
| `projects` | ✅ Available | Project details, repository operations (files, branches, commits). | `getProject`, `listProjects`, `getProjectFile`, `listProjectFiles`, `getProjectBranches`, `getProjectCommits` |
| `issues` | ✅ Available | Issue management (CRUD, comments, labels, milestones). | `getIssue`, `listIssues`, `getIssueComments`, `getIssueLabels`, `createIssue`, `updateIssue`, `createIssueComment`, `updateIssueComment`, `getMilestone`, `listMilestones`, `createMilestone`, `updateMilestone` |
| `merge_requests` | ✅ Available | Merge request operations (CRUD, comments). | `getMergeRequest`, `listMergeRequests`, `getMergeRequestComments`, `createMergeRequest`, `updateMergeRequest`, `createMergeRequestComment`, `updateMergeRequestComment` |

### Planned Toolsets

The following toolsets are planned but not yet implemented:

| Toolset | Status | Description |
|---------|--------|-------------|
| `security` | 🔜 Planned | Tools for accessing GitLab security scan results (SAST, DAST, Secret Detection, etc.). |
| `users` | 🔜 Planned | Tools for looking up GitLab user information. |
| `search` | 🔜 Planned | Tools for utilizing GitLab's scoped search capabilities (projects, issues, MRs, code). |

**Note:** All available toolsets are enabled by default if `GITLAB_TOOLSETS` is not set or set to `"all"`. Planned toolsets will be ignored if specified until they are implemented.

### Specifying Toolsets

Pass an allow-list of desired toolsets (comma-separated):

1. **Using Command Line Argument** (when running binary directly):
   ```bash
   ./gitlab-mcp-server stdio --toolsets issues,merge_requests,projects
   ```

2. **Using Environment Variable**:
   ```bash
   export GITLAB_TOOLSETS="issues,merge_requests,projects"
   ./gitlab-mcp-server stdio
   ```
   *(The environment variable `GITLAB_TOOLSETS` takes precedence over the flag.)*

3. **Using Docker**:
   ```bash
   docker run -i --rm \
     -e GITLAB_TOKEN=<your-token> \
     -e GITLAB_HOST=<your-gitlab-url_or_empty> \
     -e GITLAB_TOOLSETS="issues,merge_requests,projects" \
     gitlab-mcp-server:latest
   ```

### The "all" Toolset

Use the special value `all` to explicitly enable all available toolsets:

```bash
./gitlab-mcp-server stdio --toolsets all
# or
export GITLAB_TOOLSETS="all"
./gitlab-mcp-server stdio
# or with Docker
docker run -i --rm \
  -e GITLAB_TOKEN=<your-token> \
  -e GITLAB_TOOLSETS="all" \
  gitlab-mcp-server:latest
```

## Read-Only Mode 🔒

You can restrict the server to read-only operations using the `--read-only` flag or `GITLAB_READ_ONLY` environment variable. When enabled, only read operations are available, and all write operations (create, update, delete) are disabled.

```bash
./gitlab-mcp-server stdio --read-only
# or
export GITLAB_READ_ONLY=true
./gitlab-mcp-server stdio
# or with Docker
docker run -i --rm \
  -e GITLAB_TOKEN=<your-token> \
  -e GITLAB_READ_ONLY=true \
  gitlab-mcp-server:latest
```

## Command Logging 🔍

For debugging purposes, you can enable logging of all MCP JSON-RPC protocol messages (requests and responses) to stderr.

**⚠️ WARNING**: Command logging may expose sensitive data. Only enable in secure environments. Tokens and passwords are redacted, but other sensitive data may leak.

```bash
./gitlab-mcp-server stdio --enable-command-logging
# or
export GITLAB_ENABLE_COMMAND_LOGGING=true
./gitlab-mcp-server stdio
# or with Docker
docker run -i --rm \
  -e GITLAB_TOKEN=<your-token> \
  -e GITLAB_ENABLE_COMMAND_LOGGING=true \
  gitlab-mcp-server:latest
```

**Note**: This feature is intended for development and debugging. When enabled, all JSON-RPC messages will be logged at DEBUG level. Ensure your `--log-level` is set to `debug` to see the logs.

## GitLab Self-Managed Instances 🏢

To connect to a self-managed GitLab instance instead of `gitlab.com`, use the `--gitlab-host` flag or the `GITLAB_HOST` environment variable. Provide the base URL of your instance (e.g., `https://gitlab.example.com`).

* **Flag:** `./gitlab-mcp-server stdio --gitlab-host https://gitlab.example.com`
* **Environment Variable:** `export GITLAB_HOST="https://gitlab.example.com"`
* **Docker:** `docker run -i --rm -e GITLAB_TOKEN=... -e GITLAB_HOST="https://gitlab.example.com" gitlab-mcp-server:latest`

If the variable/flag is empty or omitted, the server defaults to `https://gitlab.com`.

## Dynamic Tool Discovery 💡

Dynamic toolset discovery allows the MCP host (like VS Code or Claude) to list available toolsets and enable them selectively in response to user needs. This prevents overwhelming the language model with too many tools initially and improves performance.

### How It Works

When dynamic tool discovery is enabled:
- The server starts with **only 2 tools** available: `list_available_toolsets` and `enable_toolset`
- You can query which toolsets are available and their descriptions
- Toolsets are loaded on-demand when you enable them
- Once enabled, all tools from that toolset become available

### Using Dynamic Tool Discovery

Enable it via:

* **Flag:** `./gitlab-mcp-server stdio --dynamic-toolsets`
* **Environment Variable:** `export GITLAB_DYNAMIC_TOOLSETS=true`
* **Docker:** `docker run -i --rm -e GITLAB_TOKEN=... -e GITLAB_DYNAMIC_TOOLSETS=true gitlab-mcp-server:latest`

### Available Discovery Tools

#### `list_available_toolsets`
Lists all available GitLab MCP toolsets that can be enabled.

**Example Output:**
```
Available Toolsets (8):
- token_management: Tools for managing GitLab tokens and authentication. [6 tools] (enabled)
- project_config: Tools for managing GitLab project configuration and auto-detection. [4 tools] (disabled)
- projects: Tools for interacting with GitLab projects, repositories, branches, commits, tags. [6 tools] (disabled)
- issues: Tools for CRUD operations on GitLab issues, comments, labels. [10 tools] (disabled)
- merge_requests: Tools for CRUD operations on GitLab merge requests, comments, approvals, diffs. [7 tools] (disabled)
- security: Tools for accessing GitLab security scan results (SAST, DAST, etc.). [0 tools] (disabled)
- users: Tools for looking up GitLab user information. [0 tools] (disabled)
- search: Tools for utilizing GitLab's scoped search capabilities. [0 tools] (disabled)
```

#### `enable_toolset`
Enables a specific GitLab MCP toolset, making its tools available.

**Parameters:**
- `toolset` (required, string): Name of the toolset to enable (e.g., 'projects', 'issues', 'merge_requests')

**Example Workflow:**
1. Start server with `--dynamic-toolsets`
2. Call `list_available_toolsets` to see available toolsets
3. Call `enable_toolset` with toolset name (e.g., "projects") to load it
4. Use the newly available tools from that toolset
5. Repeat step 3 for additional toolsets as needed

**Note:** Token management and project configuration toolsets are enabled by default even in dynamic mode, as they're essential for server operation.

## i18n / Overriding Descriptions 🌍

Tool names and descriptions can be customized or translated to better suit your workflow or language preferences.

### How It Works

1. **Generate Translation Template**: Run the server with `--export-translations` flag
2. **Customize Descriptions**: Edit the generated `gitlab-mcp-server-config.json` file
3. **Restart Server**: The server automatically loads translations on startup

### Generating Translation Template

Create a configuration file with all available translation keys:

```bash
./gitlab-mcp-server stdio --export-translations
```

This creates `gitlab-mcp-server-config.json` in the same directory as the binary with all 29+ translation keys.

### Example: Russian Translation

Create a `gitlab-mcp-server-config.json` file:

```json
{
  "TOOL_GET_PROJECT_DESCRIPTION": "Получает детали конкретного проекта GitLab.",
  "TOOL_LIST_PROJECTS_DESCRIPTION": "Возвращает список проектов GitLab с возможностью фильтрации.",
  "TOOL_GET_ISSUE_DESCRIPTION": "Получает информацию о конкретной задаче GitLab.",
  "TOOL_CREATE_ISSUE_DESCRIPTION": "Создает новую задачу в проекте GitLab."
}
```

### Example: Custom English Descriptions

Make tool descriptions more specific for your team:

```json
{
  "TOOL_GET_ISSUE_DESCRIPTION": "Fetch issue details including assignees, labels, and milestone status",
  "TOOL_CREATE_ISSUE_DESCRIPTION": "Create a new issue. Required: title. Optional: description, assignee, labels, milestone",
  "TOOL_LIST_ISSUES_DESCRIPTION": "List issues with filters. Supports: scope (assigned_to_me, created_by_me, all), labels, milestone, state, search"
}
```

### Translation Keys Reference

The server supports 29+ translation keys covering:
- **Projects** (6 keys): `getProject`, `listProjects`, `getProjectFile`, `listProjectFiles`, `getProjectBranches`, `getProjectCommits`
- **Issues** (8 keys): `getIssue`, `listIssues`, `getIssueComments`, `getIssueLabels`, `createIssue`, `updateIssue`, `createIssueComment`, `updateIssueComment`
- **Merge Requests** (7 keys): `getMergeRequest`, `listMergeRequests`, `getMergeRequestComments`, `createMergeRequest`, `updateMergeRequest`, `createMergeRequestComment`, `updateMergeRequestComment`
- **Milestones** (4 keys): `getMilestone`, `listMilestones`, `createMilestone`, `updateMilestone`
- **Token Management** (4+ keys): `listTokens`, `validateToken`, `addToken`, `updateToken`, `removeToken`

### Configuration File Location

The server looks for `gitlab-mcp-server-config.json` in:
1. **Standalone binary**: Same directory as the binary
2. **Docker**: Mount the config file to `/app/gitlab-mcp-server-config.json`

**Docker Example:**
```bash
docker run -i --rm \
  -v $(pwd)/gitlab-mcp-server-config.json:/app/gitlab-mcp-server-config.json \
  -e GITLAB_TOKEN=<your-token> \
  gitlab-mcp-server:latest
```

## Documentation 📚

For detailed information on specific features, see:

- **[Token Management](docs/TOKEN_MANAGEMENT.md)** - How tokens are validated, tracked, and managed
- **[Multi-Server Setup](docs/MULTI_SERVER_SETUP.md)** - Configure multiple GitLab instances
- **[GitHub Detection](docs/GITHUB_DETECTION.md)** - Why GitHub isn't supported and error handling
- **[Project Configuration](docs/PROJECT_CONFIG.md)** - Using `.gmcprc` files for project-specific settings

### Quick Links

| Topic | Description |
|-------|-------------|
| **Token Validation** | Automatic token validation on startup, expiration tracking, and 401 error handling |
| **Multiple GitLab Servers** | Configure work, personal, and other GitLab instances simultaneously |
| **Project Auto-Detection** | Automatically detect GitLab project from Git remotes |
| **GitHub Detection** | Clear error messages when GitHub repositories are detected |
| **Runtime Token Management** | Add, update, remove tokens using MCP tools (`addToken`, `updateToken`, etc.) |

## Contributing & License 🤝

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

This project is released under the [MIT License](LICENSE).
