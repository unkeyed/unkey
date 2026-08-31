package keys

import (
	"context"

	"github.com/unkeyed/unkey/pkg/zen"
)

// KeyService defines the interface for key management operations.
// It provides methods for key creation, retrieval, and validation.
type KeyService interface {
	// Get retrieves a sha256 hashed key and returns a KeyVerifier for validation
	Get(ctx context.Context, sess *zen.Session, hash string) (*KeyVerifier, error)

	// GetRootKey retrieves and validates a root key from the session
	GetRootKey(ctx context.Context, sess *zen.Session) (*KeyVerifier, error)

	// GetMigrated retrieves a key using rawKey and migrationID
	// If migration is pending, it performs on-demand migration and returns a KeyVerifier for further validation.
	GetMigrated(ctx context.Context, sess *zen.Session, rawKey string, migrationID string) (*KeyVerifier, error)

	// CreateKey generates a key for legacy API routes.
	//
	// Deprecated: Use [KeyService.CreateKeyV1] for new key creation paths.
	CreateKey(ctx context.Context, req CreateKeyRequest) (CreateKeyResponse, error)

	// CreateKeyV1 generates an API key in the version 1 plaintext format.
	CreateKeyV1(ctx context.Context, req CreateKeyV1Request) (CreateKeyV1Response, error)
}

// VerifyResponse contains the result of a successful key verification.
type VerifyResponse struct {
	AuthorizedWorkspaceID string // The workspace ID that the key is authorized for
	KeyID                 string // The unique identifier of the key
}

// CreateKeyRequest specifies the parameters for creating a legacy API key.
//
// Deprecated: Use [CreateKeyV1Request] for new key creation paths.
type CreateKeyRequest struct {
	Prefix     string // Optional prefix to prepend to the key (e.g., "test_", "prod_")
	ByteLength int    // Length of the random bytes to generate (16-255)
}

// CreateKeyResponse contains a generated legacy key and its metadata.
//
// Deprecated: Use [CreateKeyV1Response] for new key creation paths.
type CreateKeyResponse struct {
	Key   string // The complete plaintext key (prefix + encoded random bytes)
	Hash  string // SHA-256 hash of the key for secure storage
	Start string // The first four encoded random characters.
}

// CreateKeyV1Request specifies the prefix for a version 1 plaintext key.
type CreateKeyV1Request struct {
	Prefix string // Prefix must match ^[A-Za-z0-9_]{0,6}[A-Za-z0-9]$.
}

// CreateKeyV1Response contains a version 1 plaintext key and its storage metadata.
type CreateKeyV1Response struct {
	Key    string // Complete plaintext key.
	Hash   string // SHA-256 hash of Key.
	Prefix string // Validated user-controlled prefix.
	Start  string // First four characters of the random field.
	End    string // Final four characters of Key.
}
