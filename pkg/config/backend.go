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
	Resolve(ctx context.Context, ref string) (string, error)
	Store(ctx context.Context, name, secret string) (ref string, err error)
	Delete(ctx context.Context, ref string) error
	Scheme() string
}

// ParseRef splits a ref like "scheme://opaque" into its parts.
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

// NewBackendRegistry creates an empty registry.
func NewBackendRegistry() *BackendRegistry {
	return &BackendRegistry{backends: map[string]SecretBackend{}}
}

// Register adds a backend to the registry. Returns an error if the scheme
// is empty or already registered.
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

// Resolve dispatches to the backend matching ref's scheme and returns the
// resolved secret value.
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

// Store dispatches to the backend matching scheme and stores the secret
// under name, returning the resulting ref.
func (r *BackendRegistry) Store(ctx context.Context, scheme, name, secret string) (string, error) {
	b, err := r.get(scheme)
	if err != nil {
		return "", err
	}
	return b.Store(ctx, name, secret)
}

// Delete dispatches to the backend matching ref's scheme and removes the secret.
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

// Schemes returns the sorted list of registered backend schemes.
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
