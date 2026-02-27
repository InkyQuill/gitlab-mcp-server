package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeJoinPath(t *testing.T) {
	base := "/home/user/project"

	tests := []struct {
		name    string
		base    string
		path    string
		want    string
		wantErr bool
	}{
		{"normal file", base, "config.json", "/home/user/project/config.json", false},
		{"nested file", base, "subdir/config.json", "/home/user/project/subdir/config.json", false},
		{"dot prefix", base, "./config.json", "/home/user/project/config.json", false},
		{"traversal attack", base, "../etc/passwd", "", true},
		{"traversal attack nested", base, "subdir/../../etc/passwd", "", true},
		{"absolute path", base, "/etc/passwd", "", true},
		{"double slash", base, "subdir//config.json", "/home/user/project/subdir/config.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SafeJoinPath(tt.base, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("SafeJoinPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SafeJoinPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"normal", "config.json", false},
		{"nested", "subdir/config.json", false},
		{"traversal", "../etc/passwd", true},
		{"traversal in middle", "subdir/../../etc/passwd", true},
		{"traversal at start", "../config.json", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidatePath(tt.path); (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSafeReadFile(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Create a test file
	testData := []byte("test content")
	testFile := "test.txt"
	testPath := filepath.Join(tmpDir, testFile)
	if err := os.WriteFile(testPath, testData, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a subdirectory with a file
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}
	subFile := filepath.Join(subDir, "sub.txt")
	if err := os.WriteFile(subFile, []byte("sub content"), 0644); err != nil {
		t.Fatalf("failed to create subfile: %v", err)
	}

	tests := []struct {
		name    string
		baseDir string
		file    string
		want    []byte
		wantErr bool
	}{
		{"read normal file", tmpDir, "test.txt", testData, false},
		{"read nested file", tmpDir, "subdir/sub.txt", []byte("sub content"), false},
		{"traversal attack", tmpDir, "../etc/passwd", nil, true},
		{"absolute path", tmpDir, "/etc/passwd", nil, true},
		{"directory instead_of_file", tmpDir, "subdir", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SafeReadFile(tt.baseDir, tt.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("SafeReadFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != string(tt.want) {
				t.Errorf("SafeReadFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeWriteFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		baseDir string
		file    string
		data    []byte
		wantErr bool
	}{
		{"write normal file", tmpDir, "output.txt", []byte("test data"), false},
		{"write nested file", tmpDir, "subdir/output.txt", []byte("nested data"), false},
		{"traversal attack", tmpDir, "../etc/passwd", []byte("malicious"), true},
		{"absolute path", tmpDir, "/etc/passwd", []byte("malicious"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SafeWriteFile(tt.baseDir, tt.file, tt.data, 0644)
			if (err != nil) != tt.wantErr {
				t.Errorf("SafeWriteFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify file was written correctly
				fullPath := filepath.Join(tt.baseDir, tt.file)
				got, err := os.ReadFile(fullPath)
				if err != nil {
					t.Errorf("failed to read written file: %v", err)
					return
				}
				if string(got) != string(tt.data) {
					t.Errorf("file content = %v, want %v", got, tt.data)
				}
			}
		})
	}
}
