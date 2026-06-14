# Project Detection Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make project auto-detection choose GitLab remotes predictably, match configured server hosts, and report ambiguity instead of guessing.

**Architecture:** Replace the current first-URL-wins parser with a small candidate model that preserves remote name, URL, project path, and host. Keep low-level Git parsing in `pkg/gitlab/project_config.go`, add configured-server selection helpers next to MCP project tools, and let CLI/MCP callers decide whether to write `.gmcprc` or return detection details. This slice does not add TLS settings, job-token auth, or output envelope normalization.

**Tech Stack:** Go 1.24, `gitlab.com/gitlab-org/api/client-go`, existing `pkg/gitlab` project-config helpers, `pkg/config.Manager`, Cobra project CLI, `testify`, `go test`.

---

## Scope

This is the next Phase 1 slice after explicit `server` routing/read-only guards. It implements the roadmap requirement:

- "Improve project detection to match remotes against configured hosts and remote aliases more like `glab`."
- "Project detection reports ambiguity instead of guessing."

Included:

- Parse all GitLab-like remotes into structured candidates.
- Preserve remote names (`origin`, `upstream`, custom names) and raw URLs.
- Prefer configured server host matches when a server list is available.
- Prefer common remote aliases only when that does not hide ambiguity.
- Return clear ambiguity errors listing candidate remotes and hosts.
- Update CLI `project init` and MCP `project_config` tools to use the same selection behavior.
- Update docs to describe ambiguity and `--server` recovery.

Not included:

- Making strict resolver default.
- Self-managed TLS/CA/mTLS/skip-verify settings.
- CI/job-token auth.
- Adding new project/repository lifecycle tools.
- Replacing all project outputs with a new response envelope.

## File Map

- Modify: `pkg/gitlab/project_config.go`
  - Add `GitRemoteCandidate`, remote parsing, host normalization, and deterministic selection helpers.
  - Keep `DetectProjectFromGit() (projectID, gitlabHost string, err error)` for compatibility.
- Modify: `pkg/gitlab/project_config_test.go`
  - Add candidate parser, ambiguity, remote-priority, host normalization, and compatibility tests.
- Modify: `pkg/gitlab/project_tools.go`
  - Use candidate selection for MCP `setCurrentProject`, `getCurrentProject`, `detectProject`, and `autoDetectAndSetProject`.
  - Replace single-server host lookup with ambiguity-aware matching.
- Modify: `pkg/gitlab/project_tools_test.go`
  - Cover ambiguous server host matches and selected server output.
- Modify: `cmd/project/init.go`
  - Use candidate selection with configured servers and `--server` recovery.
- Modify: `cmd/project/detect.go`
  - Print selected remote name and ambiguity errors.
- Modify: `cmd/project/project.go`
  - Keep existing wrapper function names stable: `readConfig`, `writeConfig`, and `detectFromGit`.
- Modify: `cmd/project/project_test.go`
  - Cover multi-remote selection and ambiguity behavior.
- Modify: `docs/PROJECT_CONFIG.md`
  - Document remote matching, configured-server matching, and ambiguity recovery.
- Modify: `docs/MULTI_SERVER_SETUP.md`
  - Update project init notes for multiple matching remotes/servers.

## Domain Model

Add this type in `pkg/gitlab/project_config.go`:

```go
type GitRemoteCandidate struct {
	RemoteName string
	URL        string
	ProjectID  string
	Host       string
}
```

Keep these compatibility APIs:

```go
func DetectProjectFromGit() (projectID, gitlabHost string, err error)
func parseGitRemotes(configData []byte) (projectID, gitlabHost string, err error)
func parseGitLabURL(url string) (projectID, host string, err error)
```

Add these new APIs:

```go
func ParseGitRemoteCandidates(configData []byte) ([]GitRemoteCandidate, error)
func SelectGitRemoteCandidate(candidates []GitRemoteCandidate, allowedHosts []string) (GitRemoteCandidate, error)
func DetectProjectCandidateFromGit(allowedHosts []string) (GitRemoteCandidate, error)
func NormalizeGitLabHost(host string) string
```

Selection rules:

1. Ignore non-parseable, non-GitHub remotes.
2. GitHub remotes remain hard errors.
3. If `allowedHosts` is non-empty, keep only candidates whose normalized host matches one configured host.
4. If zero candidates remain, return a "no GitLab remote found" error.
5. If exactly one candidate remains, select it.
6. If multiple remain and exactly one uses a preferred remote name, select it.
7. Preferred remote names are `origin`, then `gitlab`, then `upstream`.
8. If multiple candidates remain after preferences, return an ambiguity error listing `remote`, `host`, and `projectId`.

## Task 1: Parse Remote Candidates

**Files:**
- Modify: `pkg/gitlab/project_config.go`
- Modify: `pkg/gitlab/project_config_test.go`

- [ ] **Step 1: Write failing candidate parser tests**

Add to `pkg/gitlab/project_config_test.go`:

```go
func TestParseGitRemoteCandidates(t *testing.T) {
	configData := []byte(`[remote "upstream"]
	url = https://gitlab.com/group/upstream.git
[remote "origin"]
	url = git@gitlab.example.com:team/repo.git
[remote "mirror"]
	url = /srv/git/local.git
`)

	candidates, err := ParseGitRemoteCandidates(configData)
	require.NoError(t, err)
	require.Len(t, candidates, 2)

	assert.Equal(t, GitRemoteCandidate{
		RemoteName: "upstream",
		URL:        "https://gitlab.com/group/upstream.git",
		ProjectID:  "group/upstream",
		Host:       "https://gitlab.com",
	}, candidates[0])
	assert.Equal(t, GitRemoteCandidate{
		RemoteName: "origin",
		URL:        "git@gitlab.example.com:team/repo.git",
		ProjectID:  "team/repo",
		Host:       "https://gitlab.example.com",
	}, candidates[1])
}

func TestParseGitRemoteCandidates_GitHubHardError(t *testing.T) {
	configData := []byte(`[remote "origin"]
	url = https://github.com/owner/repo.git
`)

	candidates, err := ParseGitRemoteCandidates(configData)
	require.Error(t, err)
	assert.Nil(t, candidates)
	assert.Contains(t, err.Error(), "GitHub repository detected")
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./pkg/gitlab -run 'TestParseGitRemoteCandidates' -count=1
```

Expected: FAIL because `GitRemoteCandidate` and `ParseGitRemoteCandidates` do not exist.

- [ ] **Step 3: Implement candidate parsing**

Add to `pkg/gitlab/project_config.go` near `parseGitRemotes`:

```go
type GitRemoteCandidate struct {
	RemoteName string
	URL        string
	ProjectID  string
	Host       string
}

func ParseGitRemoteCandidates(configData []byte) ([]GitRemoteCandidate, error) {
	lines := bytes.Split(configData, []byte{'\n'})
	candidates := make([]GitRemoteCandidate, 0)
	currentRemote := ""

	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if bytes.HasPrefix(trimmed, []byte("[remote ")) {
			currentRemote = parseRemoteSectionName(string(trimmed))
			continue
		}
		if currentRemote == "" || !bytes.HasPrefix(trimmed, []byte("url = ")) {
			continue
		}

		rawURL := strings.TrimSpace(string(trimmed[6:]))
		projectID, host, err := parseGitLabURL(rawURL)
		if err != nil {
			return nil, err
		}
		if projectID == "" || host == "" {
			continue
		}
		candidates = append(candidates, GitRemoteCandidate{
			RemoteName: currentRemote,
			URL:        rawURL,
			ProjectID:  projectID,
			Host:       host,
		})
	}

	return candidates, nil
}

func parseRemoteSectionName(section string) string {
	const prefix = `[remote "`
	const suffix = `"]`
	if !strings.HasPrefix(section, prefix) || !strings.HasSuffix(section, suffix) {
		return ""
	}
	return strings.TrimSuffix(strings.TrimPrefix(section, prefix), suffix)
}
```

- [ ] **Step 4: Run parser tests**

Run:

```bash
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./pkg/gitlab -run 'TestParseGitRemoteCandidates' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/gitlab/project_config.go pkg/gitlab/project_config_test.go
git commit -m "feat: parse git remote candidates"
```

## Task 2: Select Remote Candidate Deterministically

**Files:**
- Modify: `pkg/gitlab/project_config.go`
- Modify: `pkg/gitlab/project_config_test.go`

- [ ] **Step 1: Write failing selection tests**

Add to `pkg/gitlab/project_config_test.go`:

```go
func TestSelectGitRemoteCandidate_MatchesAllowedHost(t *testing.T) {
	candidates := []GitRemoteCandidate{
		{RemoteName: "origin", ProjectID: "oss/repo", Host: "https://gitlab.com"},
		{RemoteName: "work", ProjectID: "team/repo", Host: "https://gitlab.example.com"},
	}

	selected, err := SelectGitRemoteCandidate(candidates, []string{"https://gitlab.example.com/"})
	require.NoError(t, err)
	assert.Equal(t, "work", selected.RemoteName)
	assert.Equal(t, "team/repo", selected.ProjectID)
}

func TestSelectGitRemoteCandidate_PrefersOriginWhenNoHostFilter(t *testing.T) {
	candidates := []GitRemoteCandidate{
		{RemoteName: "upstream", ProjectID: "group/upstream", Host: "https://gitlab.com"},
		{RemoteName: "origin", ProjectID: "group/repo", Host: "https://gitlab.com"},
	}

	selected, err := SelectGitRemoteCandidate(candidates, nil)
	require.NoError(t, err)
	assert.Equal(t, "origin", selected.RemoteName)
	assert.Equal(t, "group/repo", selected.ProjectID)
}

func TestSelectGitRemoteCandidate_AmbiguousWhenMultipleRemain(t *testing.T) {
	candidates := []GitRemoteCandidate{
		{RemoteName: "work", ProjectID: "team/api", Host: "https://gitlab.example.com"},
		{RemoteName: "mirror", ProjectID: "team/api-mirror", Host: "https://gitlab.example.com"},
	}

	_, err := SelectGitRemoteCandidate(candidates, []string{"https://gitlab.example.com"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous GitLab remotes")
	assert.Contains(t, err.Error(), "work")
	assert.Contains(t, err.Error(), "mirror")
}

func TestNormalizeGitLabHost(t *testing.T) {
	assert.Equal(t, "https://gitlab.example.com", NormalizeGitLabHost("https://GitLab.Example.com/"))
	assert.Equal(t, "https://gitlab.example.com", NormalizeGitLabHost("gitlab.example.com"))
}
```

- [ ] **Step 2: Run tests to verify failure**

```bash
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./pkg/gitlab -run 'TestSelectGitRemoteCandidate|TestNormalizeGitLabHost' -count=1
```

Expected: FAIL because selection helpers do not exist.

- [ ] **Step 3: Implement host normalization and selection**

Add to `pkg/gitlab/project_config.go`:

```go
func NormalizeGitLabHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(strings.TrimSuffix(host, "/")))
	if host == "" {
		return ""
	}
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}
	return host
}

func SelectGitRemoteCandidate(candidates []GitRemoteCandidate, allowedHosts []string) (GitRemoteCandidate, error) {
	filtered := filterCandidatesByAllowedHosts(candidates, allowedHosts)
	if len(filtered) == 0 {
		return GitRemoteCandidate{}, fmt.Errorf("no GitLab remote found in .git/config")
	}
	if len(filtered) == 1 {
		return filtered[0], nil
	}

	for _, preferred := range []string{"origin", "gitlab", "upstream"} {
		matches := make([]GitRemoteCandidate, 0, 1)
		for _, candidate := range filtered {
			if candidate.RemoteName == preferred {
				matches = append(matches, candidate)
			}
		}
		if len(matches) == 1 {
			return matches[0], nil
		}
	}

	return GitRemoteCandidate{}, fmt.Errorf("ambiguous GitLab remotes: %s", formatRemoteCandidates(filtered))
}

func filterCandidatesByAllowedHosts(candidates []GitRemoteCandidate, allowedHosts []string) []GitRemoteCandidate {
	if len(allowedHosts) == 0 {
		return candidates
	}
	allowed := make(map[string]struct{}, len(allowedHosts))
	for _, host := range allowedHosts {
		normalized := NormalizeGitLabHost(host)
		if normalized != "" {
			allowed[normalized] = struct{}{}
		}
	}
	filtered := make([]GitRemoteCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if _, ok := allowed[NormalizeGitLabHost(candidate.Host)]; ok {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

func formatRemoteCandidates(candidates []GitRemoteCandidate) string {
	parts := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		parts = append(parts, fmt.Sprintf("%s=%s on %s", candidate.RemoteName, candidate.ProjectID, candidate.Host))
	}
	return strings.Join(parts, ", ")
}
```

- [ ] **Step 4: Run selection tests**

```bash
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./pkg/gitlab -run 'TestSelectGitRemoteCandidate|TestNormalizeGitLabHost' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/gitlab/project_config.go pkg/gitlab/project_config_test.go
git commit -m "feat: select gitlab remote candidates"
```

## Task 3: Preserve Compatibility Detection APIs

**Files:**
- Modify: `pkg/gitlab/project_config.go`
- Modify: `pkg/gitlab/project_config_test.go`

- [ ] **Step 1: Write compatibility and ambiguity tests**

Add to `pkg/gitlab/project_config_test.go`:

```go
func TestParseGitRemotes_UsesCandidateSelection(t *testing.T) {
	configData := []byte(`[remote "upstream"]
	url = https://gitlab.com/group/upstream.git
[remote "origin"]
	url = https://gitlab.com/group/repo.git
`)

	projectID, host, err := parseGitRemotes(configData)
	require.NoError(t, err)
	assert.Equal(t, "group/repo", projectID)
	assert.Equal(t, "https://gitlab.com", host)
}

func TestDetectProjectCandidateFromGit_Ambiguous(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	require.NoError(t, os.Mkdir(gitDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "config"), []byte(`[remote "work"]
	url = https://gitlab.example.com/team/api.git
[remote "mirror"]
	url = https://gitlab.example.com/team/api-mirror.git
`), 0644))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	_, err = DetectProjectCandidateFromGit([]string{"https://gitlab.example.com"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous GitLab remotes")
}
```

- [ ] **Step 2: Run tests to verify failure**

```bash
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./pkg/gitlab -run 'TestParseGitRemotes_UsesCandidateSelection|TestDetectProjectCandidateFromGit_Ambiguous' -count=1
```

Expected: FAIL because compatibility functions still use first URL or new function is missing.

- [ ] **Step 3: Implement compatibility through candidates**

Replace `DetectProjectFromGit` and `parseGitRemotes` bodies in `pkg/gitlab/project_config.go` with:

```go
func DetectProjectFromGit() (projectID, gitlabHost string, err error) {
	candidate, err := DetectProjectCandidateFromGit(nil)
	if err != nil {
		return "", "", err
	}
	return candidate.ProjectID, candidate.Host, nil
}

func DetectProjectCandidateFromGit(allowedHosts []string) (GitRemoteCandidate, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return GitRemoteCandidate{}, fmt.Errorf("failed to get working directory: %w", err)
	}

	gitDir := findGitDir(cwd)
	if gitDir == "" {
		return GitRemoteCandidate{}, fmt.Errorf("not a Git repository (or any parent up to mount point)")
	}

	configData, err := os.ReadFile(filepath.Join(gitDir, "config"))
	if err != nil {
		return GitRemoteCandidate{}, fmt.Errorf("failed to read .git/config: %w", err)
	}

	candidates, err := ParseGitRemoteCandidates(configData)
	if err != nil {
		return GitRemoteCandidate{}, fmt.Errorf("failed to parse Git remotes: %w", err)
	}

	candidate, err := SelectGitRemoteCandidate(candidates, allowedHosts)
	if err != nil {
		return GitRemoteCandidate{}, err
	}
	return candidate, nil
}

func parseGitRemotes(configData []byte) (projectID, gitlabHost string, err error) {
	candidates, err := ParseGitRemoteCandidates(configData)
	if err != nil {
		return "", "", err
	}
	candidate, err := SelectGitRemoteCandidate(candidates, nil)
	if err != nil {
		if strings.Contains(err.Error(), "no GitLab remote found") {
			return "", "", nil
		}
		return "", "", err
	}
	return candidate.ProjectID, candidate.Host, nil
}
```

- [ ] **Step 4: Run project config tests**

```bash
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./pkg/gitlab -run 'TestParseGitRemotes|TestDetectProjectFromGit|TestDetectProjectCandidateFromGit' -count=1
```

Expected: PASS. If existing "multiple remotes, GitLab first" test now expects `origin`, update that test name/expectation to the new deterministic selection rule.

- [ ] **Step 5: Commit**

```bash
git add pkg/gitlab/project_config.go pkg/gitlab/project_config_test.go
git commit -m "fix: detect gitlab remotes without guessing"
```

## Task 4: Match Configured Servers for CLI Project Init

**Files:**
- Modify: `cmd/project/init.go`
- Modify: `cmd/project/detect.go`
- Modify: `cmd/project/project_test.go`

- [ ] **Step 1: Write failing CLI tests**

Add to `cmd/project/project_test.go`:

```go
func TestDetectFromGit_PrefersOriginWithMultipleGitLabRemotes(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	require.NoError(t, os.Mkdir(gitDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "config"), []byte(`[remote "upstream"]
	url = https://gitlab.com/group/upstream.git
[remote "origin"]
	url = https://gitlab.com/group/repo.git
`), 0644))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	projectID, gitlabHost, err := detectFromGit()
	require.NoError(t, err)
	assert.Equal(t, "group/repo", projectID)
	assert.Equal(t, "https://gitlab.com", gitlabHost)
}
```

- [ ] **Step 2: Run CLI test**

```bash
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./cmd/project -run 'TestDetectFromGit_PrefersOriginWithMultipleGitLabRemotes' -count=1
```

Expected: PASS once Task 3 compatibility is implemented; FAIL if first-URL behavior remains.

- [ ] **Step 3: Add configured-host detection helper**

In `cmd/project/init.go`, add:

```go
func configuredHostsFromManager(mgr *pkgConfig.Manager) []string {
	if mgr == nil {
		return nil
	}
	servers := mgr.ListServers()
	hosts := make([]string, 0, len(servers))
	for _, server := range servers {
		if server.Host != "" {
			hosts = append(hosts, server.Host)
		}
	}
	return hosts
}
```

Update `runInit` detection branch:

```go
var mgr *pkgConfig.Manager
mgr, _ = pkgConfig.NewManager("")
if len(args) > 0 {
	projectID = args[0]
} else {
	candidate, err := gitlab.DetectProjectCandidateFromGit(configuredHostsFromManager(mgr))
	if err != nil {
		return fmt.Errorf("failed to detect project from Git remote: %w\n\n"+
			"Please specify projectId explicitly:\n"+
			"  gitlab-mcp-server project init <projectId>", err)
	}
	projectID = candidate.ProjectID
	gitlabHost = candidate.Host
}
```

Then reuse the same `mgr` later instead of creating a second manager.

- [ ] **Step 4: Update detect command output**

In `cmd/project/detect.go`, replace the detection call with:

```go
candidate, err := gitlab.DetectProjectCandidateFromGit(nil)
if err != nil {
	return fmt.Errorf("failed to detect project: %w", err)
}
projectID := candidate.ProjectID
gitlabHost := candidate.Host
```

Print the selected remote:

```go
fmt.Fprintf(cmd.OutOrStdout(), "  Remote:      %s\n", candidate.RemoteName)
```

- [ ] **Step 5: Run CLI package tests**

```bash
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./cmd/project -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add cmd/project/init.go cmd/project/detect.go cmd/project/project_test.go
git commit -m "fix: use configured hosts for project init"
```

## Task 5: Match Servers in MCP Project Tools

**Files:**
- Modify: `pkg/gitlab/project_tools.go`
- Modify: `pkg/gitlab/project_tools_test.go`

- [ ] **Step 1: Write failing server match tests**

Add to `pkg/gitlab/project_tools_test.go`:

```go
func TestFindServerByHost_Ambiguous(t *testing.T) {
	store := NewTokenStore()
	require.NoError(t, store.AddToken("work", &TokenMetadata{GitLabHost: "https://gitlab.example.com"}))
	require.NoError(t, store.AddToken("mirror", &TokenMetadata{GitLabHost: "https://gitlab.example.com/"}))

	server, err := findServerByHost("https://gitlab.example.com", store)
	require.Error(t, err)
	assert.Empty(t, server)
	assert.Contains(t, err.Error(), "multiple configured servers match host")
}

func TestFindServerByHost_NormalizesHost(t *testing.T) {
	store := NewTokenStore()
	require.NoError(t, store.AddToken("work", &TokenMetadata{GitLabHost: "https://GitLab.Example.com/"}))

	server, err := findServerByHost("gitlab.example.com", store)
	require.NoError(t, err)
	assert.Equal(t, "work", server)
}
```

- [ ] **Step 2: Run tests to verify failure**

```bash
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./pkg/gitlab -run 'TestFindServerByHost' -count=1
```

Expected: FAIL because `findServerByHost` currently returns only `string`.

- [ ] **Step 3: Update `findServerByHost`**

Change `findServerByHost` in `pkg/gitlab/project_tools.go`:

```go
func findServerByHost(gitlabHost string, tokenStore *TokenStore) (string, error) {
	if tokenStore == nil || gitlabHost == "" {
		return "", nil
	}

	normalizedHost := NormalizeGitLabHost(gitlabHost)
	matches := make([]string, 0, 1)
	for name, metadata := range tokenStore.ListTokens() {
		if metadata == nil {
			continue
		}
		if NormalizeGitLabHost(metadata.GitLabHost) == normalizedHost {
			matches = append(matches, name)
		}
	}
	if len(matches) == 0 {
		return "", nil
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple configured servers match host %s: %s", gitlabHost, strings.Join(matches, ", "))
	}
	return matches[0], nil
}
```

Update call sites to handle the error. For `setCurrentProject`, if auto-detect finds ambiguous servers, return an MCP error result:

```go
matchedServer, matchErr := findServerByHost(detectedHost, tokenStore)
if matchErr != nil {
	return mcp.NewToolResultError(matchErr.Error()), nil
}
server = matchedServer
```

- [ ] **Step 4: Use candidate detection in MCP tools**

Replace these four `DetectProjectFromGit()` calls in `pkg/gitlab/project_tools.go` with `DetectProjectCandidateFromGit(nil)`:

- `SetCurrentProject` auto-detection branch.
- `GetCurrentProject` no-config auto-detection branch.
- `DetectProject` handler.
- `AutoDetectAndSetProject` handler.

For each replacement, assign fields explicitly:

```go
candidate, err := DetectProjectCandidateFromGit(nil)
if err != nil {
	return mcp.NewToolResultError(fmt.Sprintf("Failed to detect project: %v", err)), nil
}
projectID := candidate.ProjectID
gitlabHost := candidate.Host
```

For JSON results from `detectProject` and `autoDetectAndSetProject`, include:

```go
"remoteName": candidate.RemoteName,
```

Keep existing `projectId` and `gitlabHost` fields unchanged.

- [ ] **Step 5: Run MCP project tool tests**

```bash
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./pkg/gitlab -run 'TestFindServerByHost|TestSetCurrentProjectHandler|TestGetCurrentProjectHandler|TestDetectProject|TestAutoDetect' -count=1
```

Expected: PASS. If exact test names differ, run:

```bash
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./pkg/gitlab -run 'Project|FindServerByHost' -count=1
```

- [ ] **Step 6: Commit**

```bash
git add pkg/gitlab/project_tools.go pkg/gitlab/project_tools_test.go
git commit -m "fix: report project server match ambiguity"
```

## Task 6: Docs, Snapshots, and Verification

**Files:**
- Modify: `docs/PROJECT_CONFIG.md`
- Modify: `docs/MULTI_SERVER_SETUP.md`
- Modify if snapshots change: `pkg/gitlab/__toolsnaps__/*.snap`

- [ ] **Step 1: Update project config docs**

In `docs/PROJECT_CONFIG.md`, replace the `project init` paragraph with:

```markdown
`project init` reads every Git remote, extracts GitLab candidates, and matches remote hosts against configured server hosts. If exactly one configured host matches, that server is recorded in `.gmcprc`. If multiple remotes or multiple configured servers could match, the command fails with a list of candidates; rerun with `--server <name>` or pass `projectId` explicitly.
```

In troubleshooting, add:

```markdown
**`project init` says "ambiguous GitLab remotes".** More than one remote matched the configured GitLab hosts and no single preferred remote could be selected. Rerun with `gitlab-mcp-server project init <projectId> --server <name>`, or remove/rename the extra remote.
```

- [ ] **Step 2: Update multi-server docs**

In `docs/MULTI_SERVER_SETUP.md`, update the project init note to mention ambiguity:

```markdown
`project init` matches Git remotes against configured server hosts. If multiple remotes or servers match, it fails closed and asks for `--server <name>` instead of guessing.
```

- [ ] **Step 3: Refresh snapshots if schema changed**

If Task 5 only adds output fields, snapshots should not change. If any tool descriptions or inputs changed, run:

```bash
UPDATE_TOOLSNAPS=true TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./pkg/gitlab -run 'Test.*Tool.*Snapshot|TestInitToolsets_RegisteredToolSchemaSnapshots' -count=1
```

Expected: snapshots update only when schemas changed. Review any snapshot diff manually before committing.

- [ ] **Step 4: Full verification**

Run:

```bash
go fmt ./cmd/project ./pkg/gitlab
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./pkg/gitlab ./cmd/project -count=1
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./... -count=1
TMPDIR=/var/tmp/gitlab-mcp-test-tmp make test-toolsnaps
git diff --exit-code -- go.mod go.sum
```

Expected:

- all tests pass
- tool snapshots pass
- no `go.mod` or `go.sum` changes

- [ ] **Step 5: Commit**

```bash
git add docs/PROJECT_CONFIG.md docs/MULTI_SERVER_SETUP.md pkg/gitlab/__toolsnaps__
git commit -m "docs: describe project detection ambiguity"
```

If snapshots did not change, use:

```bash
git add docs/PROJECT_CONFIG.md docs/MULTI_SERVER_SETUP.md
git commit -m "docs: describe project detection ambiguity"
```

## Verification Checklist

Run these before final handoff:

```bash
go fmt ./cmd/project ./pkg/gitlab
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./pkg/gitlab ./cmd/project -count=1
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./... -count=1
TMPDIR=/var/tmp/gitlab-mcp-test-tmp make test-toolsnaps
git status --short --branch
git log --oneline --decorate -8
```

Expected:

- project detection tests pass
- full test suite passes
- tool schema snapshots pass
- working tree is clean except for intentional branch commits

## Rollback Notes

If ambiguity handling breaks too many callers, keep `ParseGitRemoteCandidates` and `GitRemoteCandidate`, but temporarily make `DetectProjectFromGit` select the first candidate only after emitting an explicit error type from `DetectProjectCandidateFromGit`. Do not remove candidate parsing once added; future strict-by-default work depends on it.
