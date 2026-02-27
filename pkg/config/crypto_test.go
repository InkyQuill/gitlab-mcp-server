package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCryptoManager_EncryptDecrypt(t *testing.T) {
	cm, err := NewCryptoManager(true)
	require.NoError(t, err)
	require.True(t, cm.IsEnabled(), "CryptoManager should be enabled")

	testCases := []struct {
		name  string
		token string
	}{
		{
			name:  "simple token",
			token: "glpat-1234567890abcdef",
		},
		{
			name:  "token with special characters",
			token: "glpat-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~:/?#[]@!$&'()*+,;=",
		},
		{
			name:  "long token",
			token: strings.Repeat("a", 100),
		},
		{
			name:  "empty token",
			token: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := cm.Encrypt(tc.token)
			require.NoError(t, err)

			// Encrypted value should be different from original (except empty string)
			if tc.token != "" {
				assert.NotEqual(t, tc.token, encrypted, "Encrypted token should differ from plaintext")
			}

			// Decrypt
			decrypted, err := cm.Decrypt(encrypted)
			require.NoError(t, err)
			assert.Equal(t, tc.token, decrypted, "Decrypted token should match original")
		})
	}
}

func TestCryptoManager_Encrypt_DecryptTwice(t *testing.T) {
	cm, err := NewCryptoManager(true)
	require.NoError(t, err)

	plaintext := "glpat-mysecrettoken"

	// Encrypt twice - should produce different ciphertexts due to nonce
	encrypted1, err := cm.Encrypt(plaintext)
	require.NoError(t, err)

	encrypted2, err := cm.Encrypt(plaintext)
	require.NoError(t, err)

	// Ciphertexts should be different (due to random nonce)
	assert.NotEqual(t, encrypted1, encrypted2, "Multiple encryptions should produce different ciphertexts")

	// But both should decrypt to the same plaintext
	decrypted1, err := cm.Decrypt(encrypted1)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted1)

	decrypted2, err := cm.Decrypt(encrypted2)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted2)
}

func TestCryptoManager_Disabled(t *testing.T) {
	cm, err := NewCryptoManager(false)
	require.NoError(t, err)
	require.False(t, cm.IsEnabled(), "CryptoManager should be disabled")

	token := "glpat-1234567890abcdef"

	// Encrypt should return plaintext as-is
	encrypted, err := cm.Encrypt(token)
	require.NoError(t, err)
	assert.Equal(t, token, encrypted)

	// Decrypt should return ciphertext as-is
	decrypted, err := cm.Decrypt(token)
	require.NoError(t, err)
	assert.Equal(t, token, decrypted)
}

func TestCryptoManager_IsEncrypted(t *testing.T) {
	cm, err := NewCryptoManager(true)
	require.NoError(t, err)

	token := "glpat-1234567890abcdef"

	// Before encryption
	assert.False(t, cm.IsEncrypted(token))

	// After encryption
	encrypted, err := cm.Encrypt(token)
	require.NoError(t, err)
	assert.True(t, cm.IsEncrypted(encrypted))
}

func TestCryptoManager_DecryptOldFormat(t *testing.T) {
	cm, err := NewCryptoManager(true)
	require.NoError(t, err)

	// Simulate old unencrypted token
	oldToken := "glpat-1234567890abcdef"

	// Decrypt should handle old format gracefully
	decrypted, err := cm.Decrypt(oldToken)
	require.NoError(t, err)
	assert.Equal(t, oldToken, decrypted)
}

func TestCryptoManager_Re_encrypt(t *testing.T) {
	cm, err := NewCryptoManager(true)
	require.NoError(t, err)

	token := "glpat-1234567890abcdef"

	// First encryption
	encrypted1, err := cm.Encrypt(token)
	require.NoError(t, err)

	// Re-encrypt should work
	reEncrypted, err := cm.Re_encrypt(encrypted1)
	require.NoError(t, err)

	// Decrypt the re-encrypted value
	decrypted, err := cm.Decrypt(reEncrypted)
	require.NoError(t, err)
	assert.Equal(t, token, decrypted)
}

func TestCryptoManager_Re_encryptOldFormat(t *testing.T) {
	cm, err := NewCryptoManager(true)
	require.NoError(t, err)

	// Old unencrypted token
	oldToken := "glpat-1234567890abcdef"

	// Re-encrypt should encrypt old format tokens
	reEncrypted, err := cm.Re_encrypt(oldToken)
	require.NoError(t, err)

	// Should now be encrypted
	assert.NotEqual(t, oldToken, reEncrypted)
	assert.True(t, cm.IsEncrypted(reEncrypted))

	// And decrypt back to original
	decrypted, err := cm.Decrypt(reEncrypted)
	require.NoError(t, err)
	assert.Equal(t, oldToken, decrypted)
}

func TestCryptoManager_Re_encryptDisabled(t *testing.T) {
	cm, err := NewCryptoManager(false)
	require.NoError(t, err)

	token := "glpat-1234567890abcdef"

	// Re-encrypt should return as-is when encryption is disabled
	reEncrypted, err := cm.Re_encrypt(token)
	require.NoError(t, err)
	assert.Equal(t, token, reEncrypted)
}

func TestCryptoManager_InvalidBase64(t *testing.T) {
	cm, err := NewCryptoManager(true)
	require.NoError(t, err)

	// Invalid base64 should be returned as-is
	invalidBase64 := "not-valid-base64!!!"
	decrypted, err := cm.Decrypt(invalidBase64)
	require.NoError(t, err)
	assert.Equal(t, invalidBase64, decrypted)
}

func TestCryptoManager_DifferentInstances(t *testing.T) {
	// Create two instances with encryption enabled
	cm1, err := NewCryptoManager(true)
	require.NoError(t, err)

	cm2, err := NewCryptoManager(true)
	require.NoError(t, err)

	token := "glpat-1234567890abcdef"

	// Encrypt with first instance
	encrypted, err := cm1.Encrypt(token)
	require.NoError(t, err)

	// Second instance should be able to decrypt
	// (since they share the same keyring)
	decrypted, err := cm2.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, token, decrypted)
}

func TestCryptoManager_EmptyPlaintext(t *testing.T) {
	cm, err := NewCryptoManager(true)
	require.NoError(t, err)

	// Empty string should encrypt and decrypt correctly
	encrypted, err := cm.Encrypt("")
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted, "Empty string should produce non-empty ciphertext")

	decrypted, err := cm.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, "", decrypted)
}

func TestCryptoManager_UnicodeToken(t *testing.T) {
	cm, err := NewCryptoManager(true)
	require.NoError(t, err)

	// Token with unicode characters
	token := "glpat-token-with-unicode-"

	encrypted, err := cm.Encrypt(token)
	require.NoError(t, err)

	decrypted, err := cm.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, token, decrypted)
}

func BenchmarkCryptoManager_Encrypt(b *testing.B) {
	cm, err := NewCryptoManager(true)
	if err != nil {
		b.Fatal(err)
	}

	token := "glpat-1234567890abcdefghijklmnopqrstuvwxyz"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cm.Encrypt(token)
	}
}

func BenchmarkCryptoManager_Decrypt(b *testing.B) {
	cm, err := NewCryptoManager(true)
	if err != nil {
		b.Fatal(err)
	}

	token := "glpat-1234567890abcdefghijklmnopqrstuvwxyz"
	encrypted, err := cm.Encrypt(token)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cm.Decrypt(encrypted)
	}
}
