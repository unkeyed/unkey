// Package keys provides functionality for API key verification and management.
// It handles the verification of API keys against the database, including checking
// their validity, permissions, and associated workspace details.
package keys

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/zen"
)

// KeyService defines the interface for API key operations.
// It provides methods to verify API keys and root keys.
type KeyService interface {
	// Verify validates an API key hash against the database.
	// It checks if the key exists, is not deleted, is enabled, and 
	// has a valid associated workspace.
	//
	// Returns a VerifyResponse containing the authorized workspace ID and key ID,
	// or an error if verification fails for any reason.
	Verify(ctx context.Context, hash string) (VerifyResponse, error)
	
	// VerifyRootKey validates a root API key using bearer authentication from a session.
	// It extracts the key from the session, verifies it, and updates the session
	// with the authorized workspace ID.
	//
	// Returns a VerifyResponse and an error if verification fails.
	VerifyRootKey(ctx context.Context, sess *zen.Session) (VerifyResponse, error)
}

// VerifyResponse contains the result of a successful key verification.
type VerifyResponse struct {
	// AuthorizedWorkspaceID is the ID of the workspace the key is authorized to access.
	// For regular keys, this is the key's workspace. For root keys, this is the
	// workspace specified in ForWorkspaceID.
	AuthorizedWorkspaceID string
	
	// KeyID is the unique identifier of the verified key.
	KeyID                 string
}
