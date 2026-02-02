// Package encryption provides cryptographic utilities for encrypting and
// decrypting sensitive data in the Unkey system.
//
// The package includes:
//   - AES-256-GCM encryption/decryption for general-purpose encryption
//   - Workspace-specific encryption for isolating sensitive data per workspace
//
// # Workspace Encryption
//
// The WorkspaceEncryption type provides workspace-scoped encryption for
// sensitive tokens like Stripe API credentials. It derives unique encryption
// keys per workspace from a master key using HMAC-SHA256, ensuring that:
//   - Each workspace's data is encrypted with a unique key
//   - No per-workspace keys need to be stored
//   - Workspace isolation is cryptographically enforced
//
// Example usage:
//
//	masterKey := []byte("...32-byte-master-key...")
//	we, err := encryption.NewWorkspaceEncryption(masterKey)
//	if err != nil {
//	    return err
//	}
//
//	// Encrypt a Stripe token for a workspace
//	encrypted, err := we.EncryptToken("ws_123", "sk_live_abc123")
//	if err != nil {
//	    return err
//	}
//
//	// Decrypt the token
//	token, err := we.DecryptToken("ws_123", encrypted)
//	if err != nil {
//	    return err
//	}
//
// # Security Properties
//
// The encryption implementation provides:
//   - Confidentiality: AES-256-GCM encryption
//   - Authenticity: GCM authentication tag prevents tampering
//   - Workspace isolation: Derived keys prevent cross-workspace decryption
//   - Nonce uniqueness: Random nonces prevent replay attacks
package encryption
