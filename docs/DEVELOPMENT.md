# Development guide

Internal documentation for contributors. For user-facing setup, start at [INSTALLATION.md](INSTALLATION.md) and [CONFIGURATION.md](CONFIGURATION.md).

## Repo layout

```
.
├── cmd/
│   ├── gitlab-mcp-server/   # main.go: root cobra command, startup wiring
│   ├── config/              # `config` subcommand (init/add/list/…) and TUI
│   ├── project/             # `project` subcommand (init/detect/status)
│   └── install/             # `install` subcommand (Claude/VS Code/Cursor)
├── pkg/
│   ├── config/              # global config file, backends, crypto, path utils
│   ├── gitlab/              # MCP tool implementations + client pool + resolvers
│   ├── toolsets/            # generic Toolset / ToolsetGroup (GitLab-agnostic)
│   ├── translations/        # i18n key map and dump/load
│   └── log/                 # iolog (MCP frame logger), redaction
├── internal/toolsnaps/      # snapshot testing of tool JSON schemas
├── tests/integration/       # end-to-end tests against a real GitLab instance
├── docs/                    # user + dev documentation (this tree)
├── scripts/install.js       # legacy Node.js installer (superseded by `install` cmd)
├── setup.sh / setup.bat     # convenience wrappers around make + install
└── Dockerfile               # builder (golang:1.25-alpine) → scratch-ish runtime
```

## High-level architecture

```
┌──────────────┐   JSON-RPC over stdio    ┌──────────────────────────────┐
│ IDE / client │ ───────────────────────▶ │  stdioServer (mark3labs/mcp) │
└──────────────┘                           │            │                 │
                                           │   MCPServer registers tools  │
                                           │            │                 │
                                           │   Toolset handlers call      │
                                           │   getClient(ctx) → *gl.Client │
                                           └──────────────┬───────────────┘
                                                          │
                                   ┌──────────────────────┼────────────────────┐
                                   │                      │                    │
                          ┌────────▼────────┐  ┌──────────▼─────────┐  ┌──────▼──────┐
                          │   ClientPool    │  │ ClientResolver /   │  │  TokenStore │
                          │  name → *Client │  │ StrictResolver     │  │  metadata   │
                          └────────┬────────┘  └────────────────────┘  └─────────────┘
                                   │
                          ┌────────▼────────┐
                          │ config.Manager  │
                          │  (global file,  │
                          │   BackendReg.)  │
                          └────────┬────────┘
                                   │
       ┌───────────────────────────┼───────────────────────────┐
       │                           │                           │
┌──────▼──────┐           ┌────────▼────────┐       ┌──────────▼──────────┐
│  Keyring    │           │ EncryptedFile   │       │  ExternalCmdBackend │
│  Backend    │           │  (AES-256-GCM)  │       │  op/pass/vault/…    │
└─────────────┘           └─────────────────┘       └─────────────────────┘
```

Key boundaries:

- **`cmd/*`** assembles everything; no business logic.
- **`pkg/config`** owns the on-disk file and secret backends. No GitLab knowledge.
- **`pkg/gitlab`** owns GitLab API calls and MCP tool definitions. Consumes a `GetClientFn(ctx) (*gl.Client, error)` from a resolver; it doesn't know about the config file directly.
- **`pkg/toolsets`** is a GitLab-agnostic generic group — Name, Description, read/write tool lists, enable/disable, read-only propagation. Reusable if another MCP server grows from this codebase.

## Startup sequence (read `cmd/gitlab-mcp-server/main.go`)

1. Parse flags/env via cobra + viper.
2. Initialize logger, signal handling, translations.
3. Build the backend registry (keyring always; `external` and `file://` registered if the config references them).
4. Load the global config (`config.NewManagerWithRegistry`).
5. Create `TokenStore` (optionally memguard-backed).
6. Create `ClientPool`, add one client per configured server via `AddServerFromConfig`.
7. Fall back to `GITLAB_TOKEN`/`GITLAB_HOST` if the config has no servers (deprecated path).
8. Validate the default token on startup (`/user` call) — warnings only, not fatal.
9. Build the resolver: `ClientResolver` (legacy, default) or `StrictResolver` (env `GITLAB_MCP_STRICT_RESOLVER=1`).
10. `gitlab.InitToolsets(...)` creates all `Toolset`s and adds tools; tool handlers capture the resolver's `GetClientFn`.
11. If `--dynamic-toolsets`, register only the two discovery tools; otherwise register every enabled tool.
12. `server.NewStdioServer(mcpServer).Listen(ctx, stdin, stdout)`.

## Adding a new tool

Each tool is a function returning `(mcp.Tool, server.ToolHandlerFunc)`. The pattern, in `pkg/gitlab/<area>.go`:

```go
func GetFoo(getClient GetClientFn, t map[string]string) (mcp.Tool, server.ToolHandlerFunc) {
    tool := mcp.NewTool("getFoo",
        mcp.WithDescription(translations.Translate(t, "TOOL_GET_FOO_DESCRIPTION")),
        mcp.WithString("fooId", mcp.Required(), mcp.Description("…")),
        // … more params
    )
    handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        fooID, err := requiredParam[string](&req, "fooId")
        if err != nil { return mcp.NewToolResultError(err.Error()), nil }
        client, err := getClient(ctx)
        if err != nil { return nil, err }
        // … call GitLab
    }
    return tool, handler
}
```

Then register it in `pkg/gitlab/toolsets.go` under the appropriate toolset's `AddReadTools` / `AddWriteTools`. Write-classified tools are filtered out when `--read-only` is set globally or per-server.

After changing a tool's schema, regenerate the snapshot:

```bash
make update-toolsnaps
```

Snapshot files live in `pkg/gitlab/__toolsnaps__/` and are committed. `make test-toolsnaps` fails if any tool's schema drifts without a snapshot update — this is the guard against accidental LLM-visible API changes.

## Adding a new secret backend

Implement `config.SecretBackend` (`Resolve`/`Store`/`Delete`/`Scheme`), register an instance with the `BackendRegistry` in `cmd/gitlab-mcp-server/main.go`, and update the docs. `ExternalCmdBackend` is a multi-scheme implementation — see `RegisterAll` for the pattern when one backend needs to handle several URI schemes.

## Adding a new CLI subcommand

- Create `cmd/<name>/` with a package exporting `Cmd *cobra.Command`.
- In `cmd/gitlab-mcp-server/main.go`, import it and add to `rootCmd.AddCommand(...)`.
- Bind any flags to viper keys with a `GITLAB_` env-var prefix if you want env-based configuration.

## Conventions

- `gofmt` + `goimports` are mandatory.
- `golangci-lint run` must pass; `.golangci.yml` captures the rules.
- Errors are wrapped with `fmt.Errorf("context: %w", err)`.
- Interfaces are defined where they are consumed (small, 1–3 methods).
- No `panic` outside `main` init paths.
- Tests follow the table-driven pattern; `go test -race ./...` must stay green.
- Constructor injection: functions take a logger, a `GetClientFn`, etc., rather than reaching for globals.

## Running the server from source

```bash
go run ./cmd/gitlab-mcp-server stdio
# or build once and iterate:
make build && ./bin/gitlab-mcp-server stdio
```

Feed it JSON-RPC on stdin manually for quick sanity checks, or point your IDE at the `bin/` binary (`gitlab-mcp-server install claude --binary-path "$(pwd)/bin/gitlab-mcp-server"`).

## Tests

```bash
make test                    # all tests
go test -race ./pkg/...      # race detector
go test -cover ./pkg/...     # coverage
make test-toolsnaps          # just the schema snapshot checks
make update-toolsnaps        # regenerate snapshots (after intentional schema changes)
```

Integration tests in `tests/integration/` hit a real GitLab instance via `GITLAB_TEST_TOKEN` / `GITLAB_TEST_PROJECT_ID` env vars; they're skipped by default in CI.

See [../TESTING.md](../TESTING.md) for more on test organization and coverage targets.

## Release flow

- `.goreleaser.yml` builds binaries for Linux/macOS/Windows (amd64/arm64/armv7), `.deb`/`.rpm`/Arch packages, and archives.
- `.github/workflows/release.yml` fires on tag push, runs goreleaser, attaches artifacts to the GitHub Release.
- Version is injected into the binary via `-X main.version/commit/date` ldflags.
- Changelog lives in [CHANGELOG.md](../CHANGELOG.md). Keep it in sync with tags; v2.0 was a major consolidation, v2.1 introduced the backend registry and `tokenRef` model.

## Known debts / watch-outs

- Two files share the name `gitlab-mcp-server-config.json` — the global config (in `~/.gitlab-mcp-server/`) and the i18n file (next to the binary). They live in different directories but the name collision surprises newcomers.
- The legacy Node.js installer (`scripts/install.js`) is still referenced from the Makefile and `setup.sh`. The Go `install` command supersedes it for Claude/VS Code/Cursor; keep the JS one around for any flow it still handles better (e.g. custom prompts) until replaced.
- `GITLAB_TOKEN`/`GITLAB_HOST` env-var fallback path is deprecated. Don't extend it; migrate callers to the config file.
- `ClientResolver` (legacy) will be removed once `StrictResolver` is default. Grep for `GITLAB_MCP_STRICT_RESOLVER` to see the toggle.

## See also

- [../CONTRIBUTING.md](../CONTRIBUTING.md) — contribution workflow, branch model, review conventions.
- [../TESTING.md](../TESTING.md) — testing philosophy and examples.
- [../ROADMAP.md](../ROADMAP.md) — planned work.
- [CLI_REFERENCE.md](CLI_REFERENCE.md) — user-facing CLI surface.
- [CONFIGURATION.md](CONFIGURATION.md) — config schema and precedence.
