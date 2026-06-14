# Glab Parity Inventory

**Date:** 2026-06-14
**Status:** Phase 0 inventory
**Purpose:** Compact mapping from current `glab` command families to current and planned MCP capabilities.

## Reading Rules

- Prefer capability parity over command-name parity.
- Treat `glab mcp serve` as a coverage reference, not an implementation model.
- Keep list tools compact and get/view tools detailed.
- Keep secret-bearing and local-terminal workflows out of default MCP output.
- Use `server` routing and read-only guards for every accepted GitLab API domain.

## Status Taxonomy

| Status | Meaning |
|---|---|
| existing | Current MCP tools cover the practical `glab` family well enough for normal agent use. |
| harden | Current MCP tools partially cover the family, but need safer output, missing actions, or consistency work. |
| add | No adequate modeled MCP tool exists and the domain is in roadmap scope. |
| defer | Potentially useful later, but specialized, high-risk, or not agent-critical now. |
| non-goal | CLI-local, interactive, secret-printing, or terminal-only behavior that should not be modeled as MCP tools. |

## Current MCP Toolset Summary

| MCP toolset | Current coverage | Notes |
|---|---|---|
| projects | Projects, repository files, branches, commits. | Harden project lifecycle, members, settings, archive/fork later. |
| issues | Issues, issue labels, issue notes, milestones. | Harden close/reopen/subscribe/todos and compact output. |
| merge_requests | MR get/list/create/update and notes. | Add approvals, merge/rebase, diffs, related issues, todos. |
| pipeline_jobs | Pipeline job list/get/trace and cancel/retry/play actions. | Add CI lint/run/trigger/artifacts/schedules later. |
| search | Unified GitLab search across supported scopes. | Normalize output and pagination. |
| users | Current user, users, status, project users, admin state actions. | Harden admin isolation and add events later. |
| tags | Repository tag list/get/create/delete/get-commit. | Releases remain missing. |
| security | Security scan result readers. | Split large payloads where needed. |
| project_config | `.gmcprc` read/write and project detection helpers. | Harden remote matching in Phase 1. |
| token_management | Runtime token validation/update/remove/notifications. | Persistent auth belongs to CLI config commands. |

## Command-Family Matrix

| glab family | Status | Current MCP coverage | Desired owner | Next scoped action |
|---|---|---|---|---|
| `alias` | `non-goal` | None. | none | CLI command aliases are local shell convenience, not GitLab resources. |
| `api` | `add` | None modeled; callers use specific tools only. | `api` | Add scoped generic API passthrough with method/path/server, JSON limits, and path guards. |
| `attestation` | `defer` | None. | future specialized toolset | Defer experimental provenance verification until artifact workflows are in scope. |
| `auth` | `harden` | Runtime token management exists; no interactive login MCP flow. | `token_management`, CLI config | Add CI/job-token runtime auth mode; keep interactive login and secret printing non-goals. |
| `changelog` | `add` | Commits can be listed, but GitLab changelog generation is missing. | `releases` or `changelog` | Add generate/view changelog API support if it fits release workflows. |
| `ci` | `harden` | `pipeline_jobs` covers jobs, traces, cancel/retry/play. | `pipeline_jobs` | Add CI lint/compile, pipeline run/trigger, artifacts, and compact pipeline list/get envelopes. |
| `cluster` | `defer` | None. | future specialized toolset | Defer Kubernetes/agent management until a concrete agent workflow proves value. |
| `completion` | `non-goal` | None. | none | Shell completion is CLI-local and not an MCP capability. |
| `config` | `harden` | Config CLI and multi-server resolver exist. | CLI config, `project_config` | Finish strict-by-default migration, remote matching, TLS, and runtime auth modes. |
| `deploy-key` | `add` | No deploy key lifecycle tools. | `keys` | Add list/get/add/delete project deploy keys with write guards. |
| `duo` | `defer` | None. | future specialized toolset | Defer GitLab AI-specific workflows; not core parity. |
| `gpg-key` | `add` | No GPG key lifecycle tools. | `keys` | Add list/get/add/delete user GPG keys where token permissions allow. |
| `help` | `non-goal` | None. | none | CLI help rendering is local documentation, not a GitLab API workflow. |
| `incident` | `add` | Issues may overlap, but no incident-specific tools. | `work_items` or `incidents` | Add list/get/create/update/close where APIs support it; keep issue compatibility clear. |
| `issue` | `harden` | `issues` covers get/list/create/update/notes/labels/milestones. | `issues` | Add close/reopen/subscribe/todos and normalize compact list vs detailed get output. |
| `iteration` | `add` | No iteration tools. | `iterations` | Add compact list/get for project/group iterations. |
| `job` | `harden` | `pipeline_jobs` covers list/get/trace/retry/play. | `pipeline_jobs` | Add artifact metadata/fetch and align job aliases with CI coverage. |
| `label` | `add` | Issue labels can be read for an issue; label lifecycle is missing. | `labels` | Add compact list/get/create/update/delete for project and group labels. |
| `mcp` | `non-goal` | This server is already native MCP with explicit tools. | none | Keep `glab mcp serve` reference-only; do not wrap CLI MCP inside MCP. |
| `milestone` | `harden` | `issues` has list plus get/create/update via `milestone`. | `issues` or `milestones` | Add delete/close/reopen where supported and normalize group/project handling. |
| `mr` | `harden` | `merge_requests` covers get/list/create/update/notes. | `merge_requests` | Add approvals, merge, rebase, diffs, related issues, todos, and output normalization. |
| `opentofu` | `defer` | None. | future specialized toolset | Defer GitLab-managed OpenTofu/Terraform state unless requested. |
| `orbit` | `defer` | None. | future specialized toolset | Defer experimental Knowledge Graph workflows. |
| `repo` | `harden` | `projects` covers project get/list, files, branches, commits. | `projects`, `repository` | Add project members/settings/archive/fork/transfer/mirror/delete in safe toolsets; `project` is a CLI alias. |
| `release` | `add` | Tags exist; releases and assets are missing. | `releases` | Add list/get/create/update/delete releases and explicit asset link/upload handling. |
| `runner` | `defer` | Jobs can be viewed, but no runner administration. | future `runners` | Defer runner administration due to elevated permissions and infrastructure impact. |
| `runner-controller` | `defer` | None. | future specialized toolset | Defer experimental runner controller management. |
| `schedule` | `add` | No pipeline schedule tools. | `pipeline_jobs` or `schedules` | Add list/get/create/update/delete/run schedules without exposing trigger secrets. |
| `search` | `existing` | `search` covers projects, issues, MRs, blobs, commits, milestones, snippets, wiki, notes. | `search` | Keep coverage; harden pagination and output shape with search normalization work. |
| `securefile` | `defer` | None. | future specialized toolset | Defer secure file upload/download because it is secret-bearing project storage. |
| `skills` | `non-goal` | None. | none | Installing bundled agent skills is local tool management, not GitLab resource parity. |
| `snippet` | `add` | Search can find snippet titles/blobs; no snippet lifecycle. | `snippets` | Add compact list/get metadata and explicit content fetch/create/update/delete. |
| `ssh-key` | `add` | No SSH key lifecycle tools. | `keys` | Add list/get/add/delete user SSH keys where token permissions allow. |
| `stack` | `defer` | None. | none initially | Defer local Git stack workflows; likely CLI/worktree-local. |
| `todo` | `add` | No todo tools. | `todos` | Add compact todo list, mark done, and supported resource todo actions. |
| `token` | `non-goal` | Runtime token management exists; GitLab access-token lifecycle is not modeled. | none | Avoid secret-bearing token creation/export workflows in MCP output. |
| `update` | `non-goal` | None. | none | CLI update checks are local binary maintenance. |
| `user` | `harden` | `users` covers lookup/status/list/project users/admin state. | `users` | Add events, improve admin isolation, and normalize outputs. |
| `variable` | `add` | No project/group variable tools. | `variables` | Add list/set/update/delete without returning secret values by default. |
| `version` | `non-goal` | Server binary exposes normal version behavior outside MCP tools. | none | CLI version display is not a GitLab resource capability. |
| `whatsnew` | `non-goal` | None. | none | CLI release-note display is local update/help behavior. |
| `work-items` | `add` | Issues overlap, but no general work item tools. | `work_items` | Add list/get/create/update/delete for project/group work items once API shape is chosen. |

## Deferred Families

| Family | Reason |
|---|---|
| attestation | Experimental artifact provenance verification belongs with a later artifact/security story. |
| duo | Agent already has a reasoning surface; GitLab Duo-specific workflows need a separate user story. |
| cluster | Kubernetes and GitLab agent management are specialized and high-risk. |
| opentofu | Valuable only for teams using GitLab-managed Terraform/OpenTofu state. |
| orbit | Experimental Knowledge Graph workflows need a separate product decision. |
| runner | Runner administration affects shared infrastructure and permissions. |
| runner-controller | Experimental runner controller management is specialized infrastructure work. |
| securefile | Secure files are secret-bearing project storage and need explicit safety design. |
| stack | Local Git stack workflows are terminal/worktree concerns and may not belong in MCP. |

## Non-Goal Families

| Family | Reason |
|---|---|
| alias | Local CLI alias management. |
| completion | Shell completion is CLI-local. |
| help | CLI help rendering is local documentation. |
| mcp | `glab mcp serve` is reference-only for this native MCP server. |
| skills | Local installation of glab agent skills. |
| token | Secret-bearing token lifecycle workflows should not be exposed by default. |
| update | Local binary update checks. |
| version | Version display is process metadata, not a GitLab resource workflow. |
| whatsnew | Local CLI release-note display. |

## Source Notes

- `glab` source inspected locally at `/tmp/glab-cli`.
- Command directories discovered with `find /tmp/glab-cli/internal/commands -mindepth 1 -maxdepth 1 -type d -printf '%f\n' | sort`; directory names are not authoritative.
- Authoritative command family names come from Cobra root registration in `/tmp/glab-cli/internal/commands/root.go` and each registered command's `Use` string; this accounts for `repo` with alias `project`, `runner-controller`, and `work-items`.
- MCP command annotation references inspected under `/tmp/glab-cli/internal/mcpannotations` and `/tmp/glab-cli/internal/commands/mcp`.
- Current MCP coverage checked against `docs/TOOLS.md`, `pkg/gitlab/toolsets.go`, and `pkg/gitlab/__toolsnaps__/registered_*.snap`.
