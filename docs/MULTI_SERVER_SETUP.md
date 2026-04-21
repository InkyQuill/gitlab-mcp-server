# Multi-server setup

Most real-world setups need more than one GitLab instance — a work instance, a personal account on GitLab.com, maybe an archived mirror. The server supports this with a single MCP entry in your IDE and multiple named servers in the global config.

## Model

- One binary, one IDE entry.
- The global config (`~/.gitlab-mcp-server/gitlab-mcp-server-config.json`) lists every server you have access to.
- Each configured server has a name, a host, a token (via backend ref), and an optional `readOnly` flag.
- At tool-call time the server is chosen by: the tool's `server` argument, then `.gmcprc`'s `server` field, then the default server.

> Pre-v2.1 setups registered one MCP entry per instance and passed `GITLAB_TOKEN`/`GITLAB_HOST` env vars. That still works but is deprecated and will be removed in v3.0. Migrate with a few `config add` calls.

## Setup

```bash
# Work (self-managed) — token from 1Password
gitlab-mcp-server config add work \
    --host https://gitlab.company.com \
    --token-ref op://Work/gitlab/token

# Personal (GitLab.com) — token prompted, stored in OS keyring
gitlab-mcp-server config add personal --host https://gitlab.com

# Archived mirror — read-only guardrail
gitlab-mcp-server config add mirror \
    --host https://gitlab.internal \
    --read-only

# Pick the default for tool calls that don't specify a server
gitlab-mcp-server config default work
```

Verify:

```bash
gitlab-mcp-server config list
gitlab-mcp-server config validate
```

## Choosing a server at call time

**Per-project pinning (recommended).** In each repo:

```bash
cd ~/projects/company-service    # uses `work`
gitlab-mcp-server project init

cd ~/projects/my-oss             # uses `personal`
gitlab-mcp-server project init
```

`project init` detects the remote host and records the matching server in `.gmcprc`.

**Explicit argument on the tool call.** Any tool accepts a `server` argument that takes precedence:

```json
{ "name": "listIssues", "arguments": { "server": "personal", "projectId": "user/oss" } }
```

**Default.** If neither above is set, the resolver uses the default server.

## Strict mode

Set `GITLAB_MCP_STRICT_RESOLVER=1` to disable implicit defaults:

- Every tool call must include a valid `server`.
- The server's host is verified against the config on every session.
- Typos surface as clear errors instead of silently hitting the default server.

Recommended once you have more than one configured server.

## Switching defaults

```bash
gitlab-mcp-server config default personal
```

Only one server is default at any time. `config list` marks the current default.

## Read-only per server

Setting `--read-only` on a server blocks every write tool when that server is the target, regardless of the process-wide `--read-only` flag. Useful for mirrors or production instances you only want to read.

## Removing a server

```bash
gitlab-mcp-server config remove old-server
```

You can't remove the default server while others exist — switch the default first. Removing a server leaves the backend secret in place (the server never auto-deletes credentials it didn't create).

## Common patterns

### Work + personal, prompt-style workflow

```bash
gitlab-mcp-server config add work     --host https://gitlab.company.com --backend keyring
gitlab-mcp-server config add personal --host https://gitlab.com         --backend keyring
gitlab-mcp-server config default work
cd ~/repos/company-service && gitlab-mcp-server project init
cd ~/repos/my-oss          && gitlab-mcp-server project init
```

### CI / headless machine

Keyring access is often unavailable on a CI runner. Store the token in an encrypted file and decrypt with a key in CI secrets:

```bash
gitlab-mcp-server config add ci --host https://gitlab.company.com --backend file
```

### Ops accounts with escalated privileges

Record the admin account under its own name and pin it explicitly via `server` only when needed:

```bash
gitlab-mcp-server config add admin --host https://gitlab.company.com --token-ref op://Admin/gitlab/token
```

Tools default to your regular account; admin-only calls pass `"server": "admin"`.

## See also

- [CONFIGURATION.md](CONFIGURATION.md)
- [PROJECT_CONFIG.md](PROJECT_CONFIG.md)
- [TOKEN_MANAGEMENT.md](TOKEN_MANAGEMENT.md)
- [SELF_HOSTED.md](SELF_HOSTED.md)
