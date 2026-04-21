# Editor Setup

The fastest way to wire the MCP server into an IDE is `gitlab-mcp-server install <claude|vscode|cursor|all>`. This page covers manual setup for clients the installer doesn't yet handle (Claude Code) and for environments where you'd rather edit JSON by hand.

> **Auth reminder.** The snippets below set `GITLAB_TOKEN`/`GITLAB_HOST` env vars, which use the **deprecated fallback path** (single-server, plaintext token in config). Prefer the global config file — see [CONFIGURATION.md](CONFIGURATION.md). Once `~/.gitlab-mcp-server/gitlab-mcp-server-config.json` exists, the IDE entry only needs `command` + `args`.

## Config file locations

| Client | Linux | macOS | Windows |
|---|---|---|---|
| Claude Desktop | `~/.config/Claude/claude_desktop_config.json` | `~/Library/Application Support/Claude/claude_desktop_config.json` | `%APPDATA%\Claude\claude_desktop_config.json` |
| Claude Code | `~/.claude.json` | `~/.claude.json` | `%USERPROFILE%\.claude.json` |
| VS Code | `~/.config/Code/User/settings.json` | `~/Library/Application Support/Code/User/settings.json` | `%APPDATA%\Code\User\settings.json` |
| Cursor | `~/.cursor/mcp.json` | `~/.cursor/mcp.json` | `%APPDATA%\Cursor\mcp.json` |

Restart the client after editing any of these.

## Claude Desktop

```json
{
  "mcpServers": {
    "gitlab": {
      "command": "/absolute/path/to/gitlab-mcp-server",
      "args": ["stdio"]
    }
  }
}
```

The `install claude` command writes this entry for you. Add `"env": {"GITLAB_TOKEN": "…"}` only if you want the env-var fallback.

## Claude Code

Claude Code manages MCP servers via `claude mcp …`:

```bash
claude mcp add gitlab -s user -- /absolute/path/to/gitlab-mcp-server stdio
claude mcp list
```

Scopes: `-s user` (all projects), `-s project` (writes `.mcp.json` in the repo root), or omit for the current directory.

Manual edit of `~/.claude.json`:

```json
{
  "mcpServers": {
    "gitlab": {
      "type": "stdio",
      "command": "/absolute/path/to/gitlab-mcp-server",
      "args": ["stdio"]
    }
  }
}
```

## VS Code

Add to user `settings.json` (`Preferences: Open User Settings (JSON)`):

```json
{
  "mcp.servers": {
    "gitlab": {
      "command": "/absolute/path/to/gitlab-mcp-server",
      "args": ["stdio"]
    }
  }
}
```

For workspace-shared config, write the same `servers` object (without the `mcp.` prefix) to `.vscode/mcp.json`.

See the [VS Code MCP docs](https://code.visualstudio.com/docs/copilot/chat/mcp-servers) for prompts and input bindings.

## Cursor

Edit `~/.cursor/mcp.json` (Linux/macOS) or `%APPDATA%\Cursor\mcp.json` (Windows):

```json
{
  "mcpServers": {
    "gitlab": {
      "command": "/absolute/path/to/gitlab-mcp-server",
      "args": ["stdio"]
    }
  }
}
```

## Docker

Replace `command` and `args` with a Docker invocation:

```json
{
  "command": "docker",
  "args": [
    "run", "-i", "--rm",
    "-e", "GITLAB_TOKEN",
    "-e", "GITLAB_HOST",
    "gitlab-mcp-server:latest"
  ],
  "env": {
    "GITLAB_TOKEN": "glpat-…",
    "GITLAB_HOST": "https://gitlab.com"
  }
}
```

Notes:
- Docker can't read your OS keyring, so the token must come through env vars (or a mounted secrets file).
- For self-managed GitLab with a private CA, see [SELF_HOSTED.md](SELF_HOSTED.md).

## Passing options to the server

Any CLI flag has an env-var equivalent (`GITLAB_READ_ONLY`, `GITLAB_TOOLSETS`, `GITLAB_DYNAMIC_TOOLSETS`, etc.). Put them in the `env` block of the IDE entry. Full list: [CONFIGURATION.md](CONFIGURATION.md#environment-variables).

Example — read-only, only issues and MRs:

```json
{
  "mcpServers": {
    "gitlab": {
      "command": "/absolute/path/to/gitlab-mcp-server",
      "args": ["stdio", "--read-only"],
      "env": {
        "GITLAB_TOOLSETS": "issues,merge_requests"
      }
    }
  }
}
```

## Troubleshooting

**Tool doesn't show up.** Restart the IDE. Confirm the JSON parses (`jq . <file>`). Confirm the binary path is absolute — IDEs often don't share your shell's `PATH`.

**Works from CLI, fails in IDE.** Usually a missing token in the IDE's environment. Either put it in `env` (fallback path) or rely on the global config file which the binary reads regardless of how it was started.

**Private CA / self-managed.** See [SELF_HOSTED.md](SELF_HOSTED.md). If using Docker, you may need `-v /path/to/ca.crt:/etc/ssl/certs/ca-certificates.crt:ro` or a custom image.
