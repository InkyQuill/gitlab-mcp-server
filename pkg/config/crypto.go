package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "gitlab-mcp-server"
	keyringUser    = "master-key"
)

// CryptoManager manages token encryption using system keyring.
type CryptoManager struct {
	useKeyring bool
	masterKey  []byte
}

// NewCryptoManager creates a new crypto manager.
// If useKeyring is true, it will attempt to use the system keyring for storing the master key.
func NewCryptoManager(useKeyring bool) (*CryptoManager, error) {
	cm := &CryptoManager{
		useKeyring: useKeyring,
	}

	if useKeyring {
		// Try to get existing master key from keyring
		key, err := keyring.Get(keyringService, keyringUser)
		if err == nil {
			cm.masterKey, err = base64.StdEncoding.DecodeString(key)
			if err != nil {
				return nil, fmt.Errorf("failed to decode master key from keyring: %w", err)
			}
		} else {
			// Create new master key and store in keyring
			cm.masterKey, err = cm.generateMasterKey()
			if err != nil {
				return nil, fmt.Errorf("failed to generate master key: %w", err)
			}
			if err := keyring.Set(keyringService, keyringUser,
				base64.StdEncoding.EncodeToString(cm.masterKey)); err != nil {
				return nil, fmt.Errorf("failed to store master key in keyring: %w", err)
			}
		}
	}

	return cm, nil
}

// generateMasterKey generates a new 256-bit master key for AES-256.
func (cm *CryptoManager) generateMasterKey() ([]byte, error) {
	key := make([]byte, 32) // 256 bits for AES-256
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

// Encrypt encrypts a plaintext token using AES-256-GCM.
// If encryption is not enabled, returns the plaintext as-is.
func (cm *CryptoManager) Encrypt(plaintext string) (string, error) {
	if !cm.useKeyring || cm.masterKey == nil {
		return plaintext, nil // No encryption if keyring is not enabled
	}

	block, err := aes.NewCipher(cm.masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Use GCM for authenticated encryption
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a ciphertext token.
// If the token is not encrypted (e.g., from an older config), returns it as-is.
func (cm *CryptoManager) Decrypt(ciphertext string) (string, error) {
	if !cm.useKeyring || cm.masterKey == nil {
		return ciphertext, nil // Return as-is if encryption is not enabled
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		// Not a valid base64 string - likely not encrypted (old format)
		return ciphertext, nil
	}

	block, err := aes.NewCipher(cm.masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		// Decryption failed - possibly old format or corrupted
		return ciphertext, nil
	}

	return string(plaintext), nil
}

// IsEnabled returns true if encryption is enabled and a master key is available.
func (cm *CryptoManager) IsEnabled() bool {
	return cm.useKeyring && cm.masterKey != nil
}

// IsEncrypted checks if a given token appears to be encrypted.
// This is a heuristic check - valid base64 doesn't guarantee encryption.
func (cm *CryptoManager) IsEncrypted(token string) bool {
	if !cm.IsEnabled() {
		return false
	}
	_, err := base64.StdEncoding.DecodeString(token)
	return err == nil
}

// Re_encrypt re-encrypts a token with the current master key.
// Useful if the master key has changed.
func (cm *CryptoManager) Re_encrypt(encryptedToken string) (string, error) {
	if !cm.IsEnabled() {
		return encryptedToken, nil
	}

	// Try to decrypt first
	plaintext, err := cm.Decrypt(encryptedToken)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt token: %w", err)
	}

	// Re-encrypt
	return cm.Encrypt(plaintext)
}
