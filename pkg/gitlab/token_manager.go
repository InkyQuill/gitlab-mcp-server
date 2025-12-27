package gitlab

import (
	"context"
	"fmt"
	"sync"
	"time"

	gl "gitlab.com/gitlab-org/api/client-go"
)

// TokenMetadata stores information about a GitLab access token
type TokenMetadata struct {
	Token          string     `json:"-"`                // Token value (not persisted to JSON)
	Name           string     `json:"name"`              // Server/token name (e.g., "work", "personal")
	GitLabHost     string     `json:"gitlabHost"`        // GitLab host URL
	ExpiresAt      *time.Time `json:"expiresAt,omitempty"` // Token expiration time
	CreatedAt      time.Time  `json:"createdAt"`         // When token was added to store
	LastValidated  time.Time  `json:"lastValidated"`     // Last successful validation
	UserID         int        `json:"userId,omitempty"`  // GitLab user ID
	Username       string     `json:"username,omitempty"` // GitLab username
	IsExpiredFlag  bool       `json:"isExpired"`         // Flag indicating if token is expired
}

// IsExpired checks if the token is expired or close to expiration
func (tm *TokenMetadata) IsExpired() bool {
	if tm.IsExpiredFlag {
		return true
	}
	if tm.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*tm.ExpiresAt)
}

// DaysUntilExpiry returns the number of days until token expiration
// Returns negative value if already expired, 0 if no expiry set
func (tm *TokenMetadata) DaysUntilExpiry() int {
	if tm.ExpiresAt == nil {
		return 0
	}
	duration := time.Until(*tm.ExpiresAt)
	return int(duration.Hours() / 24)
}

// TokenStore manages multiple GitLab tokens
type TokenStore struct {
	tokens map[string]*TokenMetadata // key: token name
	mu     sync.RWMutex
}

// NewTokenStore creates a new token store
func NewTokenStore() *TokenStore {
	return &TokenStore{
		tokens: make(map[string]*TokenMetadata),
	}
}

// AddToken adds or updates a token in the store
func (ts *TokenStore) AddToken(name string, metadata *TokenMetadata) error {
	if name == "" {
		return fmt.Errorf("token name cannot be empty")
	}
	if metadata == nil {
		return fmt.Errorf("token metadata cannot be nil")
	}

	ts.mu.Lock()
	defer ts.mu.Unlock()

	metadata.Name = name
	ts.tokens[name] = metadata

	return nil
}

// GetToken retrieves a token by name
func (ts *TokenStore) GetToken(name string) (*TokenMetadata, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	token, ok := ts.tokens[name]
	if !ok {
		return nil, fmt.Errorf("token '%s' not found", name)
	}

	return token, nil
}

// ListTokens returns all tokens in the store
func (ts *TokenStore) ListTokens() map[string]*TokenMetadata {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*TokenMetadata, len(ts.tokens))
	for k, v := range ts.tokens {
		result[k] = v
	}

	return result
}

// RemoveToken removes a token from the store
func (ts *TokenStore) RemoveToken(name string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if _, ok := ts.tokens[name]; !ok {
		return fmt.Errorf("token '%s' not found", name)
	}

	delete(ts.tokens, name)
	return nil
}

// ValidateToken validates a token by calling GitLab API
// Returns TokenMetadata with user information if successful
func (ts *TokenStore) ValidateToken(ctx context.Context, name string, glClient *gl.Client) (*TokenMetadata, error) {
	// Call GitLab API /user to validate token
	user, resp, err := glClient.Users.CurrentUser(gl.WithContext(ctx))
	if err != nil {
		if resp != nil && resp.StatusCode == 401 {
			return nil, fmt.Errorf("token '%s' is invalid or expired (401)", name)
		}
		return nil, fmt.Errorf("failed to validate token '%s': %w", name, err)
	}

	// Update token metadata with user info
	ts.mu.Lock()
	defer ts.mu.Unlock()

	token, ok := ts.tokens[name]
	if !ok {
		// Token not in store, create new metadata
		token = &TokenMetadata{
			Name:           name,
			CreatedAt:      time.Now(),
			LastValidated:  time.Now(),
			IsExpiredFlag:  false,
		}
		ts.tokens[name] = token
	}

	// Update user information
	token.UserID = user.ID
	token.Username = user.Username
	token.LastValidated = time.Now()
	token.IsExpiredFlag = false

	return token, nil
}

// CheckAllTokens validates all stored tokens and returns results
func (ts *TokenStore) CheckAllTokens(ctx context.Context, getClientFunc func(name string) (*gl.Client, error)) []TokenValidationResult {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	results := make([]TokenValidationResult, 0, len(ts.tokens))

	for name := range ts.tokens {
		result := TokenValidationResult{
			TokenName: name,
			Success:   false,
		}

		// Get client for this token
		client, err := getClientFunc(name)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to get client: %v", err)
			results = append(results, result)
			continue
		}

		// Validate token
		metadata, err := ts.ValidateToken(ctx, name, client)
		if err != nil {
			result.Error = err.Error()
			result.IsExpired = true
		} else {
			result.Success = true
			result.UserID = metadata.UserID
			result.Username = metadata.Username
			result.DaysUntilExpiry = metadata.DaysUntilExpiry()
		}

		results = append(results, result)
	}

	return results
}

// TokenValidationResult represents the result of token validation
type TokenValidationResult struct {
	TokenName         string `json:"tokenName"`
	Success           bool   `json:"success"`
	Error             string `json:"error,omitempty"`
	IsExpired         bool   `json:"isExpired"`
	UserID            int    `json:"userId,omitempty"`
	Username          string `json:"username,omitempty"`
	DaysUntilExpiry   int    `json:"daysUntilExpiry,omitempty"`
}
