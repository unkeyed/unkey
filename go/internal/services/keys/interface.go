package keys

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/zen"
)

// KeyService defines the interface for key management operations.
// It provides methods for key creation, retrieval, and validation.
type KeyService interface {
	// Get retrieves a key and returns a KeyVerifier for validation
	Get(ctx context.Context, sess *zen.Session, hash string) (*KeyVerifier, func(), error)

	// GetMigrated retrieves and verifies a key that has been migrated using the raw key and migration ID
	GetMigrated(ctx context.Context, sess *zen.Session, rawKey string, migrationID string) (*KeyVerifier, func(), error)

	// GetRootKey retrieves and validates a root key from the session
	GetRootKey(ctx context.Context, sess *zen.Session) (*KeyVerifier, func(), error)

	// CreateKey generates a new secure API key
	CreateKey(ctx context.Context, req CreateKeyRequest) (CreateKeyResponse, error)
}

// VerifyResponse contains the result of a successful key verification.
type VerifyResponse struct {
	AuthorizedWorkspaceID string // The workspace ID that the key is authorized for
	KeyID                 string // The unique identifier of the key
}

// CreateKeyRequest specifies the parameters for creating a new API key.
type CreateKeyRequest struct {
	Prefix     string // Optional prefix to prepend to the key (e.g., "test_", "prod_")
	ByteLength int    // Length of the random bytes to generate (16-255)
}

// CreateKeyResponse contains the generated key and its metadata.
type CreateKeyResponse struct {
	Key   string // The complete plaintext key (prefix + encoded random bytes)
	Hash  string // SHA-256 hash of the key for secure storage
	Start string // The start of the key for indexing and display purposes
}
