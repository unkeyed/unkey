// Package portalrbac defines the portal's public capability vocabulary.
//
// A workspace owner creates a portal session with a small, stable vocabulary of
// capabilities ("keys:read", "keys:reroll", ...) scoped to a set of keyspaces.
// Portal handlers authorize these capabilities directly and separately enforce
// the session's workspace, keyspace, and external identity scope.
package portalrbac

const (
	// CapKeysRead lets the end user list and read their own keys.
	CapKeysRead = "keys:read"

	// CapKeysCreate is unfinished: no route creates keys and createSession does
	// not accept the scope. Kept for whoever builds it.
	CapKeysCreate = "keys:create"

	// CapKeysReroll lets the end user rotate the secret of an existing key.
	CapKeysReroll = "keys:reroll"

	// CapAnalyticsRead is half built: v2_portal_get_verifications authorizes it,
	// but there is no analytics view and createSession does not accept the scope.
	CapAnalyticsRead = "analytics:read"
)
