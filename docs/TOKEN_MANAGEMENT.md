# Token management

How `gitlab-mcp-server` handles tokens: validation on startup, expiry warnings, the `token_management` MCP tools, and rotation flows.

## Where tokens live

- **Primary:** the global config (`~/.gitlab-mcp-server/gitlab-mcp-server-config.json`), with the actual secret resolved through a backend (`keyring://`, `file://`, external command). See [CONFIGURATION.md](CONFIGURATION.md#secret-backends).
- **Fallback:** the `GITLAB_TOKEN` env var (deprecated, single-server only).
- **Ephemeral:** runtime-only updates made via `updateToken` (MCP tool) — live for the current process only.

## Startup validation

On every start, the default client makes a `GET /user` call:

```
INFO Token validated successfully for user inky (ID: 42) on server 'work'
WARN Token will expire in 14 days. Please create a new token and update it.
```

If validation fails, the server logs a warning and continues running — individual tool calls still fail with a 401, but the process stays up so you can fix it without an IDE restart.

Re-validate every server on demand:

```bash
gitlab-mcp-server config validate
```

## Expiry warnings

When the GitLab API returns an expiry timestamp for the token, the server surfaces a warning if fewer than 30 days remain. GitLab's maximum PAT lifetime is 1 year. No warning is shown when GitLab reports no expiry.

## Runtime MCP tools (`token_management` toolset)

The toolset is enabled by default. It exposes these tools:

| Tool | Mode | Purpose |
|---|---|---|
| `listTokens` | read | List tokens tracked in the runtime store, with user/host/expiry metadata. |
| `validateToken` | read | Re-validate a named token (or all of them) against `/user`. |
| `getNotifications` | read | Fetch accumulated notifications (validation warnings, expiry warnings, 401s). |
| `updateToken` | write | Replace the token for an existing server in the runtime store. Persists in memory only. |
| `removeToken` | write | Drop a token from the runtime store. Doesn't touch the config file. |
| `clearNotifications` | write | Clear the notification buffer. |

> `addToken` exists in the code as a **deprecated** path (accepting tokens from an LLM is unsafe). It is not registered in the MCP toolset in current builds; use the CLI (`gitlab-mcp-server config add`) instead.

### Example: `listTokens`

```json
{
  "tokens": [
    {
      "name": "work",
      "gitlabHost": "https://gitlab.company.com",
      "userId": 42,
      "username": "inky",
      "createdAt": "2026-01-10T10:30:00Z",
      "lastValidated": "2026-04-20T09:12:44Z",
      "expiresAt": "2027-01-10T00:00:00Z",
      "isExpired": false,
      "daysUntilExpiry": 265
    }
  ],
  "count": 1
}
```

### Example: `updateToken`

Rotate the token in-place for the current process:

```json
{ "name": "updateToken", "arguments": { "name": "work", "token": "glpat-new-token-value" } }
```

For persistent rotation, see below.

## Rotation

Preferred, persistent flow:

```bash
# 1. Generate a new token in GitLab (with the same scopes)
# 2. Update the secret in its backend
#    - keyring-backed:
gitlab-mcp-server config remove work
gitlab-mcp-server config add work --host https://gitlab.company.com
#    - backend-ref-backed (e.g. 1Password):
op item edit "gitlab" password=glpat-new-token-value   # or whatever your workflow is
gitlab-mcp-server config validate
# 3. Restart any running MCP client to pick up the new token
```

Hot rotation in a long-running session:

```
call updateToken { name: "work", token: "glpat-new-token-value" }
```

The new value is used for subsequent calls until the process exits; the on-disk config is unchanged.

## 401 handling

When a tool call returns `401 Unauthorized`, the server rewrites the error with a hint:

```
Authentication failed (401). Your GitLab token may be expired. Use listTokens / validateToken to inspect, and updateToken (or the config CLI) to rotate.
```

A notification is also recorded so you can retrieve it later with `getNotifications`.

## Security notes

- Prefer `--token-ref` over plaintext `--token` when adding servers; `--token` exposes the secret in shell history.
- Scope tokens minimally: `read_api` + `read_repository` for read-only, `api` for full control.
- When running under `--use-secure-memory`, tokens live in memguard-protected memory and are never swapped to disk.
- Config file permissions are `0600`; the containing directory is `0700`. Don't override these.

## See also

- [CONFIGURATION.md](CONFIGURATION.md)
- [CLI_REFERENCE.md](CLI_REFERENCE.md)
- [MULTI_SERVER_SETUP.md](MULTI_SERVER_SETUP.md)
