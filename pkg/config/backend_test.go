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
	assert.Contains(t, err.Error(), "keyring")
}

func TestBackendRegistry_DuplicateScheme(t *testing.T) {
	r := NewBackendRegistry()
	require.NoError(t, r.Register(&stubBackend{scheme: "keyring"}))
	err := r.Register(&stubBackend{scheme: "keyring"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}
