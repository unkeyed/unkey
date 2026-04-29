// Package portal_session is a placeholder for the future auth scheme
// used by the dashboard portal. It is NOT wired into the dispatcher today
// and contains no working code; its only purpose is to document the shape
// a third resolver would take so the next person to implement it doesn't
// redesign the contract.
package portal_session

import (
	"context"
	"errors"

	"github.com/unkeyed/unkey/pkg/auth"
	"github.com/unkeyed/unkey/pkg/zen"
)

// Resolver is a stub for the cookie-based portal session scheme. All calls
// return matched=false so the chain falls through; this lets the file be
// safely imported and referenced during the design phase without changing
// runtime behavior.
type Resolver struct{}

// ErrNotImplemented is returned by NewResolver until the scheme is wired up.
// Callers should check for it explicitly so a partially-built run.go can
// fail loudly rather than silently authenticate every cookie request.
var ErrNotImplemented = errors.New("auth/portal_session: not implemented")

// NewResolver currently returns ErrNotImplemented to make accidental wiring
// obvious. Replace the body with real construction (cookie parser, workspace
// lookup, role-to-permission mapper) when implementing.
func NewResolver() (*Resolver, error) {
	return nil, ErrNotImplemented
}

// Try always returns (nil, nil, nil) today, leaving the dispatcher to fall
// through to whatever comes next or to emit the standard "no credentials"
// error. When implemented this should:
//   - return (nil, nil, nil) when no portal session cookie is present
//   - return (p, emit, nil) with Scheme: SchemeSession and a populated
//     Authorizer (auth.GrantedPermissions(perms) is fine unless the
//     scheme grows per-request outcome tracking)
//   - return (nil, _, err) on a present-but-invalid cookie (mirroring the
//     JWT resolver's contract) so a bad session doesn't silently fall
//     through to a misleading downstream error.
func (r *Resolver) Try(ctx context.Context, sess *zen.Session) (*auth.Principal, auth.Emit, error) {
	return nil, nil, nil
}
