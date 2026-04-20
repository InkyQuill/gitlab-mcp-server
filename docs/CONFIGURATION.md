# Configuration Guide

Complete guide to configuring GitLab MCP Server for your development environment.

## Global Configuration

### File Location

GitLab MCP Server stores its global configuration in:

| Platform | Location |
|----------|----------|
| Linux | `~/.config/gitlab-mcp-server/config.json` |
| macOS | `~/.config/gitlab-mcp-server/config.json` |
| Windows | `%APPDATA%\gitlab-mcp-server\config.json` |

The configuration directory is created automatically when you run:

```bash
gitlab-mcp-server config init
```

### Config Structure

The global configuration file defines your GitLab servers and their access tokens:

```json
{
  "default": "personal",
  "servers": {
    "personal": {
      "host": "https://gitlab.com",
      "token": "glpat-xxxxxxxxxxxxxxxxxxxx",
      "readOnly": false
    },
    "work": {
      "host": "https://gitlab.company.com",
      "token": "glpat-yyyyyyyyyyyyyyyyyyyy",
      "readOnly": false
    }
  }
}
```

**Fields:**
- `default` - Name of the default server to use
- `servers` - Map of server configurations
  - `host` - GitLab instance URL
  - `token` - Personal Access Token
  - `readOnly` - Whether to restrict to read-only operations

### Token Encryption

GitLab MCP Server supports optional token encryption using your system's secure keyring. When enabled, tokens are encrypted with AES-256-GCM before being stored in the configuration file.

**Platform Support:**
| Platform | Keyring Backend |
|----------|-----------------|
| Linux | dbus/libsecret (gnome-keyring, KDE Wallet) |
| macOS | System Keychain |
| Windows | Windows Credential Manager |

**Enabling Encryption:**

```bash
# Interactive - will prompt for encryption preference
gitlab-mcp-server config init

# Non-interactive with encryption
gitlab-mcp-server config init --encrypt-tokens --name work --host https://gitlab.com --token glpat-xxxxx

# Add server with encryption (enables encryption if not already enabled)
gitlab-mcp-server config add personal --host https://gitlab.com --token glpat-xxxxx --encrypt
```

**How It Works:**

1. When encryption is enabled, a 256-bit master key is generated and stored in your system keyring
2. Each token is encrypted with AES-256-GCM before being written to the config file
3. Tokens are automatically decrypted when needed (no manual action required)
4. Encrypted tokens appear as random base64 data in the config file

**Encrypted Config Example:**

```json
{
  "version": "1.0",
  "servers": {
    "personal": {
      "name": "personal",
      "host": "https://gitlab.com",
      "token": "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY3ODkwYWJjZGVm",
      "readOnly": false,
      "isDefault": true
    }
  }
}
```

**Migration:**

If you have existing tokens and want to enable encryption:

1. Export your current tokens
2. Re-initialize with encryption enabled
3. Re-add your servers - existing unencrypted tokens will still work

**Disabling Encryption:**

To disable encryption, simply remove the master key from your keyring or re-initialize without the `--encrypt-tokens` flag. New tokens will be stored in plaintext.

### Example: Multiple GitLab Instances

Configure multiple GitLab instances (e.g., work and personal):

```bash
# Interactive setup
gitlab-mcp-server config init
```

Or manually edit `~/.config/gitlab-mcp-server/config.json`:

```json
{
  "default": "work",
  "servers": {
    "work": {
      "host": "https://gitlab.company.com",
      "token": "glpat-worktoken123",
      "readOnly": false
    },
    "personal": {
      "host": "https://gitlab.com",
      "token": "glpat-personaltoken456",
      "readOnly": true
    },
    "client": {
      "host": "https://gitlab.client.com",
      "token": "glpat-clienttoken789",
      "readOnly": false
    }
  }
}
```

## Project Configuration (.gmcprc)

### Auto-Detection

The server can automatically detect your GitLab project from Git remotes. Use the CLI command:

```bash
cd your-project
gitlab-mcp-server project init
```

This creates a `.gmcprc` file with the detected project information.

### Manual Configuration

Create a `.gmcprc` file manually in your project root:

```json
{
  "projectId": "mygroup/myproject",
  "gitlabHost": "https://gitlab.com",
  "tokenName": "personal"
}
```

**Fields:**
- `projectId` (required) - Project path (`group/project`) or numeric ID
- `gitlabHost` (optional) - GitLab instance URL
- `tokenName` (optional) - Name of server from global config

### Search Path (Upward Directory Search)

The server searches for `.gmcprc` files starting from the current directory and moving upward:

```
/home/user/projects/myproject/src/package/
  -> Check: /home/user/projects/myproject/src/package/.gmcprc
  -> Check: /home/user/projects/myproject/src/.gmcprc
  -> Check: /home/user/projects/myproject/.gmcprc      <- Found!
  -> Check: /home/user/projects/.gmcprc
  -> Check: /home/user/.gmcprc
```

This allows you to:
- Have a single `.gmcprc` at your project root
- Override settings in subdirectories if needed

## Server Resolution

### Resolution Order

When a tool needs a GitLab client, the server resolves it in this order:

1. **Explicit `tokenName`** in `.gmcprc` - Use the named server
2. **Host matching** - Find server with matching `gitlabHost`
3. **Default server** - Use the globally configured default
4. **First available** - Use the first configured server
5. **Error** - Fail if no servers are configured

### Example: Resolution Flow

**Given:**
- Global config with servers: `personal` (gitlab.com), `work` (gitlab.company.com)
- `.gmcprc`: `{"projectId": "team/project", "tokenName": "work"}`

**Flow:**
```
1. Tool called: getProject
2. Read .gmcprc -> tokenName = "work"
3. Get client "work" from pool
4. Use "work" token and host
```

**Without tokenName:**

**Given:**
- `.gmcprc`: `{"projectId": "team/project", "gitlabHost": "https://gitlab.company.com"}`

**Flow:**
```
1. Tool called: getProject
2. Read .gmcprc -> gitlabHost = "https://gitlab.company.com"
3. Find server with matching host -> "work"
4. Use "work" token and host
```

## Environment Variables

You can configure the server using environment variables instead of global config. This is useful for:
- Containerized deployments
- CI/CD pipelines
- Temporary overrides

| Variable | Description | Example |
|----------|-------------|---------|
| `GITLAB_TOKEN` | GitLab access token | `glpat-xxxxxx` |
| `GITLAB_HOST` | GitLab host URL | `https://gitlab.com` |
| `GITLAB_READ_ONLY` | Enable read-only mode | `true` |
| `GITLAB_TOOLSETS` | Enabled toolsets (comma-separated) | `issues,merge_requests` |
| `GITLAB_DYNAMIC_TOOLSETS` | Enable dynamic tool discovery | `true` |
| `GITLAB_LOG_LEVEL` | Logging level | `debug` |
| `GITLAB_LOG_FILE` | Log file path | `/tmp/gitlab-mcp.log` |
| `GITLAB_ENABLE_COMMAND_LOGGING` | Log JSON-RPC messages | `true` |

### Example: Environment-Based Configuration

```bash
export GITLAB_TOKEN="glpat-xxxxxxxxxxxx"
export GITLAB_HOST="https://gitlab.com"
export GITLAB_TOOLSETS="projects,issues,merge_requests"

gitlab-mcp-server stdio
```

## IDE Configuration

### VS Code

**User-level** (`~/.config/Code/User/settings.json`):

```json
{
  "mcp.servers": {
    "gitlab": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "glpat-xxxxxxxxxxxx",
        "GITLAB_HOST": "https://gitlab.com"
      }
    }
  }
}
```

**Workspace-level** (`.vscode/settings.json`):

```json
{
  "mcp.servers": {
    "gitlab": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "${env:GITLAB_TOKEN}",
        "GITLAB_HOST": "https://gitlab.company.com"
      }
    }
  }
}
```

### Claude Desktop

**Linux:** `~/.config/Claude/claude_desktop_config.json`
**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "gitlab": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "glpat-xxxxxxxxxxxx",
        "GITLAB_HOST": "https://gitlab.com"
      }
    }
  }
}
```

### Cursor

**File:** `~/.cursor/mcp.json`

```json
{
  "mcpServers": {
    "gitlab": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "glpat-xxxxxxxxxxxx",
        "GITLAB_HOST": "https://gitlab.com"
      }
    }
  }
}
```

## Advanced Configuration

### Toolsets

Control which tool categories are enabled:

```bash
# Via CLI
gitlab-mcp-server stdio --toolsets projects,issues,merge_requests

# Via environment
export GITLAB_TOOLSETS="projects,issues,merge_requests"

# Via IDE config
"env": {
  "GITLAB_TOOLSETS": "projects,issues,merge_requests"
}
```

**Available toolsets:**
- `projects` - Project management tools
- `issues` - Issue tracking tools
- `merge_requests` - Merge request tools
- `search` - Search and discovery tools
- `users` - User management tools
- `security` - Security scanning tools
- `token_management` - Token management tools
- `project_config` - Project configuration tools

### Read-Only Mode

Restrict the server to read-only operations:

```bash
# Via CLI
gitlab-mcp-server stdio --read-only

# Via environment
export GITLAB_READ_ONLY="true"

# Via IDE config
"env": {
  "GITLAB_READ_ONLY": "true"
}
```

In read-only mode, write operations (create, update, delete) are blocked.

### Dynamic Tool Discovery

Enable on-demand toolset loading to reduce initial context:

```bash
# Via CLI
gitlab-mcp-server stdio --dynamic-toolsets

# Via environment
export GITLAB_DYNAMIC_TOOLSETS="true"

# Via IDE config
"env": {
  "GITLAB_DYNAMIC_TOOLSETS": "true"
}
```

With dynamic mode:
- Server starts with only discovery tools
- Toolsets are loaded on-demand via MCP tools
- Reduces initial context window usage

### Logging

Configure logging behavior:

```bash
# Debug level to file
gitlab-mcp-server stdio --log-level debug --log-file /tmp/gitlab-mcp.log

# Via environment
export GITLAB_LOG_LEVEL="debug"
export GITLAB_LOG_FILE="/tmp/gitlab-mcp.log"
```

**Log levels:** `debug`, `info` (default), `warn`, `error`

### Command Logging

Enable JSON-RPC message logging for debugging:

```bash
gitlab-mcp-server stdio --enable-command-logging
```

**Warning:** This may log sensitive data. Use only for debugging.

## Migration from installer.js

### Importing Existing Configuration

If you previously used the Node.js installer, your configuration is in your IDE's MCP settings. The new CLI-based configuration provides:

1. **Centralized server management** - All servers in one config file
2. **CLI commands** - Easy addition, removal, and validation
3. **Project auto-detection** - Automatic `.gmcprc` creation

### Manual Migration

1. **Export your existing tokens** from IDE config
2. **Run CLI initialization:**
   ```bash
   gitlab-mcp-server config init
   ```
3. **Add each server** with your existing tokens
4. **Update IDE config** to use environment variables instead of inline tokens

**Before (inline tokens):**
```json
{
  "mcpServers": {
    "gitlab": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "glpat-xxxxxxxxxxxx",
        "GITLAB_HOST": "https://gitlab.com"
      }
    }
  }
}
```

**After (using global config):**
```json
{
  "mcpServers": {
    "gitlab": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"]
    }
  }
}
```

Tokens are now managed in `~/.config/gitlab-mcp-server/config.json`.

## Security Best Practices

### Token Encryption

Enable system keyring encryption for enhanced security:

```bash
# Enable encryption during initialization
gitlab-mcp-server config init --encrypt-tokens

# Or enable when adding a new server
gitlab-mcp-server config add work --host https://gitlab.com --token glpat-xxxxx --encrypt
```

**Benefits:**
- Tokens encrypted with AES-256-GCM
- Master key stored in system keyring
- Automatic decryption when needed
- Protected against config file theft

### Token Storage

1. **Global config** - Tokens stored in `~/.config/gitlab-mcp-server/config.json`
2. **File permissions** - Config file should have restricted permissions (600)
3. **Environment variables** - Alternative for temporary configurations
4. **Keyring encryption** - Recommended for production environments

### Token Scopes

Only request the scopes you need:

| Scope | When to Use |
|-------|-------------|
| `api` | Full API access (recommended for most use cases) |
| `read_api` | Read-only API access |
| `read_repository` | Read repository files |
| `write_repository` | Write to repository |

### Read-Only Mode

Enable read-only mode for sensitive projects:

```json
{
  "servers": {
    "production": {
      "host": "https://gitlab.company.com",
      "token": "glpat-production",
      "readOnly": true
    }
  }
}
```

## Troubleshooting

### Config Not Found

**Problem:** `No GitLab servers configured`

**Solution:**
```bash
gitlab-mcp-server config init
```

### Invalid Token

**Problem:** `Token validation failed: 401 Unauthorized`

**Solutions:**
1. Verify token in GitLab settings
2. Check token expiration
3. Update token: `gitlab-mcp-server config add <name> --token <new-token>`

### Wrong Server Used

**Problem:** Tools using wrong GitLab instance

**Solutions:**
1. Check `.gmcprc` has correct `tokenName` or `gitlabHost`
2. Verify global config default: `gitlab-mcp-server config list`
3. Set default: `gitlab-mcp-server config default <name>`

### Permission Denied

**Problem:** `Cannot read config file`

**Solution:**
```bash
chmod 600 ~/.config/gitlab-mcp-server/config.json
```

## See Also

- [CLI Reference](CLI_REFERENCE.md) - Complete command-line reference
- [Installation Guide](INSTALLATION.md) - Installation and setup
- [Project Configuration](PROJECT_CONFIG.md) - Using `.gmcprc` files
- [Multi-Server Setup](MULTI_SERVER_SETUP.md) - Configuring multiple GitLab instances
- [Token Management](TOKEN_MANAGEMENT.md) - Managing access tokens
