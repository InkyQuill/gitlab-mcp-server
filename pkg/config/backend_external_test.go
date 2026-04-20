package config

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runCmd runs an arbitrary command from `dir` and returns combined stdout+stderr.
func runCmd(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

// buildArgvEcho compiles a throwaway helper binary that writes each argv
// element on its own line, followed by a final "SECRET_VALUE" line. Returns the
// binary path. Skips on Windows.
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
func main() {
    for _, a := range os.Args[1:] {
        fmt.Fprintln(os.Stderr, a)
    }
    fmt.Println("SECRET_VALUE")
}
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
	assert.Equal(t, "SECRET_VALUE", got)
}

func TestExternalCmdBackend_ShellMetacharsArePassedLiterally(t *testing.T) {
	bin := buildArgvEcho(t)
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

func TestExternalCmdBackend_ReturnsVerbatimStdoutTrimmed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX helper")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	bin := filepath.Join(dir, "multiline")
	require.NoError(t, os.WriteFile(src, []byte(`package main
import "fmt"
func main() {
    fmt.Print("line1\nline2\nline3\n\n\n")
}
`), 0600))
	out, err := runCmd(dir, "go", "build", "-o", bin, src)
	require.NoError(t, err, "build: %s", out)

	b := NewExternalCmdBackend(map[string]string{"op": bin + " %s"})
	got, err := b.Resolve(context.Background(), "op://ignored")
	require.NoError(t, err)
	assert.Equal(t, "line1\nline2\nline3", got,
		"multi-line output must be returned verbatim with only trailing whitespace trimmed")
}

func TestExternalCmdBackend_IncludesStderrOnFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX helper")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	bin := filepath.Join(dir, "failer")
	require.NoError(t, os.WriteFile(src, []byte(`package main
import ("fmt"; "os")
func main() {
    fmt.Fprintln(os.Stderr, "vault is locked: run 'op signin'")
    os.Exit(1)
}
`), 0600))
	out, err := runCmd(dir, "go", "build", "-o", bin, src)
	require.NoError(t, err, "build: %s", out)

	b := NewExternalCmdBackend(map[string]string{"op": bin + " %s"})
	_, err = b.Resolve(context.Background(), "op://ignored")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vault is locked",
		"stderr must be surfaced in the wrapped error so operators can debug")
}

func TestExternalCmdBackend_AppliesDefaultTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX helper")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	bin := filepath.Join(dir, "hang")
	require.NoError(t, os.WriteFile(src, []byte(`package main
import "time"
func main() { time.Sleep(30 * time.Second) }
`), 0600))
	out, err := runCmd(dir, "go", "build", "-o", bin, src)
	require.NoError(t, err, "build: %s", out)

	// The backend's default 10s timeout is too long for a unit test. We pass
	// a ctx with a 200ms deadline. This exercises the "ctx with deadline" path
	// (the wrapping logic skips injecting a default when one is already set).
	b := NewExternalCmdBackend(map[string]string{"op": bin + " %s"})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = b.Resolve(ctx, "op://ignored")
	elapsed := time.Since(start)
	require.Error(t, err, "hung command with timed-out ctx must return an error")
	assert.Less(t, elapsed, 2*time.Second,
		"resolve must honor the caller's ctx deadline, not wait for the 10s default")
}
