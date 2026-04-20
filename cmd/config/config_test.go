package config

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestManager creates a test config manager with a temp file
func setupTestManager(t *testing.T) *config.Manager {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.json")

	m, err := config.NewManager(configPath)
	require.NoError(t, err)

	return m
}

// setupTestManagerWithServers creates a test config manager with sample servers
func setupTestManagerWithServers(t *testing.T) *config.Manager {
	m := setupTestManager(t)

	// Add test servers
	err := m.AddServer(&config.ServerConfig{
		Name:     "default",
		Host:     "https://gitlab.com",
		Token:    "test-token-1",
		ReadOnly: false,
		UserID:   123,
		Username: "testuser1",
	})
	require.NoError(t, err)

	err = m.AddServer(&config.ServerConfig{
		Name:     "work",
		Host:     "https://gitlab.example.com",
		Token:    "test-token-2",
		ReadOnly: true,
		UserID:   456,
		Username: "testuser2",
	})
	require.NoError(t, err)

	err = m.Save()
	require.NoError(t, err)

	return m
}

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already has https",
			input:    "https://gitlab.com",
			expected: "https://gitlab.com",
		},
		{
			name:     "already has http",
			input:    "http://gitlab.com",
			expected: "http://gitlab.com",
		},
		{
			name:     "no protocol",
			input:    "gitlab.com",
			expected: "https://gitlab.com",
		},
		{
			name:     "with path",
			input:    "gitlab.example.com",
			expected: "https://gitlab.example.com",
		},
		{
			name:     "with spaces",
			input:    "  gitlab.com  ",
			expected: "https://gitlab.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeHost(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewCmd(t *testing.T) {
	m := setupTestManager(t)
	cmd := NewCmd(m)

	assert.Equal(t, "config", cmd.Use)
	assert.NotNil(t, cmd)
	assert.Equal(t, 6, len(cmd.Commands()))
}

func TestInitCommandFlags(t *testing.T) {
	m := setupTestManager(t)
	cmd := newInitCmd(m)

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup("name"))
	assert.NotNil(t, cmd.Flags().Lookup("host"))
	assert.NotNil(t, cmd.Flags().Lookup("token"))
	assert.NotNil(t, cmd.Flags().Lookup("read-only"))
	assert.NotNil(t, cmd.Flags().Lookup("non-interactive"))
}

func TestAddCommandFlags(t *testing.T) {
	m := setupTestManager(t)
	cmd := newAddCmd(m)

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup("host"))
	assert.NotNil(t, cmd.Flags().Lookup("token"))
	assert.NotNil(t, cmd.Flags().Lookup("read-only"))
}

func TestListCommandFlags(t *testing.T) {
	m := setupTestManager(t)
	cmd := newListCmd(m)

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup("json"))
}

func TestRemoveCommandFlags(t *testing.T) {
	m := setupTestManager(t)
	cmd := newRemoveCmd(m)

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup("force"))
}

func TestValidateCommand(t *testing.T) {
	m := setupTestManager(t)
	cmd := newValidateCmd(m)

	// Check no flags for validate
	assert.Equal(t, "validate [name]", cmd.Use)
}

func TestCreateGitLabClient(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid gitlab.com",
			host:    "https://gitlab.com",
			token:   "test-token",
			wantErr: false,
		},
		{
			name:    "valid custom host",
			host:    "https://gitlab.example.com",
			token:   "test-token",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := createGitLabClient(tt.host, tt.token)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestRunList_Empty(t *testing.T) {
	m := setupTestManager(t)
	var buf stringBuf

	err := runList(m, &buf)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "No servers configured")
}

func TestRunList_WithServers(t *testing.T) {
	m := setupTestManagerWithServers(t)
	var buf stringBuf

	err := runList(m, &buf)
	assert.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "default")
	assert.Contains(t, output, "work")
	assert.Contains(t, output, "https://gitlab.com")
	assert.Contains(t, output, "https://gitlab.example.com")
}

func TestRunValidateOne_NonExistent(t *testing.T) {
	m := setupTestManager(t)
	var buf stringBuf

	ctx := context.Background()
	err := validateOne(ctx, m, &buf, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	// Buffer is empty because function returns early
	assert.Empty(t, buf.String())
}

func TestRunDefault_NonExistent(t *testing.T) {
	m := setupTestManager(t)

	err := runDefault(m, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunDefault_Success(t *testing.T) {
	m := setupTestManagerWithServers(t)

	err := runDefault(m, "work")
	assert.NoError(t, err)

	// Verify
	server, _ := m.GetServer("work")
	assert.True(t, server.IsDefault)

	defaultServer, _ := m.GetDefaultServer()
	assert.Equal(t, "work", defaultServer.Name)
}

func TestRunRemove_NonExistent(t *testing.T) {
	m := setupTestManager(t)

	err := runRemove(m, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunRemove_DefaultServer(t *testing.T) {
	m := setupTestManagerWithServers(t)

	// Try to remove default server (should fail)
	err := runRemove(m, "default")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot remove default server")
}

// stringBuf is a simple buffer for testing
type stringBuf struct {
	buf []byte
}

func (s *stringBuf) Write(p []byte) (n int, err error) {
	s.buf = append(s.buf, p...)
	return len(p), nil
}

func (s *stringBuf) String() string {
	return string(s.buf)
}

func TestListTableOutput(t *testing.T) {
	m := setupTestManagerWithServers(t)
	servers := m.ListServers()

	var buf stringBuf
	err := listTableOutput(servers, &buf)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "HOST")
	assert.Contains(t, output, "default")
	assert.Contains(t, output, "work")
}

func TestListJSONOutput(t *testing.T) {
	m := setupTestManagerWithServers(t)
	servers := m.ListServers()

	var buf stringBuf
	err := listJSONOutput(servers, &buf)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "\"name\"")
	assert.Contains(t, output, "\"host\"")
	assert.Contains(t, output, "default")
	assert.Contains(t, output, "work")
}

// Helper for read from pipe
func (s *stringBuf) ReadFrom(r interface {
	Read([]byte) (int, error)
}) (int64, error) {
	n := int64(0)
	b := make([]byte, 1024)
	for {
		nn, err := r.Read(b)
		if nn > 0 {
			s.buf = append(s.buf, b[:nn]...)
			n += int64(nn)
		}
		if err != nil {
			return n, err
		}
	}
}
