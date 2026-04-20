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
	b, err := NewEncryptedFileBackend(path, cm)
	require.NoError(t, err)
	ctx := context.Background()

	ref, err := b.Store(ctx, "work", "glpat-xyz")
	require.NoError(t, err)
	assert.Equal(t, "file://"+path+"#work", ref)

	ref2, err := b.Store(ctx, "personal", "glpat-abc")
	require.NoError(t, err)

	got, err := b.Resolve(ctx, ref)
	require.NoError(t, err)
	assert.Equal(t, "glpat-xyz", got)

	got2, err := b.Resolve(ctx, ref2)
	require.NoError(t, err)
	assert.Equal(t, "glpat-abc", got2)

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "glpat-xyz")
	assert.NotContains(t, string(raw), "glpat-abc")

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

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
	b, err := NewEncryptedFileBackend(path, cm)
	require.NoError(t, err)

	_, err = b.Resolve(context.Background(), "file://"+path+"#x")
	require.ErrorIs(t, err, ErrSecretNotFound)
}

func TestEncryptedFileBackend_ParseFileRef(t *testing.T) {
	p, entry, err := parseFileRef("file:///tmp/x.enc#work")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/x.enc", p)
	assert.Equal(t, "work", entry)

	_, _, err = parseFileRef("file:///tmp/x.enc")
	require.Error(t, err)
	_, _, err = parseFileRef("keyring://work")
	require.Error(t, err)
}

func TestEncryptedFileBackend_RequiresEnabledCrypto(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.enc")

	disabled, err := NewCryptoManager(false) // useKeyring=false → IsEnabled()==false
	require.NoError(t, err)
	_, err = NewEncryptedFileBackend(path, disabled)
	require.Error(t, err, "disabled CryptoManager must be rejected")

	_, err = NewEncryptedFileBackend(path, nil)
	require.Error(t, err, "nil CryptoManager must be rejected")
}

func TestEncryptedFileBackend_EnforcesModeOnExistingFile(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")

	// Pre-create the file with wide-open perms to simulate a misbehaving admin
	// or a restore-from-backup situation.
	require.NoError(t, os.WriteFile(path, []byte("{}"), 0644))
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0644), info.Mode().Perm(), "precondition: file starts 0644")

	cm, err := NewCryptoManager(true)
	require.NoError(t, err)
	b, err := NewEncryptedFileBackend(path, cm)
	require.NoError(t, err)

	_, err = b.Store(context.Background(), "work", "glpat-x")
	require.NoError(t, err)

	info, err = os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm(),
		"Store must tighten mode to 0600 even when file pre-existed at 0644")
}

func TestEncryptedFileBackend_NoTmpLeftBehindOnSuccess(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")

	cm, err := NewCryptoManager(true)
	require.NoError(t, err)
	b, err := NewEncryptedFileBackend(path, cm)
	require.NoError(t, err)

	_, err = b.Store(context.Background(), "work", "glpat-x")
	require.NoError(t, err)

	_, err = os.Stat(path + ".tmp")
	require.True(t, os.IsNotExist(err), "temp file must not be left behind after a successful Store")
}
