package encryption

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// WorkspaceEncryption provides workspace-specific encryption for sensitive data
// like Stripe tokens. It derives workspace-specific keys from a master key
// using HKDF (HMAC-based Key Derivation Function).
type WorkspaceEncryption struct {
	masterKey []byte
}

// NewWorkspaceEncryption creates a new workspace encryption service with the
// provided master key. The master key should be at least 32 bytes for AES-256.
func NewWorkspaceEncryption(masterKey []byte) (*WorkspaceEncryption, error) {
	if len(masterKey) < 32 {
		return nil, fmt.Errorf("master key must be at least 32 bytes, got %d", len(masterKey))
	}

	return &WorkspaceEncryption{
		masterKey: masterKey,
	}, nil
}

// deriveWorkspaceKey derives a workspace-specific encryption key from the master
// key and workspace ID using HMAC-SHA256. This ensures each workspace has a
// unique encryption key while avoiding the need to store per-workspace keys.
func (w *WorkspaceEncryption) deriveWorkspaceKey(workspaceID string) ([]byte, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace ID cannot be empty")
	}

	// Use HMAC-SHA256 to derive a workspace-specific key
	// This provides strong key derivation without external dependencies
	h := hmac.New(sha256.New, w.masterKey)
	h.Write([]byte("unkey-billing-encryption:"))
	h.Write([]byte(workspaceID))

	return h.Sum(nil), nil
}

// EncryptToken encrypts a token (like Stripe access token or refresh token)
// for a specific workspace. Returns base64-encoded nonce and ciphertext
// concatenated with a separator for easy storage.
func (w *WorkspaceEncryption) EncryptToken(workspaceID, token string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("token cannot be empty")
	}

	workspaceKey, err := w.deriveWorkspaceKey(workspaceID)
	if err != nil {
		return "", fmt.Errorf("failed to derive workspace key: %w", err)
	}

	nonce, ciphertext, err := Encrypt(workspaceKey, []byte(token))
	if err != nil {
		return "", fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Concatenate nonce and ciphertext with base64 encoding
	// Format: base64(nonce):base64(ciphertext)
	encoded := base64.StdEncoding.EncodeToString(nonce) + ":" + base64.StdEncoding.EncodeToString(ciphertext)
	return encoded, nil
}

// DecryptToken decrypts a token that was encrypted with EncryptToken.
// The encrypted parameter should be in the format returned by EncryptToken.
func (w *WorkspaceEncryption) DecryptToken(workspaceID, encrypted string) (string, error) {
	if encrypted == "" {
		return "", fmt.Errorf("encrypted token cannot be empty")
	}

	workspaceKey, err := w.deriveWorkspaceKey(workspaceID)
	if err != nil {
		return "", fmt.Errorf("failed to derive workspace key: %w", err)
	}

	// Split the encoded string into nonce and ciphertext
	var nonceB64, ciphertextB64 string
	for i := 0; i < len(encrypted); i++ {
		if encrypted[i] == ':' {
			nonceB64 = encrypted[:i]
			ciphertextB64 = encrypted[i+1:]
			break
		}
	}

	if nonceB64 == "" || ciphertextB64 == "" {
		return "", fmt.Errorf("invalid encrypted token format: missing separator")
	}

	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	plaintext, err := Decrypt(workspaceKey, nonce, ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt token: %w", err)
	}

	return string(plaintext), nil
}
