// Package auth defines the contract every API handler relies on for
// authentication and authorization, regardless of whether the caller
// presented a root key, a JWT, or (eventually) a session cookie. Every
// scheme produces the same Principal shape (data + an Authorizer hook),
// so handlers don't need to switch on scheme.
package auth

import (
	"context"

	"github.com/unkeyed/unkey/pkg/rbac"
)

// Scheme identifies which authentication mechanism produced a Principal.
// Useful for audit logs and for handlers that want to restrict by scheme
// (e.g. "this endpoint requires a root key, no cookie sessions").
type Scheme string

const (
	SchemeRootKey Scheme = "root_key"
	SchemeJWT     Scheme = "jwt"
	SchemeSession Scheme = "session"
)

// Authorizer is the per-scheme behavior plugged into a Principal so
// Authorize can run scheme-aware side effects. The root-key scheme uses
// this to flip the KeyVerifier's status to StatusInsufficientPermissions
// on a denied check, so the deferred ClickHouse emit records the right
// outcome instead of "VALID".
//
// Stateless schemes (JWT, future portal sessions) wrap their granted
// slice in GrantedPermissions, which just delegates to rbac.Check.
type Authorizer interface {
	Authorize(ctx context.Context, q rbac.PermissionQuery) error
}

// Principal is the resolved authenticated state for one request. After
// Authenticator.Authenticate returns one, the bearer has been verified,
// the workspace has been computed, and any cross-cutting checks
// (workspace rate limiting, etc.) have run.
type Principal struct {
	// Scheme records which mechanism authenticated this request.
	Scheme Scheme

	// ID is the stable identifier of the caller. For root keys it's the key
	// ID; for JWTs it's the sub claim; for cookies it's the user ID. Used
	// as ActorID in audit logs.
	ID string

	// DisplayName is a human-readable label for audit logs (key name, user
	// email, "dashboard"). May be empty when the scheme has nothing useful
	// to surface.
	DisplayName string

	// WorkspaceID is the tenant this request is authorized to act on.
	// Always non-empty after a successful resolution; handlers use it as
	// the tenant filter on every query they issue.
	WorkspaceID string

	// Permissions is the flat list of granted permission strings in
	// "resource.id.action" form. Read by handlers that need to enumerate
	// granted permissions directly (e.g. analytics filtering). For
	// tuple-based access decisions, call Principal.Authorize instead so
	// scheme-specific outcome tracking runs.
	Permissions []string

	// Authorizer is the scheme's authorization implementation. Always
	// non-nil after a resolver returns. Resolvers point this at whichever
	// type tracks per-request outcome (KeyVerifier for root keys, the
	// stateless GrantedPermissions for JWT/session schemes).
	Authorizer Authorizer
}

// Authorize evaluates q against the Principal's granted permissions via
// the scheme's Authorizer. Use this in handlers; do not call rbac.Check
// directly, or scheme-specific outcome tracking (e.g. ClickHouse outcome
// for root keys) will record the wrong result.
func (p *Principal) Authorize(ctx context.Context, q rbac.PermissionQuery) error {
	return p.Authorizer.Authorize(ctx, q)
}

// GrantedPermissions is the default Authorizer for stateless schemes
// (JWT, portal session). It carries the granted slice and delegates to
// rbac.Check, which produces the standard InsufficientPermissions fault
// on denial. No outcome tracking, because these schemes have no
// per-request emit to update.
type GrantedPermissions []string

// Authorize implements Authorizer for stateless schemes.
func (g GrantedPermissions) Authorize(_ context.Context, q rbac.PermissionQuery) error {
	return rbac.Check(q, g)
}
