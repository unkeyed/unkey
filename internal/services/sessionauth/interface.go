package sessionauth

import (
	"context"

	"github.com/unkeyed/unkey/pkg/jwt"
)

// SessionClaims represents the JWT claims from a WorkOS access token.
type SessionClaims struct {
	jwt.RegisteredClaims
	OrgID       string   `json:"org_id"`
	SessionID   string   `json:"sid"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

// SessionResult contains the authenticated session information resolved
// from a JWT access token.
type SessionResult struct {
	// WorkspaceID is the Unkey workspace ID resolved from the JWT's org_id claim.
	WorkspaceID string

	// UserID is the subject (sub) from the JWT, typically a WorkOS user ID.
	UserID string

	// OrgID is the WorkOS organization ID from the JWT.
	OrgID string

	// Role is the user's role within the organization.
	Role string

	// Permissions are the user's permissions from the JWT.
	Permissions []string
}

// Service authenticates session tokens (JWTs) and resolves them to workspace context.
type Service interface {
	// CanHandle reports whether this service should attempt to authenticate the
	// given token. JWKS implementations check if the token looks like a JWT;
	// local implementations accept any token.
	CanHandle(token string) bool

	// Authenticate validates the given token and returns session information.
	// The token is expected to be a JWT access token (without the "Bearer " prefix).
	Authenticate(ctx context.Context, token string) (*SessionResult, error)
}
