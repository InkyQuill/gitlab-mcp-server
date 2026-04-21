# GitLab MCP Server

[![Tests](https://github.com/InkyQuill/gitlab-mcp-server/actions/workflows/test.yml/badge.svg)](https://github.com/InkyQuill/gitlab-mcp-server/actions/workflows/test.yml)
[![Lint](https://github.com/InkyQuill/gitlab-mcp-server/actions/workflows/lint.yml/badge.svg)](https://github.com/InkyQuill/gitlab-mcp-server/actions/workflows/lint.yml)
[![Security](https://github.com/InkyQuill/gitlab-mcp-server/actions/workflows/security.yml/badge.svg)](https://github.com/InkyQuill/gitlab-mcp-server/actions/workflows/security.yml)
[![codecov](https://codecov.io/gh/InkyQuill/gitlab-mcp-server/branch/main/graph/badge.svg)](https://codecov.io/gh/InkyQuill/gitlab-mcp-server)
[![Go Report Card](https://goreportcard.com/badge/github.com/InkyQuill/gitlab-mcp-server)](https://goreportcard.com/report/github.com/InkyQuill/gitlab-mcp-server)
[![Go Version](https://img.shields.io/badge/go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A [Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction) server that exposes GitLab as a tool surface for AI coding assistants (Claude Desktop, Claude Code, Cursor, VS Code, and any other MCP-capable client).

It speaks JSON-RPC over stdio, authenticates against GitLab.com or self-managed GitLab, and offers ~40 action-based tools covering projects, issues, merge requests, pipelines, search, users, security scans, and more.

> Fork of [LuisCusihuaman/gitlab-mcp-server](https://github.com/LuisCusihuaman/gitlab-mcp-server), substantially rewritten. See [CHANGELOG.md](CHANGELOG.md) for history and v2.0 migration notes.

## Highlights

- **Action-based tools** — related operations consolidated under one tool via an `action` parameter, reducing context-window cost for LLMs.
- **Multi-server** — register several GitLab instances (work, personal, self-hosted) side by side; tools pick the right one per call.
- **Pluggable secret backends** — tokens live in the OS keyring, an encrypted file, or an external command (e.g. `op`, `pass`, `vault`). Plaintext tokens in config are legacy only.
- **Project auto-detection** — reads `.git/config` to infer the GitLab project for the current working directory.
- **Dynamic toolsets** — optional lazy loading so clients start with a discovery tool and opt in to categories on demand.
- **Read-only mode** — at the server or per-server level, for safe exploratory use.
- **Secure memory mode** — optional memguard-backed encrypted memory for tokens.
- **Thorough tests** — ~89% coverage, race-detected, schema-snapshotted.

## Quick Start

```bash
# 1. Install (requires Go 1.25+)
go install github.com/InkyQuill/gitlab-mcp-server@latest

# 2. Register with your IDE (auto-detects Claude Desktop / VS Code / Cursor)
gitlab-mcp-server install claude     # or: vscode | cursor | all

# 3. Add a GitLab server and token (token is prompted, stored in OS keyring)
gitlab-mcp-server config add work --host https://gitlab.com

# 4. In any project directory, pin the current GitLab project
cd your-project
gitlab-mcp-server project init

# 5. Restart the IDE
```

Detailed walkthrough: [docs/INSTALLATION.md](docs/INSTALLATION.md).

## CLI overview

| Command | Purpose |
|---|---|
| `stdio` | Start the MCP server (invoked by the IDE, rarely run manually). |
| `install [claude\|vscode\|cursor\|all]` | Write an MCP server entry into the IDE's config file (with backup). Subcommands: `status`, `path`, `uninstall`. |
| `config init` | Interactive creation of the global config. |
| `config add <name> --host <url> [--token-ref <ref>]` | Add a server; token is prompted (no echo) unless a backend ref is supplied. |
| `config list`, `config default <name>`, `config remove <name>`, `config validate` | Manage servers. |
| `config backends` | Manage pluggable secret backends. |
| `config -i` | Terminal UI for configuration. |
| `project init\|detect\|status` | Create / inspect the project's `.gmcprc`. |
| `version` | Show version, commit, build date. |

Full reference: [docs/CLI_REFERENCE.md](docs/CLI_REFERENCE.md).

## Configuration

The server reads configuration (in this order of precedence):

1. Global config file — `~/.gitlab-mcp-server/gitlab-mcp-server-config.json`
2. Environment variables (prefix `GITLAB_`)
3. Command-line flags

### Environment variables

| Variable | Maps to flag | Purpose |
|---|---|---|
| `GITLAB_TOKEN` | `--gitlab-token` | Token for the fallback (single-server) path. **Deprecated** — prefer the global config. |
| `GITLAB_HOST` | `--gitlab-host` | GitLab host for the fallback path. |
| `GITLAB_TOOLSETS` | `--toolsets` | Comma-separated toolset names, or `all`. |
| `GITLAB_READ_ONLY` | `--read-only` | `true` to disable all write tools globally. |
| `GITLAB_DYNAMIC_TOOLSETS` | `--dynamic-toolsets` | `true` for lazy toolset loading. |
| `GITLAB_ENABLE_COMMAND_LOGGING` | `--enable-command-logging` | Log every JSON-RPC frame (redacted; use with care). |
| `GITLAB_LOG_LEVEL` | `--log-level` | `debug` / `info` / `warn` / `error`. |
| `GITLAB_LOG_FILE` | `--log-file` | Write logs to this path instead of stderr. |
| `GITLAB_USE_SECURE_MEMORY` | `--use-secure-memory` | Store tokens in memguard-protected memory. |
| `GITLAB_EXPORT_TRANSLATIONS` | `--export-translations` | Dump translation keys and exit. |
| `GITLAB_MCP_STRICT_RESOLVER` | _(no flag)_ | `1` to disable fallback resolution; every request must specify a known server. |

See [docs/CONFIGURATION.md](docs/CONFIGURATION.md) for the full schema, secret backend setup, and precedence rules.

## Toolsets

Ten toolsets, ~40 tools total. Pass a subset via `--toolsets` (default: `all`).

| Toolset | Tools |
|---|---|
| `projects` | `getProject`, `listProjects`, `getProjectFile`, `listProjectFiles`, `getProjectBranches`, `getProjectCommits` |
| `issues` | `getIssue`, `listIssues`, `getIssueLabels`, `createIssue`, `updateIssue`, `issueComment` (list/create/update), `milestone` (get/create/update), `listMilestones` |
| `merge_requests` | `getMergeRequest`, `listMergeRequests`, `createMergeRequest`, `updateMergeRequest`, `mergeRequestComment` (list/create/update) |
| `pipeline_jobs` | `pipelineJob` (list/get/trace), `pipeline` (cancel/retry), `retryPipelineJob`, `playPipelineJob` |
| `search` | `search` (unified; `resourceType` = projects/issues/merge_requests/blobs/commits/milestones/snippet_titles/snippet_blobs/wiki_blobs/notes) |
| `users` | `getCurrentUser`, `getUser`, `getUserStatus`, `listUsers`, `listProjectUsers`, `manageUserState` (block/unblock/ban/unban/activate/deactivate/approve) |
| `tags` | `tag` (get/create/delete/getCommit), `listRepositoryTags` |
| `security` | `getProjectSAST`, `getProjectDAST`, `getProjectDependencyScanning`, `getProjectContainerScanning`, `getProjectSecretDetection`, `getProjectLicenseCompliance` |
| `token_management` | `listTokens`, `validateToken`, `updateToken`, `removeToken`, `getNotifications`, `clearNotifications` |
| `project_config` | `getCurrentProject`, `setCurrentProject` |

Complete tool documentation with parameters and examples: [docs/TOOLS.md](docs/TOOLS.md).

## Documentation

### Getting started
- [Installation](docs/INSTALLATION.md) — prerequisites, install methods, IDE registration, verification.
- [Configuration](docs/CONFIGURATION.md) — global config, secret backends, env vars, precedence.
- [Editor setup](docs/EDITOR_SETUP.md) — manual IDE integration if `install` can't cover your case.
- [CLI reference](docs/CLI_REFERENCE.md) — every command, every flag.

### Core features
- [Tools reference](docs/TOOLS.md) — all tools with parameters and examples.
- [Token management](docs/TOKEN_MANAGEMENT.md) — lifecycle, expiry warnings, rotation.
- [Multi-server setup](docs/MULTI_SERVER_SETUP.md) — work/personal/self-hosted side by side.
- [Project configuration](docs/PROJECT_CONFIG.md) — `.gmcprc` format and auto-detection.

### Advanced
- [Dynamic tool discovery](docs/DYNAMIC_TOOLS.md) — lazy toolset loading.
- [Self-hosted GitLab](docs/SELF_HOSTED.md) — CA certs, hostnames, gotchas.
- [Internationalization](docs/I18N.md) — translating tool descriptions.
- [GitHub detection](docs/GITHUB_DETECTION.md) — why GitHub URLs are rejected.

### For developers
- [Development guide](docs/DEVELOPMENT.md) — architecture, package layout, how to add tools.
- [Contributing](CONTRIBUTING.md) — workflow, conventions.
- [Testing](TESTING.md) — how tests are organized, snapshot discipline.
- [Roadmap](ROADMAP.md) — upcoming work.
- [Changelog](CHANGELOG.md) — release history and breaking changes.

## Building from source

```bash
git clone https://github.com/InkyQuill/gitlab-mcp-server.git
cd gitlab-mcp-server
make build        # produces bin/gitlab-mcp-server
make test         # all tests
make docker-build # gitlab-mcp-server:latest image
```

Requires Go 1.25 or newer. `make help` lists every target.

## Troubleshooting

**`gitlab-mcp-server: command not found`** — ensure `$GOPATH/bin` (typically `$HOME/go/bin`) is on `PATH`.

**Token validation fails** — check that the token has the `api` and `read_repository` scopes and has not expired. Use `gitlab-mcp-server config validate` to re-verify all configured servers.

**Tool not appearing in IDE** — restart the IDE after `install`; confirm the path in the IDE's MCP config points at the actual binary (`which gitlab-mcp-server`).

More in [docs/INSTALLATION.md](docs/INSTALLATION.md#troubleshooting).

## License

MIT — see [LICENSE](LICENSE).
