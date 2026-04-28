// Package auth defines the contract every API handler relies on for
// authentication, regardless of whether the caller presented a root key,
// a JWT, or (eventually) a session cookie. The contract is intentionally
// a single struct, not an interface: every authentication scheme produces
// the same shape of resolved data, so an interface adds ceremony without
// abstraction power.
package auth

// Scheme identifies which authentication mechanism produced a Principal.
// Useful for audit logs and for handlers that want to restrict by scheme
// (e.g. "this endpoint requires a root key, no cookie sessions").
type Scheme string

const (
	SchemeRootKey Scheme = "root_key"
	SchemeJWT     Scheme = "jwt"
	SchemeCookie  Scheme = "cookie"
)

// Principal is the resolved authenticated state for one request. After
// Dispatcher.Authenticate returns one, the bearer has been verified, the
// workspace has been computed, and any cross-cutting checks (workspace
// rate limiting, etc.) have run. Handlers consume Principal as plain data.
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
	// "resource.id.action" form. Pass to rbac.Check for tuple-based access
	// decisions; pass directly to handlers that need to enumerate granted
	// permissions (e.g. analytics filtering).
	Permissions []string
}
