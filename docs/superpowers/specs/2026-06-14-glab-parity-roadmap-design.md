# Glab Parity Roadmap

**Date:** 2026-06-14
**Status:** Roadmap draft for review
**Purpose:** Define a compact, agent-readable roadmap for bringing `gitlab-mcp-server` to practical capability parity with `glab`, while keeping safer multi-instance behavior and MCP-native output contracts.

## Context

This project already has native MCP tools for common GitLab workflows, a multi-server config model, secret backends, project detection, dynamic toolsets, and a strict resolver option.

`glab` is useful as a parity reference, but it should not define this server's architecture. Its current `glab mcp serve` implementation dynamically exposes annotated Cobra commands as generic MCP tools, runs `glab` as a subprocess, accepts nested `args` and `flags`, auto-adds JSON output where possible, and truncates raw output by rune count. That model is broad, but it is still CLI-shaped and experimental.

This server should remain a native MCP server with explicit, structured GitLab tools.

## Definition of Parity

Parity means an LLM agent can safely accomplish the same practical GitLab work that a human can perform with `glab`, especially for day-to-day project work and instance management.

Parity does not mean:

- Mirroring every `glab` command name.
- Reusing `glab` as a subprocess backend.
- Exposing interactive, secret-printing, local Git checkout, or terminal-only workflows as MCP tools.
- Treating specialized or experimental `glab` areas as mandatory.

## Design Principles

1. **Capability parity over command parity.** Tools should model GitLab resources and agent workflows, not terminal syntax.
2. **Compact by default.** List tools return compact structured rows. Get/view tools return the full useful item slice.
3. **Large payloads are explicit.** Diffs, traces, artifacts, blobs, logs, raw markdown, and vulnerability details require separate tools or explicit flags.
4. **Instance safety comes first.** No tool expansion should depend on ambiguous host or token selection.
5. **Tasks stay separately plannable.** Each roadmap item should become its own future spec and implementation plan.
6. **Specialized domains are reference/backlog unless agent-critical.** Areas like Duo, cluster agents, OpenTofu, Orbit, Skills, and Stack are tracked but not early roadmap commitments.

## Output Contract

All toolsets should follow the same response policy.

### List Tools

List tools return compact rows only:

- stable identifiers: numeric ID, IID, full path, key name, or slug
- display name/title
- state/status
- web URL when GitLab provides one
- updated or created timestamp
- small status fields needed to choose the next action

List tools should not include:

- long descriptions
- note bodies
- full diffs
- full job traces
- file contents
- complete vulnerability payloads
- secret values

### Get/View Tools

Get/view tools return the full useful item slice:

- all fields needed to understand or modify the item
- permissions/approval/status fields when relevant
- links and related IDs
- recent metadata
- body/description if it is core to the item

Very large fields should still be split into explicit fetch tools.

### Pagination and Limits

Every list tool should expose consistent pagination parameters and return pagination metadata. Defaults should be conservative. Agents must be able to continue from the previous page without guessing.

### Error Shape

Errors should be structured enough for agents to recover:

- error code/category
- human message
- target server
- target project/group when known
- recovery hint when useful

## Roadmap Phases

### Phase 0: Parity Inventory

**Goal:** Produce and maintain a compact matrix mapping `glab` command families to MCP capabilities.

**Tasks:**

- Build a `glab` parity matrix from docs and source annotations.
- Classify each command family as `existing`, `harden`, `add`, `defer`, or `non-goal`.
- Record why specialized areas are deferred or excluded.
- Add the current MCP tool inventory and toolset ownership.

**Acceptance checks:**

- Matrix includes every current `glab` top-level command family.
- Matrix identifies which MCP toolset should own each accepted domain.
- Matrix is small enough for an agent to read without loading command docs.

### Phase 1: Multi-Instance Foundation

**Goal:** Make several GitLab instances safe and predictable before broadening the tool surface.

**Tasks:**

- Add first-class server metadata to the client pool: name, host, API host, read-only flag, username, token type, TLS settings.
- Ensure every normal GitLab tool can accept an explicit `server` argument.
- Make resolver behavior strict by default for multi-server configs or provide a migration path to strict-by-default.
- Remove silent fallback paths that can hit the wrong instance.
- Enforce per-server read-only mode at write-tool execution time, not only registration time.
- Improve project detection to match remotes against configured hosts and remote aliases more like `glab`.
- Add self-managed instance settings: API host override, CA certificate, mTLS certificate/key, skip TLS verification with clear warnings.
- Add CI/job-token auth as a separate non-persistent runtime mode where safe.

**Acceptance checks:**

- A tool call cannot silently use a different server than requested or pinned.
- Read-only server configuration blocks write actions even when the tool is registered.
- Project detection reports ambiguity instead of guessing.
- Self-managed TLS settings are testable without real private infrastructure.

### Phase 2: Output and Schema Normalization

**Goal:** Standardize tool responses before adding more domains.

**Tasks:**

- Define shared list/get response envelope conventions.
- Normalize pagination arguments and metadata across list tools.
- Split large payloads out of existing list/get tools where needed.
- Add stable output tests or snapshots for compact list rows and detailed get responses.
- Audit current docs that claim every tool accepts `server` and align implementation with docs.

**Acceptance checks:**

- Existing high-traffic list tools are compact.
- Existing get/view tools contain enough detail for update decisions.
- Tool snapshots prove schema consistency.

### Phase 3: Existing Domain Hardening

**Goal:** Bring current domains closer to glab parity before adding many new toolsets.

**Domains:**

- projects and repositories
- issues and issue notes
- merge requests, approvals, notes, diffs, merge/rebase actions
- pipelines and jobs
- tags and milestones
- users and project members
- search
- security reports

**Tasks:**

- Fill missing get/list/update/close/reopen/subscribe/todo-style actions where they are part of core GitLab work.
- Add missing merge request approval, merge, rebase, diff, related issues, and todo capabilities.
- Add CI lint/compile, pipeline trigger/run, pipeline delete, artifacts metadata, and trace fetch parity.
- Add project members add/remove/update where safe.
- Ensure every destructive action has a read-only guard and clear confirmation-oriented naming.

**Acceptance checks:**

- A common issue/MR/CI workflow can be completed without falling back to `glab`.
- List commands stay compact under realistic project sizes.
- Dangerous actions are classified as write/destructive and blocked in read-only contexts.

### Phase 4: Missing Core GitLab Domains

**Goal:** Add high-value GitLab resource domains that `glab` exposes and agents commonly need.

**Task cards:**

#### Releases

- List releases compactly.
- Get full release details.
- Create, update, delete releases.
- Upload/link assets without dumping binary content.

#### Labels

- List compact labels.
- Get label details.
- Create, update, delete labels.

#### Variables

- Support project/group variable list, set, update, delete.
- Never return secret values unless GitLab itself marks them visible and the tool is explicitly designed for that.
- Keep `glab`'s MCP exclusions around variable value exposure as a warning signal.

#### Snippets

- List compact snippets.
- Get snippet metadata and content explicitly.
- Create, update, delete snippets.

#### Schedules and Triggers

- List pipeline schedules.
- Get schedule details.
- Create, update, delete, and run schedules.
- Manage trigger tokens without exposing token values.

#### Deploy Keys

- List and get deploy keys.
- Add, enable, disable, delete deploy keys.

#### SSH and GPG Keys

- List and get user keys.
- Add and delete keys when token permissions allow.

#### Todos

- List todos compactly.
- Mark done.
- Create todo actions for supported resources where available.

#### Work Items and Incidents

- List and get work items/incidents.
- Create, update, close/delete where APIs support it.
- Keep issue compatibility clear.

#### Secure Files

- List and get metadata.
- Create/remove entries.
- Download content only through explicit file/content fetch tools with size limits.

#### Changelog

- Generate changelog from GitLab API.
- Keep output bounded by range and explicit version/ref parameters.

#### API Passthrough

- Provide a carefully scoped generic API tool for endpoints not yet modeled.
- Require explicit method/path/server.
- Block or warn on secret-prone paths.
- Return structured JSON with size limits.

**Acceptance checks:**

- Each domain has list/get separation.
- Secret-bearing domains do not leak values by default.
- Generic API passthrough is not a substitute for core modeled tools.

### Phase 5: Instance and Administration Management

**Goal:** Let an agent manage a GitLab instance where credentials allow it.

**Task cards:**

#### Groups and Namespaces

- List/search groups.
- Get group details.
- Manage group members.
- Read group projects.

#### Users and Admin

- Harden existing user tools.
- Add user events where useful.
- Keep admin-only state changes clearly destructive.

#### Runners

- List runners and managers compactly.
- Get runner details and jobs.
- Assign, unassign, update, delete runners.
- Add runner-controller support only if it maps cleanly to GitLab APIs and does not expose registration tokens.

#### Protected Resources

- Manage protected branches and tags.
- Include approval/permission details in get responses.

#### Project Settings

- Read and update safe project settings.
- Archive/unarchive, transfer, fork, mirror, and delete with strong destructive classification.

#### Tokens

- List/revoke tokens where GitLab APIs support it.
- Create/rotate only when returned values can be handled without exposing secrets to the LLM, or mark as non-goal.

#### Audit and Events

- Add project/group/user events where available.
- Keep output compact and filterable by date, actor, action, and target.

**Acceptance checks:**

- Admin-capable tools are isolated in clearly named toolsets.
- Read-only mode blocks all mutating instance-management actions.
- Token-creating workflows do not print secrets into MCP output.

### Phase 6: Specialized Reference Backlog

**Goal:** Track glab areas that may be useful later without letting them bloat core parity.

**Default classification:** defer unless a future user story proves agent value.

| Area | Classification | Notes |
|---|---|---|
| `duo` | defer | Agent already has its own reasoning surface; GitLab Duo may be useful only for explicit GitLab AI workflows. |
| `cluster` | defer | Kubernetes agent management is specialized and high-risk. |
| `opentofu` | defer | Useful only for teams using GitLab-managed Terraform/OpenTofu state. |
| `orbit` | defer | Experimental/specialized. |
| `skills` | defer | Glab-specific feature surface. |
| `stack` | defer | Local Git workflow concerns may not belong in this MCP server. |
| `attestation` | defer | Could become useful for supply-chain verification, but not core instance management. |
| `runner-controller` | partial defer | Track runner-controller read/list first; token creation remains sensitive. |
| `glab mcp` | reference only | Useful as coverage and annotation inspiration, not an implementation target. |

## Capability Matrix Seed

The full Phase 0 inventory lives in [2026-06-14-glab-parity-inventory.md](2026-06-14-glab-parity-inventory.md). The table below is a high-level seed retained for roadmap readability.

| Glab family | Roadmap status | MCP direction |
|---|---|---|
| `api` | add | Scoped generic API passthrough. |
| `auth` | harden/add | Config and runtime auth modes, not interactive login tools. |
| `config` | harden | Multi-instance config, TLS, secret backends. |
| `project` / `repo` | harden/add | Projects, repo files, members, settings, archive/fork/transfer/mirror. |
| `issue` / `issuable` | harden | Issues, notes, close/reopen, subscribe, todos. |
| `mr` | harden | MRs, approvals, merge/rebase, diffs, notes, todos. |
| `ci` / `job` | harden | Pipelines, jobs, lint/compile, run/trigger, traces, artifacts. |
| `release` | add | Releases and assets. |
| `label` | add | Project/group labels. |
| `milestone` | harden | Complete milestone lifecycle. |
| `variable` | add carefully | No secret value leakage by default. |
| `snippet` | add | Metadata list and explicit content fetch. |
| `schedule` | add | Pipeline schedules. |
| `deploy-key` | add | Deploy key lifecycle. |
| `ssh-key` / `gpg-key` | add | User key lifecycle. |
| `todo` | add | Todo list/done and resource todo actions. |
| `user` | harden/add | User info, events, admin state. |
| `runner` | add | Runner lifecycle and jobs. |
| `runner-controller` | defer/partial | Avoid token exposure. |
| `work-items` / `incident` | add | Work item and incident lifecycle. |
| `securefile` | add carefully | Metadata first, content explicit. |
| `changelog` | add | Generated changelog output. |
| specialized families | defer | See Phase 6. |

## Documentation Requirements

Each future domain spec should include:

- current `glab` command references
- current MCP tool inventory
- desired toolset ownership
- list/get output shapes
- write/destructive classification
- server-routing behavior
- secret-handling rules
- tests and snapshots

## Source Notes

References inspected while drafting:

- `glab` README and documentation for command families and `glab mcp serve`.
- Local clone at `/tmp/glab-cli` on 2026-06-14.
- `glab` MCP implementation: `/tmp/glab-cli/internal/commands/mcp/serve`.
- `glab` MCP command annotations: `/tmp/glab-cli/internal/mcpannotations`.
- Current server docs and code: `docs/TOOLS.md`, `docs/MULTI_SERVER_SETUP.md`, `docs/PROJECT_CONFIG.md`, `pkg/gitlab`, `pkg/toolsets`, and `pkg/config`.

## Next Step

Phase 1's first routing/read-only slice is implemented in [../plans/2026-06-14-multi-instance-routing-foundation.md](../plans/2026-06-14-multi-instance-routing-foundation.md). The next planning artifact is Phase 0 inventory: [../plans/2026-06-14-parity-inventory.md](../plans/2026-06-14-parity-inventory.md). After the inventory is complete, choose between the remaining Phase 1 slices (`project detection`, `strict-by-default migration`, `self-managed TLS`, `CI/job-token auth`) and Phase 2 output normalization.
