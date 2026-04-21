# Configuration

This document describes the global configuration file, secret backends, environment variables, and precedence rules used by `gitlab-mcp-server`.

For per-project settings (`.gmcprc`) see [PROJECT_CONFIG.md](PROJECT_CONFIG.md). For IDE-specific configuration see [EDITOR_SETUP.md](EDITOR_SETUP.md).

## Configuration sources and precedence

The server merges configuration from several places. At startup:

1. **Global config file** — `~/.gitlab-mcp-server/gitlab-mcp-server-config.json`. Holds one or more named servers and (optionally) a backend template map. If it contains servers, they are loaded into the client pool.
2. **Environment variables** — prefix `GITLAB_`, e.g. `GITLAB_TOKEN`, `GITLAB_HOST`. These feed the single-server **fallback** path used when the config file has no servers. `GITLAB_TOKEN` is **deprecated** and will be removed in v3.0.
3. **Command-line flags** — override the corresponding env vars for this process.

At tool-call time, a resolver picks a GitLab client:

- **Legacy (default).** Uses `server` from the tool arguments when present; otherwise falls back to the configured default.
- **Strict.** Set `GITLAB_MCP_STRICT_RESOLVER=1` to require every call to name a valid configured server and verify that the server's host matches its config (no fallbacks). Recommended if you use multiple servers and want tools to fail loudly on typos.

## The global config file

### Location

`~/.gitlab-mcp-server/gitlab-mcp-server-config.json` on every OS (hidden directory under `$HOME`). The directory is created the first time you run `config init` / `config add`.

> Note: this is **not** `~/.config/gitlab-mcp-server/…`. It sits directly under your home directory.

File permissions are set to `0600` and the directory to `0700`.

### Schema

```jsonc
{
  "version": "2.0",                 // "1.0" for plaintext-token configs, "2.0" once any server uses tokenRef
  "servers": {
    "work": {
      "name": "work",
      "host": "https://gitlab.company.com",
      "tokenRef": "keyring://work", // v2.0+: pointer into a secret backend
      "token": "",                  // legacy plaintext; present in v1.0 only
      "readOnly": false,
      "isDefault": true,
      "userId": 42,                 // populated on validate
      "username": "inky",           // populated on validate
      "lastValidated": "2026-04-20T09:12:44Z"
    },
    "personal": { "…": "…" }
  },
  "backends": {                     // optional: external-command secret templates
    "external": {
      "op":    "op read %s",
      "pass":  "pass show %s",
      "vault": "vault kv get -field=token %s"
    }
  }
}
```

Field notes:

- **`tokenRef`** takes precedence over `token` when both are set. A ref has the form `scheme://opaque` (e.g. `keyring://work`, `file:///path/to/tokens.age#work`, `op://Work/gitlab/token`).
- **`token`** is the legacy plaintext field. If the config was originally v1.0 and encryption was enabled, this value is AES-256-GCM ciphertext and the server decrypts it on load. New installs should always use `tokenRef`.
- **`readOnly`** on a single server restricts that server alone. The process-wide `--read-only` flag restricts every server.
- **`isDefault`** picks the server used when tools don't specify one (legacy resolver).
- The file version auto-bumps to `"2.0"` when a `tokenRef` first appears; the prior file is backed up to `gitlab-mcp-server-config.json.bak`.

### Commands that touch this file

```bash
gitlab-mcp-server config init                  # interactive setup (TUI)
gitlab-mcp-server config add <name> --host URL # add a server (token prompted or --token-ref)
gitlab-mcp-server config list                  # show servers (redacted tokens)
gitlab-mcp-server config default <name>        # change default
gitlab-mcp-server config validate              # contact /user on every server
gitlab-mcp-server config remove <name>         # delete a server
gitlab-mcp-server config backends              # inspect registered backends
gitlab-mcp-server config -i                    # terminal UI
```

Full CLI reference: [CLI_REFERENCE.md](CLI_REFERENCE.md).

## Secret backends

Tokens should not live as plaintext in the config. The server resolves them through pluggable backends, dispatched by URI scheme.

| Scheme | Backend | Ref format | Notes |
|---|---|---|---|
| `keyring://` | OS keyring (libsecret / Keychain / Credential Manager) | `keyring://<name>` | Default for interactive `config add`. |
| `file://` | AES-256-GCM encrypted file | `file:///absolute/path#entry` | Master key lives in the keyring. Useful when the keyring is headless/unavailable. Registered automatically when any server's `tokenRef` starts with `file://`. |
| `op://`, `pass://`, `vault://`, … | External command | `<scheme>://<opaque>` | Runs the template from `backends.external`. `%s` is substituted with the opaque portion. `exec.Command` is used (not `sh -c`), so shell metacharacters are harmless. |

### Registering an external command backend

Edit the config file directly, or use `config backends` subcommands, to add templates. Example config fragment:

```json
"backends": {
  "external": {
    "op":    "op read %s",
    "pass":  "pass show %s",
    "vault": "vault kv get -field=token %s"
  }
}
```

Then reference secrets normally:

```bash
gitlab-mcp-server config add work --host https://gitlab.company.com \
    --token-ref op://Work/gitlab/token
```

External commands run with a 10-second timeout and their stdout is trimmed and used as the secret.

### Choosing a backend for interactive adds

```bash
gitlab-mcp-server config add work --host https://gitlab.com --backend keyring   # default
gitlab-mcp-server config add work --host https://gitlab.com --backend file      # encrypted file
```

If you pass `--token-ref`, the backend flag is ignored (the ref already points into a specific backend).

### Legacy encryption of the `token` field

For configs that still use the plaintext `token` field, a `--encrypt` flag on `config add` enables AES-256-GCM encryption of that field with a master key stored in the keyring. This is the old v1.0 path; prefer `tokenRef` for all new setups.

## Environment variables

All env vars use the prefix `GITLAB_` and are read by viper. Every variable below has a corresponding flag on the `stdio` command.

| Variable | Flag | Default | Meaning |
|---|---|---|---|
| `GITLAB_TOKEN` | `--gitlab-token` | _(unset)_ | Fallback single-server token. **Deprecated**; remove by v3.0. |
| `GITLAB_HOST` | `--gitlab-host` | `https://gitlab.com` | Host for the fallback token. |
| `GITLAB_TOOLSETS` | `--toolsets` | `all` | Comma-separated toolset names. |
| `GITLAB_READ_ONLY` | `--read-only` | `false` | Disable every write tool. |
| `GITLAB_DYNAMIC_TOOLSETS` | `--dynamic-toolsets` | `false` | Start with discovery tools only; enable toolsets on demand. |
| `GITLAB_ENABLE_COMMAND_LOGGING` | `--enable-command-logging` | `false` | Log each JSON-RPC frame to stderr (tokens are redacted, but treat the log as sensitive). |
| `GITLAB_LOG_LEVEL` | `--log-level` | `info` | `debug`, `info`, `warn`, `error`. |
| `GITLAB_LOG_FILE` | `--log-file` | _(stderr)_ | Append logs to a file instead of stderr. |
| `GITLAB_USE_SECURE_MEMORY` | `--use-secure-memory` | `false` | Store token bytes in memguard-protected, no-swap memory. |
| `GITLAB_EXPORT_TRANSLATIONS` | `--export-translations` | `false` | Write translation keys and exit. |

Special case — **not** managed by viper:

| Variable | Default | Meaning |
|---|---|---|
| `GITLAB_MCP_STRICT_RESOLVER` | _(unset)_ | Set to `1` to switch the resolver to strict mode (every tool call must specify a known server; host verified per session). |

## Read-only mode

Two levels:

- Per server: set `readOnly: true` in the config (or pass `--read-only` to `config add`). Only write tools that target that server are blocked.
- Global: pass `--read-only` to `stdio`, or set `GITLAB_READ_ONLY=true`. No write tool is even registered.

## Example configurations

### Single personal account on GitLab.com

```bash
gitlab-mcp-server config add personal --host https://gitlab.com
```

Resulting config (abridged):

```json
{
  "version": "2.0",
  "servers": {
    "personal": {
      "name": "personal",
      "host": "https://gitlab.com",
      "tokenRef": "keyring://personal",
      "isDefault": true
    }
  }
}
```

### Work + personal + self-hosted read-only mirror

```bash
gitlab-mcp-server config add work     --host https://gitlab.company.com --token-ref op://Work/gitlab/token
gitlab-mcp-server config add personal --host https://gitlab.com
gitlab-mcp-server config add mirror   --host https://gitlab.example.internal --read-only
gitlab-mcp-server config default work
```

Tools can target a specific server by passing the `server` argument (`server: "personal"`). With `GITLAB_MCP_STRICT_RESOLVER=1` the `server` argument becomes mandatory.

## See also

- [CLI_REFERENCE.md](CLI_REFERENCE.md) — every command and flag.
- [TOKEN_MANAGEMENT.md](TOKEN_MANAGEMENT.md) — token lifecycle, rotation, expiry warnings.
- [MULTI_SERVER_SETUP.md](MULTI_SERVER_SETUP.md) — running multiple GitLab instances.
- [SELF_HOSTED.md](SELF_HOSTED.md) — private CAs and self-managed quirks.
