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
	_ = path
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

	r := NewStrictResolver(pool, map[string]string{"work": "https://gitlab.other.invalid"}, logrus.New())
	_, _, err = r.Resolve(context.Background())
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "host") || errors.Is(err, ErrHostMismatch))
}
