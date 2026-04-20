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
//
//	file://<absolute-path>#<entry-name>
type EncryptedFileBackend struct {
	path   string
	crypto *CryptoManager
	mu     sync.Mutex
}

// NewEncryptedFileBackend constructs a backend writing to the given path.
// The CryptoManager must be enabled — a nil or disabled manager is rejected
// so a backend named "Encrypted" cannot silently write plaintext secrets.
func NewEncryptedFileBackend(path string, crypto *CryptoManager) (*EncryptedFileBackend, error) {
	if crypto == nil || !crypto.IsEnabled() {
		return nil, errors.New("file backend: requires an enabled CryptoManager")
	}
	return &EncryptedFileBackend{path: path, crypto: crypto}, nil
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
	// Write to a sibling temp file first, then atomically rename into place.
	// This both prevents torn-writes on crash and enforces 0600 on every write
	// (os.WriteFile only applies perm when CREATING the file).
	tmp := e.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0600); err != nil {
		return fmt.Errorf("file backend: write: %w", err)
	}
	if err := os.Rename(tmp, e.path); err != nil {
		_ = os.Remove(tmp) // best-effort cleanup on rename failure
		return fmt.Errorf("file backend: rename: %w", err)
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
