# Multi-Server Setup Guide

This guide explains how to configure GitLab MCP Server to work with multiple GitLab instances (e.g., work and personal GitLab).

## Overview

The GitLab MCP Server supports connecting to multiple GitLab instances simultaneously. Each server configuration has:
- A unique name (e.g., "work", "personal")
- Its own access token
- Its own GitLab host URL
- Optional read-only mode

## Installation

### Single Server (Default)

For a single GitLab server, run the installer:

```bash
node scripts/install.js
```

When prompted:
```
Configure multiple GitLab servers? (y/n, default: n): [press Enter]
```

This creates a single MCP server named `gitlab`.

### Multiple Servers

For multiple GitLab servers:

```bash
node scripts/install.js
```

When prompted:
```
Configure multiple GitLab servers? (y/n, default: n): y
```

Then configure each server:

```
=== Configuring Server 1 ===
Server name (e.g., 'work', 'personal', 'gitlab'): work
Select mode [local/docker] (default: local): [press Enter]
GitLab host URL for 'work' (default: https://gitlab.com, press Enter to use default): https://gitlab.example.com
GitLab access token for 'work': [paste token]
Enable read-only mode? (y/n, default: n): [press Enter]

Add another server? (y/n, default: n): y

=== Configuring Server 2 ===
Server name (e.g., 'work', 'personal', 'gitlab'): personal
Select mode [local/docker] (default: local): [press Enter]
GitLab host URL for 'personal' (default: https://gitlab.com, press Enter to use default): [press Enter]
GitLab access token for 'personal': [paste token]
Enable read-only mode? (y/n, default: n): y

Add another server? (y/n, default: n): [press Enter]
```

## Configuration Files

After installation, your MCP configuration will contain multiple servers:

### Claude Desktop / Cursor (`~/.claude.json`)

```json
{
  "mcpServers": {
    "work": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "glpat-worktoken",
        "GITLAB_HOST": "https://gitlab.example.com"
      }
    },
    "personal": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "glpat-personaltoken",
        "GITLAB_HOST": "https://gitlab.com",
        "GITLAB_READ_ONLY": "true"
      }
    }
  }
}
```

### VS Code (`~/.config/Code/User/settings.json` or workspace `.vscode/settings.json`)

```json
{
  "mcp.servers": {
    "work": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "glpat-worktoken",
        "GITLAB_HOST": "https://gitlab.example.com"
      }
    },
    "personal": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "glpat-personaltoken",
        "GITLAB_HOST": "https://gitlab.com",
        "GITLAB_READ_ONLY": "true"
      }
    }
  }
}
```

## Project Configuration

To specify which GitLab server to use for a project, create a `.gmcprc` file in your project directory:

### Automatic Server Selection

Create `.gmcprc` in your project root:

```json
{
  "projectId": "123",
  "gitlabHost": "https://gitlab.example.com",
  "tokenName": "work"
}
```

The MCP server will automatically use the "work" server when working in this project.

### Alternative: Host-Based Matching

If you don't specify `tokenName`, the server will match by `gitlabHost`:

```json
{
  "projectId": "456",
  "gitlabHost": "https://gitlab.example.com"
}
```

The server will find the client configured with that host.

### Project Auto-Detection

Use the `detectProject` tool to auto-detect from Git remotes:

```
> detectProject
```

This analyzes your `.git/config` and creates/updates `.gmcprc` automatically.

## How It Works

### Client Pool

On startup, the server initializes a **Client Pool** with all configured servers:

```
INFO  Added client 'work' to pool
INFO  Added client 'personal' to pool
INFO  Client resolver initialized with default server 'work'
```

### Client Resolver

When an MCP tool is called, the **Client Resolver** determines which GitLab client to use:

1. Read `.gmcprc` → get `tokenName`
2. If `tokenName` exists → use that client
3. Else, check `gitlabHost` → find matching client by host
4. Fall back to default server

### Example Flow

```
User calls: getProject {projectId: "123"}

Resolver:
  1. Read .gmcprc → {"tokenName": "work", ...}
  2. Get client "work" from pool
  3. Call GitLab API with "work" client's token
  4. Return results
```

## Usage Examples

### Working with Multiple Projects

**Project A** (work GitLab):
```bash
cd /path/to/project-a
echo '{"projectId":"123","tokenName":"work"}' > .gmcprc
# All MCP tools now use the 'work' GitLab server
```

**Project B** (personal GitLab):
```bash
cd /path/to/project-b
echo '{"projectId":"456","tokenName":"personal"}' > .gmcprc
# All MCP tools now use the 'personal' GitLab server
```

### Manual Server Selection

Use `setCurrentProject` to configure manually:

```
> setCurrentProject {
  "projectId": "789",
  "gitlabHost": "https://gitlab.example.com",
  "tokenName": "work"
}
```

### List All Configured Tokens

```
> listTokens
```

Response:
```json
{
  "tokens": [
    {
      "name": "work",
      "gitlabHost": "https://gitlab.example.com",
      "userId": 12345,
      "username": "john_doe",
      "isExpired": false
    },
    {
      "name": "personal",
      "gitlabHost": "https://gitlab.com",
      "userId": 67890,
      "username": "johndoe",
      "isExpired": false
    }
  ],
  "count": 2
}
```

## Runtime Token Management

You can add, update, or remove servers at runtime using MCP tools:

### Add a New Server

```
> addToken {
  "name": "sideproject",
  "token": "glpat-newtoken",
  "gitlabHost": "https://gitlab.sideproject.com"
}
```

**Note:** Runtime additions are lost when the server restarts. For permanent configuration, re-run the installer.

### Update a Server Token

```
> updateToken {
  "name": "work",
  "token": "glpat-newworktoken"
}
```

### Remove a Server

```
> removeToken {
  "name": "sideproject"
}
```

## Migration from Single-Server

If you're currently using a single-server setup and want to add more servers:

1. **Backup your current configuration:**
   ```bash
   cp ~/.claude.json ~/.claude.json.backup
   ```

2. **Re-run the installer:**
   ```bash
   node scripts/install.js
   ```

3. **Select multi-server mode:**
   ```
   Configure multiple GitLab servers? (y/n): y
   ```

4. **Configure your existing server first** (use your current name or "default")

5. **Add additional servers** as needed

6. **Restart your development environment** (VS Code, Claude Desktop, etc.)

Your existing `.gmcprc` files will continue to work without modification.

## Troubleshooting

### Wrong server being used

**Problem:** Tools are using the wrong GitLab server.

**Solution:** Check your `.gmcprc`:
```bash
cat .gmcprc
```

Ensure `tokenName` or `gitlabHost` matches your intended server.

### Server not found

**Problem:** `Client 'work' not found in pool`

**Solution:**
1. Check MCP configuration has the server defined
2. Restart the MCP server
3. Check logs: `listTokens` tool

### All projects using same server

**Problem:** Multiple projects use the same server despite different `.gmcprc` files.

**Solution:** Ensure each `.gmcprc` has a unique `tokenName`:
```json
{
  "projectId": "123",
  "tokenName": "work"
}
```

## Advanced Configuration

### Custom Default Server

By default, the first configured server is the default. To change this, set the `GITLAB_SERVER_NAME` environment variable:

```json
{
  "mcpServers": {
    "work": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "glpat-work",
        "GITLAB_HOST": "https://gitlab.example.com",
        "GITLAB_SERVER_NAME": "work"
      }
    }
  }
}
```

### Read-Only Mode

Enable read-only mode per server to prevent accidental modifications:

```json
{
  "mcpServers": {
    "personal": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "glpat-personal",
        "GITLAB_HOST": "https://gitlab.com",
        "GITLAB_READ_ONLY": "true"
      }
    }
  }
}
```

When read-only mode is enabled, write operations (createIssue, updateMergeRequest, etc.) will be blocked.

## Best Practices

1. **Server Naming**: Use descriptive names (work, personal, client-name, etc.)
2. **Token Scope**: Give each token only the permissions it needs
3. **Read-Only**: Enable read-only for personal/review accounts
4. **Documentation**: Document which server to use for each project in `.gmcprc`
5. **Token Rotation**: Update tokens before expiry using `updateToken`
6. **Configuration Backup**: Keep backups of your MCP configuration files

## Example Workflow

### Setup
```bash
# Install with multiple servers
node scripts/install.js
# Configure: work, personal, client1

# Configure project for work server
cd ~/projects/work-project
echo '{"projectId":"123","tokenName":"work"}' > .gmcprc
detectProject

# Configure project for personal server
cd ~/projects/side-project
echo '{"projectId":"456","tokenName":"personal"}' > .gmcprc
detectProject
```

### Daily Use
```bash
# Work on work project
cd ~/projects/work-project
# All tools use 'work' server automatically

# Work on personal project
cd ~/projects/side-project
# All tools use 'personal' server automatically
```

### Maintenance
```bash
# Check all tokens
listTokens

# Update expired token
updateToken {"name":"work","token":"glpat-newtoken"}

# Check notifications
getNotifications
```
