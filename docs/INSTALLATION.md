# Installation

This guide walks through installing the GitLab MCP server, registering it with your IDE, and verifying the setup.

Expected time: **5 minutes**.

## 1. Prerequisites

- **A GitLab access token.** A [Personal](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html), [Project](https://docs.gitlab.com/ee/user/project/settings/project_access_tokens.html), or [Group](https://docs.gitlab.com/ee/user/group/settings/group_access_tokens.html) access token works. Required scopes: **`api`** (covers everything the tools can do) or the narrower pair `read_api` + `read_repository` for read-only use. If you intend to push commits via tools, add `write_repository`.
- **An MCP-capable client.** Tested: Claude Desktop, Claude Code, Cursor, VS Code (with an MCP extension). Any client that speaks stdio MCP will work.
- **One of:**
  - Go **1.25 or newer** (for `go install`), **or**
  - A pre-built binary from the [Releases](https://github.com/InkyQuill/gitlab-mcp-server/releases) page, **or**
  - Docker (for containerized use).

## 2. Install the binary

### Option A — `go install` (recommended)

```bash
go install github.com/InkyQuill/gitlab-mcp-server@latest
```

The binary lands in `$(go env GOBIN)` (typically `~/go/bin`). Make sure that directory is on your `PATH`:

```bash
export PATH="$HOME/go/bin:$PATH"   # add to ~/.bashrc, ~/.zshrc, etc.
```

Verify:

```bash
gitlab-mcp-server version
```

### Option B — download a release

Grab the archive for your platform from [Releases](https://github.com/InkyQuill/gitlab-mcp-server/releases), extract it, and move `gitlab-mcp-server` somewhere on your `PATH` (e.g. `/usr/local/bin`).

`.deb`, `.rpm`, and Arch Linux packages are published alongside each release.

### Option C — build from source

```bash
git clone https://github.com/InkyQuill/gitlab-mcp-server.git
cd gitlab-mcp-server
make build                  # produces bin/gitlab-mcp-server
sudo install bin/gitlab-mcp-server /usr/local/bin/
```

### Option D — Docker

```bash
docker build -t gitlab-mcp-server:latest .
# or pull a published image when available
```

Running under Docker requires passing the token and mounting any config you want the server to read. See [docs/SELF_HOSTED.md](SELF_HOSTED.md) and [docs/CONFIGURATION.md](CONFIGURATION.md) for the env-var fallback path.

## 3. Register with your IDE

Let the `install` command write the MCP entry for you. It detects installed clients, updates their config JSON in place, and writes a `.backup` next to the original.

```bash
gitlab-mcp-server install claude     # Claude Desktop
gitlab-mcp-server install vscode     # VS Code
gitlab-mcp-server install cursor     # Cursor
gitlab-mcp-server install all        # every detected client

gitlab-mcp-server install status     # which clients are installed
gitlab-mcp-server install --dry-run  # preview the change without writing
```

Config file locations (managed for you):

| Client | Linux | macOS | Windows |
|---|---|---|---|
| Claude Desktop | `~/.config/Claude/claude_desktop_config.json` | `~/Library/Application Support/Claude/claude_desktop_config.json` | `%APPDATA%\Claude\claude_desktop_config.json` |
| VS Code | `~/.config/Code/User/settings.json` | `~/Library/Application Support/Code/User/settings.json` | `%APPDATA%\Code\User\settings.json` |
| Cursor | `~/.cursor/mcp.json` (Linux/macOS) | same | `%APPDATA%\Cursor\mcp.json` |

Prefer to edit configs by hand (or configure Claude Code, which is not yet auto-installed)? See [docs/EDITOR_SETUP.md](EDITOR_SETUP.md) for copy-paste snippets.

## 4. Configure a GitLab server

The server authenticates against one or more named GitLab instances recorded in `~/.gitlab-mcp-server/gitlab-mcp-server-config.json`.

### Interactive (recommended)

```bash
gitlab-mcp-server config init
```

Walks you through host, token, and backend selection. The token is entered with no echo and is never written to the config file in plaintext by default — it goes to the OS keyring (or another backend you choose).

### Non-interactive

```bash
# Token prompted (no echo), stored in OS keyring:
gitlab-mcp-server config add work --host https://gitlab.com

# Token already lives in a backend (1Password, pass, Vault, …):
gitlab-mcp-server config add work --host https://gitlab.com --token-ref op://Work/gitlab/token

# Multiple servers — the first one added is default; change with:
gitlab-mcp-server config default personal
```

List, validate, or remove later:

```bash
gitlab-mcp-server config list
gitlab-mcp-server config validate
gitlab-mcp-server config remove old-server
```

Secret backend setup (keyring, encrypted file, external command like `op`/`pass`/`vault`) is covered in [docs/CONFIGURATION.md](CONFIGURATION.md#secret-backends).

## 5. (Optional) Pin the current project

In a project directory, create `.gmcprc` so tools know which GitLab project `cwd` maps to:

```bash
cd your-project
gitlab-mcp-server project init       # detects from .git/config
gitlab-mcp-server project status     # shows the effective config
```

Details: [docs/PROJECT_CONFIG.md](PROJECT_CONFIG.md).

## 6. Restart the IDE and verify

Most clients only load MCP config at startup. After restarting:

1. Check that the GitLab MCP server appears in the client's tool list (method varies per client).
2. Ask the assistant to run `getCurrentUser` — it should return your GitLab username.

From the shell, a quick sanity check:

```bash
gitlab-mcp-server config validate   # contacts each server's /user endpoint
```

## Troubleshooting

**`gitlab-mcp-server: command not found`**
Ensure `~/go/bin` (or wherever the binary went) is on `PATH`. In IDE configs, prefer an absolute path (`which gitlab-mcp-server`) so the IDE's shell environment doesn't matter.

**Permission denied**
`chmod +x ~/go/bin/gitlab-mcp-server` (Linux/macOS).

**Token validation fails on startup**
Re-run `gitlab-mcp-server config validate`. Common causes: expired token, wrong host, scopes missing (`api` is the safe default), or a corporate proxy stripping TLS. For self-managed GitLab with a private CA, see [docs/SELF_HOSTED.md](SELF_HOSTED.md).

**Tool appears but errors "server not found"**
You have multiple servers configured and no default. Set one with `gitlab-mcp-server config default <name>`, or pass `server` explicitly on tool calls. Strict mode (`GITLAB_MCP_STRICT_RESOLVER=1`) requires explicit `server` on every call.

**IDE doesn't pick up the server**
Confirm the IDE restarted, the config file is valid JSON (the installer creates a `.backup` you can diff against), and the path to the binary is correct. `gitlab-mcp-server install status` reports whether each client's config file exists.

## Next steps

- [docs/CONFIGURATION.md](CONFIGURATION.md) — config schema, backends, environment variables.
- [docs/CLI_REFERENCE.md](CLI_REFERENCE.md) — every command and flag.
- [docs/TOOLS.md](TOOLS.md) — what each tool does and how to call it.
- [docs/MULTI_SERVER_SETUP.md](MULTI_SERVER_SETUP.md) — work + personal + self-hosted.
- [docs/TOKEN_MANAGEMENT.md](TOKEN_MANAGEMENT.md) — rotation and expiry handling.
