# Project configuration (`.gmcprc`)

A `.gmcprc` file in your repository tells the server which GitLab project the current directory maps to, and (optionally) which configured server it belongs to. Once present, tools can omit `projectId` and the server resolves it from the file.

## Location and lookup

The server searches upward from the current working directory for `.gmcprc`, stopping at the filesystem root — the same algorithm `git` uses for `.git/`. So `.gmcprc` at the repo root covers every subdirectory.

## Schema

```json
{
  "projectId":   "mygroup/myproject",
  "server":      "work",
  "lastUpdated": "2026-04-20T09:12:44Z"
}
```

| Field | Required | Description |
|---|---|---|
| `projectId` | yes | GitLab project ID (numeric) or full path (`group/subgroup/project`). |
| `server` | no | Name of a configured server (e.g. `work`, `personal`). Omit to use the default server. |
| `lastUpdated` | auto | ISO 8601 timestamp; written by the CLI / `setCurrentProject`. |

### Deprecated fields

Old `.gmcprc` files may contain these; they still work in v2.x but print a stderr warning on first load and will be removed in v3.0.

- `tokenName` — promoted to `server` automatically.
- `gitlabHost` — no longer needed; each configured server already carries its host.

## CLI

```bash
gitlab-mcp-server project init     # auto-detect from .git/config and write .gmcprc
gitlab-mcp-server project detect   # detect without writing
gitlab-mcp-server project status   # show the effective .gmcprc for the current directory
```

`project init` looks at `.git/config` remote URLs, matches them against known GitLab hosts, and extracts `group/project`. If the remote host matches a configured server's host, that server is recorded as well.

## Tools

When the project_config toolset is enabled, the MCP server exposes:

- **`getCurrentProject`** — returns the effective `.gmcprc` (or `{found: false}`).
- **`setCurrentProject`** — writes `.gmcprc` in the current working directory. Parameters: `projectId` (required), `server` (optional).
- **`detectProject`** — detect without writing.
- **`autoDetectAndSetProject`** — detect and write in one call.

## Resolution order for `projectId`

Tools that accept a `projectId` argument resolve it in this order:

1. Explicit `projectId` in the tool arguments.
2. `.gmcprc` (nearest ancestor of the server's working directory).
3. Error if neither is available.

## Examples

Single GitLab.com project:

```json
{ "projectId": "mygroup/myproject" }
```

Self-hosted project pinned to a specific configured server:

```json
{ "projectId": "engineering/backend", "server": "work" }
```

## Security notes

`.gmcprc` contains no secrets — only IDs and a server name. It's safe to commit. Tokens live in the global config or a secret backend; see [CONFIGURATION.md](CONFIGURATION.md#secret-backends).

## Troubleshooting

**`project init` says "no GitLab remote found".** The remote URL must point at a known GitLab host. For private instances, make sure you've run `config add … --host <your-host>` first; `project init` will match against your configured servers.

**Tool still asks for `projectId` even though `.gmcprc` exists.** The server only reads `.gmcprc` from its own working directory. Make sure the MCP client launches the server in the repo directory (most clients do). You can also call `setCurrentProject` once to write the file from inside a session.

**GitHub URL detected.** The server intentionally rejects GitHub remotes with a friendly error. See [GITHUB_DETECTION.md](GITHUB_DETECTION.md).
