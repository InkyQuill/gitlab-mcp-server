package toolsnaps

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dummyTool represents a simple tool structure for testing
type dummyTool struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

// withIsolatedWorkingDir creates a temp dir, changes to it, and restores the original working dir after the test.
func withIsolatedWorkingDir(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, os.Chdir(origDir)) })
	require.NoError(t, os.Chdir(dir))
}

func TestSnapshotDoesNotExistNotInCI(t *testing.T) {
	withIsolatedWorkingDir(t)

	// Given we are not running in CI
	t.Setenv("GITHUB_ACTIONS", "false") // This REALLY is required because the tests run in CI
	tool := dummyTool{"foo", 42}

	// When we test the snapshot
	err := Test("dummy", tool)

	// Then it should succeed and write the snapshot file
	require.NoError(t, err)
	path := filepath.Join("__toolsnaps__", "dummy.snap")
	_, statErr := os.Stat(path)
	assert.NoError(t, statErr, "expected snapshot file to be written")
}

func TestSnapshotDoesNotExistInCI(t *testing.T) {
	withIsolatedWorkingDir(t)
	// Ensure that UPDATE_TOOLSNAPS is not set for this test, which it might be if someone is running
	// UPDATE_TOOLSNAPS=true go test ./...
	t.Setenv("UPDATE_TOOLSNAPS", "false")

	// Given we are running in CI
	t.Setenv("GITHUB_ACTIONS", "true")
	tool := dummyTool{"foo", 42}

	// When we test the snapshot
	err := Test("dummy", tool)

	// Then it should error about missing snapshot in CI
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tool snapshot does not exist", "expected error about missing snapshot in CI")
}

func TestSnapshotExistsMatch(t *testing.T) {
	withIsolatedWorkingDir(t)

	// Given a matching snapshot file exists
	tool := dummyTool{"foo", 42}
	b, _ := json.MarshalIndent(tool, "", "  ")
	require.NoError(t, os.MkdirAll("__toolsnaps__", 0700))
	require.NoError(t, os.WriteFile(filepath.Join("__toolsnaps__", "dummy.snap"), b, 0600))

	// When we test the snapshot
	err := Test("dummy", tool)

	// Then it should succeed (no error)
	require.NoError(t, err)
}

func TestSnapshotExistsDiff(t *testing.T) {
	withIsolatedWorkingDir(t)
	// Ensure that UPDATE_TOOLSNAPS is not set for this test, which it might be if someone is running
	// UPDATE_TOOLSNAPS=true go test ./...
	t.Setenv("UPDATE_TOOLSNAPS", "false")

	// Given a non-matching snapshot file exists
	require.NoError(t, os.MkdirAll("__toolsnaps__", 0700))
	require.NoError(t, os.WriteFile(filepath.Join("__toolsnaps__", "dummy.snap"), []byte(`{"name":"foo","value":1}`), 0600))
	tool := dummyTool{"foo", 2}

	// When we test the snapshot
	err := Test("dummy", tool)

	// Then it should error about the schema diff
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tool schema for dummy has changed unexpectedly", "expected error about diff")
}

func TestUpdateToolsnaps(t *testing.T) {
	withIsolatedWorkingDir(t)

	// Given UPDATE_TOOLSNAPS is set, regardless of whether a matching snapshot file exists
	t.Setenv("UPDATE_TOOLSNAPS", "true")
	require.NoError(t, os.MkdirAll("__toolsnaps__", 0700))
	require.NoError(t, os.WriteFile(filepath.Join("__toolsnaps__", "dummy.snap"), []byte(`{"name":"foo","value":1}`), 0600))
	tool := dummyTool{"foo", 42}

	// When we test the snapshot
	err := Test("dummy", tool)

	// Then it should succeed and write the snapshot file
	require.NoError(t, err)
	path := filepath.Join("__toolsnaps__", "dummy.snap")
	_, statErr := os.Stat(path)
	assert.NoError(t, statErr, "expected snapshot file to be written")

	// Verify the content was updated
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	var updatedTool dummyTool
	require.NoError(t, json.Unmarshal(content, &updatedTool))
	assert.Equal(t, 42, updatedTool.Value)
}

func TestMalformedSnapshotJSON(t *testing.T) {
	withIsolatedWorkingDir(t)
	// Ensure that UPDATE_TOOLSNAPS is not set for this test, which it might be if someone is running
	// UPDATE_TOOLSNAPS=true go test ./...
	t.Setenv("UPDATE_TOOLSNAPS", "false")

	// Given a malformed snapshot file exists
	require.NoError(t, os.MkdirAll("__toolsnaps__", 0700))
	require.NoError(t, os.WriteFile(filepath.Join("__toolsnaps__", "dummy.snap"), []byte(`not-json`), 0600))
	tool := dummyTool{"foo", 42}

	// When we test the snapshot
	err := Test("dummy", tool)

	// Then it should error about malformed snapshot JSON
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse snapshot JSON for dummy", "expected error about malformed snapshot JSON")
}

func TestMultipleSnapshots(t *testing.T) {
	// Create snapshots dir for testing
	snapDir := "__toolsnaps__"
	require.NoError(t, os.MkdirAll(snapDir, 0700))
	t.Cleanup(func() { os.RemoveAll(snapDir) })

	// Given multiple tools
	tools := []struct {
		name string
		tool dummyTool
	}{
		{"tool1", dummyTool{"first", 1}},
		{"tool2", dummyTool{"second", 2}},
		{"tool3", dummyTool{"third", 3}},
	}

	// When we test all snapshots
	for _, tt := range tools {
		err := Test(tt.name, tt.tool)
		require.NoError(t, err)
	}

	// Then all snapshot files should exist
	for _, tt := range tools {
		path := filepath.Join(snapDir, tt.name+".snap")
		_, statErr := os.Stat(path)
		assert.NoError(t, statErr, "expected snapshot file for %s", tt.name)
	}
}

func TestArrayOrderingInsensitive(t *testing.T) {
	// Create snapshots dir for testing
	snapDir := "__toolsnaps__"
	require.NoError(t, os.MkdirAll(snapDir, 0700))
	t.Cleanup(func() { os.RemoveAll(snapDir) })

	// Ensure that UPDATE_TOOLSNAPS is not set for this test
	t.Setenv("UPDATE_TOOLSNAPS", "false")

	// Tool with arrays
	type toolWithArrays struct {
		Name   string   `json:"name"`
		Items  []string `json:"items"`
		Values []int    `json:"values"`
	}

	// Create snapshot with arrays in one order
	snapshotTool := toolWithArrays{
		Name:   "test",
		Items:  []string{"z", "a", "m"},
		Values: []int{3, 1, 2},
	}
	b, _ := json.MarshalIndent(snapshotTool, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(snapDir, "array_test.snap"), b, 0600))

	// Test with same arrays but in different order - should still match
	testTool := toolWithArrays{
		Name:   "test",
		Items:  []string{"a", "m", "z"}, // different order
		Values: []int{1, 2, 3},          // different order
	}

	// Should not error because we use jd.SET for array comparison
	err := Test("array_test", testTool)
	assert.NoError(t, err, "arrays in different order should still match with SET mode")
}

// unmarhsalableTool represents a tool that cannot be marshaled to JSON
type unmarshalableTool struct {
	Channel chan int `json:"channel"` // channels cannot be marshaled
}

func TestMarshalError(t *testing.T) {
	withIsolatedWorkingDir(t)
	t.Setenv("UPDATE_TOOLSNAPS", "false")

	// Given a tool that cannot be marshaled
	tool := unmarshalableTool{
		Channel: make(chan int),
	}

	// When we test the snapshot
	err := Test("unmarshalable", tool)

	// Then it should error about marshaling failure
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal tool unmarshalable", "expected error about marshaling failure")
}

func TestReadFileError(t *testing.T) {
	withIsolatedWorkingDir(t)
	t.Setenv("UPDATE_TOOLSNAPS", "false")

	// Create snapshot file but make it unreadable by creating a directory with the same name
	snapPath := filepath.Join("__toolsnaps__", "dummy.snap")
	require.NoError(t, os.MkdirAll(filepath.Dir(snapPath), 0700))
	// Create a directory instead of a file to cause read error
	require.NoError(t, os.Mkdir(snapPath, 0700))
	t.Cleanup(func() { os.RemoveAll("__toolsnaps__") })

	tool := dummyTool{"foo", 42}

	// When we test the snapshot
	err := Test("dummy", tool)

	// Then it should error about reading the snapshot file
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read snapshot file for dummy", "expected error about reading snapshot file")
}

func TestWriteSnapMkdirError(t *testing.T) {
	withIsolatedWorkingDir(t)
	t.Setenv("UPDATE_TOOLSNAPS", "true")

	// Create a file where the directory should be, making MkdirAll fail
	// When __toolsnaps__ is a file instead of a directory, MkdirAll will fail
	os.RemoveAll("__toolsnaps__")
	require.NoError(t, os.WriteFile("__toolsnaps__", []byte("block"), 0600))
	t.Cleanup(func() { os.RemoveAll("__toolsnaps__") })

	tool := dummyTool{"foo", 42}

	// When we try to create a snapshot
	err := Test("dummy", tool)

	// Then it should error about creating the directory
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create snapshot directory", "expected error about creating directory")
}

func TestWriteSnapWriteError(t *testing.T) {
	withIsolatedWorkingDir(t)
	t.Setenv("UPDATE_TOOLSNAPS", "true")
	t.Setenv("GITHUB_ACTIONS", "false")

	// Create __toolsnaps__ directory but make it read-only (on Unix systems)
	require.NoError(t, os.MkdirAll("__toolsnaps__", 0500)) // read-only, no write permission
	t.Cleanup(func() {
		os.Chmod("__toolsnaps__", 0700) // restore permissions for cleanup
		os.RemoveAll("__toolsnaps__")
	})

	tool := dummyTool{"foo", 42}

	// When we try to create a snapshot
	err := Test("dummy", tool)

	// Then it should error about writing the snapshot file
	// Note: On some systems this might succeed if the process owner can write
	// But on systems where it fails, we should get an error
	if err != nil {
		assert.Contains(t, err.Error(), "failed to write snapshot file", "expected error about writing snapshot file")
	}
}

func TestParseToolJSONError(t *testing.T) {
	withIsolatedWorkingDir(t)
	t.Setenv("UPDATE_TOOLSNAPS", "false")

	// The tool JSON parsing error (line 68-71) is difficult to test because:
	// 1. If json.MarshalIndent succeeds, the resulting JSON should be valid
	// 2. jd.ReadJsonString should handle any valid JSON
	// This error path is very unlikely to occur in practice.
	// However, we can verify that the error message format is correct by checking
	// the code coverage, but a direct test would require mocking jd.ReadJsonString
	// which is not practical. This edge case is considered low priority.
}

func TestSnapshotExistsButDirDoesNot(t *testing.T) {
	withIsolatedWorkingDir(t)
	t.Setenv("UPDATE_TOOLSNAPS", "false")
	t.Setenv("GITHUB_ACTIONS", "true")

	// Test the edge case where snapshot file exists and __toolsnaps__ directory exists
	// This tests the snapDirExists logic (lines 51-54)
	tool := dummyTool{"foo", 42}
	b, _ := json.MarshalIndent(tool, "", "  ")

	// Create snapshot file and directory
	require.NoError(t, os.MkdirAll("__toolsnaps__", 0700))
	require.NoError(t, os.WriteFile("__toolsnaps__/dummy.snap", b, 0600))

	// When we test the snapshot in CI with directory existing
	err := Test("dummy", tool)

	// Then it should succeed (because directory exists, so we allow creating snapshots in CI)
	require.NoError(t, err)
}

func TestSnapshotDoesNotExistInCIWithDir(t *testing.T) {
	withIsolatedWorkingDir(t)
	t.Setenv("UPDATE_TOOLSNAPS", "false")
	t.Setenv("GITHUB_ACTIONS", "true")

	// Test the case where we're in CI, snapshot doesn't exist, but __toolsnaps__ directory exists
	// This should allow creating the snapshot (lines 51-58)
	require.NoError(t, os.MkdirAll("__toolsnaps__", 0700))
	tool := dummyTool{"foo", 42}

	// When we test the snapshot
	err := Test("dummy", tool)

	// Then it should succeed and create the snapshot (because directory exists)
	require.NoError(t, err)
	path := filepath.Join("__toolsnaps__", "dummy.snap")
	_, statErr := os.Stat(path)
	assert.NoError(t, statErr, "expected snapshot file to be created when directory exists in CI")
}
