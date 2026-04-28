// Package portal_session is a placeholder for the future cookie-based
// auth scheme used by the dashboard portal. It is NOT wired into the
// dispatcher today and contains no working code; its only purpose is to
// document the shape a third resolver would take so the next person to
// implement it doesn't redesign the contract.
//
// # Trust model
//
// A portal session cookie identifies a *user* (via the WorkOS session). It
// does NOT identify a workspace, because a single user can belong to
// multiple workspaces. The resolver therefore must:
//
//  1. Validate the session cookie (signature, expiry).
//  2. Resolve the active workspace from somewhere outside the cookie:
//     URL path (e.g. /[workspaceSlug]/...), an X-Workspace-Id header,
//     or a separate "active workspace" cookie.
//  3. Verify the user is a member of that workspace.
//  4. Compute granted permissions from the user's role within the
//     workspace (admin vs member, etc.) and surface them via Permissions().
//
// Step 2 is the open contract decision: the dispatcher does not impose
// where workspace context lives on the request. The cookie resolver picks.
//
// # Why it slots in cleanly
//
// The dispatcher walks resolvers in order and stops at the first match.
// Bearer-shaped requests (JWT or root key) match earlier resolvers and
// never reach this one. Anything else with a session cookie falls
// through to here. The only behavioral coupling is that this resolver
// must return matched=false on requests it cannot handle (no cookie, or
// cookie present but workspace context missing) so the dispatcher can
// emit the standard "no credentials" error.
//
// # What this stub deliberately does NOT do
//
//   - Signature verification (will be SignedCookie or a session DB lookup
//     depending on how the dashboard signs sessions; punt until we wire it).
//   - User-to-workspace permission resolution (depends on the workspace
//     membership tables in the dashboard DB; the resolver will need a
//     dependency on that data source).
//   - Caching of the user→permissions resolution (per-request DB hits will
//     be expensive at scale; design a cache when implementing).
//
// To activate: implement Resolver.Try, expose NewResolver(deps...), and
// append it to the chain in svc/api/run.go AFTER the JWT resolver so
// bearer auth still wins.
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
//   - return (p, emit, nil) with Scheme: SchemeCookie on success
//   - return (nil, _, err) on a present-but-invalid cookie (mirroring the
//     JWT resolver's contract) so a bad session doesn't silently fall
//     through to a misleading downstream error.
func (r *Resolver) Try(ctx context.Context, sess *zen.Session) (*auth.Principal, auth.Emit, error) {
	return nil, nil, nil
}
