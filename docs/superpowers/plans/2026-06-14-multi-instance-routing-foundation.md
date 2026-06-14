# Multi-Instance Routing Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add explicit per-tool `server` routing, client metadata, and per-server read-only execution guards so multiple GitLab instances cannot be selected accidentally.

**Architecture:** Keep existing tool handlers mostly unchanged by injecting the requested server into `context.Context` before handlers run. Teach resolvers to honor that request before `.gmcprc` or default fallback, and teach the client pool to retain server metadata used by strict host checks and read-only guards.

**Tech Stack:** Go 1.24, `mark3labs/mcp-go`, `gitlab.com/gitlab-org/api/client-go`, existing `pkg/gitlab` resolver/toolset patterns, table-driven tests with `testify`.

---

## Scope

This is the first actionable slice from the glab parity roadmap. It implements the safety foundation only:

- client metadata in `ClientPool`
- explicit `server` argument on registered GitLab MCP tools
- resolver support for request-scoped server selection
- per-server read-only guard on write tools
- focused tests and docs alignment

This slice does not add TLS settings, CI job-token auth, output envelope normalization, or new GitLab domains.

## File Map

- Modify `pkg/gitlab/client_pool.go`: store client metadata and expose lookup helpers.
- Modify `pkg/gitlab/client_pool_test.go`: cover metadata, host matching, and read-only fields.
- Create `pkg/gitlab/request_context.go`: request-scoped server context helpers and MCP wrapper.
- Create `pkg/gitlab/request_context_test.go`: cover server argument parsing and schema injection.
- Modify `pkg/gitlab/client_resolver.go`: honor explicit `server` before legacy `.gmcprc` and default fallback.
- Modify `pkg/gitlab/client_resolver_test.go`: cover explicit server success and unknown server failure.
- Modify `pkg/gitlab/strict_resolver.go`: honor explicit `server` and use pool metadata for host verification.
- Modify `pkg/gitlab/strict_resolver_test.go`: cover strict explicit server behavior.
- Modify `pkg/gitlab/toolsets.go`: wrap all registered tools with `server`, and wrap write tools with a read-only guard.
- Modify `pkg/gitlab/toolsets_test.go`: assert schemas include `server` and read-only guard rejects writes.
- Modify `cmd/gitlab-mcp-server/main.go`: pass pool-backed read-only policy and strict resolver host metadata.
- Modify `docs/TOOLS.md` and `docs/MULTI_SERVER_SETUP.md`: describe implemented behavior accurately.

## Task 1: Add Client Metadata to ClientPool

**Files:**
- Modify: `pkg/gitlab/client_pool.go`
- Modify: `pkg/gitlab/client_pool_test.go`

- [ ] **Step 1: Write failing metadata tests**

Add these tests to `pkg/gitlab/client_pool_test.go`:

```go
func TestClientPool_AddClientWithMetadata(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	cp := NewClientPool(NewTokenStore(), logger)
	client := &gl.Client{}

	info := ClientInfo{
		Name:     "work",
		Host:     "https://gitlab.example.com",
		ReadOnly: true,
		Username: "inky",
	}

	require.NoError(t, cp.AddClientWithInfo(info, client))

	gotClient, err := cp.GetClient("work")
	require.NoError(t, err)
	assert.Same(t, client, gotClient)

	gotInfo, err := cp.GetClientInfo("work")
	require.NoError(t, err)
	assert.Equal(t, "work", gotInfo.Name)
	assert.Equal(t, "https://gitlab.example.com", gotInfo.Host)
	assert.True(t, gotInfo.ReadOnly)
	assert.Equal(t, "inky", gotInfo.Username)
}

func TestClientPool_FindClientByHost(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	cp := NewClientPool(NewTokenStore(), logger)

	require.NoError(t, cp.AddClientWithInfo(ClientInfo{
		Name: "work",
		Host: "https://gitlab.example.com/",
	}, &gl.Client{}))

	name, info, ok := cp.FindClientByHost("https://gitlab.example.com")
	require.True(t, ok)
	assert.Equal(t, "work", name)
	assert.Equal(t, "https://gitlab.example.com/", info.Host)

	_, _, ok = cp.FindClientByHost("https://gitlab.other.example")
	assert.False(t, ok)
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
mkdir -p /var/tmp/gitlab-mcp-test-tmp
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./pkg/gitlab -run 'TestClientPool_(AddClientWithMetadata|FindClientByHost)' -count=1
```

Expected: FAIL because `ClientInfo`, `AddClientWithInfo`, `GetClientInfo`, and `FindClientByHost` do not exist.

- [ ] **Step 3: Implement metadata storage**

In `pkg/gitlab/client_pool.go`, update `ClientPool` and add helpers:

```go
type ClientInfo struct {
	Name     string
	Host     string
	APIHost  string
	ReadOnly bool
	UserID   int64
	Username string
}

type ClientPool struct {
	clients map[string]*gl.Client
	info    map[string]ClientInfo
	store   *TokenStore
	logger  *log.Logger
	mu      sync.RWMutex
}
```

Initialize `info` in `NewClientPool`:

```go
return &ClientPool{
	clients: make(map[string]*gl.Client),
	info:    make(map[string]ClientInfo),
	store:   store,
	logger:  logger,
}
```

Add these methods:

```go
func (cp *ClientPool) AddClientWithInfo(info ClientInfo, client *gl.Client) error {
	if info.Name == "" {
		return fmt.Errorf("client name cannot be empty")
	}
	if client == nil {
		return fmt.Errorf("client cannot be nil")
	}
	if info.APIHost == "" {
		info.APIHost = info.Host
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.clients[info.Name] = client
	cp.info[info.Name] = info
	cp.logger.Infof("Added client '%s' to pool", info.Name)
	return nil
}

func (cp *ClientPool) GetClientInfo(name string) (ClientInfo, error) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	info, ok := cp.info[name]
	if !ok {
		return ClientInfo{}, fmt.Errorf("client '%s' not found in pool", name)
	}
	return info, nil
}

func (cp *ClientPool) ListClientInfo() []ClientInfo {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	infos := make([]ClientInfo, 0, len(cp.info))
	for _, info := range cp.info {
		infos = append(infos, info)
	}
	return infos
}

func (cp *ClientPool) FindClientByHost(host string) (string, ClientInfo, bool) {
	normalized := normalizePoolHost(host)
	if normalized == "" {
		return "", ClientInfo{}, false
	}

	cp.mu.RLock()
	defer cp.mu.RUnlock()

	for name, info := range cp.info {
		if normalizePoolHost(info.Host) == normalized || normalizePoolHost(info.APIHost) == normalized {
			return name, info, true
		}
	}
	return "", ClientInfo{}, false
}

func normalizePoolHost(host string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "/"))
}
```

Update `AddClient` to delegate:

```go
func (cp *ClientPool) AddClient(name string, client *gl.Client) error {
	return cp.AddClientWithInfo(ClientInfo{Name: name}, client)
}
```

Update `RemoveClient` to delete metadata:

```go
delete(cp.clients, name)
delete(cp.info, name)
```

Update `InitializeFromEnv` and `AddServerFromConfig` to call `AddClientWithInfo`. For `AddServerFromConfig`, pass:

```go
ClientInfo{
	Name:     server.Name,
	Host:     server.Host,
	APIHost:  server.Host,
	ReadOnly: server.ReadOnly,
	UserID:   server.UserID,
	Username: server.Username,
}
```

- [ ] **Step 4: Run metadata tests**

Run:

```bash
mkdir -p /var/tmp/gitlab-mcp-test-tmp
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./pkg/gitlab -run 'TestClientPool_(AddClientWithMetadata|FindClientByHost|AddClient|GetClient|RemoveClient)' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/gitlab/client_pool.go pkg/gitlab/client_pool_test.go
git commit -m "feat: track gitlab client metadata"
```

## Task 2: Add Request-Scoped Server Selection

**Files:**
- Create: `pkg/gitlab/request_context.go`
- Create: `pkg/gitlab/request_context_test.go`

- [ ] **Step 1: Write failing tests**

Create `pkg/gitlab/request_context_test.go`:

```go
package gitlab

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestedServerContext(t *testing.T) {
	ctx := WithRequestedServer(context.Background(), "work")
	server, ok := RequestedServerFromContext(ctx)
	require.True(t, ok)
	assert.Equal(t, "work", server)
}

func TestWithServerSelectionInjectsContext(t *testing.T) {
	tool := mcp.NewTool("sample")
	var gotServer string

	_, handler := WithServerSelection(tool, func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		gotServer, _ = RequestedServerFromContext(ctx)
		return mcp.NewToolResultText("ok"), nil
	})

	result, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{"server": "personal"},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "personal", gotServer)
}

func TestWithServerSelectionAddsSchemaParameter(t *testing.T) {
	tool := mcp.NewTool("sample")
	wrapped, _ := WithServerSelection(tool, nil)

	props := wrapped.InputSchema.Properties
	require.Contains(t, props, "server")
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
go test ./pkg/gitlab -run 'TestRequestedServerContext|TestWithServerSelection' -count=1
```

Expected: FAIL because request context helpers do not exist.

- [ ] **Step 3: Implement request context helpers**

Create `pkg/gitlab/request_context.go`:

```go
package gitlab

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type requestedServerContextKey struct{}

func WithRequestedServer(ctx context.Context, name string) context.Context {
	name = strings.TrimSpace(name)
	if name == "" {
		return ctx
	}
	return context.WithValue(ctx, requestedServerContextKey{}, name)
}

func RequestedServerFromContext(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(requestedServerContextKey{}).(string)
	if !ok || name == "" {
		return "", false
	}
	return name, true
}

func RequestedServerFromRequest(request mcp.CallToolRequest) string {
	raw, ok := request.GetArguments()["server"]
	if !ok || raw == nil {
		return ""
	}
	name, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(name)
}

func WithServerSelection(tool mcp.Tool, handler server.ToolHandlerFunc) (mcp.Tool, server.ToolHandlerFunc) {
	mcp.WithString("server",
		mcp.Description("Configured GitLab server name. Overrides .gmcprc/default routing for this call."),
	)(&tool)

	if handler == nil {
		return tool, nil
	}

	return tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if serverName := RequestedServerFromRequest(request); serverName != "" {
			ctx = WithRequestedServer(ctx, serverName)
		}
		return handler(ctx, request)
	}
}
```

- [ ] **Step 4: Run request context tests**

Run:

```bash
go test ./pkg/gitlab -run 'TestRequestedServerContext|TestWithServerSelection' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/gitlab/request_context.go pkg/gitlab/request_context_test.go
git commit -m "feat: add request scoped server selection"
```

## Task 3: Teach Resolvers to Honor Explicit Server

**Files:**
- Modify: `pkg/gitlab/client_resolver.go`
- Modify: `pkg/gitlab/client_resolver_test.go`
- Modify: `pkg/gitlab/strict_resolver.go`
- Modify: `pkg/gitlab/strict_resolver_test.go`

- [ ] **Step 1: Write failing resolver tests**

Add to `pkg/gitlab/client_resolver_test.go`:

```go
func TestClientResolver_Resolve_ExplicitServerFromContext(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	pool := NewClientPool(NewTokenStore(), logger)

	workClient := &gl.Client{}
	personalClient := &gl.Client{}
	require.NoError(t, pool.AddClient("work", workClient))
	require.NoError(t, pool.AddClient("personal", personalClient))

	resolver := NewClientResolver(pool, "work", logger)
	client, name, err := resolver.Resolve(WithRequestedServer(context.Background(), "personal"))
	require.NoError(t, err)
	assert.Same(t, personalClient, client)
	assert.Equal(t, "personal", name)
}

func TestClientResolver_Resolve_ExplicitUnknownServerErrors(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	pool := NewClientPool(NewTokenStore(), logger)
	require.NoError(t, pool.AddClient("work", &gl.Client{}))

	resolver := NewClientResolver(pool, "work", logger)
	_, _, err := resolver.Resolve(WithRequestedServer(context.Background(), "missing"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requested server")
	assert.Contains(t, err.Error(), "missing")
}
```

Add to `pkg/gitlab/strict_resolver_test.go`:

```go
func TestStrictResolver_ResolvesExplicitServerWithoutProjectConfig(t *testing.T) {
	srv := newFakeGitLab(t)
	defer srv.Close()

	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	require.NoError(t, os.Chdir(dir))

	pool := NewClientPool(NewTokenStore(), logrus.New())
	client, err := gl.NewClient("x", gl.WithBaseURL(srv.URL))
	require.NoError(t, err)
	require.NoError(t, pool.AddClientWithInfo(ClientInfo{Name: "work", Host: srv.URL}, client))

	r := NewStrictResolver(pool, nil, logrus.New())
	got, name, err := r.Resolve(WithRequestedServer(context.Background(), "work"))
	require.NoError(t, err)
	assert.Equal(t, "work", name)
	assert.NotNil(t, got)
}
```

- [ ] **Step 2: Run resolver tests to verify failure**

Run:

```bash
mkdir -p /var/tmp/gitlab-mcp-test-tmp
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./pkg/gitlab -run 'Test(ClientResolver_Resolve_Explicit|StrictResolver_ResolvesExplicit)' -count=1
```

Expected: FAIL because resolvers do not read request-scoped server selection.

- [ ] **Step 3: Update legacy resolver**

At the top of `ClientResolver.Resolve`, before reading `.gmcprc`, add:

```go
if requestedServer, ok := RequestedServerFromContext(ctx); ok {
	client, err := cr.pool.GetClient(requestedServer)
	if err != nil {
		return nil, "", fmt.Errorf("requested server %q is not configured: %w", requestedServer, err)
	}
	cr.logger.Debugf("Using explicitly requested client '%s'", requestedServer)
	return client, requestedServer, nil
}
```

Add `fmt` to imports in `pkg/gitlab/client_resolver.go`.

- [ ] **Step 4: Update strict resolver**

In `pkg/gitlab/strict_resolver.go`, add a helper:

```go
func (r *StrictResolver) resolveNamed(ctx context.Context, name string) (*gl.Client, string, error) {
	client, err := r.pool.GetClient(name)
	if err != nil {
		configured := r.configuredServerNames()
		return nil, "", fmt.Errorf("strict resolver: server %q not configured; configured servers: %s",
			name, strings.Join(configured, ", "))
	}
	if err := r.verifyHost(ctx, name, client); err != nil {
		return nil, "", err
	}
	return client, name, nil
}

func (r *StrictResolver) configuredServerNames() []string {
	infos := r.pool.ListClientInfo()
	names := make([]string, 0, len(infos))
	for _, info := range infos {
		names = append(names, info.Name)
	}
	if len(names) == 0 {
		for n := range r.serverHosts {
			names = append(names, n)
		}
	}
	return names
}
```

At the top of `StrictResolver.Resolve`, add:

```go
if requestedServer, ok := RequestedServerFromContext(ctx); ok {
	return r.resolveNamed(ctx, requestedServer)
}
```

Change existing `.gmcprc` server resolution to call:

```go
return r.resolveNamed(ctx, cfg.Server)
```

Update `verifyHost` so it reads host metadata from the pool before falling back to `serverHosts`:

```go
wantHost := r.serverHosts[name]
if info, err := r.pool.GetClientInfo(name); err == nil && info.APIHost != "" {
	wantHost = info.APIHost
} else if err == nil && info.Host != "" {
	wantHost = info.Host
}
```

- [ ] **Step 5: Run resolver tests**

Run:

```bash
mkdir -p /var/tmp/gitlab-mcp-test-tmp
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./pkg/gitlab -run 'Test(ClientResolver_Resolve_Explicit|StrictResolver_)' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/gitlab/client_resolver.go pkg/gitlab/client_resolver_test.go pkg/gitlab/strict_resolver.go pkg/gitlab/strict_resolver_test.go
git commit -m "feat: resolve explicit gitlab server requests"
```

## Task 4: Add Server Parameter and Read-Only Guard to Tool Registration

**Files:**
- Modify: `pkg/gitlab/toolsets.go`
- Modify: `pkg/gitlab/toolsets_test.go`

- [ ] **Step 1: Write failing toolset tests**

Add to `pkg/gitlab/toolsets_test.go`:

```go
func TestInitToolsets_AddsServerParameter(t *testing.T) {
	tg, err := InitToolsets([]string{"projects"}, false, mockGetClientFn, nil, NewTokenStore(), nil, false, nil)
	require.NoError(t, err)

	tools := tg.Toolsets["projects"].Tools()
	require.NotEmpty(t, tools)

	for _, tool := range tools {
		assert.Contains(t, tool.Tool.InputSchema.Properties, "server", "tool %s should accept server", tool.Tool.Name)
	}
}

func TestInitToolsets_WriteGuardRejectsReadOnlyServer(t *testing.T) {
	policy := func(context.Context) (ServerPolicy, error) {
		return ServerPolicy{Name: "mirror", ReadOnly: true}, nil
	}

	tg, err := InitToolsets([]string{"issues"}, false, mockGetClientFn, nil, NewTokenStore(), nil, false, policy)
	require.NoError(t, err)

	var createIssue server.ServerTool
	for _, tool := range tg.Toolsets["issues"].Tools() {
		if tool.Tool.Name == "createIssue" {
			createIssue = tool
			break
		}
	}
	require.NotNil(t, createIssue.Handler)

	result, err := createIssue.Handler(context.Background(), mcp.CallToolRequest{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "read-only")
}
```

Add imports for `github.com/mark3labs/mcp-go/mcp` and `github.com/mark3labs/mcp-go/server`.

- [ ] **Step 2: Run toolset tests to verify failure**

Run:

```bash
go test ./pkg/gitlab -run 'TestInitToolsets_(AddsServerParameter|WriteGuardRejectsReadOnlyServer)' -count=1
```

Expected: FAIL because `InitToolsets` has no server policy parameter and tools are not wrapped.

- [ ] **Step 3: Implement server policy and wrappers**

In `pkg/gitlab/toolsets.go`, add:

```go
type ServerPolicy struct {
	Name     string
	ReadOnly bool
}

type ServerPolicyFn func(context.Context) (ServerPolicy, error)

func newGitLabReadTool(tool mcp.Tool, handler server.ToolHandlerFunc) server.ServerTool {
	tool, handler = WithServerSelection(tool, handler)
	return toolsets.NewServerTool(tool, handler)
}

func newGitLabWriteTool(policy ServerPolicyFn, tool mcp.Tool, handler server.ToolHandlerFunc) server.ServerTool {
	tool, handler = WithServerSelection(tool, handler)
	if handler == nil || policy == nil {
		return toolsets.NewServerTool(tool, handler)
	}
	return toolsets.NewServerTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		p, err := policy(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to resolve server policy: %v", err)), nil
		}
		if p.ReadOnly {
			serverName := p.Name
			if serverName == "" {
				serverName = "selected"
			}
			return mcp.NewToolResultError(fmt.Sprintf("server %q is configured read-only; write tool %q is blocked", serverName, tool.Name)), nil
		}
		return handler(ctx, request)
	})
}
```

Add `fmt` and `github.com/mark3labs/mcp-go/mcp` imports if missing.

Change the `InitToolsets` signature to:

```go
func InitToolsets(
	enabledToolsets []string,
	readOnly bool,
	getClient GetClientFn,
	logger *log.Logger,
	tokenStore *TokenStore,
	translations map[string]string,
	dynamicMode bool,
	serverPolicy ServerPolicyFn,
) (*toolsets.ToolsetGroup, error) {
```

Replace `toolsets.NewServerTool(...)` in read tool registrations with `newGitLabReadTool(...)`.

Replace `toolsets.NewServerTool(...)` in write tool registrations with `newGitLabWriteTool(serverPolicy, ...)`.

- [ ] **Step 4: Update InitToolsets call sites**

Update test call sites by adding `nil` as the final argument.

Update `cmd/gitlab-mcp-server/main.go` call site after creating `resolverFn`:

```go
serverPolicy := func(ctx context.Context) (gitlab.ServerPolicy, error) {
	client, name, err := resolveClientForPolicy(ctx, clientPool, defaultServer)
	if err != nil {
		return gitlab.ServerPolicy{}, err
	}
	_ = client
	info, err := clientPool.GetClientInfo(name)
	if err != nil {
		return gitlab.ServerPolicy{Name: name}, nil
	}
	return gitlab.ServerPolicy{Name: name, ReadOnly: info.ReadOnly}, nil
}
```

Add a private helper in `cmd/gitlab-mcp-server/main.go`:

```go
func resolveClientForPolicy(ctx context.Context, pool *gitlab.ClientPool, defaultServer string) (*gl.Client, string, error) {
	if requestedServer, ok := gitlab.RequestedServerFromContext(ctx); ok {
		client, err := pool.GetClient(requestedServer)
		return client, requestedServer, err
	}
	if defaultServer != "" {
		client, err := pool.GetClient(defaultServer)
		if err == nil {
			return client, defaultServer, nil
		}
	}
	return pool.GetDefaultClient()
}
```

Change the `InitToolsets` call:

```go
toolsetGroup, err := gitlab.InitToolsets(enabledToolsets, readOnly, resolverFn, logger, tokenStore, t, dynamicToolsets, serverPolicy)
```

- [ ] **Step 5: Run toolset tests**

Run:

```bash
go test ./pkg/gitlab -run 'TestInitToolsets' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/gitlab/toolsets.go pkg/gitlab/toolsets_test.go cmd/gitlab-mcp-server/main.go
git commit -m "feat: guard write tools for read-only servers"
```

## Task 5: Align Docs and Snapshots

**Files:**
- Modify: `docs/TOOLS.md`
- Modify: `docs/MULTI_SERVER_SETUP.md`
- Modify: `pkg/gitlab/__toolsnaps__/*.snap`

- [ ] **Step 1: Update docs text**

In `docs/TOOLS.md`, replace the opening sentence:

```markdown
Every tool accepts a `server` argument (optional) to pick which configured GitLab instance to use.
```

with:

```markdown
Every GitLab tool accepts a `server` argument (optional) to pick which configured GitLab instance to use. When omitted, the resolver uses `.gmcprc` and then the configured default in legacy mode. Strict mode rejects unresolved calls.
```

In `docs/MULTI_SERVER_SETUP.md`, ensure the “Read-only per server” section includes:

```markdown
Per-server read-only mode is enforced when a write tool runs. This means a write tool may still appear in the tool list, but calls targeting a read-only server fail before making a GitLab API request.
```

- [ ] **Step 2: Regenerate tool snapshots**

Run:

```bash
go test ./internal/toolsnaps -count=1
```

Expected: PASS and snapshot files update to include the `server` parameter where applicable.

- [ ] **Step 3: Run focused verification**

Run:

```bash
mkdir -p /var/tmp/gitlab-mcp-test-tmp
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./pkg/gitlab ./pkg/toolsets ./cmd/project ./cmd/config ./internal/toolsnaps -count=1
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add docs/TOOLS.md docs/MULTI_SERVER_SETUP.md pkg/gitlab/__toolsnaps__ internal/toolsnaps
git commit -m "docs: document explicit gitlab server routing"
```

## Task 6: Full Verification

**Files:**
- No source edits expected.

- [ ] **Step 1: Run full unit suite**

Run:

```bash
mkdir -p /var/tmp/gitlab-mcp-test-tmp
TMPDIR=/var/tmp/gitlab-mcp-test-tmp go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 2: Run formatting**

Run:

```bash
go fmt ./cmd/... ./pkg/... ./internal/...
```

Expected: command exits 0. If it changes files, commit those changes with the relevant previous task or a final formatting commit.

- [ ] **Step 3: Run module tidy check**

Run:

```bash
go mod tidy
git diff --exit-code -- go.mod go.sum
```

Expected: `go mod tidy` exits 0 and `git diff --exit-code -- go.mod go.sum` exits 0.

- [ ] **Step 4: Check final branch state**

Run:

```bash
git status --short --branch
git log --oneline --decorate -8
```

Expected: working tree clean and branch contains focused commits for metadata, request routing, resolver behavior, read-only guards, docs, and verification-only formatting if needed.

## Self-Review

Spec coverage:

- Client metadata: Task 1.
- Explicit `server` routing: Tasks 2, 3, and 4.
- Per-server read-only enforcement: Task 4.
- Docs alignment: Task 5.
- Verification: Task 6.

Scope intentionally excluded from this slice:

- TLS/self-managed settings.
- CI job-token auth.
- Output envelope normalization.
- New glab parity domains.

No placeholders remain in this plan. Function names and type names are consistent across tasks.
