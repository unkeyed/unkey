package workos

import (
	"context"
	"errors"
)

// Directory resolves active members with verified emails for notification routing.
// Implementations must be safe for concurrent use across workspaces.
type Directory interface {
	// Members returns the organization's eligible members, sorted by user ID.
	// ErrDirectoryLimit rejects incomplete or non-progressing directory traversal.
	Members(ctx context.Context, orgID string) ([]Member, error)
	// LookupMembers omits unknown, inactive, and unverified users and deduplicates IDs.
	LookupMembers(ctx context.Context, orgID string, userIDs []string) ([]Member, error)
	// CanEmail rechecks membership and requires an exact match to the verified email.
	CanEmail(ctx context.Context, orgID, userID, email string) (bool, error)
}

// Member is an active organization member with a verified account email.
type Member struct {
	UserID string
	Email  string
	Admin  bool
}

// ErrDirectoryLimit stops exhausted or non-progressing directory traversal; don't retry unchanged.
var ErrDirectoryLimit = errors.New("workos directory limit exceeded")
