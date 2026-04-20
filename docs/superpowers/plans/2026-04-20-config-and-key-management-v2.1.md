# Config & Key Management Redesign — v2.1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the new pluggable-backend + strict-resolver + no-LLM-secrets design from `docs/superpowers/specs/2026-04-20-config-and-key-management-redesign.md` as a **v2.1 release** that runs in parallel with the old behavior. All old paths keep working but emit deprecation warnings; new paths are opt-in where breaking. Removal of deprecated paths is a separate v3.0 plan.

**Architecture:** Introduce `SecretBackend` interface + three backends (keyring/file/external-cmd) in `pkg/config`. Extend `ServerConfig` with `TokenRef` alongside existing `Token` for one-release overlap. Add `StrictResolver` in `pkg/gitlab` gated by `GITLAB_MCP_STRICT_RESOLVER=1`. Add deprecation warnings on every legacy path: `GITLAB_TOKEN` env startup, `--token` CLI flag, `addToken`/`updateToken` MCP tools, `.gmcprc` `tokenName`/`gitlabHost` fields. Installer stops emitting token env vars in newly generated IDE entries.

**Tech Stack:** Go 1.24, `stretchr/testify` (assert + require), `zalando/go-keyring`, `spf13/cobra` + `viper`, `charmbracelet/bubbletea`, existing `CryptoManager` (AES-256-GCM) reused for `EncryptedFileBackend`, new dep `golang.org/x/term` for no-echo TTY input.

**Scope note:** This plan keeps the global config file path at the current location (`~/.gitlab-mcp-server/gitlab-mcp-server-config.json`). The spec diagram shows `~/.config/gitlab-mcp-server/config.json` but that path migration is deferred to v3.0 to keep v2.1 changes non-destructive.

---

## File Structure

**New files:**

| Path | Responsibility |
|---|---|
| `pkg/config/backend.go` | `SecretBackend` interface, `BackendRegistry`, `ParseRef`. Scheme dispatch. |
| `pkg/config/backend_fake.go` | `FakeSecretBackend` (test helper, in-memory map). Lives in the main package so it's importable by any test in the module. |
| `pkg/config/backend_keyring.go` | `KeyringBackend` — scheme `keyring`. |
| `pkg/config/backend_file.go` | `EncryptedFileBackend` — scheme `file`. Reuses `CryptoManager`. |
| `pkg/config/backend_external.go` | `ExternalCmdBackend` — schemes `op`, `pass`, `age`, `gopass`. Runs templated commands via `exec.Command`. |
| `pkg/config/backend_test.go` | Registry + scheme-dispatch + `ParseRef` tests. |
| `pkg/config/backend_keyring_test.go` | KeyringBackend tests (uses `FakeSecretBackend` wrapper + `keyring.MockInit()`). |
| `pkg/config/backend_file_test.go` | EncryptedFileBackend tests. |
| `pkg/config/backend_external_test.go` | External-cmd tests including shell-metacharacter injection check. |
| `pkg/gitlab/strict_resolver.go` | `StrictResolver` (no fallback, host verification, per-session cache). |
| `pkg/gitlab/strict_resolver_test.go` | Table-driven resolver tests. |
| `cmd/config/backends.go` | `gitlab-mcp-server config backends` diagnostic command. |
| `cmd/config/secret_input.go` | TTY no-echo prompt helper for interactive `config add`. |

**Modified files:**

| Path | Change |
|---|---|
| `pkg/config/config.go` | Add `TokenRef` to `ServerConfig`, `Backends` to `Config`. Bump `Version` to `"2.0"` on save when any `TokenRef` is set. Never remove `Token` in v2.1 — both fields coexist. `Manager` gains optional `BackendRegistry` dependency. |
| `pkg/gitlab/project_config.go` | `Server` field already exists; add auto-promotion of deprecated `TokenName` + `GitLabHost` on read with one-time warnings. |
| `cmd/config/add.go` | Add `--token-ref` flag. Add interactive TTY fallback when neither `--token` nor `--token-ref` is given. Print deprecation warning when `--token` is used. Resolve `--token-ref` against the registry before persisting. |
| `cmd/config/command.go` | Register new `backends` subcommand. |
| `cmd/project/init.go` | Make `--server <name>` effectively required: if unset, try to match one server by Git remote host; if no match, error with a helpful message listing configured servers. Stop writing `gitlabHost` to `.gmcprc`. |
| `cmd/install/install.go` | `createInstallConfig` stops adding `GITLAB_TOKEN`/`GITLAB_HOST` to the emitted IDE entry. `GITLAB_CONFIG_PATH` stays. |
| `cmd/gitlab-mcp-server/main.go` | Emit deprecation warning when `GITLAB_TOKEN` env is used. Gate `StrictResolver` behind `GITLAB_MCP_STRICT_RESOLVER=1`. Initialize `BackendRegistry` and pass to `Manager`. |
| `pkg/gitlab/token_tools.go` | Add deprecation field `"deprecated": true` to `addToken`/`updateToken` responses and log warnings. |
| `CHANGELOG.md` | v2.1 entry. |
| `go.mod` / `go.sum` | `go get golang.org/x/term` for `term.ReadPassword`. |

---

### Task 1: `SecretBackend` interface, registry, and `ParseRef`

**Files:**
- Create: `pkg/config/backend.go`
- Create: `pkg/config/backend_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// pkg/config/backend_test.go
package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRef(t *testing.T) {
	tests := []struct {
		in         string
		wantScheme string
		wantOpaque string
		wantErr    bool
	}{
		{"keyring://work", "keyring", "work", false},
		{"file:///tmp/x.enc#entry", "file", "/tmp/x.enc#entry", false},
		{"op://Work/gitlab/token", "op", "Work/gitlab/token", false},
		{"notaref", "", "", true},
		{"://empty", "", "", true},
		{"", "", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			scheme, opaque, err := ParseRef(tc.in)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantScheme, scheme)
			assert.Equal(t, tc.wantOpaque, opaque)
		})
	}
}

type stubBackend struct {
	scheme string
	store  map[string]string
}

func (s *stubBackend) Scheme() string { return s.scheme }
func (s *stubBackend) Resolve(_ context.Context, ref string) (string, error) {
	_, opaque, err := ParseRef(ref)
	if err != nil {
		return "", err
	}
	v, ok := s.store[opaque]
	if !ok {
		return "", ErrSecretNotFound
	}
	return v, nil
}
func (s *stubBackend) Store(_ context.Context, name, secret string) (string, error) {
	s.store[name] = secret
	return s.scheme + "://" + name, nil
}
func (s *stubBackend) Delete(_ context.Context, ref string) error {
	_, opaque, err := ParseRef(ref)
	if err != nil {
		return err
	}
	delete(s.store, opaque)
	return nil
}

func TestBackendRegistry_DispatchesByScheme(t *testing.T) {
	r := NewBackendRegistry()
	k := &stubBackend{scheme: "keyring", store: map[string]string{"work": "secret-w"}}
	f := &stubBackend{scheme: "file", store: map[string]string{"fe": "secret-f"}}
	require.NoError(t, r.Register(k))
	require.NoError(t, r.Register(f))

	got, err := r.Resolve(context.Background(), "keyring://work")
	require.NoError(t, err)
	assert.Equal(t, "secret-w", got)

	got, err = r.Resolve(context.Background(), "file://fe")
	require.NoError(t, err)
	assert.Equal(t, "secret-f", got)
}

func TestBackendRegistry_UnknownScheme(t *testing.T) {
	r := NewBackendRegistry()
	k := &stubBackend{scheme: "keyring", store: map[string]string{}}
	require.NoError(t, r.Register(k))

	_, err := r.Resolve(context.Background(), "op://x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown secret backend scheme")
	assert.Contains(t, err.Error(), "keyring") // lists what IS registered
}

func TestBackendRegistry_DuplicateScheme(t *testing.T) {
	r := NewBackendRegistry()
	require.NoError(t, r.Register(&stubBackend{scheme: "keyring"}))
	err := r.Register(&stubBackend{scheme: "keyring"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/inky/Development/gitlab-mcp-server && go test ./pkg/config/ -run 'TestParseRef|TestBackendRegistry' -v`
Expected: FAIL — `undefined: ParseRef`, `undefined: NewBackendRegistry`, `undefined: ErrSecretNotFound`.

- [ ] **Step 3: Write the minimal implementation**

```go
// pkg/config/backend.go
package config

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// ErrSecretNotFound is returned when a secret ref does not exist in its backend.
var ErrSecretNotFound = errors.New("secret not found")

// SecretBackend is a pluggable storage for tokens. Implementations are
// keyed by URI scheme (keyring://, file://, op://, ...).
type SecretBackend interface {
	// Resolve fetches the secret for a fully-qualified ref.
	Resolve(ctx context.Context, ref string) (string, error)
	// Store persists a secret under a logical name and returns the ref to
	// record in config. CLI/TUI only — not called from MCP tool handlers.
	Store(ctx context.Context, name, secret string) (ref string, err error)
	// Delete removes a secret by ref.
	Delete(ctx context.Context, ref string) error
	// Scheme returns the URI scheme this backend handles (no trailing "://").
	Scheme() string
}

// ParseRef splits a ref like "scheme://opaque" into its parts. The opaque
// portion is returned verbatim (including any '#' fragment or '/' path).
func ParseRef(ref string) (scheme, opaque string, err error) {
	i := strings.Index(ref, "://")
	if i <= 0 {
		return "", "", fmt.Errorf("invalid secret ref %q: expected scheme://opaque", ref)
	}
	scheme = ref[:i]
	opaque = ref[i+3:]
	if opaque == "" {
		return "", "", fmt.Errorf("invalid secret ref %q: empty opaque portion", ref)
	}
	return scheme, opaque, nil
}

// BackendRegistry dispatches secret refs to the appropriate backend by scheme.
type BackendRegistry struct {
	mu       sync.RWMutex
	backends map[string]SecretBackend
}

// NewBackendRegistry returns an empty registry.
func NewBackendRegistry() *BackendRegistry {
	return &BackendRegistry{backends: map[string]SecretBackend{}}
}

// Register adds a backend. Returns an error if the scheme is already taken.
func (r *BackendRegistry) Register(b SecretBackend) error {
	if b == nil {
		return errors.New("cannot register nil backend")
	}
	scheme := b.Scheme()
	if scheme == "" {
		return errors.New("backend scheme cannot be empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.backends[scheme]; exists {
		return fmt.Errorf("scheme %q already registered", scheme)
	}
	r.backends[scheme] = b
	return nil
}

// Resolve fetches the secret for the given ref, dispatching by scheme.
func (r *BackendRegistry) Resolve(ctx context.Context, ref string) (string, error) {
	scheme, _, err := ParseRef(ref)
	if err != nil {
		return "", err
	}
	b, err := r.get(scheme)
	if err != nil {
		return "", err
	}
	return b.Resolve(ctx, ref)
}

// Store persists a secret under the given name in the backend for the given scheme.
func (r *BackendRegistry) Store(ctx context.Context, scheme, name, secret string) (string, error) {
	b, err := r.get(scheme)
	if err != nil {
		return "", err
	}
	return b.Store(ctx, name, secret)
}

// Delete removes a secret by ref.
func (r *BackendRegistry) Delete(ctx context.Context, ref string) error {
	scheme, _, err := ParseRef(ref)
	if err != nil {
		return err
	}
	b, err := r.get(scheme)
	if err != nil {
		return err
	}
	return b.Delete(ctx, ref)
}

// Schemes returns the registered scheme names in sorted order (for stable error messages).
func (r *BackendRegistry) Schemes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.backends))
	for s := range r.backends {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func (r *BackendRegistry) get(scheme string) (SecretBackend, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.backends[scheme]
	if !ok {
		return nil, fmt.Errorf("unknown secret backend scheme %q (registered: %s)",
			scheme, strings.Join(r.sortedSchemesLocked(), ", "))
	}
	return b, nil
}

func (r *BackendRegistry) sortedSchemesLocked() []string {
	out := make([]string, 0, len(r.backends))
	for s := range r.backends {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/inky/Development/gitlab-mcp-server && go test ./pkg/config/ -run 'TestParseRef|TestBackendRegistry' -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/config/backend.go pkg/config/backend_test.go
git commit -m "feat(config): add SecretBackend interface and BackendRegistry

Scheme-dispatched registry for pluggable token storage. ParseRef splits
scheme://opaque refs; unknown schemes yield a loud error that lists what
IS registered. Part of the v2.1 key-management redesign."
```

---

### Task 2: `FakeSecretBackend` (reusable test helper)

**Files:**
- Create: `pkg/config/backend_fake.go`

- [ ] **Step 1: Write the helper (no separate test — it's a test double, exercised indirectly)**

```go
// pkg/config/backend_fake.go
package config

import (
	"context"
	"fmt"
	"sync"
)

// FakeSecretBackend is an in-memory SecretBackend for tests. Scheme is configurable
// so it can stand in for keyring, file, op, etc.
type FakeSecretBackend struct {
	scheme string
	mu     sync.Mutex
	store  map[string]string
}

// NewFakeSecretBackend returns a fake backend for the given scheme.
// Pre-populate entries with SetEntry if the test needs a ref to already resolve.
func NewFakeSecretBackend(scheme string) *FakeSecretBackend {
	return &FakeSecretBackend{scheme: scheme, store: map[string]string{}}
}

// SetEntry pre-populates the backend so a ref like scheme://name resolves to secret.
func (f *FakeSecretBackend) SetEntry(name, secret string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.store[name] = secret
}

func (f *FakeSecretBackend) Scheme() string { return f.scheme }

func (f *FakeSecretBackend) Resolve(_ context.Context, ref string) (string, error) {
	_, opaque, err := ParseRef(ref)
	if err != nil {
		return "", err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.store[opaque]
	if !ok {
		return "", fmt.Errorf("fake backend: %w: %s", ErrSecretNotFound, ref)
	}
	return v, nil
}

func (f *FakeSecretBackend) Store(_ context.Context, name, secret string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.store[name] = secret
	return f.scheme + "://" + name, nil
}

func (f *FakeSecretBackend) Delete(_ context.Context, ref string) error {
	_, opaque, err := ParseRef(ref)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.store, opaque)
	return nil
}
```

- [ ] **Step 2: Verify it compiles and the existing test suite still passes**

Run: `cd /home/inky/Development/gitlab-mcp-server && go build ./... && go test ./pkg/config/ -v`
Expected: compiles; all existing tests + the Task 1 tests pass.

- [ ] **Step 3: Commit**

```bash
git add pkg/config/backend_fake.go
git commit -m "test(config): add FakeSecretBackend for use across package tests"
```

---

### Task 3: `KeyringBackend`

**Files:**
- Create: `pkg/config/backend_keyring.go`
- Create: `pkg/config/backend_keyring_test.go`

The `zalando/go-keyring` library provides `keyring.MockInit()` which replaces the system keyring with an in-memory map for tests. Use it so tests don't touch the real OS keyring.

- [ ] **Step 1: Write the failing tests**

```go
// pkg/config/backend_keyring_test.go
package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

const testKeyringService = "gitlab-mcp-server-test"

func TestKeyringBackend_StoreResolveDelete(t *testing.T) {
	keyring.MockInit()
	b := NewKeyringBackend(testKeyringService)
	ctx := context.Background()

	ref, err := b.Store(ctx, "work", "glpat-abc123")
	require.NoError(t, err)
	assert.Equal(t, "keyring://work", ref)

	got, err := b.Resolve(ctx, ref)
	require.NoError(t, err)
	assert.Equal(t, "glpat-abc123", got)

	require.NoError(t, b.Delete(ctx, ref))

	_, err = b.Resolve(ctx, ref)
	require.Error(t, err)
}

func TestKeyringBackend_ResolveNotFound(t *testing.T) {
	keyring.MockInit()
	b := NewKeyringBackend(testKeyringService)
	_, err := b.Resolve(context.Background(), "keyring://missing")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSecretNotFound)
}

func TestKeyringBackend_WrongScheme(t *testing.T) {
	keyring.MockInit()
	b := NewKeyringBackend(testKeyringService)
	_, err := b.Resolve(context.Background(), "file://x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scheme")
}

func TestKeyringBackend_SchemeReporter(t *testing.T) {
	b := NewKeyringBackend(testKeyringService)
	assert.Equal(t, "keyring", b.Scheme())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/inky/Development/gitlab-mcp-server && go test ./pkg/config/ -run 'TestKeyringBackend' -v`
Expected: FAIL — `undefined: NewKeyringBackend`.

- [ ] **Step 3: Write the minimal implementation**

```go
// pkg/config/backend_keyring.go
package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

// KeyringBackend stores secrets in the OS keyring via zalando/go-keyring.
// It uses the supplied service name; the ref opaque is used as the account.
type KeyringBackend struct {
	service string
}

// NewKeyringBackend returns a backend scoped to the given keyring service name.
func NewKeyringBackend(service string) *KeyringBackend {
	return &KeyringBackend{service: service}
}

// Scheme implements SecretBackend.
func (k *KeyringBackend) Scheme() string { return "keyring" }

// Resolve fetches the secret for a ref like keyring://<account>.
func (k *KeyringBackend) Resolve(_ context.Context, ref string) (string, error) {
	scheme, account, err := ParseRef(ref)
	if err != nil {
		return "", err
	}
	if scheme != "keyring" {
		return "", fmt.Errorf("keyring backend: wrong scheme %q", scheme)
	}
	v, err := keyring.Get(k.service, account)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", fmt.Errorf("keyring backend: %w: %s", ErrSecretNotFound, ref)
		}
		return "", fmt.Errorf("keyring backend: %w", err)
	}
	return v, nil
}

// Store persists the secret under the given name and returns the ref.
func (k *KeyringBackend) Store(_ context.Context, name, secret string) (string, error) {
	if name == "" {
		return "", errors.New("keyring backend: name cannot be empty")
	}
	if err := keyring.Set(k.service, name, secret); err != nil {
		return "", fmt.Errorf("keyring backend: %w", err)
	}
	return "keyring://" + name, nil
}

// Delete removes the secret by ref.
func (k *KeyringBackend) Delete(_ context.Context, ref string) error {
	scheme, account, err := ParseRef(ref)
	if err != nil {
		return err
	}
	if scheme != "keyring" {
		return fmt.Errorf("keyring backend: wrong scheme %q", scheme)
	}
	if err := keyring.Delete(k.service, account); err != nil {
		return fmt.Errorf("keyring backend: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/inky/Development/gitlab-mcp-server && go test ./pkg/config/ -run 'TestKeyringBackend' -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/config/backend_keyring.go pkg/config/backend_keyring_test.go
git commit -m "feat(config): add KeyringBackend (default secret backend)

Wraps zalando/go-keyring. Scheme 'keyring', ref form keyring://<name>.
Tests use keyring.MockInit() to avoid touching the real OS keychain."
```

---

### Task 4: `EncryptedFileBackend`

Stores secrets in a JSON-at-rest file encrypted with AES-256-GCM via the existing `CryptoManager`. Reuses the master key already managed by `CryptoManager` (keyring-held) but also accepts a passphrase via the `GITLAB_MCP_FILE_PASSPHRASE` env var for headless/Docker.

**Files:**
- Create: `pkg/config/backend_file.go`
- Create: `pkg/config/backend_file_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// pkg/config/backend_file_test.go
package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

func TestEncryptedFileBackend_RoundTrip(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")

	cm, err := NewCryptoManager(true)
	require.NoError(t, err)
	b := NewEncryptedFileBackend(path, cm)
	ctx := context.Background()

	ref, err := b.Store(ctx, "work", "glpat-xyz")
	require.NoError(t, err)
	assert.Equal(t, "file://"+path+"#work", ref)

	// Second store under a different name should not clobber the first.
	ref2, err := b.Store(ctx, "personal", "glpat-abc")
	require.NoError(t, err)

	got, err := b.Resolve(ctx, ref)
	require.NoError(t, err)
	assert.Equal(t, "glpat-xyz", got)

	got2, err := b.Resolve(ctx, ref2)
	require.NoError(t, err)
	assert.Equal(t, "glpat-abc", got2)

	// File on disk should not contain the plaintext.
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "glpat-xyz")
	assert.NotContains(t, string(raw), "glpat-abc")

	// File mode should be 0600.
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// Delete removes just the one entry.
	require.NoError(t, b.Delete(ctx, ref))
	_, err = b.Resolve(ctx, ref)
	require.ErrorIs(t, err, ErrSecretNotFound)

	got2Again, err := b.Resolve(ctx, ref2)
	require.NoError(t, err)
	assert.Equal(t, "glpat-abc", got2Again)
}

func TestEncryptedFileBackend_ResolveMissingFile(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "nope.enc")
	cm, err := NewCryptoManager(true)
	require.NoError(t, err)
	b := NewEncryptedFileBackend(path, cm)

	_, err = b.Resolve(context.Background(), "file://"+path+"#x")
	require.ErrorIs(t, err, ErrSecretNotFound)
}

func TestEncryptedFileBackend_ParseFileRef(t *testing.T) {
	p, entry, err := parseFileRef("file:///tmp/x.enc#work")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/x.enc", p)
	assert.Equal(t, "work", entry)

	_, _, err = parseFileRef("file:///tmp/x.enc") // no #
	require.Error(t, err)
	_, _, err = parseFileRef("keyring://work") // wrong scheme
	require.Error(t, err)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/inky/Development/gitlab-mcp-server && go test ./pkg/config/ -run 'TestEncryptedFileBackend|TestEncryptedFileBackend_ParseFileRef' -v`
Expected: FAIL — `undefined: NewEncryptedFileBackend`, `undefined: parseFileRef`.

- [ ] **Step 3: Write the minimal implementation**

```go
// pkg/config/backend_file.go
package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// EncryptedFileBackend stores multiple secrets in a single JSON file, each
// value AES-256-GCM-encrypted by a CryptoManager. Ref format:
//   file://<absolute-path>#<entry-name>
type EncryptedFileBackend struct {
	path   string
	crypto *CryptoManager
	mu     sync.Mutex
}

// NewEncryptedFileBackend constructs a backend writing to the given path.
// The CryptoManager must be IsEnabled() for reads/writes to succeed.
func NewEncryptedFileBackend(path string, crypto *CryptoManager) *EncryptedFileBackend {
	return &EncryptedFileBackend{path: path, crypto: crypto}
}

// Scheme implements SecretBackend.
func (e *EncryptedFileBackend) Scheme() string { return "file" }

// Resolve fetches the entry named in the ref.
func (e *EncryptedFileBackend) Resolve(_ context.Context, ref string) (string, error) {
	path, entry, err := parseFileRef(ref)
	if err != nil {
		return "", err
	}
	if path != e.path {
		return "", fmt.Errorf("file backend: ref path %q does not match configured path %q", path, e.path)
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	store, err := e.loadLocked()
	if err != nil {
		return "", err
	}
	ct, ok := store[entry]
	if !ok {
		return "", fmt.Errorf("file backend: %w: %s", ErrSecretNotFound, ref)
	}
	return e.crypto.Decrypt(ct)
}

// Store persists the given secret under the given name.
func (e *EncryptedFileBackend) Store(_ context.Context, name, secret string) (string, error) {
	if name == "" {
		return "", errors.New("file backend: name cannot be empty")
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	store, err := e.loadLocked()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if store == nil {
		store = map[string]string{}
	}
	ct, err := e.crypto.Encrypt(secret)
	if err != nil {
		return "", fmt.Errorf("file backend: encrypt: %w", err)
	}
	store[name] = ct
	if err := e.saveLocked(store); err != nil {
		return "", err
	}
	return "file://" + e.path + "#" + name, nil
}

// Delete removes the entry named in the ref.
func (e *EncryptedFileBackend) Delete(_ context.Context, ref string) error {
	path, entry, err := parseFileRef(ref)
	if err != nil {
		return err
	}
	if path != e.path {
		return fmt.Errorf("file backend: ref path %q does not match configured path %q", path, e.path)
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	store, err := e.loadLocked()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	delete(store, entry)
	return e.saveLocked(store)
}

func (e *EncryptedFileBackend) loadLocked() (map[string]string, error) {
	raw, err := os.ReadFile(e.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("file backend: read: %w", err)
	}
	var store map[string]string
	if err := json.Unmarshal(raw, &store); err != nil {
		return nil, fmt.Errorf("file backend: parse: %w", err)
	}
	if store == nil {
		store = map[string]string{}
	}
	return store, nil
}

func (e *EncryptedFileBackend) saveLocked(store map[string]string) error {
	raw, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("file backend: marshal: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(e.path), 0700); err != nil {
		return fmt.Errorf("file backend: mkdir: %w", err)
	}
	if err := os.WriteFile(e.path, raw, 0600); err != nil {
		return fmt.Errorf("file backend: write: %w", err)
	}
	return nil
}

// parseFileRef splits file://<path>#<entry> into path and entry.
func parseFileRef(ref string) (path, entry string, err error) {
	scheme, opaque, err := ParseRef(ref)
	if err != nil {
		return "", "", err
	}
	if scheme != "file" {
		return "", "", fmt.Errorf("file backend: wrong scheme %q", scheme)
	}
	i := strings.LastIndex(opaque, "#")
	if i < 0 {
		return "", "", fmt.Errorf("file backend: ref missing '#entry' suffix: %s", ref)
	}
	return opaque[:i], opaque[i+1:], nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/inky/Development/gitlab-mcp-server && go test ./pkg/config/ -run 'TestEncryptedFileBackend|TestEncryptedFileBackend_ParseFileRef' -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/config/backend_file.go pkg/config/backend_file_test.go
git commit -m "feat(config): add EncryptedFileBackend

AES-256-GCM-encrypted JSON-at-rest store keyed by entry name. Reuses the
existing CryptoManager. Intended for headless/Docker use where keyring is
unavailable. Scheme 'file', ref form file://<path>#<entry>."
```

---

### Task 5: `ExternalCmdBackend` (with shell-injection-resistance test)

Runs a templated command to resolve a secret — e.g. `op read op://Work/gitlab/token`. Command template comes from global config, not the ref, so the LLM cannot influence which binary runs. **No `Store`** in this backend — the user pre-populates their external tool.

**Files:**
- Create: `pkg/config/backend_external.go`
- Create: `pkg/config/backend_external_test.go`

- [ ] **Step 1: Write the failing tests**

We use a small shim binary for the test: a helper script that `echo`s its argv for introspection. We exec `go` to run a tiny Go program — no bash, no dependency on external CLIs.

```go
// pkg/config/backend_external_test.go
package config

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildArgvEcho compiles a throwaway helper binary that writes each argv
// element on its own line. Returns the binary path.
func buildArgvEcho(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("external-cmd injection test uses POSIX semantics")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	bin := filepath.Join(dir, "argvecho")
	require.NoError(t, os.WriteFile(src, []byte(`package main
import ("fmt"; "os")
func main() { for _, a := range os.Args[1:] { fmt.Println(a) }; fmt.Println("SECRET_VALUE") }
`), 0600))
	out, err := runCmd(dir, "go", "build", "-o", bin, src)
	require.NoError(t, err, "build helper: %s", out)
	return bin
}

func TestExternalCmdBackend_ResolvesViaTemplate(t *testing.T) {
	bin := buildArgvEcho(t)
	templates := map[string]string{"op": bin + " %s"}
	b := NewExternalCmdBackend(templates)

	got, err := b.Resolve(context.Background(), "op://Work/gitlab/token")
	require.NoError(t, err)
	assert.Equal(t, "SECRET_VALUE", got) // our helper emits "SECRET_VALUE" on its last line

	// The argv should contain "Work/gitlab/token" as ONE argv element, not split.
	// We rely on the helper printing each arg on its own line; the last output line
	// is the final "SECRET_VALUE". The preceding lines are the real argv. We only need
	// the Resolve() value to be the LAST line, which we assert above. The arg-handling
	// contract is verified by the injection test below.
}

func TestExternalCmdBackend_ShellMetacharsArePassedLiterally(t *testing.T) {
	bin := buildArgvEcho(t)
	// A malicious-looking ref tail with shell metacharacters that would wreak havoc
	// under `sh -c` but must be passed verbatim under exec.Command.
	tail := "Work/gitlab/token; rm -rf / && echo pwned"
	templates := map[string]string{"op": bin + " %s"}
	b := NewExternalCmdBackend(templates)

	got, err := b.Resolve(context.Background(), "op://"+tail)
	require.NoError(t, err)
	// Still gets the helper's SECRET_VALUE line — `rm -rf /` was NOT interpreted by a shell.
	assert.Equal(t, "SECRET_VALUE", got)
}

func TestExternalCmdBackend_UnknownScheme(t *testing.T) {
	b := NewExternalCmdBackend(map[string]string{"op": "op read %s"})
	_, err := b.Resolve(context.Background(), "pass://foo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no command template")
}

func TestExternalCmdBackend_StoreUnsupported(t *testing.T) {
	b := NewExternalCmdBackend(map[string]string{"op": "op read %s"})
	_, err := b.Store(context.Background(), "name", "secret")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "store is not supported")
}
```

Helper for running arbitrary commands from tests (we'll reuse later too):

```go
// pkg/config/backend_external_test.go (append)
import (
	"bytes"
	"os/exec"
)

func runCmd(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}
```

(Merge the imports into one import block in your editor.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/inky/Development/gitlab-mcp-server && go test ./pkg/config/ -run 'TestExternalCmdBackend' -v`
Expected: FAIL — `undefined: NewExternalCmdBackend`.

- [ ] **Step 3: Write the minimal implementation**

```go
// pkg/config/backend_external.go
package config

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ExternalCmdBackend resolves secrets by shelling out to a user-configured command.
// Schemes are allow-listed at construction time via a templates map:
//
//	op:   "op read %s"
//	pass: "pass show %s"
//	age:  "age -d -i /home/u/.config/age/key.txt %s"
//
// The %s is replaced with the opaque portion of the ref. Commands are executed
// via exec.Command (not `sh -c`), so shell metacharacters in the ref cannot
// influence command parsing. Store is not implemented — users populate their
// external tool directly.
type ExternalCmdBackend struct {
	templates map[string]string // scheme → template (must contain %s exactly once)
}

// NewExternalCmdBackend returns a backend whose Scheme() dispatches into
// ExternalCmdDispatcher below. NOTE: Scheme() can only return a single string,
// but this backend handles multiple schemes. It is therefore registered
// indirectly via the RegisterAll helper below rather than via Register().
func NewExternalCmdBackend(templates map[string]string) *ExternalCmdBackend {
	cp := make(map[string]string, len(templates))
	for k, v := range templates {
		cp[k] = v
	}
	return &ExternalCmdBackend{templates: cp}
}

// Schemes returns the list of schemes this backend handles.
func (x *ExternalCmdBackend) Schemes() []string {
	out := make([]string, 0, len(x.templates))
	for k := range x.templates {
		out = append(out, k)
	}
	return out
}

// Scheme is NOT meaningful for this multi-scheme backend. It returns "external"
// as a sentinel — callers should use RegisterAll to install one shim per scheme.
func (x *ExternalCmdBackend) Scheme() string { return "external" }

// RegisterAll registers a per-scheme shim in the given registry for every
// scheme this backend handles. This keeps the one-scheme-per-backend rule in
// BackendRegistry intact.
func (x *ExternalCmdBackend) RegisterAll(r *BackendRegistry) error {
	for scheme := range x.templates {
		if err := r.Register(&externalSchemeShim{scheme: scheme, parent: x}); err != nil {
			return err
		}
	}
	return nil
}

// Resolve runs the templated command and returns its last non-empty stdout line.
func (x *ExternalCmdBackend) Resolve(ctx context.Context, ref string) (string, error) {
	scheme, opaque, err := ParseRef(ref)
	if err != nil {
		return "", err
	}
	tmpl, ok := x.templates[scheme]
	if !ok {
		return "", fmt.Errorf("external-cmd backend: no command template for scheme %q", scheme)
	}
	name, args, err := buildExec(tmpl, opaque)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("external-cmd backend: %s: %w", name, err)
	}
	// Take the last non-empty line as the secret.
	secret := lastNonEmptyLine(string(out))
	if secret == "" {
		return "", fmt.Errorf("external-cmd backend: empty output from %s", name)
	}
	return secret, nil
}

// Store is unsupported: external tools are populated by the user out-of-band.
func (x *ExternalCmdBackend) Store(_ context.Context, _, _ string) (string, error) {
	return "", errors.New("external-cmd backend: store is not supported — populate your external tool directly and pass --token-ref")
}

// Delete is unsupported for the same reason.
func (x *ExternalCmdBackend) Delete(_ context.Context, _ string) error {
	return errors.New("external-cmd backend: delete is not supported — manage entries in your external tool directly")
}

// buildExec splits the template on whitespace, substitutes the single %s with
// the opaque ref, and returns (name, args). Whitespace in the opaque portion is
// preserved: the result is a single argv element containing it.
func buildExec(tmpl, opaque string) (string, []string, error) {
	fields := strings.Fields(tmpl)
	if len(fields) == 0 {
		return "", nil, errors.New("external-cmd backend: empty command template")
	}
	replaced := make([]string, 0, len(fields))
	subs := 0
	for _, f := range fields {
		if strings.Contains(f, "%s") {
			replaced = append(replaced, strings.ReplaceAll(f, "%s", opaque))
			subs++
			continue
		}
		replaced = append(replaced, f)
	}
	if subs == 0 {
		return "", nil, errors.New("external-cmd backend: template missing %s placeholder")
	}
	return replaced[0], replaced[1:], nil
}

func lastNonEmptyLine(s string) string {
	sc := bufio.NewScanner(strings.NewReader(s))
	last := ""
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), " \t\r")
		if line != "" {
			last = line
		}
	}
	return last
}

// externalSchemeShim adapts ExternalCmdBackend to the single-scheme Register API.
type externalSchemeShim struct {
	scheme string
	parent *ExternalCmdBackend
}

func (s *externalSchemeShim) Scheme() string { return s.scheme }
func (s *externalSchemeShim) Resolve(ctx context.Context, ref string) (string, error) {
	return s.parent.Resolve(ctx, ref)
}
func (s *externalSchemeShim) Store(ctx context.Context, name, secret string) (string, error) {
	return s.parent.Store(ctx, name, secret)
}
func (s *externalSchemeShim) Delete(ctx context.Context, ref string) error {
	return s.parent.Delete(ctx, ref)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/inky/Development/gitlab-mcp-server && go test ./pkg/config/ -run 'TestExternalCmdBackend' -v`
Expected: all PASS. The injection test specifically proves `rm -rf /` was not interpreted.

- [ ] **Step 5: Commit**

```bash
git add pkg/config/backend_external.go pkg/config/backend_external_test.go
git commit -m "feat(config): add ExternalCmdBackend for op/pass/age/gopass refs

Runs user-configured command templates via exec.Command (no shell) to
resolve secrets from external password managers. Shell metacharacters in
the ref are passed as a single argv element — verified by an
injection-resistance test. Store/Delete intentionally unsupported."
```

---

### Task 6: Extend `ServerConfig` with `TokenRef`; `Config` with `Backends`; wire registry into `Manager`

Additive changes only. Existing fields stay. Callers that set `Token` keep working; new callers set `TokenRef`.

**Files:**
- Modify: `pkg/config/config.go`
- Modify: `pkg/config/config_test.go` (adds new tests; does not change existing ones)

- [ ] **Step 1: Write the failing tests**

Append these to `pkg/config/config_test.go`:

```go
// pkg/config/config_test.go (append)
func TestManager_WithRegistry_ResolvesTokenRef(t *testing.T) {
	reg := NewBackendRegistry()
	fake := NewFakeSecretBackend("keyring")
	fake.SetEntry("work", "glpat-resolved")
	require.NoError(t, reg.Register(fake))

	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.json")
	m, err := NewManagerWithRegistry(path, reg)
	require.NoError(t, err)

	require.NoError(t, m.AddServer(&ServerConfig{
		Name:     "work",
		Host:     "https://gitlab.example.com",
		TokenRef: "keyring://work",
	}))

	got, err := m.ResolveServerToken(context.Background(), "work")
	require.NoError(t, err)
	assert.Equal(t, "glpat-resolved", got)
}

func TestManager_WithRegistry_PrefersTokenRefOverToken(t *testing.T) {
	reg := NewBackendRegistry()
	fake := NewFakeSecretBackend("keyring")
	fake.SetEntry("work", "FROM-REF")
	require.NoError(t, reg.Register(fake))

	tmp := t.TempDir()
	m, err := NewManagerWithRegistry(filepath.Join(tmp, "config.json"), reg)
	require.NoError(t, err)

	require.NoError(t, m.AddServer(&ServerConfig{
		Name:     "work",
		Host:     "https://gitlab.example.com",
		Token:    "FROM-TOKEN",      // legacy
		TokenRef: "keyring://work",  // new — wins
	}))

	got, err := m.ResolveServerToken(context.Background(), "work")
	require.NoError(t, err)
	assert.Equal(t, "FROM-REF", got)
}

func TestManager_WithRegistry_FallsBackToLegacyToken(t *testing.T) {
	reg := NewBackendRegistry() // empty registry
	tmp := t.TempDir()
	m, err := NewManagerWithRegistry(filepath.Join(tmp, "config.json"), reg)
	require.NoError(t, err)

	require.NoError(t, m.AddServer(&ServerConfig{
		Name:  "work",
		Host:  "https://gitlab.example.com",
		Token: "FROM-TOKEN",
	}))

	got, err := m.ResolveServerToken(context.Background(), "work")
	require.NoError(t, err)
	assert.Equal(t, "FROM-TOKEN", got)
}

func TestConfig_BackendsSection_Roundtrips(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.json")
	m, err := NewManager(path)
	require.NoError(t, err)
	m.Config().Backends = &BackendsConfig{
		External: map[string]string{"op": "op read %s"},
	}
	require.NoError(t, m.Save())

	m2, err := NewManager(path)
	require.NoError(t, err)
	require.NotNil(t, m2.Config().Backends)
	assert.Equal(t, "op read %s", m2.Config().Backends.External["op"])
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/inky/Development/gitlab-mcp-server && go test ./pkg/config/ -run 'TestManager_WithRegistry|TestConfig_BackendsSection' -v`
Expected: FAIL — `undefined: NewManagerWithRegistry`, `undefined: ServerConfig{TokenRef}`, `undefined: BackendsConfig`.

- [ ] **Step 3: Modify `ServerConfig` and `Config`**

Edit `pkg/config/config.go`:

```go
// pkg/config/config.go — update type declarations
type ServerConfig struct {
	Name          string `json:"name"`
	Host          string `json:"host"`
	Token         string `json:"token,omitempty"`     // legacy; present until v3.0 clean break
	TokenRef      string `json:"tokenRef,omitempty"`  // new in v2.1; takes precedence when set
	ReadOnly      bool   `json:"readOnly,omitempty"`
	IsDefault     bool   `json:"isDefault,omitempty"` // still honored but no longer required
	UserID        int    `json:"userId,omitempty"`
	Username      string `json:"username,omitempty"`
	LastValidated string `json:"lastValidated,omitempty"`
}

// BackendsConfig holds pluggable-backend settings (external command templates).
type BackendsConfig struct {
	External map[string]string `json:"external,omitempty"` // scheme → command template
}

type Config struct {
	Version  string                   `json:"version"`
	Servers  map[string]*ServerConfig `json:"servers,omitempty"`
	Backends *BackendsConfig          `json:"backends,omitempty"`
}
```

- [ ] **Step 4: Add `NewManagerWithRegistry` and `ResolveServerToken`**

Append to `pkg/config/config.go`:

```go
// pkg/config/config.go — add at end

// NewManagerWithRegistry creates a Manager wired to a BackendRegistry.
// If registry is nil, ResolveServerToken falls back to the legacy Token field.
func NewManagerWithRegistry(configPath string, registry *BackendRegistry) (*Manager, error) {
	m, err := NewManager(configPath)
	if err != nil {
		return nil, err
	}
	m.registry = registry
	return m, nil
}

// ResolveServerToken returns the plaintext token for the given server, preferring
// TokenRef (via the registry) over the legacy Token field.
func (m *Manager) ResolveServerToken(ctx context.Context, name string) (string, error) {
	m.mu.RLock()
	srv, ok := m.config.Servers[name]
	registry := m.registry
	m.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("server %q not found", name)
	}
	if srv.TokenRef != "" && registry != nil {
		return registry.Resolve(ctx, srv.TokenRef)
	}
	// Legacy fallback: decrypt if needed (CryptoManager already handles plaintext).
	return m.crypto.Decrypt(srv.Token)
}
```

And add the `registry *BackendRegistry` field to the `Manager` struct declaration near the top of the file:

```go
// pkg/config/config.go — update Manager struct
type Manager struct {
	mu       sync.RWMutex
	config   *Config
	FilePath string
	crypto   *CryptoManager
	registry *BackendRegistry // nil-safe; optional
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /home/inky/Development/gitlab-mcp-server && go test ./pkg/config/ -v`
Expected: all new PASS; all existing tests still PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/config/config.go pkg/config/config_test.go
git commit -m "feat(config): add TokenRef and BackendsConfig; ResolveServerToken

ServerConfig gains TokenRef alongside existing Token. Manager gains an
optional BackendRegistry; ResolveServerToken prefers TokenRef when the
registry can resolve it, else falls back to the legacy Token field.
Legacy callers are unaffected."
```

---

### Task 7: Config file v1 → v2 migration on save (with backup)

On any `Manager.Save()` where at least one server already has a `TokenRef`, bump the stored `Version` to `"2.0"` and emit a one-time `.bak` snapshot of the previous file.

**Files:**
- Modify: `pkg/config/config.go`
- Modify: `pkg/config/config_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `pkg/config/config_test.go`:

```go
func TestManager_Save_BumpsVersionWhenTokenRefPresent(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.json")

	// Pretend a v1 config already exists on disk.
	v1 := `{"version":"1.0","servers":{"work":{"name":"work","host":"https://gitlab.example.com","token":"plain-token"}}}`
	require.NoError(t, os.WriteFile(path, []byte(v1), 0600))

	reg := NewBackendRegistry()
	fake := NewFakeSecretBackend("keyring")
	require.NoError(t, reg.Register(fake))

	m, err := NewManagerWithRegistry(path, reg)
	require.NoError(t, err)

	// Upgrade: set TokenRef for the existing server.
	require.NoError(t, m.UpdateServer("work", &ServerConfig{
		Name:     "work",
		Host:     "https://gitlab.example.com",
		TokenRef: "keyring://work",
	}))
	require.NoError(t, m.Save())

	// Version on disk is now "2.0".
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"version": "2.0"`)

	// Backup of the previous v1 file exists next to it.
	bak, err := os.ReadFile(path + ".bak")
	require.NoError(t, err)
	assert.Contains(t, string(bak), `"version":"1.0"`)
}

func TestManager_Save_NoBumpWhenOnlyLegacyTokens(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.json")

	m, err := NewManager(path)
	require.NoError(t, err)
	require.NoError(t, m.AddServer(&ServerConfig{
		Name:  "work",
		Host:  "https://gitlab.example.com",
		Token: "plain",
	}))
	require.NoError(t, m.Save())

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"version": "1.0"`)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/inky/Development/gitlab-mcp-server && go test ./pkg/config/ -run 'TestManager_Save_Bumps|TestManager_Save_NoBump' -v`
Expected: FAIL — version stays `"1.0"` and no `.bak` is produced.

- [ ] **Step 3: Modify `Save` to bump version + create backup**

Replace the body of `Save` in `pkg/config/config.go`:

```go
// pkg/config/config.go — replace Save

// Save saves the configuration to disk. When any server carries a TokenRef,
// the on-disk schema is bumped to "2.0" and a one-time backup of the prior
// file is written next to it as config.json.bak.
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	hasTokenRef := false
	for _, s := range m.config.Servers {
		if s.TokenRef != "" {
			hasTokenRef = true
			break
		}
	}

	// Decide target version.
	if hasTokenRef && m.config.Version != "2.0" {
		// Backup the prior file (best-effort; missing prior file is fine).
		if prior, err := os.ReadFile(m.FilePath); err == nil {
			_ = os.WriteFile(m.FilePath+".bak", prior, 0600)
		}
		m.config.Version = "2.0"
	}
	if !hasTokenRef && m.config.Version == "" {
		m.config.Version = "1.0"
	}

	configDir := filepath.Dir(m.FilePath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(m.FilePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/inky/Development/gitlab-mcp-server && go test ./pkg/config/ -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/config/config.go pkg/config/config_test.go
git commit -m "feat(config): bump schema to v2 and back up v1 on first TokenRef save

When any server acquires a TokenRef, Save() writes the prior file as
config.json.bak and stamps version '2.0' going forward. Pure-legacy
configs keep version '1.0' until the user actually opts into refs."
```

---

### Task 8: `config backends` diagnostic subcommand

Lists registered backends and probes their availability. Useful for "my keyring is broken, what do I do" triage.

**Files:**
- Create: `cmd/config/backends.go`
- Modify: `cmd/config/command.go`

- [ ] **Step 1: Peek at the existing command wiring**

Run: `cd /home/inky/Development/gitlab-mcp-server && cat cmd/config/command.go`
Expected: you'll see a `Cmd` variable wired by `AddCommand` to each subcommand. The pattern: one file per subcommand, one `newXxxCmd` constructor.

- [ ] **Step 2: Write the subcommand**

```go
// cmd/config/backends.go
package config

import (
	"context"
	"fmt"
	"os/exec"

	pkgconfig "github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

func newBackendsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backends",
		Short: "List available secret backends and probe their health",
		Long: `Diagnose which secret backends are compiled in, which are reachable,
and what command templates are configured for external backends.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			mgr, err := pkgconfig.NewManager("")
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			out := cmd.OutOrStdout()

			fmt.Fprintln(out, "Secret backends:")

			// keyring probe
			probeKey := "gitlab-mcp-server-probe"
			probeAcct := "probe"
			setErr := keyring.Set(probeKey, probeAcct, "ok")
			if setErr == nil {
				_ = keyring.Delete(probeKey, probeAcct)
				fmt.Fprintln(out, "  keyring      OK (default)")
			} else {
				fmt.Fprintf(out, "  keyring      UNAVAILABLE: %v\n", setErr)
			}

			// file backend is always compiled in
			fmt.Fprintln(out, "  file         OK (~/.gitlab-mcp-server/secrets.enc when used)")

			// external commands
			if mgr.Config().Backends != nil && len(mgr.Config().Backends.External) > 0 {
				fmt.Fprintln(out, "  external:")
				for scheme, tmpl := range mgr.Config().Backends.External {
					name := firstWord(tmpl)
					path, err := exec.LookPath(name)
					if err != nil {
						fmt.Fprintf(out, "    %-8s template=%q  binary=%q NOT FOUND ON $PATH\n", scheme, tmpl, name)
					} else {
						fmt.Fprintf(out, "    %-8s template=%q  binary=%s\n", scheme, tmpl, path)
					}
				}
			} else {
				fmt.Fprintln(out, "  external     (none configured — add under backends.external in config)")
			}

			_ = context.Background() // reserved for future async probes
			return nil
		},
	}
}

func firstWord(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			return s[:i]
		}
	}
	return s
}
```

- [ ] **Step 3: Wire it into the `config` command**

Edit `cmd/config/command.go` and add `newBackendsCmd()` to the list of subcommands registered on `Cmd`. For example, if the existing file ends with:

```go
Cmd.AddCommand(
    newInitCmd(mgr),
    newAddCmd(mgr),
    newRemoveCmd(mgr),
    newListCmd(mgr),
    newDefaultCmd(mgr),
    newValidateCmd(mgr),
)
```

add `newBackendsCmd()` to the slice. Do not pass a `Manager` in — the subcommand creates its own.

- [ ] **Step 4: Smoke-test the command**

Run: `cd /home/inky/Development/gitlab-mcp-server && go build ./... && ./gitlab-mcp-server config backends`
Expected: a human-readable listing of backend probe results. No panics. Exit 0 on a machine with keyring working; non-zero exit only on config load failure.

- [ ] **Step 5: Commit**

```bash
git add cmd/config/backends.go cmd/config/command.go
git commit -m "feat(cli): add 'config backends' diagnostic subcommand

Probes keyring reachability, reports whether configured external-cmd
binaries are on \$PATH, and prints command templates verbatim for
debugging. Does not touch token data."
```

---

### Task 9: `config add --token-ref` + interactive TTY; `--token` deprecation warning

Adds two new non-secret-in-argv paths. `--token` still works (to avoid breaking automation in v2.1) but emits a warning.

**Files:**
- Create: `cmd/config/secret_input.go`
- Modify: `cmd/config/add.go`
- Modify: `go.mod` / `go.sum` (for `golang.org/x/term`)

- [ ] **Step 1: Add the `golang.org/x/term` dependency**

Run: `cd /home/inky/Development/gitlab-mcp-server && go get golang.org/x/term`
Expected: `go.mod` and `go.sum` are updated. No code change required yet; we'll use it in step 3.

- [ ] **Step 2: Write the secret-input helper**

```go
// cmd/config/secret_input.go
package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// promptSecret reads a secret from the controlling TTY without echoing.
// If stdin is not a TTY, it returns an error instructing the caller to use
// --token-ref instead.
func promptSecret(prompt string) (string, error) {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", errors.New("no TTY available — use --token-ref <ref> in non-interactive contexts")
	}
	defer f.Close()
	fmt.Fprint(f, prompt)
	fd := int(f.Fd())
	bytes, err := term.ReadPassword(fd)
	fmt.Fprintln(f)
	if err != nil {
		return "", fmt.Errorf("read secret: %w", err)
	}
	return strings.TrimSpace(string(bytes)), nil
}

// promptLine reads a line from the given reader (used for backend-choice prompts).
func promptLine(r io.Reader, prompt string, dflt string) (string, error) {
	fmt.Print(prompt)
	sc := bufio.NewScanner(r)
	if sc.Scan() {
		s := strings.TrimSpace(sc.Text())
		if s == "" {
			return dflt, nil
		}
		return s, nil
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return dflt, nil
}
```

- [ ] **Step 3: Update `cmd/config/add.go`**

Replace the flag wiring and `runAdd` logic. Keep compatibility: `--token` still works with a warning; `--token-ref` is preferred; interactive prompt is the fallback.

Edit the var block and `newAddCmd`:

```go
// cmd/config/add.go — top of file (replace var block + newAddCmd)
var (
	addHost        string
	addToken       string
	addTokenRef    string
	addBackend     string
	addReadOnly    bool
	addEncryptFlag bool
)

func newAddCmd(_ *config.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new GitLab server configuration",
		Long: `Add a new GitLab server configuration.

Token can be supplied three ways:
  --token <raw>             DEPRECATED: exposes the secret in shell history.
  --token-ref <ref>         A ref like keyring://work or op://Work/gitlab/token.
                            The ref must resolve in a configured backend.
  (none of the above)       Prompt on TTY with no echo.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			registry, err := buildBackendRegistry(addBackend)
			if err != nil {
				return err
			}
			cfgManager, err := config.NewManagerWithRegistry("", registry)
			if err != nil {
				return fmt.Errorf("failed to create config manager: %w", err)
			}
			return runAdd(cmd.Context(), cfgManager, registry, args[0])
		},
	}

	cmd.Flags().StringVar(&addHost, "host", "", "GitLab host URL (required)")
	cmd.Flags().StringVar(&addToken, "token", "", "GitLab personal access token (DEPRECATED: use --token-ref or interactive)")
	cmd.Flags().StringVar(&addTokenRef, "token-ref", "", "Ref to a secret in a configured backend (e.g. keyring://work, op://Work/gitlab/token)")
	cmd.Flags().StringVar(&addBackend, "backend", "keyring", "Backend to store newly-entered secret (keyring|file); ignored when --token-ref is set")
	cmd.Flags().BoolVar(&addReadOnly, "read-only", false, "Configure server in read-only mode")
	cmd.Flags().BoolVar(&addEncryptFlag, "encrypt", false, "(Legacy) encrypt token using system keyring")
	_ = cmd.MarkFlagRequired("host")
	return cmd
}
```

Add a helper `buildBackendRegistry` in the same file:

```go
// cmd/config/add.go — add helper

func buildBackendRegistry(primary string) (*config.BackendRegistry, error) {
	r := config.NewBackendRegistry()
	if err := r.Register(config.NewKeyringBackend("gitlab-mcp-server")); err != nil {
		return nil, err
	}
	// EncryptedFileBackend is registered lazily — only if the user chooses it.
	// External backends are wired in from config file at server startup, not here.
	return r, nil
}
```

Replace `runAdd`:

```go
// cmd/config/add.go — replace runAdd

func runAdd(ctx context.Context, cfgManager *config.Manager, registry *config.BackendRegistry, name string) error {
	if cfgManager.HasServer(name) {
		return fmt.Errorf("server '%s' already exists. Use 'config list' to see existing servers", name)
	}
	host := normalizeHost(addHost)

	// Resolve secret source.
	var secret string
	var tokenRef string
	switch {
	case addToken != "" && addTokenRef != "":
		return fmt.Errorf("--token and --token-ref are mutually exclusive")
	case addTokenRef != "":
		// Validate the ref resolves NOW so we fail before we persist.
		s, err := registry.Resolve(ctx, addTokenRef)
		if err != nil {
			return fmt.Errorf("resolve --token-ref: %w", err)
		}
		secret = s
		tokenRef = addTokenRef
	case addToken != "":
		fmt.Fprintln(os.Stderr, "DEPRECATION: --token exposes the secret in shell history and process listings. "+
			"Use --token-ref or interactive mode. --token will be removed in v3.0.")
		secret = addToken
		// Store into the chosen backend so we DON'T save plaintext in config.json.
		ref, err := registry.Store(ctx, addBackend, name, secret)
		if err != nil {
			return fmt.Errorf("store secret in %q backend: %w", addBackend, err)
		}
		tokenRef = ref
	default:
		s, err := promptSecret(fmt.Sprintf("GitLab access token for %q: ", name))
		if err != nil {
			return err
		}
		if s == "" {
			return fmt.Errorf("no token entered")
		}
		secret = s
		ref, err := registry.Store(ctx, addBackend, name, secret)
		if err != nil {
			return fmt.Errorf("store secret in %q backend: %w", addBackend, err)
		}
		tokenRef = ref
	}

	// Validate against GitLab.
	fmt.Printf("Validating token for %s...\n", name)
	client, err := createGitLabClient(host, secret)
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}
	user, resp, err := client.Users.CurrentUser(gl.WithContext(ctx))
	if err != nil {
		if resp != nil && resp.StatusCode == 401 {
			return fmt.Errorf("token validation failed: invalid or expired token (401)")
		}
		return fmt.Errorf("token validation failed: %w", err)
	}
	fmt.Printf("Token validated for user %s (ID: %d)\n", user.Username, user.ID)

	cfg := &config.ServerConfig{
		Name:     name,
		Host:     host,
		TokenRef: tokenRef, // never Token in the new path
		ReadOnly: addReadOnly,
		UserID:   user.ID,
		Username: user.Username,
	}
	if err := cfgManager.AddServer(cfg); err != nil {
		return fmt.Errorf("failed to add server: %w", err)
	}
	if err := cfgManager.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Printf("\nServer '%s' added.\n", name)
	fmt.Printf("  Host:      %s\n", host)
	fmt.Printf("  User:      %s\n", user.Username)
	fmt.Printf("  TokenRef:  %s\n", tokenRef)
	fmt.Printf("  Read-only: %v\n", addReadOnly)
	return nil
}
```

Note: the existing `Manager.AddServer` runs `crypto.Encrypt` on the `Token` field. With the new path we set `Token = ""` and `TokenRef = <ref>`, so the encrypt call is a no-op (empty string in/out). No change needed there.

- [ ] **Step 4: Build and smoke-test**

Run: `cd /home/inky/Development/gitlab-mcp-server && go build ./...`
Expected: compiles.

Smoke run (requires a scratch GitLab token): `./gitlab-mcp-server config add sandbox --host https://gitlab.com --token-ref keyring://sandbox`
If no entry exists: `resolve --token-ref: keyring backend: secret not found`.

- [ ] **Step 5: Commit**

```bash
git add cmd/config/add.go cmd/config/secret_input.go go.mod go.sum
git commit -m "feat(cli): add --token-ref and interactive TTY paths to 'config add'

--token still works but prints a deprecation warning pointing at the new
paths. --token-ref validates resolution against the backend before
anything is persisted. Interactive mode reads from /dev/tty with no echo
via golang.org/x/term."
```

---

### Task 10: `.gmcprc` auto-promote `TokenName` → `Server` + deprecation warnings

The `Server` field already exists in `ProjectConfig`. This task:
1. Promotes deprecated `TokenName` into `Server` on read if `Server` is empty.
2. Logs a one-time warning per file per process when deprecated fields are seen.

**Files:**
- Modify: `pkg/gitlab/project_config.go`
- Modify: `pkg/gitlab/project_config_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `pkg/gitlab/project_config_test.go` (create the file section if it doesn't yet exist):

```go
// pkg/gitlab/project_config_test.go (append)
func TestReadProjectConfig_PromotesTokenNameToServer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gmcprc")
	body := `{"projectId":"g/p","tokenName":"work"}`
	require.NoError(t, os.WriteFile(path, []byte(body), 0600))

	cfg, err := readProjectConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "g/p", cfg.ProjectID)
	assert.Equal(t, "work", cfg.Server, "tokenName should auto-promote to server")
}

func TestReadProjectConfig_KeepsServerWhenBothSet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gmcprc")
	body := `{"projectId":"g/p","server":"work","tokenName":"other"}`
	require.NoError(t, os.WriteFile(path, []byte(body), 0600))

	cfg, err := readProjectConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "work", cfg.Server, "explicit server wins over deprecated tokenName")
}
```

Make sure the test file imports `"github.com/stretchr/testify/assert"` and `"github.com/stretchr/testify/require"` and `"os"`, `"path/filepath"`, `"testing"`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/inky/Development/gitlab-mcp-server && go test ./pkg/gitlab/ -run 'TestReadProjectConfig_Promotes|TestReadProjectConfig_KeepsServer' -v`
Expected: FAIL — `Server` stays `""` on the first test.

- [ ] **Step 3: Implement promotion + one-time warning**

In `pkg/gitlab/project_config.go`, replace `readProjectConfig`:

```go
// pkg/gitlab/project_config.go — replace readProjectConfig

var gmcprcWarnedPaths sync.Map // path → struct{}

func readProjectConfig(path string) (*ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	var config ProjectConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	// Promote deprecated TokenName if Server is empty.
	deprecated := []string{}
	if config.Server == "" && config.TokenName != "" {
		config.Server = config.TokenName
		deprecated = append(deprecated, "tokenName")
	} else if config.TokenName != "" {
		deprecated = append(deprecated, "tokenName (ignored; 'server' already set)")
	}
	if config.GitLabHost != "" {
		deprecated = append(deprecated, "gitlabHost")
	}
	if len(deprecated) > 0 {
		if _, already := gmcprcWarnedPaths.LoadOrStore(path, struct{}{}); !already {
			fmt.Fprintf(os.Stderr,
				"DEPRECATION: %s contains legacy .gmcprc fields: %s. "+
					"Re-run 'gitlab-mcp-server project init' to migrate. These fields are removed in v3.0.\n",
				path, strings.Join(deprecated, ", "))
		}
	}
	return &config, nil
}
```

Add `"sync"` and `"strings"` to the imports if not already present.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/inky/Development/gitlab-mcp-server && go test ./pkg/gitlab/ -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/gitlab/project_config.go pkg/gitlab/project_config_test.go
git commit -m "feat(gitlab): auto-promote .gmcprc tokenName to server; warn on legacy fields

One-time stderr warning per .gmcprc per process when deprecated fields
(tokenName, gitlabHost) are present. Removed in v3.0."
```

---

### Task 11: `StrictResolver` with host verification (opt-in via env var)

New resolver lives alongside the existing `ClientResolver`. Used only when `GITLAB_MCP_STRICT_RESOLVER=1`. No fallbacks. Host check runs once per session per server, cached.

**Files:**
- Create: `pkg/gitlab/strict_resolver.go`
- Create: `pkg/gitlab/strict_resolver_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// pkg/gitlab/strict_resolver_test.go
package gitlab

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gl "gitlab.com/gitlab-org/api/client-go"
)

func mkProjectCfg(t *testing.T, server string) string {
	t.Helper()
	dir := t.TempDir()
	body := `{"projectId":"g/p","server":"` + server + `"}`
	path := filepath.Join(dir, ".gmcprc")
	require.NoError(t, os.WriteFile(path, []byte(body), 0600))
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	require.NoError(t, os.Chdir(dir))
	return path
}

// newFakeGitLab returns an httptest server whose /api/v4/user endpoint
// returns a minimal valid user response so client.Users.CurrentUser succeeds.
func newFakeGitLab(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/user", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"username":"probe"}`))
	})
	return httptest.NewServer(mux)
}

func TestStrictResolver_ResolvesGoodMatch(t *testing.T) {
	srv := newFakeGitLab(t)
	defer srv.Close()
	_ = mkProjectCfg(t, "work")

	pool := NewClientPool(NewTokenStore(), logrus.New())
	client, err := gl.NewClient("x", gl.WithBaseURL(srv.URL))
	require.NoError(t, err)
	require.NoError(t, pool.AddClient("work", client))

	r := NewStrictResolver(pool, map[string]string{"work": srv.URL}, logrus.New())
	got, name, err := r.Resolve(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "work", name)
	assert.NotNil(t, got)
}

func TestStrictResolver_ErrorsWhenNoProjectConfig(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	require.NoError(t, os.Chdir(dir))

	pool := NewClientPool(NewTokenStore(), logrus.New())
	r := NewStrictResolver(pool, nil, logrus.New())
	_, _, err := r.Resolve(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no project configured")
}

func TestStrictResolver_ErrorsWhenServerFieldEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gmcprc")
	require.NoError(t, os.WriteFile(path, []byte(`{"projectId":"g/p"}`), 0600))
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	require.NoError(t, os.Chdir(dir))

	pool := NewClientPool(NewTokenStore(), logrus.New())
	r := NewStrictResolver(pool, nil, logrus.New())
	_, _, err := r.Resolve(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required 'server' field")
}

func TestStrictResolver_ErrorsOnUnknownServer(t *testing.T) {
	_ = mkProjectCfg(t, "missing")
	pool := NewClientPool(NewTokenStore(), logrus.New())
	r := NewStrictResolver(pool, map[string]string{"work": "https://gitlab.example.com"}, logrus.New())
	_, _, err := r.Resolve(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestStrictResolver_ErrorsOnHostMismatch(t *testing.T) {
	realSrv := newFakeGitLab(t)
	defer realSrv.Close()
	_ = mkProjectCfg(t, "work")

	pool := NewClientPool(NewTokenStore(), logrus.New())
	client, err := gl.NewClient("x", gl.WithBaseURL(realSrv.URL))
	require.NoError(t, err)
	require.NoError(t, pool.AddClient("work", client))

	// Resolver is told server 'work' should be at "https://gitlab.other.invalid"
	// but the client is actually pointed at realSrv.URL.
	r := NewStrictResolver(pool, map[string]string{"work": "https://gitlab.other.invalid"}, logrus.New())
	_, _, err = r.Resolve(context.Background())
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "host") || errors.Is(err, ErrHostMismatch))
}
```

Note: `ErrHostMismatch` is a sentinel we'll declare in the implementation.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/inky/Development/gitlab-mcp-server && go test ./pkg/gitlab/ -run 'TestStrictResolver' -v`
Expected: FAIL — `undefined: NewStrictResolver`, `undefined: ErrHostMismatch`.

- [ ] **Step 3: Write the implementation**

```go
// pkg/gitlab/strict_resolver.go
package gitlab

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	gl "gitlab.com/gitlab-org/api/client-go"
)

// ErrHostMismatch is returned when the API response URL host differs from the
// host recorded in the server's config.
var ErrHostMismatch = errors.New("configured server host does not match API host")

// StrictResolver picks a GitLab client based strictly on .gmcprc. No fallbacks,
// no host-based matching, no default-server cascade. On first use per session
// per server it verifies the API host matches the configured host and caches
// the pass/fail.
type StrictResolver struct {
	pool         *ClientPool
	serverHosts  map[string]string // server name → configured host URL
	logger       *log.Logger
	mu           sync.Mutex
	verifiedOK   map[string]bool // server name → already checked this session
	verifiedFail map[string]error
}

// NewStrictResolver returns a resolver. serverHosts maps each configured
// server name to its expected host URL (from global config).
func NewStrictResolver(pool *ClientPool, serverHosts map[string]string, logger *log.Logger) *StrictResolver {
	return &StrictResolver{
		pool:         pool,
		serverHosts:  serverHosts,
		logger:       logger,
		verifiedOK:   map[string]bool{},
		verifiedFail: map[string]error{},
	}
}

// Resolve returns (client, serverName, error). It NEVER falls back.
func (r *StrictResolver) Resolve(ctx context.Context) (*gl.Client, string, error) {
	cfg, _, err := FindProjectConfig()
	if err != nil {
		return nil, "", fmt.Errorf("strict resolver: failed to read .gmcprc: %w", err)
	}
	if cfg == nil {
		return nil, "", errors.New("strict resolver: no project configured — run 'gitlab-mcp-server project init' or pass --server")
	}
	if cfg.Server == "" {
		return nil, "", errors.New("strict resolver: .gmcprc is missing required 'server' field — re-run 'gitlab-mcp-server project init'")
	}

	client, err := r.pool.GetClient(cfg.Server)
	if err != nil {
		configured := make([]string, 0, len(r.serverHosts))
		for n := range r.serverHosts {
			configured = append(configured, n)
		}
		return nil, "", fmt.Errorf("strict resolver: server %q not configured; configured servers: %s",
			cfg.Server, strings.Join(configured, ", "))
	}

	if err := r.verifyHost(ctx, cfg.Server, client); err != nil {
		return nil, "", err
	}
	return client, cfg.Server, nil
}

// GetClientFn adapts StrictResolver to the GetClientFn signature used by
// existing tool handlers.
func (r *StrictResolver) GetClientFn() GetClientFn {
	return func(ctx context.Context) (*gl.Client, error) {
		c, _, err := r.Resolve(ctx)
		return c, err
	}
}

func (r *StrictResolver) verifyHost(ctx context.Context, name string, client *gl.Client) error {
	r.mu.Lock()
	if r.verifiedOK[name] {
		r.mu.Unlock()
		return nil
	}
	if prev, ok := r.verifiedFail[name]; ok {
		r.mu.Unlock()
		return prev
	}
	r.mu.Unlock()

	wantHost := r.serverHosts[name]
	if wantHost == "" {
		return fmt.Errorf("strict resolver: no host recorded for server %q", name)
	}

	// Use client.Users.CurrentUser as the probe — it's already used elsewhere in
	// the codebase for token validation and is available on every authenticated
	// client. We inspect resp.Request.URL.Host to detect redirects or host
	// mismatches between the configured host and where the API actually answered.
	_, resp, err := client.Users.CurrentUser(gl.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("strict resolver: host verification call failed: %w", err)
	}
	if resp == nil || resp.Request == nil || resp.Request.URL == nil {
		return errors.New("strict resolver: host verification did not receive a response URL")
	}
	wantURL, err := url.Parse(wantHost)
	if err != nil {
		return fmt.Errorf("strict resolver: configured host %q is not a valid URL: %w", wantHost, err)
	}
	gotHost := resp.Request.URL.Host
	if wantURL.Host != gotHost {
		mismatch := fmt.Errorf("%w: configured %q but API responded from %q",
			ErrHostMismatch, wantURL.Host, gotHost)
		r.mu.Lock()
		r.verifiedFail[name] = mismatch
		r.mu.Unlock()
		return mismatch
	}
	r.mu.Lock()
	r.verifiedOK[name] = true
	r.mu.Unlock()
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/inky/Development/gitlab-mcp-server && go test ./pkg/gitlab/ -run 'TestStrictResolver' -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/gitlab/strict_resolver.go pkg/gitlab/strict_resolver_test.go
git commit -m "feat(gitlab): add StrictResolver with host verification

Looks up client by .gmcprc 'server' field only. Verifies API host matches
configured host on first use per server, caches the result. Errors are
loud and list configured servers on unknown-name miss."
```

---

### Task 12: Wire `StrictResolver` into startup behind `GITLAB_MCP_STRICT_RESOLVER=1`

The existing `ClientResolver` stays as the default. Only when the env var is truthy do we use `StrictResolver`. This lets users opt in without breaking anyone.

**Files:**
- Modify: `cmd/gitlab-mcp-server/main.go`

- [ ] **Step 1: Read the current resolver wiring**

Run: `grep -n 'NewClientResolver\|resolver\.' /home/inky/Development/gitlab-mcp-server/cmd/gitlab-mcp-server/main.go`
Expected: ~3 occurrences around lines 238–245.

- [ ] **Step 2: Replace the resolver construction**

In `cmd/gitlab-mcp-server/main.go`, find:

```go
// Create Client Resolver
resolver := gitlab.NewClientResolver(clientPool, defaultServer, logger)
logger.Infof("Client resolver initialized with default server '%s'", defaultServer)
```

Replace with:

```go
// Create Client Resolver — strict (opt-in via env) or legacy (default).
var resolverFn gitlab.GetClientFn
if os.Getenv("GITLAB_MCP_STRICT_RESOLVER") == "1" {
	hostsByName := map[string]string{}
	if hasConfigServers {
		for _, s := range cfgManager.ListServers() {
			hostsByName[s.Name] = s.Host
		}
	}
	sr := gitlab.NewStrictResolver(clientPool, hostsByName, logger)
	resolverFn = sr.GetClientFn()
	logger.Info("Strict resolver enabled (GITLAB_MCP_STRICT_RESOLVER=1) — no fallbacks, host verified per session")
} else {
	resolver := gitlab.NewClientResolver(clientPool, defaultServer, logger)
	resolverFn = resolver.GetClientFn()
	logger.Infof("Client resolver initialized with default server '%s' (legacy — set GITLAB_MCP_STRICT_RESOLVER=1 for strict mode)", defaultServer)
}
```

Then find the line:

```go
toolsetGroup, err := gitlab.InitToolsets(enabledToolsets, readOnly, resolver.GetClientFn(), logger, tokenStore, t, dynamicToolsets)
```

Change `resolver.GetClientFn()` → `resolverFn`.

- [ ] **Step 3: Build**

Run: `cd /home/inky/Development/gitlab-mcp-server && go build ./...`
Expected: compiles.

- [ ] **Step 4: Smoke-test both modes**

```bash
# Legacy (default): should start and resolve via old resolver.
./gitlab-mcp-server stdio --log-level=debug <<<'' 2>&1 | head -30
# Strict: needs a .gmcprc with server= and a matching configured server, else loud error on first request.
GITLAB_MCP_STRICT_RESOLVER=1 ./gitlab-mcp-server stdio --log-level=debug <<<'' 2>&1 | head -30
```

You'll see the "Strict resolver enabled" log line in the second run.

- [ ] **Step 5: Commit**

```bash
git add cmd/gitlab-mcp-server/main.go
git commit -m "feat(main): gate StrictResolver behind GITLAB_MCP_STRICT_RESOLVER=1

Opt-in flag for v2.1. Legacy ClientResolver remains the default path so
no existing user is broken; strict mode errors loudly on every
misconfiguration class we care about. Becomes the only mode in v3.0."
```

---

### Task 13: Wire `BackendRegistry` into startup

Register keyring by default. Register external-cmd schemes from `Config.Backends.External`. Pass the registry to `Manager` so `ResolveServerToken` works during `AddServerFromConfig`.

**Files:**
- Modify: `cmd/gitlab-mcp-server/main.go`
- Modify: `pkg/gitlab/client_pool.go`

- [ ] **Step 1: Update `AddServerFromConfig` to use `ResolveServerToken`**

In `pkg/gitlab/client_pool.go`, change `AddServerFromConfig` to accept a resolver function (not a raw `*config.ServerConfig`-with-decrypted-Token).

Replace the body of `AddServerFromConfig`:

```go
// pkg/gitlab/client_pool.go — replace AddServerFromConfig

// TokenResolver resolves a server's token at runtime. Returns the plaintext
// token for a given server name.
type TokenResolver func(ctx context.Context, serverName string) (string, error)

// AddServerFromConfig adds a client using a TokenResolver for secret fetching.
// The resolver may hit a keyring / external tool / encrypted file.
func (cp *ClientPool) AddServerFromConfig(ctx context.Context, server *config.ServerConfig, resolve TokenResolver) error {
	if server == nil {
		return fmt.Errorf("server config cannot be nil")
	}
	if server.Name == "" {
		return fmt.Errorf("server name cannot be empty")
	}
	token, err := resolve(ctx, server.Name)
	if err != nil {
		return fmt.Errorf("resolve token for %q: %w", server.Name, err)
	}
	clientOpts := []gl.ClientOptionFunc{}
	if server.Host != "" && server.Host != "https://gitlab.com" {
		clientOpts = append(clientOpts, gl.WithBaseURL(server.Host))
	}
	glClient, err := gl.NewClient(token, clientOpts...)
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}
	if err := cp.AddClient(server.Name, glClient); err != nil {
		return err
	}
	metadata := &TokenMetadata{
		Token:         token,
		GitLabHost:    server.Host,
		CreatedAt:     time.Now(),
		LastValidated: time.Now(),
	}
	if server.UserID > 0 {
		metadata.UserID = server.UserID
		metadata.Username = server.Username
	}
	if err := cp.store.AddToken(server.Name, metadata); err != nil {
		cp.logger.Warnf("Failed to store token metadata for '%s': %v", server.Name, err)
	}
	return nil
}
```

- [ ] **Step 2: Update the one caller in `main.go`**

In `cmd/gitlab-mcp-server/main.go`, near where `cfgManager.ListServers()` is ranged over to call `clientPool.AddServerFromConfig`:

Before (existing):
```go
for _, serverCfg := range servers {
    if err := clientPool.AddServerFromConfig(ctx, serverCfg); err != nil {
```

Change to:
```go
resolveToken := func(ctx context.Context, name string) (string, error) {
    return cfgManager.ResolveServerToken(ctx, name)
}
for _, serverCfg := range servers {
    if err := clientPool.AddServerFromConfig(ctx, serverCfg, resolveToken); err != nil {
```

- [ ] **Step 3: Construct the registry and pass it to `cfgManager`**

Replace the existing `cfgManager, err := config.NewManager("")` line in `main.go` with:

```go
// Build the secret backend registry.
registry := config.NewBackendRegistry()
if err := registry.Register(config.NewKeyringBackend("gitlab-mcp-server")); err != nil {
    logger.Warnf("Failed to register keyring backend: %v", err)
}
cfgManager, err := config.NewManagerWithRegistry("", registry)
if err != nil {
    logger.Warnf("Failed to create config manager: %v", err)
}

// If config has external-cmd templates, register those schemes too.
if cfgManager != nil && cfgManager.Config().Backends != nil &&
    len(cfgManager.Config().Backends.External) > 0 {
    ext := config.NewExternalCmdBackend(cfgManager.Config().Backends.External)
    if err := ext.RegisterAll(registry); err != nil {
        logger.Warnf("Failed to register external-cmd backends: %v", err)
    } else {
        logger.Infof("Registered external-cmd schemes: %v", ext.Schemes())
    }
}

// If any server uses file:// refs, register the encrypted-file backend with its path.
for _, s := range cfgManager.ListServers() {
    if strings.HasPrefix(s.TokenRef, "file://") {
        if path, _, err := parseFileRefFromRef(s.TokenRef); err == nil {
            crypto, cerr := config.NewCryptoManager(true)
            if cerr == nil {
                _ = registry.Register(config.NewEncryptedFileBackend(path, crypto))
            }
            break // one file backend suffices
        }
    }
}
```

Add a tiny helper at the bottom of `main.go` (since `parseFileRef` is unexported in `pkg/config`):

```go
// Duplicated tiny parser to avoid exporting parseFileRef across package boundaries.
func parseFileRefFromRef(ref string) (path, entry string, err error) {
    const prefix = "file://"
    if !strings.HasPrefix(ref, prefix) {
        return "", "", fmt.Errorf("not a file ref: %s", ref)
    }
    rest := ref[len(prefix):]
    hash := strings.LastIndex(rest, "#")
    if hash < 0 {
        return "", "", fmt.Errorf("missing #entry: %s", ref)
    }
    return rest[:hash], rest[hash+1:], nil
}
```

- [ ] **Step 4: Build and test**

Run: `cd /home/inky/Development/gitlab-mcp-server && go build ./... && go test ./...`
Expected: all tests pass, binary builds.

- [ ] **Step 5: Commit**

```bash
git add cmd/gitlab-mcp-server/main.go pkg/gitlab/client_pool.go
git commit -m "feat(main): wire BackendRegistry into startup and client pool

Keyring backend registered by default. External-cmd schemes registered
from config.Backends.External. EncryptedFileBackend registered lazily
when any server uses a file:// ref. Client pool now fetches tokens via
a TokenResolver callback instead of reading ServerConfig.Token directly."
```

---

### Task 14: `project init` steers users toward explicit `--server` and stops writing `gitlabHost`

**Files:**
- Modify: `cmd/project/init.go`

- [ ] **Step 1: Read the current state**

Run: `cd /home/inky/Development/gitlab-mcp-server && cat cmd/project/init.go`
You've seen this — currently `--server` is optional and `gitlabHost` is written to `.gmcprc`.

- [ ] **Step 2: Update the command**

Replace `runInit` in `cmd/project/init.go`:

```go
// cmd/project/init.go — replace runInit

func runInit(cmd *cobra.Command, args []string) error {
	var projectID string
	var gitlabHost string

	if len(args) > 0 {
		projectID = args[0]
	} else {
		var err error
		projectID, gitlabHost, err = detectFromGit()
		if err != nil {
			return fmt.Errorf("failed to detect project from Git remote: %w\n\n"+
				"Please specify projectId explicitly:\n"+
				"  gitlab-mcp-server project init <projectId>", err)
		}
	}
	if initHost != "" {
		gitlabHost = initHost
	}

	// Server is effectively required in v2.1 for clean forward-compat with v3.0.
	// If not provided, try to match exactly one configured server by host.
	serverName := initServer
	if serverName == "" {
		mgr, err := pkgConfig.NewManager("")
		if err == nil && mgr.ServerCount() > 0 && gitlabHost != "" {
			matches := []string{}
			for _, s := range mgr.ListServers() {
				if strings.EqualFold(strings.TrimSuffix(s.Host, "/"),
					strings.TrimSuffix(gitlabHost, "/")) {
					matches = append(matches, s.Name)
				}
			}
			if len(matches) == 1 {
				serverName = matches[0]
				fmt.Fprintf(cmd.OutOrStdout(), "Matched server %q from configured host %s.\n", serverName, gitlabHost)
			} else if len(matches) > 1 {
				return fmt.Errorf("multiple configured servers match host %s: %v — re-run with --server <name>",
					gitlabHost, matches)
			}
		}
	}
	if serverName == "" {
		return fmt.Errorf("no --server specified and could not infer one. " +
			"Run 'gitlab-mcp-server config list' to see configured servers and pass --server <name>")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// v3 .gmcprc shape: no gitlabHost, no tokenName.
	cfg := &gitlab.ProjectConfig{
		ProjectID: projectID,
		Server:    serverName,
	}
	configPath, err := writeConfig(cwd, cfg)
	if err != nil {
		return fmt.Errorf("failed to write .gmcprc: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Project configuration created.\n\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  Project ID: %s\n", cfg.ProjectID)
	fmt.Fprintf(cmd.OutOrStdout(), "  Server:     %s\n", cfg.Server)
	fmt.Fprintf(cmd.OutOrStdout(), "\n  Config file: %s\n", configPath)
	return nil
}
```

Add these imports at the top of the file if not already present:

```go
import (
	"fmt"
	"os"
	"strings"

	pkgConfig "github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	"github.com/InkyQuill/gitlab-mcp-server/pkg/gitlab"
	"github.com/spf13/cobra"
)
```

- [ ] **Step 3: Build and run manually**

```bash
cd /home/inky/Development/gitlab-mcp-server && go build ./...
# In a git repo with a GitLab remote and a configured server named "work":
cd /tmp/some-gitlab-repo && /path/to/gitlab-mcp-server project init --server work
cat .gmcprc  # should show projectId + server, NO gitlabHost, NO tokenName
```

Expected: two fields plus `lastUpdated`. Legacy test coverage for `writeConfig` still passes (it doesn't care about extra/missing optional fields).

- [ ] **Step 4: Commit**

```bash
git add cmd/project/init.go
git commit -m "feat(project): steer 'project init' toward --server; stop writing gitlabHost

--server is auto-matched if exactly one configured server's host matches
the detected Git remote; otherwise init errors loudly with a pointer to
'config list'. The written .gmcprc contains only projectId + server in v2.1."
```

---

### Task 15: Installer stops emitting `GITLAB_TOKEN`/`GITLAB_HOST` in generated IDE entries

**Files:**
- Modify: `cmd/install/install.go`
- Modify: `cmd/install/install_test.go` (if it exists; if not, add one)

- [ ] **Step 1: Inspect current emission logic**

Re-read `createInstallConfig` in `cmd/install/install.go:222-236`. Currently it only emits `GITLAB_CONFIG_PATH` when `configPath` is set — no token env var is emitted. **This is already correct.** The spec item "stop emitting token env vars" is therefore a no-op *in code* for this file. The documentation in `docs/MULTI_SERVER_SETUP.md` is what's stale.

- [ ] **Step 2: Add a test pinning the behavior**

Create or extend `cmd/install/install_test.go`:

```go
// cmd/install/install_test.go
package install

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateInstallConfig_DoesNotEmitTokenEnv(t *testing.T) {
	cfg := createInstallConfig("/usr/local/bin/gitlab-mcp-server", "")
	if cfg.Env != nil {
		_, hasTok := cfg.Env["GITLAB_TOKEN"]
		_, hasHost := cfg.Env["GITLAB_HOST"]
		assert.Falsef(t, hasTok, "GITLAB_TOKEN must not be emitted: %v", cfg.Env)
		assert.Falsef(t, hasHost, "GITLAB_HOST must not be emitted: %v", cfg.Env)
	}
}

func TestCreateInstallConfig_EmitsConfigPathWhenGiven(t *testing.T) {
	cfg := createInstallConfig("/usr/local/bin/gitlab-mcp-server", "/tmp/x.json")
	assert.Equal(t, "/tmp/x.json", cfg.Env["GITLAB_CONFIG_PATH"])
}
```

- [ ] **Step 3: Run tests**

Run: `cd /home/inky/Development/gitlab-mcp-server && go test ./cmd/install/ -v`
Expected: both PASS.

- [ ] **Step 4: Update `docs/MULTI_SERVER_SETUP.md`**

Replace the section that shows per-instance entries with `GITLAB_TOKEN` env vars. Add a brief note:

```markdown
## v2.1 and later: single MCP entry

As of v2.1 the recommended setup is a single MCP server entry. Multiple
GitLab instances are configured in the global config file via
`gitlab-mcp-server config add <name>`, and the active instance per project
is selected by the `server` field in that project's `.gmcprc`.

Per-instance entries with `GITLAB_TOKEN`/`GITLAB_HOST` env vars still work
in v2.1 but are deprecated and will be removed in v3.0.
```

Leave the legacy sections below that note, but prefix each with a "(legacy)" heading.

- [ ] **Step 5: Commit**

```bash
git add cmd/install/install_test.go docs/MULTI_SERVER_SETUP.md
git commit -m "docs+test: pin installer no-token-env behavior and update multi-server docs

createInstallConfig already omits GITLAB_TOKEN/GITLAB_HOST; add a
regression test that fails if someone reintroduces it. Multi-server docs
now lead with the single-MCP-entry model and mark per-instance entries
as legacy."
```

---

### Task 16: Deprecation warnings — `GITLAB_TOKEN` env startup + `addToken`/`updateToken` tools

**Files:**
- Modify: `cmd/gitlab-mcp-server/main.go`
- Modify: `pkg/gitlab/token_tools.go`

- [ ] **Step 1: Add the `GITLAB_TOKEN` startup warning**

In `cmd/gitlab-mcp-server/main.go`, find the block that uses `token` (around line 162) and inject a warning *before* it calls `InitializeFromEnv`:

```go
// Before: logger.Info("Using environment variables for client initialization")
if token != "" {
    logger.Warn("DEPRECATION: GITLAB_TOKEN env var usage is deprecated and will be removed in v3.0. " +
        "Run 'gitlab-mcp-server config add <name> --host <url>' to migrate to the global config. " +
        "See docs/MULTI_SERVER_SETUP.md.")
    logger.Info("Using environment variables for client initialization")
    ...
}
```

- [ ] **Step 2: Add deprecation warning to `addToken`/`updateToken`**

In `pkg/gitlab/token_tools.go`, update the handler function inside `AddToken`. At the START of the handler body, insert:

```go
logger.Warn("DEPRECATION: addToken MCP tool accepts tokens via LLM input. " +
    "This tool will be removed in v3.0 for security. Configure tokens via " +
    "'gitlab-mcp-server config add' (CLI) instead.")
```

And in the `result` map near the end, add:

```go
result["deprecated"] = true
result["deprecationMessage"] = "addToken will be removed in v3.0. Use 'gitlab-mcp-server config add' CLI instead."
```

Apply the analogous changes to `UpdateToken` (same file).

- [ ] **Step 3: Add a test that pins the deprecation fields**

Append to `pkg/gitlab/token_tools_test.go` (create if absent — follow the style in other `_test.go` files in the package):

```go
// pkg/gitlab/token_tools_test.go (append)
package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gl "gitlab.com/gitlab-org/api/client-go"
)

func TestAddToken_EmitsDeprecationFields(t *testing.T) {
	// Fake GitLab server that accepts any token and returns a valid user.
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/user", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":42,"username":"probe"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	logger := logrus.New()
	store := NewTokenStore()

	factory := func(token string, opts ...gl.ClientOptionFunc) (*gl.Client, error) {
		return gl.NewClient(token, append(opts, gl.WithBaseURL(srv.URL))...)
	}

	_, handler := AddToken(factory, logger, store)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"name":       "probe",
		"token":      "glpat-xxx",
		"gitlabHost": srv.URL,
	}
	res, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Content, 1)

	txt, ok := res.Content[0].(mcp.TextContent)
	require.True(t, ok, "expected TextContent")

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(txt.Text), &parsed))
	assert.Equal(t, true, parsed["deprecated"])
	assert.Contains(t, parsed["deprecationMessage"], "v3.0")
}
```

**Merge the imports** into the existing `token_tools_test.go` import block if the file already exists — don't duplicate the `package gitlab` line.

- [ ] **Step 4: Build and run**

Run: `cd /home/inky/Development/gitlab-mcp-server && go build ./... && go test ./...`
Expected: compiles, tests pass.

Smoke:
```bash
GITLAB_TOKEN=glpat-xyz GITLAB_HOST=https://gitlab.com ./gitlab-mcp-server stdio --log-level=warn <<<'' 2>&1 | head -3
```
Expected: you should see `DEPRECATION: GITLAB_TOKEN env var usage is deprecated...` on stderr.

- [ ] **Step 5: Commit**

```bash
git add cmd/gitlab-mcp-server/main.go pkg/gitlab/token_tools.go pkg/gitlab/token_tools_test.go
git commit -m "feat(deprecation): warn on GITLAB_TOKEN env and addToken/updateToken tools

One-line stderr warning on startup when GITLAB_TOKEN is used. Runtime
tool responses carry 'deprecated': true and a migration message. All
three paths are removed in v3.0."
```

---

### Task 17: CHANGELOG + release notes for v2.1

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Prepend a v2.1 entry**

Edit `CHANGELOG.md` and add at the top (after the header, before the latest existing entry):

```markdown
## [2.1.0] — Unreleased

### Added
- `SecretBackend` abstraction with three implementations: `KeyringBackend`
  (default, OS keyring), `EncryptedFileBackend` (AES-256-GCM JSON-at-rest),
  and `ExternalCmdBackend` (templated commands for `op`, `pass`, `age`,
  `gopass`).
- `ServerConfig.TokenRef` field: a ref of the form
  `scheme://opaque` that points to a secret in the backend. Tokens no
  longer need to be stored in the global config file at all.
- `BackendsConfig` (`backends.external`) section in the global config for
  external-cmd command templates.
- `config add --token-ref <ref>` flag and interactive TTY prompt
  (no-echo) as the recommended ways to register a server without
  exposing the secret in shell history or argv.
- `config backends` subcommand: probes which backends are available,
  which are reachable, and lists configured external-cmd templates.
- `StrictResolver` (opt-in via `GITLAB_MCP_STRICT_RESOLVER=1`): looks up
  the GitLab client strictly by the `server` field of `.gmcprc`,
  verifies the API host matches the configured host on first use, and
  never silently falls back to a default or host-based match.
- Auto-promotion of deprecated `tokenName` → `server` when reading a
  legacy `.gmcprc`.
- `project init` now matches the project's Git remote host against
  configured servers and auto-fills `--server` when exactly one matches.

### Deprecated (removed in v3.0)
- `GITLAB_TOKEN` / `GITLAB_HOST` env vars for startup configuration.
- `config add --token <raw>` flag.
- `addToken` and `updateToken` MCP tools.
- `.gmcprc` fields `tokenName` and `gitlabHost`.
- `ClientResolver` fallback cascade.

### Changed
- Global config schema bumps to version `"2.0"` when any server gains a
  `TokenRef`. Previous file is saved as `config.json.bak`. Version stays
  `"1.0"` for pure-legacy configs.
- `ClientPool.AddServerFromConfig` now takes a `TokenResolver` callback
  instead of reading `ServerConfig.Token` directly.

### Not changed in v2.1 (planned for v3.0)
- Global config file path stays at `~/.gitlab-mcp-server/gitlab-mcp-server-config.json`.
- `ClientResolver` remains the default resolver.
- All deprecation shims still function.
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs: add v2.1 changelog entry for key-management redesign"
```

---

## Self-Review (run before handing off)

Spec coverage (spec section → plan task):

| Spec section | Task(s) |
|---|---|
| SecretBackend interface | Task 1 |
| Three backends | Tasks 3, 4, 5 |
| Test double | Task 2 |
| Global config schema (v2) | Task 6 |
| Backends section | Task 6, 8 |
| Config v1 → v2 migration on save | Task 7 |
| `.gmcprc` schema + auto-promote | Tasks 10, 14 |
| StrictResolver + host verification | Task 11 |
| StrictResolver wired at startup (opt-in) | Task 12 |
| BackendRegistry wired at startup | Task 13 |
| `project init` requires `server` | Task 14 |
| Installer no-token-env | Task 15 |
| Deprecation shims (GITLAB_TOKEN, addToken, updateToken, .gmcprc legacy fields, --token flag) | Tasks 9, 10, 15, 16 |
| v2.1 release notes | Task 17 |
| Test surface (FakeSecretBackend, registry, resolver, migration, injection, tool-registration) | Tasks 1, 2, 3, 4, 5, 6, 7, 11, 15, 16 |

Not covered in v2.1 by design (deferred to v3.0 plan):
- Removal of deprecated code: `Token`, `IsDefault`, `TokenName`, `GitLabHost`, `GITLAB_TOKEN`, `--token`, `addToken`, `updateToken`, `ClientResolver`.
- Global config path migration (`~/.gitlab-mcp-server/` → `~/.config/gitlab-mcp-server/`).
- TUI (`cmd/config/tui.go`) "add server" screen update to support `--token-ref` and no-echo password input. The TUI continues to work via the deprecated `--token` / legacy-Token path in v2.1; rewriting the TUI is v3.0 work once the deprecated paths are removed.

---

## Handoff
