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

### Building the Server

#### Docker Build

To build the Docker image:

```bash
docker build -t gitlab-mcp-server:latest .
```

#### Standalone Build

To build the binary from source:

```bash
git clone <repository-url>
cd gitlab-mcp-server
go build -o gitlab-mcp-server ./cmd/gitlab-mcp-server
```

The binary will be created as `gitlab-mcp-server` in the current directory.

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
      "gitlab": {
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
      "gitlab": {
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
    "gitlab": {
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
    "Gitlab Local": {
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
    "Gitlab Local": {
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
claude mcp add gitlab -s user -e GITLAB_TOKEN=<YOUR_TOKEN> -e GITLAB_HOST=<YOUR_GITLAB_URL_OR_EMPTY> -- docker run -i --rm -e GITLAB_TOKEN -e GITLAB_HOST gitlab-mcp-server:latest
```

**Using Standalone Binary:**

```bash
claude mcp add gitlab -s user -e GITLAB_TOKEN=<YOUR_TOKEN> -e GITLAB_HOST=<YOUR_GITLAB_URL_OR_EMPTY> -- /path/to/gitlab-mcp-server stdio
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
    "gitlab": {
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
    "Gitlab Local": {
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
    "Gitlab Local": {
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

Replace `/home/inky/Development/gitlab-mcp-server/gitlab-mcp-server` with the actual path to your binary.

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

## GitLab Self-Managed Instances 🏢

To connect to a self-managed GitLab instance instead of `gitlab.com`, use the `--gitlab-host` flag or the `GITLAB_HOST` environment variable. Provide the base URL of your instance (e.g., `https://gitlab.example.com`).

* **Flag:** `./gitlab-mcp-server stdio --gitlab-host https://gitlab.example.com`
* **Environment Variable:** `export GITLAB_HOST="https://gitlab.example.com"`
* **Docker:** `docker run -i --rm -e GITLAB_TOKEN=... -e GITLAB_HOST="https://gitlab.example.com" gitlab-mcp-server:latest`

If the variable/flag is empty or omitted, the server defaults to `https://gitlab.com`.

## Dynamic Tool Discovery 💡

*(This feature might be implemented later, following the pattern from github-mcp-server)*

Instead of starting with a fixed set of enabled tools, dynamic toolset discovery allows the MCP host (like VS Code or Claude) to list available toolsets and enable them selectively in response to user needs. This can prevent overwhelming the language model with too many tools initially.

### Using Dynamic Tool Discovery

If implemented, enable it via:

* **Flag:** `./gitlab-mcp-server stdio --dynamic-toolsets`
* **Environment Variable:** `export GITLAB_DYNAMIC_TOOLSETS=1`
* **Docker:** `docker run -i --rm -e GITLAB_TOKEN=... -e GITLAB_DYNAMIC_TOOLSETS=1 gitlab-mcp-server:latest`

When enabled, the server initially exposes only minimal tools, including tools to list and enable other toolsets dynamically.

## i18n / Overriding Descriptions 🌍

Tool names and descriptions can be customized or translated. Create a `gitlab-mcp-server-config.json` file in the *same directory* as the server binary (or mount it into the container).

The file should contain a JSON object mapping internal translation keys (which correspond to tool names/descriptions) to your desired strings.

**Example `gitlab-mcp-server-config.json`:**
```json
{
  "TOOL_GET_ISSUE_DESCRIPTION": "Fetch details for a specific GitLab issue.",
  "TOOL_CREATE_MERGE_REQUEST_USER_TITLE": "Open New MR"
}
```

You can generate a template file containing all current translation keys by running the server with the `--export-translations` flag:

```bash
./gitlab-mcp-server --export-translations
# This will create/update gitlab-mcp-server-config.json
```

This flag preserves existing overrides while adding any new keys introduced in the server.

## Contributing & License 🤝

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

This project is released under the [MIT License](LICENSE).
