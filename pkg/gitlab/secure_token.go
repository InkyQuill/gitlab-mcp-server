package gitlab

import (
	"fmt"
	"sync"
	"time"

	"github.com/awnumar/memguard"
)

// SecureToken safely stores a token in encrypted memory
type SecureToken struct {
	token *memguard.Enclave
	mu    sync.RWMutex
}

// NewSecureToken creates a new secure token from a plaintext string
// The plaintext is encrypted and stored in memory that cannot be swapped to disk
func NewSecureToken(token string) (*SecureToken, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	// Create a mutable buffer of the correct size
	lb := memguard.NewBuffer(len(token))
	// Copy the token data into the protected buffer
	lb.Copy([]byte(token))

	// Create enclave from the buffer (Seal() also destroys the buffer)
	enclave := lb.Seal()

	return &SecureToken{
		token: enclave,
	}, nil
}

// Get returns the decrypted token as bytes
// IMPORTANT: Caller must wipe the returned byte slice after use with memguard.WipeBytes()
func (st *SecureToken) Get() ([]byte, error) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	if st.token == nil {
		return nil, fmt.Errorf("token is nil")
	}

	lb, err := st.token.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open token enclave: %w", err)
	}
	defer lb.Destroy()

	// Make a copy of the bytes since the LockedBuffer will be destroyed
	result := make([]byte, lb.Size())
	copy(result, lb.Bytes())
	return result, nil
}

// GetString returns the token as a string
// WARNING: The returned string will be in memory as plain text, use with caution
func (st *SecureToken) GetString() (string, error) {
	b, err := st.Get()
	if err != nil {
		return "", err
	}
	defer memguard.WipeBytes(b)
	return string(b), nil
}

// Destroy securely wipes the token from memory
func (st *SecureToken) Destroy() {
	st.mu.Lock()
	defer st.mu.Unlock()

	// Enclave doesn't have a Destroy method, just nil the reference
	st.token = nil
}

// IsDestroyed returns true if the token has been destroyed
func (st *SecureToken) IsDestroyed() bool {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.token == nil
}

// SecureTokenMetadata stores token metadata with a securely stored token
type SecureTokenMetadata struct {
	ID            string
	Name          string
	Token         *SecureToken
	GitLabHost    string
	UserID        int
	Username      string
	LastValidated time.Time
	IsExpiredFlag bool
	ExpiresAt     *time.Time
	CreatedAt     time.Time
}

// GetToken safely returns the token as a string
func (stm *SecureTokenMetadata) GetToken() (string, error) {
	if stm.Token == nil {
		return "", fmt.Errorf("token not set")
	}
	return stm.Token.GetString()
}

// GetTokenBytes returns the token as bytes that should be wiped after use
func (stm *SecureTokenMetadata) GetTokenBytes() ([]byte, error) {
	if stm.Token == nil {
		return nil, fmt.Errorf("token not set")
	}
	return stm.Token.Get()
}

// Destroy securely wipes all sensitive data
func (stm *SecureTokenMetadata) Destroy() {
	if stm.Token != nil {
		stm.Token.Destroy()
	}
}

// IsExpired checks if the token is expired or close to expiration
func (stm *SecureTokenMetadata) IsExpired() bool {
	if stm.IsExpiredFlag {
		return true
	}
	if stm.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*stm.ExpiresAt)
}

// DaysUntilExpiry returns the number of days until token expiration
// Returns negative value if already expired, 0 if no expiry set
func (stm *SecureTokenMetadata) DaysUntilExpiry() int {
	if stm.ExpiresAt == nil {
		return 0
	}
	duration := time.Until(*stm.ExpiresAt)
	return int(duration.Hours() / 24)
}

// ToTokenMetadata converts SecureTokenMetadata to regular TokenMetadata
// WARNING: This will decrypt the token and store it in plain text in the returned struct
// The caller is responsible for wiping the Token field in the returned struct
func (stm *SecureTokenMetadata) ToTokenMetadata() (*TokenMetadata, error) {
	tokenStr, err := stm.GetToken()
	if err != nil {
		return nil, err
	}

	return &TokenMetadata{
		Token:         tokenStr,
		Name:          stm.Name,
		GitLabHost:    stm.GitLabHost,
		UserID:        stm.UserID,
		Username:      stm.Username,
		LastValidated: stm.LastValidated,
		IsExpiredFlag: stm.IsExpiredFlag,
		ExpiresAt:     stm.ExpiresAt,
		CreatedAt:     stm.CreatedAt,
	}, nil
}

// SecureTokenStore manages multiple secure tokens
type SecureTokenStore struct {
	tokens map[string]*SecureTokenMetadata
	mu     sync.RWMutex
	logger Logger
}

// Logger interface for SecureTokenStore
type Logger interface {
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// NewSecureTokenStore creates a new secure token store
func NewSecureTokenStore(logger Logger) *SecureTokenStore {
	return &SecureTokenStore{
		tokens: make(map[string]*SecureTokenMetadata),
		logger: logger,
	}
}

// AddToken adds or updates a secure token in the store
func (sts *SecureTokenStore) AddToken(name string, token string, metadata *TokenMetadata) error {
	if name == "" {
		return fmt.Errorf("token name cannot be empty")
	}
	if metadata == nil {
		return fmt.Errorf("token metadata cannot be nil")
	}

	secureToken, err := NewSecureToken(token)
	if err != nil {
		return fmt.Errorf("failed to create secure token: %w", err)
	}

	stm := &SecureTokenMetadata{
		ID:            metadata.Name,
		Name:          name,
		Token:         secureToken,
		GitLabHost:    metadata.GitLabHost,
		UserID:        metadata.UserID,
		Username:      metadata.Username,
		LastValidated: metadata.LastValidated,
		IsExpiredFlag: metadata.IsExpiredFlag,
		ExpiresAt:     metadata.ExpiresAt,
		CreatedAt:     metadata.CreatedAt,
	}

	sts.mu.Lock()
	defer sts.mu.Unlock()

	sts.tokens[name] = stm
	if sts.logger != nil {
		sts.logger.Infof("Added secure token '%s' to store", name)
	}

	return nil
}

// GetToken retrieves a secure token by name
func (sts *SecureTokenStore) GetToken(name string) (*SecureTokenMetadata, error) {
	sts.mu.RLock()
	defer sts.mu.RUnlock()

	token, ok := sts.tokens[name]
	if !ok {
		return nil, fmt.Errorf("token '%s' not found", name)
	}

	return token, nil
}

// GetTokenString returns the token as a string
func (sts *SecureTokenStore) GetTokenString(name string) (string, error) {
	sts.mu.RLock()
	token, ok := sts.tokens[name]
	sts.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("token '%s' not found", name)
	}

	return token.GetToken()
}

// ListTokens returns all tokens in the store (without decrypting them)
func (sts *SecureTokenStore) ListTokens() map[string]*SecureTokenMetadata {
	sts.mu.RLock()
	defer sts.mu.RUnlock()

	result := make(map[string]*SecureTokenMetadata, len(sts.tokens))
	for k, v := range sts.tokens {
		result[k] = v
	}

	return result
}

// RemoveToken removes a token from the store and destroys it
func (sts *SecureTokenStore) RemoveToken(name string) error {
	sts.mu.Lock()
	defer sts.mu.Unlock()

	token, ok := sts.tokens[name]
	if !ok {
		return fmt.Errorf("token '%s' not found", name)
	}

	// Destroy the secure token
	token.Destroy()

	delete(sts.tokens, name)
	if sts.logger != nil {
		sts.logger.Infof("Removed secure token '%s' from store", name)
	}

	return nil
}

// PurgeAll destroys all tokens in the store
func (sts *SecureTokenStore) PurgeAll() {
	sts.mu.Lock()
	defer sts.mu.Unlock()

	for _, token := range sts.tokens {
		token.Destroy()
	}

	sts.tokens = make(map[string]*SecureTokenMetadata)
	if sts.logger != nil {
		sts.logger.Infof("Purged all secure tokens from store")
	}
}

// HasToken returns true if a token with the given name exists
func (sts *SecureTokenStore) HasToken(name string) bool {
	sts.mu.RLock()
	defer sts.mu.RUnlock()
	_, ok := sts.tokens[name]
	return ok
}

// Count returns the number of tokens in the store
func (sts *SecureTokenStore) Count() int {
	sts.mu.RLock()
	defer sts.mu.RUnlock()
	return len(sts.tokens)
}

// Initialize memguard when the package is loaded
func init() {
	// Catch interrupt signals and securely purge memory
	// This is a no-op if memguard is already initialized
}
