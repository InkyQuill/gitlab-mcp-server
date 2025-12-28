# Editor Setup Guide

This guide covers configuring the GitLab MCP Server for different development environments.

## VS Code / VS Code Agent Mode

### Using Docker

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

### Using Standalone Binary

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

## Claude Desktop

The configuration file location varies by operating system:

- **Linux:** `~/.config/Claude/claude_desktop_config.json`
- **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

If the file doesn't exist, create it with the following structure.

### Using Docker

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

### Using Standalone Binary

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

## Claude Code

Claude Code uses a command-line interface to manage MCP servers. The configuration file location varies by operating system:

- **macOS/Linux:** `~/.claude.json`
- **Windows:** `%USERPROFILE%\.claude.json`

### Using Command Line (Recommended)

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

### Using Direct Configuration File Editing

Alternatively, you can directly edit the configuration file:

**Using Docker:**

```json
{
  "mcpServers": {
    "gitlab-go-mcp": {
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

## Cursor

Create or edit the file `~/.cursor/mcp.json` with the following configuration.

### Using Docker

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

### Using Standalone Binary

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

Replace `/path/to/gitlab-mcp-server` with the actual path to your binary (typically `bin/gitlab-mcp-server` relative to the project root).

**Note:** The automated installer (`make install-mcp` or `./setup.sh`) will automatically configure all supported environments with the correct paths and settings.

## Additional Configuration Options

### Read-Only Mode

Enable read-only mode to prevent accidental modifications:

```json
{
  "mcpServers": {
    "gitlab-go-mcp": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "<YOUR_TOKEN>",
        "GITLAB_READ_ONLY": "true"
      }
    }
  }
}
```

### Custom Toolsets

Enable only specific toolsets:

```json
{
  "mcpServers": {
    "gitlab-go-mcp": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "<YOUR_TOKEN>",
        "GITLAB_TOOLSETS": "issues,merge_requests,projects"
      }
    }
  }
}
```

### Dynamic Tool Discovery

Enable dynamic toolset loading:

```json
{
  "mcpServers": {
    "gitlab-go-mcp": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "<YOUR_TOKEN>",
        "GITLAB_DYNAMIC_TOOLSETS": "true"
      }
    }
  }
}
```

## Troubleshooting

### Server Not Appearing

**Problem:** MCP server doesn't appear in the tool list

**Solution:**
- Verify the configuration file syntax (valid JSON)
- Check file location matches your editor
- Restart the editor/application
- Check logs for errors

### Connection Errors

**Problem:** Cannot connect to GitLab MCP server

**Solution:**
- Verify binary path is correct
- Check binary has execute permissions
- Ensure Docker is running (if using Docker)
- Verify environment variables are set correctly

### Token Issues

**Problem:** Authentication errors

**Solution:**
- Verify token is valid in GitLab
- Check token has required scopes
- Ensure `GITLAB_HOST` matches your instance
- See [Token Management](TOKEN_MANAGEMENT.md) for details

