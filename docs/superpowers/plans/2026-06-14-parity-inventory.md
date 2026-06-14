# Glab Parity Inventory Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce a compact, agent-readable parity inventory that maps every current `glab` command family to existing MCP coverage, desired ownership, and the next scoped action.

**Architecture:** Keep this as a documentation and inventory slice only. Use the local `glab` clone and the current MCP tool catalog as sources, then commit a stable matrix document that future implementation plans can reference without re-reading `glab` source. Do not add new MCP tools in this slice.

**Tech Stack:** Markdown documentation, local `glab` source at `/tmp/glab-cli`, current MCP docs/snapshots in `docs/TOOLS.md` and `pkg/gitlab/__toolsnaps__`, standard shell tools (`rg`, `find`, `sort`, `go test` for validation).

---

## Scope

This implements Phase 0 from `docs/superpowers/specs/2026-06-14-glab-parity-roadmap-design.md`.

Included:

- Build a compact `glab` command-family inventory from the local clone.
- Cross-check current MCP toolsets and registered tool schemas.
- Classify each command family as `existing`, `harden`, `add`, `defer`, or `non-goal`.
- Assign desired MCP toolset ownership for accepted domains.
- Record why specialized areas are deferred or excluded.
- Keep the final matrix small enough for an agent to read quickly.

Not included:

- Adding new MCP tools.
- Changing output envelopes.
- Implementing project detection hardening.
- Implementing TLS, job-token auth, or new admin/domain toolsets.

## File Map

- Create: `docs/superpowers/specs/2026-06-14-glab-parity-inventory.md`
  - The compact parity inventory artifact.
  - Contains source notes, status taxonomy, current MCP inventory summary, and the command-family matrix.
- Modify: `docs/superpowers/specs/2026-06-14-glab-parity-roadmap-design.md`
  - Replace or augment the seed matrix note with a link to the full inventory artifact.
  - Keep the roadmap as high-level strategy; do not duplicate the full matrix there.

## Status Taxonomy

Use these exact statuses in the inventory matrix:

| Status | Meaning |
|---|---|
| `existing` | Current MCP tools cover the practical `glab` family well enough for normal agent use. |
| `harden` | Current MCP tools partially cover the family, but need safer output, missing actions, or consistency work. |
| `add` | No adequate modeled MCP tool exists and the domain is in roadmap scope. |
| `defer` | Potentially useful later, but specialized, high-risk, or not agent-critical now. |
| `non-goal` | CLI-local, interactive, secret-printing, or terminal-only behavior that should not be modeled as MCP tools. |

## Task 1: Capture Glab Command Families

**Files:**
- Read: `/tmp/glab-cli`
- No repository files modified in this task.

- [ ] **Step 1: Verify the local `glab` clone exists**

Run:

```bash
test -d /tmp/glab-cli && git -C /tmp/glab-cli status --short --branch
```

Expected: output starts with:

```text
## main...origin/main
```

If `/tmp/glab-cli` is missing, run:

```bash
git clone https://gitlab.com/gitlab-org/cli.git /tmp/glab-cli
```

- [ ] **Step 2: Capture top-level command directories**

Run:

```bash
find /tmp/glab-cli/internal/commands -mindepth 1 -maxdepth 1 -type d -printf '%f\n' | sort
```

Expected: a sorted list of top-level command directories such as `api`, `auth`, `ci`, `config`, `issue`, `mr`, `project`, `release`, `user`, and specialized families. Directory names are discovery hints; root command `Use` strings are authoritative for the matrix.

- [ ] **Step 3: Capture annotated MCP command references**

Run:

```bash
rg -n 'mcp|MCP|Annotation|Use:' /tmp/glab-cli/internal/mcpannotations /tmp/glab-cli/internal/commands/mcp /tmp/glab-cli/internal/commands | sed -n '1,220p'
```

Expected: output that identifies the `glab mcp serve` implementation and command annotations. Use this only as source material; do not copy large chunks into repository docs.

- [ ] **Step 4: Write a temporary command-family checklist**

Create a local scratch file outside the repository:

```bash
find /tmp/glab-cli/internal/commands -mindepth 1 -maxdepth 1 -type d -printf '%f\n' \
  | sed -e '/^issuable$/d' \
        -e 's/^project$/repo/' \
        -e 's/^runnercontroller$/runner-controller/' \
        -e 's/^workitems$/work-items/' \
  | sort > /tmp/glab-command-families.txt
```

Expected:

```bash
wc -l /tmp/glab-command-families.txt
```

prints a non-zero command family count.

Do not commit `/tmp/glab-command-families.txt`.

## Task 2: Capture Current MCP Inventory

**Files:**
- Read: `docs/TOOLS.md`
- Read: `pkg/gitlab/__toolsnaps__/registered_*.snap`
- No repository files modified in this task.

- [ ] **Step 1: Capture current toolsets from docs**

Run:

```bash
rg -n '^### `|^\\| `[^`]+` \\|' docs/TOOLS.md
```

Expected: output lists documented toolsets and tools, including `projects`, `issues`, `merge_requests`, `pipeline_jobs`, `search`, `users`, `tags`, `security`, `token_management`, and `project_config`.

- [ ] **Step 2: Capture registered MCP schemas**

Run:

```bash
find pkg/gitlab/__toolsnaps__ -maxdepth 1 -type f -name 'registered_*.snap' -printf '%f\n' | sort
```

Expected: output includes registered snapshots such as:

```text
registered_getProject.snap
registered_listIssues.snap
registered_listMergeRequests.snap
registered_pipelineJob.snap
registered_search.snap
```

- [ ] **Step 3: Capture toolset ownership from code**

Run:

```bash
rg -n 'NewToolset\\(|AddReadTools\\(|AddWriteTools\\(' pkg/gitlab/toolsets.go
```

Expected: output shows the toolset names and registration blocks that will be cited in the inventory source notes.

## Task 3: Create the Parity Inventory Artifact

**Files:**
- Create: `docs/superpowers/specs/2026-06-14-glab-parity-inventory.md`

- [ ] **Step 1: Create the inventory document**

Create `docs/superpowers/specs/2026-06-14-glab-parity-inventory.md` with this structure:

```markdown
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
| `existing` | Current MCP tools cover the practical `glab` family well enough for normal agent use. |
| `harden` | Current MCP tools partially cover the family, but need safer output, missing actions, or consistency work. |
| `add` | No adequate modeled MCP tool exists and the domain is in roadmap scope. |
| `defer` | Potentially useful later, but specialized, high-risk, or not agent-critical now. |
| `non-goal` | CLI-local, interactive, secret-printing, or terminal-only behavior that should not be modeled as MCP tools. |

## Current MCP Toolset Summary

| MCP toolset | Current coverage | Notes |
|---|---|---|
| `projects` | Projects, repository files, branches, commits. | Harden repo/project lifecycle and settings later. |
| `issues` | Issues, issue labels, issue notes, milestones. | Harden close/reopen/subscribe/todos and compact output. |
| `merge_requests` | MR get/list/create/update and notes. | Add approvals, merge/rebase, diffs, related issues, todos. |
| `pipeline_jobs` | Pipeline job list/get/trace and cancel/retry/play actions. | Add CI lint/run/trigger/artifacts/schedules later. |
| `search` | Unified GitLab search across supported scopes. | Normalize output and pagination. |
| `users` | Current user, users, status, project users, admin state actions. | Harden admin isolation and add events later. |
| `tags` | Repository tag list/get/create/delete/get-commit. | Releases remain missing. |
| `security` | Security scan result readers. | Split large payloads where needed. |
| `project_config` | `.gmcprc` read/write and project detection helpers. | Harden remote matching in Phase 1. |
| `token_management` | Runtime token validation/update/remove/notifications. | Persistent auth belongs to CLI config commands. |

## Command-Family Matrix

| `glab` family | Status | Current MCP coverage | Desired owner | Next scoped action |
|---|---|---|---|---|
```

- [ ] **Step 2: Fill the matrix rows**

Add one row per current canonical `glab` top-level family from `/tmp/glab-command-families.txt`. Use Cobra `Use` strings from root-registered commands as canonical names; directory names are only a discovery aid.

Use these baseline classifications unless the fresh source inspection proves a family was renamed or removed:

```markdown
| `api` | `add` | None modeled; callers use specific tools only. | `api` toolset | Add scoped generic API passthrough with method/path/server, JSON size limits, and secret-prone path guards. |
| `auth` | `harden` | Runtime token management and config CLI exist; no interactive login MCP flow. | `token_management`, CLI config | Add CI/job-token runtime auth mode; keep interactive login and secret printing non-goals. |
| `ci` | `harden` | `pipeline_jobs` covers jobs, traces, cancel/retry/play. | `pipeline_jobs` | Add CI lint/compile, pipeline run/trigger, artifact metadata/fetch, and compact pipeline list/get envelopes. |
| `cluster` | `defer` | None. | future specialized toolset | Defer Kubernetes/agent management until a concrete agent workflow proves value. |
| `completion` | `non-goal` | None. | none | Shell completion is CLI-local and not an MCP capability. |
| `config` | `harden` | Config CLI and multi-server resolver exist. | CLI config, `project_config` | Finish strict-by-default migration, project remote matching, self-managed TLS, and runtime auth modes. |
| `duo` | `defer` | None. | future specialized toolset | Defer GitLab AI-specific workflows; not core parity. |
| `incident` | `add` | Issues may overlap, but no incident-specific tools. | `work_items` or `incidents` | Add list/get/create/update/close where GitLab APIs support it; keep issue compatibility clear. |
| `issue` | `harden` | `issues` covers get/list/create/update/notes/labels/milestones. | `issues` | Add close/reopen/subscribe/todos and normalize compact list vs detailed get output. |
| `label` | `add` | Issue labels can be read for an issue; project/group label lifecycle is missing. | `labels` | Add compact list/get/create/update/delete for project and group labels. |
| `mr` | `harden` | `merge_requests` covers get/list/create/update/notes. | `merge_requests` | Add approvals, merge, rebase, diffs, related issues, todos, and output normalization. |
| `opentofu` | `defer` | None. | future specialized toolset | Defer GitLab-managed OpenTofu state unless requested. |
| `release` | `add` | Tags exist; releases and assets are missing. | `releases` | Add list/get/create/update/delete releases and explicit asset link/upload handling. |
| `repo` | `harden` | `projects`, repository files, branches, commits, tags. | `projects`, `repository` | Add project members/settings/archive/fork/transfer/mirror/delete in safe toolsets; `project` is a CLI alias. |
| `schedule` | `add` | No pipeline schedule tools. | `pipeline_jobs` or `schedules` | Add list/get/create/update/delete/run schedules without exposing trigger secrets. |
| `snippet` | `add` | Search can find snippet titles/blobs; no snippet lifecycle. | `snippets` | Add compact list/get metadata and explicit content fetch/create/update/delete. |
| `ssh-key` | `add` | No SSH key lifecycle tools. | `keys` | Add list/get/add/delete user SSH keys where token permissions allow. |
| `gpg-key` | `add` | No GPG key lifecycle tools. | `keys` | Add list/get/add/delete user GPG keys where token permissions allow. |
| `stack` | `defer` | None. | none initially | Defer local Git stack workflows; likely CLI-local. |
| `todo` | `add` | No todo tools. | `todos` | Add compact todo list, mark done, and supported resource todo actions. |
| `user` | `harden` | `users` covers lookup/status/list/project users/admin state. | `users` | Add events, improve admin isolation, and normalize outputs. |
| `variable` | `add` | No project/group variable tools. | `variables` | Add list/set/update/delete without returning secret values by default. |
| `version` | `non-goal` | Server binary exposes normal version behavior outside MCP tools. | none | CLI version display is not a GitLab resource capability. |
```

If `/tmp/glab-command-families.txt` contains additional current families, add rows for them using these rules:

- Put specialized or experimental GitLab domains in `defer` unless they are already in Phase 4 or Phase 5.
- Put shell/local-only command helpers in `non-goal`.
- Put day-to-day GitLab resources in `add` or `harden`.

- [ ] **Step 3: Add deferred and non-goal rationale sections**

After the matrix, add:

```markdown
## Deferred Families

| Family | Reason |
|---|---|
| `duo` | Agent already has a reasoning surface; GitLab Duo-specific workflows need a separate user story. |
| `cluster` | Kubernetes and GitLab agent management are specialized and high-risk. |
| `opentofu` | Valuable only for teams using GitLab-managed Terraform/OpenTofu state. |
| `stack` | Local Git stack workflows are terminal/worktree concerns and may not belong in MCP. |

## Non-Goal Families

| Family | Reason |
|---|---|
| `completion` | Shell completion is CLI-local. |
| `version` | Version display is process metadata, not a GitLab resource workflow. |
```

Add any additional non-goal rows found in the current `glab` clone.

- [ ] **Step 4: Add source notes**

Add:

```markdown
## Source Notes

- `glab` source inspected locally at `/tmp/glab-cli`.
- Command directories discovered with `find /tmp/glab-cli/internal/commands -mindepth 1 -maxdepth 1 -type d -printf '%f\n' | sort`; directory names are not authoritative.
- Authoritative command family names come from Cobra root registration in `/tmp/glab-cli/internal/commands/root.go` and each registered command's `Use` string.
- MCP command annotation references inspected under `/tmp/glab-cli/internal/mcpannotations` and `/tmp/glab-cli/internal/commands/mcp`.
- Current MCP coverage checked against `docs/TOOLS.md`, `pkg/gitlab/toolsets.go`, and `pkg/gitlab/__toolsnaps__/registered_*.snap`.
```

## Task 4: Link Roadmap to the Full Inventory

**Files:**
- Modify: `docs/superpowers/specs/2026-06-14-glab-parity-roadmap-design.md`

- [ ] **Step 1: Update the Capability Matrix Seed intro**

Find:

```markdown
## Capability Matrix Seed
```

Directly below that heading, add:

```markdown
The full Phase 0 inventory lives in [2026-06-14-glab-parity-inventory.md](2026-06-14-glab-parity-inventory.md). The table below is a high-level seed retained for roadmap readability.
```

- [ ] **Step 2: Update the Next Step section**

In `## Next Step`, replace:

```markdown
After this roadmap is approved, create the first implementation-planning artifact for **Phase 0: Parity Inventory** or **Phase 1: Multi-Instance Foundation**. Phase 1 is the recommended engineering start because safe server selection affects every later domain.
```

with:

```markdown
Phase 1's first routing/read-only slice is implemented in [../plans/2026-06-14-multi-instance-routing-foundation.md](../plans/2026-06-14-multi-instance-routing-foundation.md). The next planning artifact is Phase 0 inventory: [../plans/2026-06-14-parity-inventory.md](../plans/2026-06-14-parity-inventory.md). After the inventory is complete, choose between the remaining Phase 1 slices (`project detection`, `strict-by-default migration`, `self-managed TLS`, `CI/job-token auth`) and Phase 2 output normalization.
```

## Task 5: Validate Inventory Compactness and Coverage

**Files:**
- Read: `docs/superpowers/specs/2026-06-14-glab-parity-inventory.md`
- Read: `docs/superpowers/specs/2026-06-14-glab-parity-roadmap-design.md`

- [ ] **Step 1: Check every `glab` family has a matrix row**

Run:

```bash
while read -r family; do
  rg -q "\\| \\`${family}\\` \\|" docs/superpowers/specs/2026-06-14-glab-parity-inventory.md || echo "missing: ${family}"
done < /tmp/glab-command-families.txt
```

Expected: no output.

- [ ] **Step 2: Check all matrix statuses are from the taxonomy**

Run:

```bash
awk -F'|' '/^\\| `/{gsub(/`| /,"",$3); if ($3 !~ /^(existing|harden|add|defer|non-goal)$/) print $0}' docs/superpowers/specs/2026-06-14-glab-parity-inventory.md
```

Expected: no output.

- [ ] **Step 3: Check the inventory is compact enough**

Run:

```bash
wc -l docs/superpowers/specs/2026-06-14-glab-parity-inventory.md
```

Expected: fewer than 180 lines. If it is longer, tighten prose before committing.

- [ ] **Step 4: Check roadmap links**

Run:

```bash
rg -n 'glab-parity-inventory|parity-inventory|multi-instance-routing-foundation' docs/superpowers/specs/2026-06-14-glab-parity-roadmap-design.md
```

Expected: output includes links to both the inventory artifact and the completed Phase 1 routing plan.

## Task 6: Commit

**Files:**
- Add: `docs/superpowers/specs/2026-06-14-glab-parity-inventory.md`
- Modify: `docs/superpowers/specs/2026-06-14-glab-parity-roadmap-design.md`
- Add: `docs/superpowers/plans/2026-06-14-parity-inventory.md`

- [ ] **Step 1: Review diff**

Run:

```bash
git diff -- docs/superpowers/specs/2026-06-14-glab-parity-inventory.md docs/superpowers/specs/2026-06-14-glab-parity-roadmap-design.md docs/superpowers/plans/2026-06-14-parity-inventory.md
```

Expected: only the new inventory plan, new inventory spec, and roadmap link update.

- [ ] **Step 2: Commit**

Run:

```bash
git add docs/superpowers/specs/2026-06-14-glab-parity-inventory.md docs/superpowers/specs/2026-06-14-glab-parity-roadmap-design.md docs/superpowers/plans/2026-06-14-parity-inventory.md
git commit -m "docs: add glab parity inventory plan"
```

Expected: commit succeeds.

## Verification

Run these commands before final handoff:

```bash
test -d /tmp/glab-cli
find /tmp/glab-cli/internal/commands -mindepth 1 -maxdepth 1 -type d -printf '%f\n' \
  | sed -e '/^issuable$/d' \
        -e 's/^project$/repo/' \
        -e 's/^runnercontroller$/runner-controller/' \
        -e 's/^workitems$/work-items/' \
  | sort > /tmp/glab-command-families.txt
while read -r family; do
  rg -q "\\| \\`${family}\\` \\|" docs/superpowers/specs/2026-06-14-glab-parity-inventory.md || echo "missing: ${family}"
done < /tmp/glab-command-families.txt
awk -F'|' '/^\\| `/{gsub(/`| /,"",$3); if ($3 !~ /^(existing|harden|add|defer|non-goal)$/) print $0}' docs/superpowers/specs/2026-06-14-glab-parity-inventory.md
wc -l docs/superpowers/specs/2026-06-14-glab-parity-inventory.md
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./... -count=1
```

Expected:

- no missing command-family output
- no invalid status output
- inventory line count below 180
- full Go test suite passes

## Implementation Notes

- Keep the inventory actionable but not implementation-specific. The matrix should choose ownership and next slices, not define full APIs.
- Do not duplicate `docs/TOOLS.md`; summarize current coverage in one row per toolset.
- If `glab` has command families not listed in this plan, add them to the matrix instead of forcing them into existing rows.
- Keep `duo`, `cluster`, `opentofu`, `stack`, and similarly specialized domains deferred unless the current `glab` source shows they became core day-to-day project workflows.
- Keep `glab mcp` as reference-only; this server remains native MCP with explicit structured tools.
