package gitlab

import (
	"testing"
	"time"

	"github.com/awnumar/memguard"
)

// TestNewSecureToken tests creating a new secure token
func TestNewSecureToken(t *testing.T) {
	tokenStr := "glpat-test123456789012345"
	st, err := NewSecureToken(tokenStr)
	if err != nil {
		t.Fatalf("NewSecureToken() error = %v", err)
	}

	if st == nil {
		t.Fatal("NewSecureToken() returned nil")
	}

	// Test Get
	token, err := st.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(token) != tokenStr {
		t.Errorf("Get() = %v, want %v", string(token), tokenStr)
	}

	// Wipe the decrypted token
	memguard.WipeBytes(token)

	// Test GetString
	tokenStr2, err := st.GetString()
	if err != nil {
		t.Fatalf("GetString() error = %v", err)
	}

	if tokenStr2 != tokenStr {
		t.Errorf("GetString() = %v, want %v", tokenStr2, tokenStr)
	}
}

// TestNewSecureTokenEmpty tests that empty tokens return an error
func TestNewSecureTokenEmpty(t *testing.T) {
	_, err := NewSecureToken("")
	if err == nil {
		t.Error("NewSecureToken() with empty string should return error")
	}
}

// TestSecureTokenDestroy tests destroying a secure token
func TestSecureTokenDestroy(t *testing.T) {
	st, err := NewSecureToken("test-token")
	if err != nil {
		t.Fatalf("NewSecureToken() error = %v", err)
	}

	st.Destroy()

	if !st.IsDestroyed() {
		t.Error("IsDestroyed() should return true after Destroy()")
	}

	// After destruction, Get should return an error
	_, err = st.Get()
	if err == nil {
		t.Error("Get() after Destroy() should return error")
	}
}

// TestSecureTokenMetadata tests the SecureTokenMetadata struct
func TestSecureTokenMetadata(t *testing.T) {
	tokenStr := "glpat-testtoken123456789"
	secureToken, err := NewSecureToken(tokenStr)
	if err != nil {
		t.Fatalf("NewSecureToken() error = %v", err)
	}

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	stm := &SecureTokenMetadata{
		ID:            "test-id",
		Name:          "test-token",
		Token:         secureToken,
		GitLabHost:    "https://gitlab.example.com",
		UserID:        123,
		Username:      "testuser",
		LastValidated: time.Now(),
		IsExpiredFlag: false,
		ExpiresAt:     &expiresAt,
		CreatedAt:     time.Now(),
	}

	// Test GetToken
	retrievedToken, err := stm.GetToken()
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}

	if retrievedToken != tokenStr {
		t.Errorf("GetToken() = %v, want %v", retrievedToken, tokenStr)
	}

	// Test GetTokenBytes
	tokenBytes, err := stm.GetTokenBytes()
	if err != nil {
		t.Fatalf("GetTokenBytes() error = %v", err)
	}

	if string(tokenBytes) != tokenStr {
		t.Errorf("GetTokenBytes() = %v, want %v", string(tokenBytes), tokenStr)
	}

	// Wipe the bytes
	memguard.WipeBytes(tokenBytes)

	// Test DaysUntilExpiry
	days := stm.DaysUntilExpiry()
	if days < 29 || days > 30 {
		t.Errorf("DaysUntilExpiry() = %v, want ~30", days)
	}

	// Test IsExpired
	if stm.IsExpired() {
		t.Error("IsExpired() should return false for non-expired token")
	}

	// Test Destroy
	stm.Destroy()
	if !stm.Token.IsDestroyed() {
		t.Error("Token should be destroyed after Destroy()")
	}
}

// TestSecureTokenMetadataExpired tests expired token detection
func TestSecureTokenMetadataExpired(t *testing.T) {
	pastTime := time.Now().Add(-24 * time.Hour)
	stm := &SecureTokenMetadata{
		Name:          "expired-token",
		ExpiresAt:     &pastTime,
		IsExpiredFlag: false,
	}

	if !stm.IsExpired() {
		t.Error("IsExpired() should return true for expired token")
	}

	days := stm.DaysUntilExpiry()
	if days >= 0 {
		t.Errorf("DaysUntilExpiry() for expired token should be negative, got %v", days)
	}
}

// TestSecureTokenStore tests the SecureTokenStore
func TestSecureTokenStore(t *testing.T) {
	// Create a test logger
	testLogger := &testLogger{}
	store := NewSecureTokenStore(testLogger)

	metadata := &TokenMetadata{
		Name:          "test-token",
		GitLabHost:    "https://gitlab.com",
		UserID:        456,
		Username:      "testuser2",
		LastValidated: time.Now(),
		CreatedAt:     time.Now(),
	}

	err := store.AddToken("test", "glpat-testtoken123456789", metadata)
	if err != nil {
		t.Fatalf("AddToken() error = %v", err)
	}

	// Test HasToken
	if !store.HasToken("test") {
		t.Error("HasToken() should return true for added token")
	}

	// Test Count
	if store.Count() != 1 {
		t.Errorf("Count() = %v, want 1", store.Count())
	}

	// Test GetToken
	stm, err := store.GetToken("test")
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}

	if stm.Name != "test" {
		t.Errorf("GetToken().Name = %v, want 'test'", stm.Name)
	}

	// Test GetTokenString
	tokenStr, err := store.GetTokenString("test")
	if err != nil {
		t.Fatalf("GetTokenString() error = %v", err)
	}

	if tokenStr != "glpat-testtoken123456789" {
		t.Errorf("GetTokenString() = %v, want 'glpat-testtoken123456789'", tokenStr)
	}

	// Test ListTokens
	tokens := store.ListTokens()
	if len(tokens) != 1 {
		t.Errorf("ListTokens() = %v tokens, want 1", len(tokens))
	}

	// Test RemoveToken
	err = store.RemoveToken("test")
	if err != nil {
		t.Fatalf("RemoveToken() error = %v", err)
	}

	if store.HasToken("test") {
		t.Error("HasToken() should return false after RemoveToken()")
	}

	if store.Count() != 0 {
		t.Errorf("Count() after RemoveToken() = %v, want 0", store.Count())
	}
}

// TestSecureTokenStoreAddEmptyName tests adding token with empty name
func TestSecureTokenStoreAddEmptyName(t *testing.T) {
	store := NewSecureTokenStore(nil)

	err := store.AddToken("", "glpat-test123456789", &TokenMetadata{})
	if err == nil {
		t.Error("AddToken() with empty name should return error")
	}
}

// TestSecureTokenStoreAddNilMetadata tests adding token with nil metadata
func TestSecureTokenStoreAddNilMetadata(t *testing.T) {
	store := NewSecureTokenStore(nil)

	err := store.AddToken("test", "glpat-test123456789", nil)
	if err == nil {
		t.Error("AddToken() with nil metadata should return error")
	}
}

// TestSecureTokenStoreGetNotFound tests getting non-existent token
func TestSecureTokenStoreGetNotFound(t *testing.T) {
	store := NewSecureTokenStore(nil)

	_, err := store.GetToken("nonexistent")
	if err == nil {
		t.Error("GetToken() with non-existent name should return error")
	}

	_, err = store.GetTokenString("nonexistent")
	if err == nil {
		t.Error("GetTokenString() with non-existent name should return error")
	}
}

// TestSecureTokenStoreRemoveNotFound tests removing non-existent token
func TestSecureTokenStoreRemoveNotFound(t *testing.T) {
	store := NewSecureTokenStore(nil)

	err := store.RemoveToken("nonexistent")
	if err == nil {
		t.Error("RemoveToken() with non-existent name should return error")
	}
}

// TestSecureTokenStorePurgeAll tests purging all tokens
func TestSecureTokenStorePurgeAll(t *testing.T) {
	store := NewSecureTokenStore(nil)

	// Add multiple tokens
	metadata := &TokenMetadata{
		Name:      "test",
		CreatedAt: time.Now(),
	}

	store.AddToken("token1", "glpat-token1", metadata)
	store.AddToken("token2", "glpat-token2", metadata)
	store.AddToken("token3", "glpat-token3", metadata)

	if store.Count() != 3 {
		t.Errorf("Count() before PurgeAll() = %v, want 3", store.Count())
	}

	store.PurgeAll()

	if store.Count() != 0 {
		t.Errorf("Count() after PurgeAll() = %v, want 0", store.Count())
	}
}

// TestToTokenMetadata tests converting SecureTokenMetadata to TokenMetadata
func TestToTokenMetadata(t *testing.T) {
	tokenStr := "glpat-testtoken123456789"
	secureToken, err := NewSecureToken(tokenStr)
	if err != nil {
		t.Fatalf("NewSecureToken() error = %v", err)
	}

	stm := &SecureTokenMetadata{
		Name:          "test-token",
		Token:         secureToken,
		GitLabHost:    "https://gitlab.example.com",
		UserID:        789,
		Username:      "testuser3",
		LastValidated: time.Now(),
		CreatedAt:     time.Now(),
	}

	tm, err := stm.ToTokenMetadata()
	if err != nil {
		t.Fatalf("ToTokenMetadata() error = %v", err)
	}

	if tm.Token != tokenStr {
		t.Errorf("ToTokenMetadata().Token = %v, want %v", tm.Token, tokenStr)
	}

	if tm.Name != "test-token" {
		t.Errorf("ToTokenMetadata().Name = %v, want 'test-token'", tm.Name)
	}

	if tm.GitLabHost != "https://gitlab.example.com" {
		t.Errorf("ToTokenMetadata().GitLabHost = %v, want 'https://gitlab.example.com'", tm.GitLabHost)
	}

	if tm.UserID != 789 {
		t.Errorf("ToTokenMetadata().UserID = %v, want 789", tm.UserID)
	}

	if tm.Username != "testuser3" {
		t.Errorf("ToTokenMetadata().Username = %v, want 'testuser3'", tm.Username)
	}
}

// TestToTokenMetadataNilToken tests converting with nil token
func TestToTokenMetadataNilToken(t *testing.T) {
	stm := &SecureTokenMetadata{
		Name:      "test-token",
		Token:     nil,
		CreatedAt: time.Now(),
	}

	_, err := stm.ToTokenMetadata()
	if err == nil {
		t.Error("ToTokenMetadata() with nil Token should return error")
	}
}

// testLogger is a simple logger implementation for testing
type testLogger struct {
	infoLogs  []string
	warnLogs  []string
	debugLogs []string
	errorLogs []string
}

func (l *testLogger) Infof(format string, args ...interface{}) {
	l.infoLogs = append(l.infoLogs, format)
}

func (l *testLogger) Warnf(format string, args ...interface{}) {
	l.warnLogs = append(l.warnLogs, format)
}

func (l *testLogger) Debugf(format string, args ...interface{}) {
	l.debugLogs = append(l.debugLogs, format)
}

func (l *testLogger) Errorf(format string, args ...interface{}) {
	l.errorLogs = append(l.errorLogs, format)
}

// TestSecureTokenStoreWithNilLogger tests that store works with nil logger
func TestSecureTokenStoreWithNilLogger(t *testing.T) {
	store := NewSecureTokenStore(nil)

	metadata := &TokenMetadata{
		Name:      "test",
		CreatedAt: time.Now(),
	}

	err := store.AddToken("test", "glpat-test123456789", metadata)
	if err != nil {
		t.Fatalf("AddToken() error = %v", err)
	}

	if store.Count() != 1 {
		t.Errorf("Count() = %v, want 1", store.Count())
	}

	store.PurgeAll() // Should not panic with nil logger
}
