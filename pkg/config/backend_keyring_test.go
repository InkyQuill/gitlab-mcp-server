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
