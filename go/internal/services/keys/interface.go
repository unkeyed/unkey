package keys

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/zen"
)

type KeyService interface {
	Get(ctx context.Context, sess *zen.Session, hash string) (*KeyVerifier, error)
	GetRootKey(ctx context.Context, sess *zen.Session) (*KeyVerifier, error)
	CreateKey(ctx context.Context, req CreateKeyRequest) (CreateKeyResponse, error)
}

type VerifyResponse struct {
	AuthorizedWorkspaceID string
	KeyID                 string
}

type CreateKeyRequest struct {
	// Key generation parameters
	Prefix     string
	ByteLength int
}

type CreateKeyResponse struct {
	Key   string // The plaintext key
	Hash  string // SHA-256 hash for storage
	Start string // Key prefix for indexing
}
