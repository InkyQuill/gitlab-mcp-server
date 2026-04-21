# CLI Reference

Complete reference for `gitlab-mcp-server` commands. For higher-level task guides see [INSTALLATION.md](INSTALLATION.md) and [CONFIGURATION.md](CONFIGURATION.md).

## Usage

```text
gitlab-mcp-server [global flags] <command> [args]
```

Running the binary with no command prints help. `--version` and the `version` subcommand print the version.

## Global flags

Global flags are accepted by the root command and inherited by subcommands. Most are only meaningful for `stdio`.

| Flag | Env var | Default | Description |
|---|---|---|---|
| `--toolsets` | `GITLAB_TOOLSETS` | `all` | Comma-separated toolset names. |
| `--read-only` | `GITLAB_READ_ONLY` | `false` | Disable every write tool. |
| `--gitlab-host` | `GITLAB_HOST` | `https://gitlab.com` | Fallback host when no config file exists. |
| `--gitlab-token` | `GITLAB_TOKEN` | _(unset)_ | Fallback token. **Deprecated** — use the config file. |
| `--log-level` | `GITLAB_LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error`. |
| `--log-file` | `GITLAB_LOG_FILE` | _(stderr)_ | Append logs to a file. |
| `--enable-command-logging` | `GITLAB_ENABLE_COMMAND_LOGGING` | `false` | Log JSON-RPC frames (treat as sensitive). |
| `--dynamic-toolsets` | `GITLAB_DYNAMIC_TOOLSETS` | `false` | Start with discovery tools; enable toolsets on demand. |
| `--export-translations` | `GITLAB_EXPORT_TRANSLATIONS` | `false` | Write translation keys and exit. |
| `--use-secure-memory` | `GITLAB_USE_SECURE_MEMORY` | `false` | Store tokens in memguard-protected memory. |

Not bound to a flag: `GITLAB_MCP_STRICT_RESOLVER=1` switches the resolver to strict mode (see [CONFIGURATION.md](CONFIGURATION.md#configuration-sources-and-precedence)).

## Commands

### `stdio`

Start the MCP server, reading JSON-RPC requests on stdin and writing responses on stdout. Invoked automatically by IDE clients.

```bash
gitlab-mcp-server stdio
gitlab-mcp-server stdio --read-only --toolsets projects,issues
```

### `config`

Manage the global configuration file at `~/.gitlab-mcp-server/gitlab-mcp-server-config.json`. Run with `-i` (or with no subcommand) to launch the interactive TUI.

#### `config init`

Guided, interactive setup. Creates the config directory and prompts for server name, host, token (no echo), and backend choice.

```bash
gitlab-mcp-server config init
```

#### `config add <name>`

Add a new server non-interactively.

```bash
gitlab-mcp-server config add <name> --host <url> [flags]
```

| Flag | Description |
|---|---|
| `--host` (required) | GitLab host URL. Added `https://` automatically if missing. |
| `--token-ref <ref>` | Use an existing secret ref (e.g. `op://Work/gitlab/token`). |
| `--token <raw>` | **Deprecated** — exposes the token in shell history. |
| `--backend <keyring\|file>` | Backend used when the token is entered interactively or via `--token`. Default: `keyring`. Ignored when `--token-ref` is set. |
| `--read-only` | Restrict this server to read-only operations. |
| `--encrypt` | (Legacy) encrypt the plaintext `token` field using the keyring master key. Prefer `--token-ref`. |

If neither `--token` nor `--token-ref` is given, the token is read from the terminal with echo disabled.

#### `config list`

List all configured servers with redacted token info.

#### `config default <name>`

Set the named server as default.

#### `config validate`

Contact `/user` on every configured server to verify tokens and populate `userId` / `username` / `lastValidated`.

#### `config remove <name>`

Delete a server. The default server cannot be removed while another server exists — change the default first.

#### `config backends`

Inspect and manage registered secret backends.

### `project`

Manage per-project configuration (`.gmcprc`) — the file that tells the server which GitLab project maps to the current working directory. Full format: [PROJECT_CONFIG.md](PROJECT_CONFIG.md).

#### `project init`

Detect the GitLab project from `.git/config` and write `.gmcprc` in the repo root.

#### `project detect`

Run detection without writing anything.

#### `project status`

Show the effective `.gmcprc` for the current directory (walks upward, like git).

### `install`

Register the MCP server with an IDE by editing its JSON config. A `.backup` is written next to the original before modification.

```bash
gitlab-mcp-server install [claude|vscode|cursor|all] [flags]
```

| Flag | Description |
|---|---|
| `--dry-run` | Print the new config; don't write. |
| `--binary-path <path>` | Override the binary path written into the IDE config (default: resolved from `os.Executable`, symlinks followed). |
| `--config-path <path>` | Custom MCP config file path (overrides OS default). |

Subcommands:

- `install status` — report which clients are detected.
- `install path` — print the path to the bundled Node.js installer script (legacy flow).
- `install uninstall` — print manual removal instructions for each client.

For manual IDE integration (including Claude Code), see [EDITOR_SETUP.md](EDITOR_SETUP.md).

### `version`

Print the version, commit, and build date.

```bash
gitlab-mcp-server version
```

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success. |
| 1 | Command error (bad flag, config error, validation failure, etc.). The error is printed to stderr. |

## See also

- [INSTALLATION.md](INSTALLATION.md) — install, IDE registration, verification.
- [CONFIGURATION.md](CONFIGURATION.md) — global config, env vars, secret backends, precedence.
- [TOOLS.md](TOOLS.md) — the MCP tools exposed by `stdio`.
