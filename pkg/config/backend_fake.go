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
