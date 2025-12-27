package gitlab

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	gl "gitlab.com/gitlab-org/api/client-go"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientResolver(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	pool := NewClientPool(store, logger)

	cr := NewClientResolver(pool, "default", logger)

	assert.NotNil(t, cr)
	assert.Same(t, pool, cr.pool)
	assert.Equal(t, "default", cr.defaultServer)
	assert.Same(t, logger, cr.logger)
}

func TestClientResolver_Resolve_NoConfig(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	pool := NewClientPool(store, logger)

	// Add default client
	mockClient := &gl.Client{}
	err := pool.AddClient("default", mockClient)
	require.NoError(t, err)

	cr := NewClientResolver(pool, "default", logger)

	// Create a temp directory with no .gmcprc
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Resolve should return default client
	client, name, err := cr.Resolve(context.Background())
	require.NoError(t, err)
	assert.Same(t, mockClient, client)
	assert.Equal(t, "default", name)
}

func TestClientResolver_Resolve_WithTokenName(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	pool := NewClientPool(store, logger)

	// Add multiple clients
	mockClient1 := &gl.Client{}
	mockClient2 := &gl.Client{}
	err := pool.AddClient("default", mockClient1)
	require.NoError(t, err)
	err = pool.AddClient("work-token", mockClient2)
	require.NoError(t, err)

	cr := NewClientResolver(pool, "default", logger)

	// Create a temp directory with .gmcprc
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gmcprc")
	configContent := `{
  "projectId": "group/project",
  "gitlabHost": "https://gitlab.example.com",
  "tokenName": "work-token"
}`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Resolve should use tokenName from config
	client, name, err := cr.Resolve(context.Background())
	require.NoError(t, err)
	assert.Same(t, mockClient2, client, "Should use client specified by tokenName")
	assert.Equal(t, "work-token", name)
}

func TestClientResolver_Resolve_InvalidTokenNameFallback(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	pool := NewClientPool(store, logger)

	// Add default client only
	mockClient := &gl.Client{}
	err := pool.AddClient("default", mockClient)
	require.NoError(t, err)

	cr := NewClientResolver(pool, "default", logger)

	// Create a temp directory with .gmcprc pointing to non-existent token
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gmcprc")
	configContent := `{
  "projectId": "group/project",
  "tokenName": "non-existent-token"
}`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Resolve should fall back to default
	client, name, err := cr.Resolve(context.Background())
	require.NoError(t, err)
	assert.Same(t, mockClient, client)
	assert.Equal(t, "default", name)
}

func TestClientResolver_Resolve_HostMatching(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	pool := NewClientPool(store, logger)

	// Add clients for different hosts
	mockClient1 := &gl.Client{}
	mockClient2 := &gl.Client{}
	err := pool.AddClient("default", mockClient1)
	require.NoError(t, err)
	err = pool.AddClient("https://gitlab.example.com", mockClient2)
	require.NoError(t, err)

	cr := NewClientResolver(pool, "default", logger)

	// Create a temp directory with .gmcprc specifying host
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gmcprc")
	configContent := `{
  "projectId": "group/project",
  "gitlabHost": "https://gitlab.example.com"
}`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Resolve should match by host
	client, name, err := cr.Resolve(context.Background())
	require.NoError(t, err)
	assert.Same(t, mockClient2, client, "Should use client matching host")
	assert.Equal(t, "https://gitlab.example.com", name)
}

func TestClientResolver_Resolve_HostMatchingFallback(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	pool := NewClientPool(store, logger)

	// Add only default client
	mockClient := &gl.Client{}
	err := pool.AddClient("default", mockClient)
	require.NoError(t, err)

	cr := NewClientResolver(pool, "default", logger)

	// Create a temp directory with .gmcprc specifying non-existent host
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gmcprc")
	configContent := `{
  "projectId": "group/project",
  "gitlabHost": "https://gitlab.example.com"
}`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Resolve should fall back to default
	client, name, err := cr.Resolve(context.Background())
	require.NoError(t, err)
	assert.Same(t, mockClient, client)
	assert.Equal(t, "default", name)
}

func TestClientResolver_Resolve_GitLabComHost(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	pool := NewClientPool(store, logger)

	// Add clients
	mockClient1 := &gl.Client{}
	mockClient2 := &gl.Client{}
	err := pool.AddClient("default", mockClient1)
	require.NoError(t, err)
	err = pool.AddClient("https://gitlab.com", mockClient2)
	require.NoError(t, err)

	cr := NewClientResolver(pool, "default", logger)

	// Create a temp directory with .gmcprc specifying gitlab.com
	// Note: gitlab.com hosts skip host matching logic (line 59)
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gmcprc")
	configContent := `{
  "projectId": "group/project",
  "gitlabHost": "https://gitlab.com"
}`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Resolve should skip host matching for gitlab.com and use default
	client, name, err := cr.Resolve(context.Background())
	require.NoError(t, err)
	assert.Same(t, mockClient1, client, "Should skip host matching for gitlab.com")
	assert.Equal(t, "default", name)
}

func TestClientResolver_Resolve_DefaultServer(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	pool := NewClientPool(store, logger)

	// Add clients
	mockClient1 := &gl.Client{}
	mockClient2 := &gl.Client{}
	err := pool.AddClient("default", mockClient1)
	require.NoError(t, err)
	err = pool.AddClient("custom-server", mockClient2)
	require.NoError(t, err)

	cr := NewClientResolver(pool, "custom-server", logger)

	// Create a temp directory with .gmcprc with no specific token
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gmcprc")
	configContent := `{
  "projectId": "group/project"
}`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Resolve should use defaultServer from resolver
	client, name, err := cr.Resolve(context.Background())
	require.NoError(t, err)
	assert.Same(t, mockClient2, client, "Should use defaultServer")
	assert.Equal(t, "custom-server", name)
}

func TestClientResolver_Resolve_DefaultServerNotFound(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	pool := NewClientPool(store, logger)

	// Add only default client
	mockClient := &gl.Client{}
	err := pool.AddClient("default", mockClient)
	require.NoError(t, err)

	cr := NewClientResolver(pool, "non-existent-server", logger)

	// Create a temp directory with .gmcprc
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gmcprc")
	configContent := `{
  "projectId": "group/project"
}`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Resolve should fall back to GetDefaultClient when defaultServer not found
	client, name, err := cr.Resolve(context.Background())
	require.NoError(t, err)
	assert.Same(t, mockClient, client)
	assert.Equal(t, "default", name)
}

func TestClientResolver_Resolve_NoClients(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	pool := NewClientPool(store, logger)

	// Don't add any clients
	cr := NewClientResolver(pool, "default", logger)

	// Create a temp directory with .gmcprc
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gmcprc")
	configContent := `{
  "projectId": "group/project"
}`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Resolve should return error when no clients available
	client, name, err := cr.Resolve(context.Background())
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Empty(t, name)
	assert.Contains(t, err.Error(), "no clients available")
}

func TestClientResolver_GetClientFn(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	pool := NewClientPool(store, logger)

	// Add client
	mockClient := &gl.Client{}
	err := pool.AddClient("default", mockClient)
	require.NoError(t, err)

	cr := NewClientResolver(pool, "default", logger)
	getClientFn := cr.GetClientFn()

	// Create temp directory (no config)
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// GetClientFn should return client without name
	ctx := context.Background()
	client, err := getClientFn(ctx)
	require.NoError(t, err)
	assert.Same(t, mockClient, client)
}

func TestClientResolver_GetClientFn_Error(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	pool := NewClientPool(store, logger)

	// Don't add any clients
	cr := NewClientResolver(pool, "default", logger)
	getClientFn := cr.GetClientFn()

	// Create temp directory (no config)
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// GetClientFn should return error
	ctx := context.Background()
	client, err := getClientFn(ctx)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestClientResolver_ResolveForProject(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	pool := NewClientPool(store, logger)

	// Add client
	mockClient := &gl.Client{}
	err := pool.AddClient("default", mockClient)
	require.NoError(t, err)

	cr := NewClientResolver(pool, "default", logger)

	// Create temp directory (no config)
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// ResolveForProject should behave like Resolve but with project logging
	ctx := context.Background()
	projectID := "group/project"
	client, name, err := cr.ResolveForProject(ctx, projectID)
	require.NoError(t, err)
	assert.Same(t, mockClient, client)
	assert.Equal(t, "default", name)
}

func TestClientResolver_ResolveForProject_Error(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	pool := NewClientPool(store, logger)

	// Don't add any clients
	cr := NewClientResolver(pool, "default", logger)

	// Create temp directory (no config)
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// ResolveForProject should return error
	ctx := context.Background()
	projectID := "group/project"
	client, name, err := cr.ResolveForProject(ctx, projectID)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Empty(t, name)
}

func TestClientResolver_PriorityOrder(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	pool := NewClientPool(store, logger)

	// Add multiple clients
	mockDefault := &gl.Client{}
	mockHost := &gl.Client{}
	mockToken := &gl.Client{}

	err := pool.AddClient("default", mockDefault)
	require.NoError(t, err)
	err = pool.AddClient("https://gitlab.example.com", mockHost)
	require.NoError(t, err)
	err = pool.AddClient("specific-token", mockToken)
	require.NoError(t, err)

	cr := NewClientResolver(pool, "default", logger)

	// Create .gmcprc with both tokenName and gitlabHost
	// Priority 1: tokenName should win
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gmcprc")
	configContent := `{
  "projectId": "group/project",
  "gitlabHost": "https://gitlab.example.com",
  "tokenName": "specific-token"
}`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// tokenName should take priority over host matching
	client, name, err := cr.Resolve(context.Background())
	require.NoError(t, err)
	assert.Same(t, mockToken, client, "tokenName should have highest priority")
	assert.Equal(t, "specific-token", name)
}

func TestClientResolver_MultiLevelFallback(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)
	store := NewTokenStore()
	pool := NewClientPool(store, logger)

	// Add clients
	mockDefault := &gl.Client{}
	err := pool.AddClient("default", mockDefault)
	require.NoError(t, err)

	cr := NewClientResolver(pool, "non-existent-default", logger)

	// Create .gmcprc with invalid tokenName and non-matching host
	// Should fallback through: tokenName -> host -> defaultServer -> GetDefaultClient
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gmcprc")
	configContent := `{
  "projectId": "group/project",
  "gitlabHost": "https://nonexistent.example.com",
  "tokenName": "non-existent-token"
}`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Should fallback all the way to GetDefaultClient
	client, name, err := cr.Resolve(context.Background())
	require.NoError(t, err)
	assert.Same(t, mockDefault, client)
	assert.Equal(t, "default", name)
}
