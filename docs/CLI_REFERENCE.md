# GitLab MCP Server - CLI Reference

Complete reference for all GitLab MCP Server command-line interface commands.

## Installation

```bash
go install github.com/InkyQuill/gitlab-mcp-server@latest
```

Or build from source:

```bash
git clone https://github.com/InkyQuill/gitlab-mcp-server.git
cd gitlab-mcp-server
make build
```

The binary will be created at `bin/gitlab-mcp-server`.

## Quick Start

```bash
# 1. Register MCP server in your IDE
gitlab-mcp-server install claude

# 2. Configure GitLab servers
gitlab-mcp-server config init

# 3. Set up project
cd my-project
gitlab-mcp-server project init

# 4. Start server (usually done automatically by IDE)
gitlab-mcp-server stdio
```

## Global Options

These options can be used with any command:

| Flag | Environment Variable | Description | Default |
|------|---------------------|-------------|---------|
| `--gitlab-token` | `GITLAB_TOKEN` | GitLab Personal Access Token | Required |
| `--gitlab-host` | `GITLAB_HOST` | GitLab host URL | `https://gitlab.com` |
| `--toolsets` | `GITLAB_TOOLSETS` | Comma-separated toolsets to enable | All toolsets |
| `--read-only` | `GITLAB_READ_ONLY` | Enable read-only mode | `false` |
| `--log-level` | `GITLAB_LOG_LEVEL` | Log level (debug, info, warn, error) | `info` |
| `--log-file` | `GITLAB_LOG_FILE` | Path to log file | stderr |
| `--enable-command-logging` | `GITLAB_ENABLE_COMMAND_LOGGING` | Log JSON-RPC requests/responses | `false` |
| `--dynamic-toolsets` | `GITLAB_DYNAMIC_TOOLSETS` | Enable dynamic tool discovery | `false` |
| `--export-translations` | `GITLAB_EXPORT_TRANSLATIONS` | Export translations and exit | `false` |

## Commands

### config - Manage GitLab server configurations

Manage your GitLab server configurations and tokens.

#### config init

Initialize GitLab server configuration. This is an interactive command that guides you through setting up one or more GitLab servers.

```bash
gitlab-mcp-server config init
```

**Prompts:**
- Server name (e.g., 'work', 'personal', 'gitlab')
- GitLab host URL (default: `https://gitlab.com`)
- GitLab access token
- Read-only mode preference
- Additional servers (optional)

**Example output:**

```
=== GitLab MCP Server Configuration ===

Server name (e.g., 'work', 'personal', 'gitlab'): personal
GitLab host URL (default: https://gitlab.com): [press Enter]
GitLab access token: [paste token]
Enable read-only mode? (y/n, default: n): n

Configuration saved to ~/.config/gitlab-mcp-server/config.json
```

#### config add

Add a new GitLab server configuration.

```bash
gitlab-mcp-server config add <name> [flags]
```

**Flags:**
- `--host` - GitLab host URL (default: `https://gitlab.com`)
- `--token` - GitLab Personal Access Token (required)
- `--read-only` - Enable read-only mode

**Examples:**

```bash
# Add a work GitLab instance
gitlab-mcp-server config add work \
  --host https://gitlab.company.com \
  --token glpat-xxxxxxxxxxxx

# Add a personal read-only server
gitlab-mcp-server config add personal \
  --token glpat-yyyyyyyyyyyy \
  --read-only
```

#### config list

List all configured GitLab servers.

```bash
gitlab-mcp-server config list
```

**Example output:**

```
Configured GitLab Servers:

  personal (default)
    Host: https://gitlab.com
    User: john_doe
    Read-only: false

  work
    Host: https://gitlab.company.com
    User: john.doe
    Read-only: false
```

#### config remove

Remove a GitLab server configuration.

```bash
gitlab-mcp-server config remove <name>
```

**Example:**

```bash
gitlab-mcp-server config remove old-server
```

**Warning:** This removes the server from your configuration. Projects using this server will fall back to other configured servers or fail.

#### config default

Set the default GitLab server.

```bash
gitlab-mcp-server config default <name>
```

**Example:**

```bash
gitlab-mcp-server config default work
```

#### config validate

Validate one or all GitLab server configurations.

```bash
gitlab-mcp-server config validate [name]
```

**Examples:**

```bash
# Validate all servers
gitlab-mcp-server config validate

# Validate specific server
gitlab-mcp-server config validate work
```

**Example output:**

```
Validating GitLab server configurations...

  personal: OK (user: john_doe)
  work: OK (user: john.doe)
```

### project - Manage project .gmcprc files

Manage project-specific configuration files for automatic project detection.

#### project init

Initialize project configuration by detecting from Git remote or prompting for details.

```bash
gitlab-mcp-server project init [flags]
```

**Flags:**
- `--project-id` - Manually specify project ID (e.g., `group/project`)
- `--host` - GitLab host URL

**Examples:**

```bash
# Auto-detect from Git remote
gitlab-mcp-server project init

# Specify project manually
gitlab-mcp-server project init --project-id mygroup/myproject

# Self-hosted GitLab
gitlab-mcp-server project init --project-id team/project --host https://gitlab.company.com
```

**Creates:** `.gmcprc` file in current directory

**Example output:**

```
Detecting GitLab project from Git remote...
Found project: mygroup/myproject on https://gitlab.com
Creating .gmcprc file...
Project configured successfully!
```

#### project detect

Detect GitLab project from Git remote without creating configuration file.

```bash
gitlab-mcp-server project detect
```

**Example output:**

```
Git remote detected:
  Host: https://gitlab.com
  Project: mygroup/myproject
  URL: https://gitlab.com/mygroup/myproject.git

To configure, run: gitlab-mcp-server project init
```

#### project status

Show current project configuration status.

```bash
gitlab-mcp-server project status
```

**Example output:**

```
Project Configuration Status:

  .gmcprc file: Found
  Project ID: mygroup/myproject
  GitLab Host: https://gitlab.com
  Token Name: personal
  Last Updated: 2025-02-27 10:30:00

Git Remote:
  URL: git@gitlab.com:mygroup/myproject.git
  Detected Project: mygroup/myproject
```

### install - Register MCP server in IDE

Register the GitLab MCP server with your development environment.

#### install claude

Register the server with Claude Desktop.

```bash
gitlab-mcp-server install claude [flags]
```

**Flags:**
- `--config-path` - Custom config file path
- `--server-name` - Server name (default: `gitlab-mcp-server`)

**Example:**

```bash
gitlab-mcp-server install claude
```

**Modifies:**
- Linux: `~/.config/Claude/claude_desktop_config.json`
- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`

#### install vscode

Register the server with VS Code.

```bash
gitlab-mcp-server install vscode [flags]
```

**Flags:**
- `--scope` - Configuration scope: `user` (default) or `workspace`
- `--server-name` - Server name (default: `gitlab-mcp-server`)

**Examples:**

```bash
# User-level configuration
gitlab-mcp-server install vscode

# Workspace-level configuration
gitlab-mcp-server install vscode --scope workspace
```

**Modifies:**
- User: `~/.config/Code/User/settings.json`
- Workspace: `.vscode/settings.json`

#### install cursor

Register the server with Cursor.

```bash
gitlab-mcp-server install cursor [flags]
```

**Flags:**
- `--server-name` - Server name (default: `gitlab-mcp-server`)

**Example:**

```bash
gitlab-mcp-server install cursor
```

**Modifies:** `~/.cursor/mcp.json`

### stdio - Start MCP server

Start the MCP server using stdio transport. This is the main command used by IDEs to communicate with the server.

```bash
gitlab-mcp-server stdio [flags]
```

**Flags:** See [Global Options](#global-options)

**Examples:**

```bash
# Start with default settings
gitlab-mcp-server stdio

# Start with specific toolsets
gitlab-mcp-server stdio --toolsets issues,merge_requests

# Start in read-only mode
gitlab-mcp-server stdio --read-only

# Start with debug logging
gitlab-mcp-server stdio --log-level debug --log-file /tmp/mcp.log

# Start with dynamic tool discovery
gitlab-mcp-server stdio --dynamic-toolsets
```

**Note:** This command is typically called automatically by your IDE's MCP integration. You don't usually need to run it manually.

### version - Show version info

Display version information for the GitLab MCP Server.

```bash
gitlab-mcp-server version
```

**Example output:**

```
GitLab MCP Server
Version: 2.0.0
Commit: abc123def
Build Date: 2025-02-27T10:30:00Z
```

## Configuration

### Global Config Location

GitLab MCP Server stores its configuration in:

- **Linux/macOS:** `~/.config/gitlab-mcp-server/config.json`
- **Windows:** `%APPDATA%\gitlab-mcp-server\config.json`

### Config Structure

```json
{
  "default": "personal",
  "servers": {
    "personal": {
      "host": "https://gitlab.com",
      "token": "glpat-xxxxxxxxxxxx",
      "readOnly": false
    },
    "work": {
      "host": "https://gitlab.company.com",
      "token": "glpat-yyyyyyyyyyyy",
      "readOnly": false
    }
  }
}
```

### .gmcprc Format

The `.gmcprc` file stores project-specific configuration:

```json
{
  "projectId": "mygroup/myproject",
  "gitlabHost": "https://gitlab.com",
  "tokenName": "personal",
  "lastUpdated": "2025-02-27T10:30:00Z"
}
```

**Fields:**
- `projectId` (required) - GitLab project path or numeric ID
- `gitlabHost` (optional) - GitLab instance URL
- `tokenName` (optional) - Name of server to use from global config
- `lastUpdated` (auto-generated) - Last modification timestamp

### Server Resolution Order

When a tool needs a GitLab client, the server resolves it in this order:

1. **Explicit `tokenName`** in `.gmcprc` - Use the named server
2. **Host matching** - Find server with matching `gitlabHost`
3. **Default server** - Use the globally configured default
4. **First available** - Use the first configured server
5. **Error** - Fail if no servers are configured

## Examples

### First-Time Setup

```bash
# Install the binary
go install github.com/InkyQuill/gitlab-mcp-server@latest

# Configure GitLab servers
gitlab-mcp-server config init

# Register with Claude Desktop
gitlab-mcp-server install claude

# Restart Claude Desktop
```

### Working with Multiple Projects

```bash
# Project 1 - Personal GitLab
cd ~/projects/personal-project
gitlab-mcp-server project init

# Project 2 - Work GitLab
cd ~/projects/work-project
gitlab-mcp-server project init --host https://gitlab.company.com
```

### Dynamic Tool Discovery

```bash
# Start server with dynamic toolsets
gitlab-mcp-server stdio --dynamic-toolsets
```

This starts with only discovery tools enabled. You can then enable toolsets on-demand via MCP tools.

### Debugging

```bash
# Enable debug logging
gitlab-mcp-server stdio --log-level debug --log-file /tmp/gitlab-mcp-debug.log

# Enable JSON-RPC logging (for debugging MCP protocol)
gitlab-mcp-server stdio --enable-command-logging
```

## Troubleshooting

### Command Not Found

**Problem:** `gitlab-mcp-server: command not found`

**Solutions:**
- Ensure `~/go/bin` is in your PATH
- Use the full path: `~/go/bin/gitlab-mcp-server`
- Reinstall with: `go install github.com/InkyQuill/gitlab-mcp-server@latest`

### Permission Denied

**Problem:** `Permission denied` when running the binary

**Solution:**
```bash
chmod +x $(which gitlab-mcp-server)
```

### Config Not Found

**Problem:** `No GitLab servers configured`

**Solution:**
```bash
gitlab-mcp-server config init
```

### Project Not Detected

**Problem:** `Could not detect GitLab project from Git remote`

**Solutions:**
- Ensure you're in a Git repository
- Check the remote is a GitLab URL
- Specify manually: `gitlab-mcp-server project init --project-id group/project`

### Token Validation Failed

**Problem:** `Token validation failed: 401 Unauthorized`

**Solutions:**
- Verify token is valid in GitLab settings
- Check token hasn't expired
- Ensure token has required scopes (`api`, `read_repository`)
- Update token: `gitlab-mcp-server config add <name> --token <new-token>`

## See Also

- [Installation Guide](INSTALLATION.md) - Detailed installation instructions
- [Configuration Guide](CONFIGURATION.md) - Advanced configuration options
- [Tools Reference](TOOLS.md) - Complete MCP tools reference
- [Project Configuration](PROJECT_CONFIG.md) - Using `.gmcprc` files
- [Multi-Server Setup](MULTI_SERVER_SETUP.md) - Configuring multiple GitLab instances
