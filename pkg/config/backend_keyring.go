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
