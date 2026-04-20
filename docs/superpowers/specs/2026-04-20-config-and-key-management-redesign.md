# Config & Key Management Redesign

**Date:** 2026-04-20
**Target release:** v2.1 (deprecation) → v3.0 (clean break)
**Status:** Design approved, ready for planning

## Problem

Four intertwined pain points in the current config/token handling:

1. **Tokens sprawl across four storage locations** — global config JSON (plaintext by default), IDE MCP config env vars (always plaintext), in-memory `TokenStore`, optional `memguard` enclaves. The `GITLAB_TOKEN` env-var startup path bypasses the whole "secure config" system.
2. **Two conflicting multi-instance mental models exist simultaneously.** Per-instance IDE entries vs. single entry with a global multi-server config. Docs describe both. The word "server" is overloaded (MCP server vs GitLab instance). `.gmcprc` carries both `tokenName` and `server` fields that mean the same thing.
3. **No enforcement of "this project → this instance only".** `ClientResolver.Resolve` has a 4-step fallback cascade that will happily use the wrong instance if the intended one is missing or misnamed. Nothing checks that a `.gmcprc` is pointed at the matching host.
4. **LLM can read and write secrets via MCP tools.** `addToken` and `updateToken` accept raw token strings as parameters. Any prompt injection can exfiltrate or replace them.

## Goals

- One source of truth for GitLab instance config.
- Tokens never stored on disk as plaintext.
- Tokens never visible to the LLM.
- Project directory strictly binds to exactly one configured instance.
- Failures are loud; no silent fallback paths that mask misconfiguration.

## Non-goals

- Team/shared config sync. Per-user setup only.
- OAuth / browser-based token issuance. Users still create a PAT in GitLab and register it.
- Token rotation automation. Out of scope; revalidation stays manual via `validateToken`.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ IDE MCP entry: `gitlab-mcp-server stdio` — no secrets       │
└──────────────────────┬──────────────────────────────────────┘
                       │ reads on startup
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ Global config  ~/.config/gitlab-mcp-server/config.json      │
│   version: "2.0"                                            │
│   servers:                                                  │
│     work:     { host, username, tokenRef: "keyring://work" }│
│     personal: { host, username, tokenRef: "op://…" }        │
│   backends:                                                 │
│     external: { op: "op read %s", pass: "pass show %s" }    │
└──────────────────────┬──────────────────────────────────────┘
                       │ resolved once per server at startup
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ SecretBackend registry (scheme → backend)                   │
│   keyring:// → KeyringBackend       (default)               │
│   file://    → EncryptedFileBackend (headless/docker)       │
│   op://, pass://, age://, gopass:// → ExternalCmdBackend    │
└──────────────────────┬──────────────────────────────────────┘
                       │ plaintext secret in memory only
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ Runtime: ClientPool + StrictResolver                        │
│   .gmcprc requires `server` (single field)                  │
│   On first request per session: verify API host matches     │
│   No fallback. No default. No GITLAB_TOKEN env.             │
└─────────────────────────────────────────────────────────────┘
```

## Component design

### 1. `SecretBackend` interface

Defined in `pkg/config`, small and focused:

```go
type SecretBackend interface {
    Resolve(ctx context.Context, ref string) (string, error)
    Store(ctx context.Context, name, secret string) (ref string, err error)
    Delete(ctx context.Context, ref string) error
    Scheme() string
}
```

**Ref format:** `<scheme>://<opaque>` — e.g. `keyring://work`, `file://~/.config/gitlab-mcp-server/secrets.enc#work`, `op://Work/gitlab/token`.

**BackendRegistry** dispatches by scheme. Unknown scheme returns a loud error naming the compiled-in backends.

`Store` and `Delete` are CLI/TUI-only. MCP tool code imports only `Resolve`.

### 2. Three concrete backends

**KeyringBackend** (default). Wraps `zalando/go-keyring` (already a dependency). Service `gitlab-mcp-server`, account = server name. Ref `keyring://<name>`.

**EncryptedFileBackend**. Scheme `file`. Reuses existing `CryptoManager` (AES-256-GCM). Master key sourced from:
- Keyring (preferred), or
- `GITLAB_MCP_FILE_PASSPHRASE` env var (for headless/Docker/CI).

Ref `file://<path>#<entryName>`. File format: JSON map of `entryName → base64(ciphertext)`. Mode 0600.

**ExternalCmdBackend**. Schemes from a configurable allow-list (`op`, `pass`, `age`, `gopass`). Command templates live in global config under `backends.external.<scheme>`, e.g. `op: "op read %s"`. Resolution shells out via `exec.Command` (not `sh -c`) with `%s` substituted with the ref suffix. Command templates are not user-data; they're only writable by the config owner (file mode 0600), so the LLM cannot influence which command runs. **`Store` is not implemented** for this backend — the user is expected to populate their external tool directly; the CLI only persists the ref.

### 3. Global config schema (v2)

```go
type Config struct {
    Version  string                   `json:"version"` // "2.0"
    Servers  map[string]*ServerConfig `json:"servers"`
    Backends *BackendsConfig          `json:"backends,omitempty"`
}

type ServerConfig struct {
    Name          string `json:"name"`
    Host          string `json:"host"`
    TokenRef      string `json:"tokenRef"`           // replaces Token
    ReadOnly      bool   `json:"readOnly,omitempty"`
    UserID        int    `json:"userId,omitempty"`
    Username      string `json:"username,omitempty"`
    LastValidated string `json:"lastValidated,omitempty"`
    // Removed: Token, IsDefault
}

type BackendsConfig struct {
    External map[string]string `json:"external,omitempty"` // scheme → command template
}
```

**Removed fields:** `ServerConfig.Token`, `ServerConfig.IsDefault`. The default-server concept is gone entirely — resolution is purely `.gmcprc`-driven.

### 4. `.gmcprc` schema (v3)

```go
type ProjectConfig struct {
    ProjectID   string    `json:"projectId"`
    Server      string    `json:"server"`       // required
    LastUpdated time.Time `json:"lastUpdated"`
    // Removed: GitLabHost, TokenName
}
```

`GitLabHost` is gone — it's derived from the resolved server's config. `TokenName` is gone — it was a duplicate of `Server`.

### 5. `StrictResolver`

Replaces `ClientResolver`. Resolution order, no fallbacks:

1. Find `.gmcprc` walking up from cwd.
2. If missing → error: `no project configured. Run 'gitlab-mcp-server project init' or pass --server explicitly.`
3. Read `server`. If empty → error: `.gmcprc missing required 'server' field. Re-run 'gitlab-mcp-server project init'.`
4. Look up server in ClientPool. If missing → error listing configured servers.
5. **Host verification (first call per session, cached):**
   - Call `/api/v4/version`.
   - Compare response URL host to `server.host`.
   - Mismatch → error naming both hosts; refuse to use the client.
   - Pass → cache success for this resolver instance.
6. Return client.

No `--default` flag, no `GITLAB_TOKEN` env, no host-based matching, no "first available" fallback.

### 6. MCP tool surface

**Removed:**
- `addToken`
- `updateToken`

**Kept (no secrets in / out):**
- `listTokens` — name, host, username, validation status, expiry. No token value.
- `validateToken` — triggers revalidation by name.
- `removeToken` — removes from runtime store and backend; does not return the secret.
- `getNotifications`, `clearNotifications`.

**Project tools unchanged in signature**, but `setCurrentProject` now:
- Requires the `server` parameter.
- Validates that `server` exists in config before writing `.gmcprc`.
- Writes no `tokenName`, no `gitlabHost`.

A registration test asserts that `addToken` and `updateToken` are **not** registered in any toolset group (guard against accidental re-introduction).

### 7. CLI / TUI

**`config add` changes:**
- `--token <raw>` flag removed. Attempting to use it exits with: `use --token-ref or interactive mode; --token removed in v3.0 to prevent shell history leaks.`
- Interactive path: prompts on `/dev/tty` with no-echo; asks for backend choice (`keyring` default).
- Non-interactive path: `--token-ref <ref>` points at an already-populated backend entry. CLI validates ref resolves and token works against GitLab before saving.

**New `config backends` command:** lists compiled-in backends, probes availability (keyring reachable?, external commands on `$PATH`?), shows the default.

**TUI:** "add server" screen adopts the same two paths. Secret input uses `textinput.EchoMode = EchoPassword`.

### 8. Installer

`install.go:createInstallConfig` stops emitting the `env` block by default. Resulting IDE entry:

```json
{"command": "/usr/local/bin/gitlab-mcp-server", "args": ["stdio"]}
```

If `--config-path` is passed, `GITLAB_CONFIG_PATH` stays (it's a path, not a secret). All other env vars dropped from generated entries.

### 9. Config file v1 → v2 migration on startup

(Note: this is the config file's own schema version, not the release version. Config v1 is what today's v2.x releases write; config v2 is the new shape. Config v2 first ships in release v2.1 and is the only shape in v3.0.)

Triggered when `config.Version == "1.0"` is seen on load:

1. Print migration banner with one-line summary to stderr.
2. For each server with a non-empty `Token`:
   - If token looks encrypted (was `--encrypt`-added) and keyring has the master key → decrypt.
   - Else treat as plaintext.
3. For each recovered plaintext token: store via `KeyringBackend.Store(name, secret)` → get back a `tokenRef`.
4. Rewrite config with `TokenRef` set, `Token` removed, `Version = "2.0"`, `IsDefault` dropped.
5. Save atomically.

**Failure mode:** if any backend op fails, write new config as `config.json.migrated` **next to** the old one, leave the old one untouched, print clear instructions. Never corrupt user state.

### 10. v2.1 deprecation shims (one release only)

- `GITLAB_TOKEN` env path: still works, emits `DEPRECATION: GITLAB_TOKEN env var will be removed in v3.0. Run 'gitlab-mcp-server config add' and 'gitlab-mcp-server project init' to migrate.` on every startup.
- `config add --token <raw>`: still accepted, prints `DEPRECATION: --token flag exposes secrets via shell history and process listing. Use interactive mode or --token-ref.`
- `addToken` / `updateToken` MCP tools: still registered, but each invocation returns a deprecation warning field in its response and logs the warning.
- `.gmcprc` with `tokenName`: auto-promoted to `server` on read, one-time log warning per file per session.
- `.gmcprc` with `gitlabHost` only: deprecated matcher runs once to suggest a server name, then errors with migration instructions.

All shims and their tests get removed in v3.0.

## Testing

- **`FakeSecretBackend`** — in-memory map, injected into config tests. Real keyring is never touched in unit tests.
- **BackendRegistry** — scheme dispatch, unknown scheme error shape.
- **StrictResolver** — table-driven: missing `.gmcprc`, empty `server`, unknown server, host mismatch, host match (cached second call doesn't re-hit API).
- **Migration** — config v1 plaintext → v2; config v1 encrypted (keyring present) → v2; keyring-unavailable failure path writes `.migrated` sibling and leaves original intact.
- **`config add` CLI** — `--token` rejected with clear message; `--token-ref` path validates against mock GitLab; interactive path reads from a test TTY.
- **Tool registration** — assertion that `addToken`/`updateToken` are absent from toolset groups in v3 builds.
- **External-cmd injection** — feed a ref with shell metacharacters; verify `exec.Command` is called with args (not `sh -c`) and metacharacters are passed literally.

## Rollout

**v2.1** (one release before v3.0):
- Ship SecretBackend + three backends in parallel with existing storage.
- Config migration writes v2 shape on save but keeps reading v1.
- Add all deprecation warnings listed in §10.
- `config add` interactive prompt and `--token-ref` ship here.
- New `StrictResolver` is gated behind `GITLAB_MCP_STRICT_RESOLVER=1` opt-in.
- Release notes spell out the v3.0 breaking changes and the one-command migration path.

**v3.0** (clean break):
- Remove `GITLAB_TOKEN` / `GITLAB_HOST` startup path.
- Remove `ServerConfig.Token`, `ServerConfig.IsDefault`, `ProjectConfig.TokenName`, `ProjectConfig.GitLabHost`.
- Remove `addToken`, `updateToken` MCP tools.
- Remove `config add --token <raw>` flag.
- `StrictResolver` is the only resolver.
- `ClientResolver` (legacy) deleted.

## Risks and mitigations

| Risk | Mitigation |
|---|---|
| Users have existing configs with plaintext tokens on disk | Automatic v1 → v2 migration on first v2.1 run; backup written to `config.json.bak` before rewrite. |
| Keyring unavailable in some Linux/WSL/Docker environments | `EncryptedFileBackend` with env-var passphrase is the documented headless path. `config backends` command diagnoses. |
| External-cmd template could be abused if global config is writable by an attacker | Enforce mode 0600 on global config; refuse to load if permissions are broader. |
| Host verification adds a round-trip on first tool call per session | Cached after first success; failure is what we actually want — it means the project is pointed at the wrong server. |
| Breaking change hits every existing user | v2.1 deprecation window with loud warnings; documented one-command migration (`gitlab-mcp-server config migrate`). |

## Open questions (tracked as plan inputs, not design blockers)

- Exact wording of deprecation warnings — defer to writing-plans.
- Should `config migrate` be an explicit subcommand on top of auto-migration? Probably yes, for users who want to preview the change without starting the server. Defer to plan.
- Do we want a `config doctor` command that runs all backend probes + a dry resolve for the current `.gmcprc`? Deferrable.

## What this fixes

| Pain | Fix |
|---|---|
| Bad key storage | `Token` field gone from disk. Secrets live in keyring / external tool / encrypted file. Plaintext-JSON path removed. `GITLAB_TOKEN` env removed. |
| Clunky multi-instance | Single MCP entry, single source of truth (global config), single resolver path, single `server` field in `.gmcprc`. |
| No "this instance only" | `.gmcprc` `server` field is required; no fallback; host verified against API on first use. |
| LLM sees keys | `addToken`/`updateToken` stripped from MCP surface in v3.0. CLI/TUI only, TTY-prompted. LLM only sees ref strings and metadata. |
