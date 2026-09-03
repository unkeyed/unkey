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

	// CapKeysCreate was never finished: no portal route creates keys, and
	// portal.createSession does not accept the scope, so nothing can grant it.
	// Kept as the name to reuse if the feature is built.
	CapKeysCreate = "keys:create"

	// CapKeysReroll lets the end user rotate the secret of an existing key.
	CapKeysReroll = "keys:reroll"

	// CapAnalyticsRead was only half built: v2_portal_get_verifications
	// authorizes it, but the portal has no analytics view and
	// portal.createSession does not accept the scope, so nothing can grant it.
	// Kept as the name to reuse if the feature is finished.
	CapAnalyticsRead = "analytics:read"
)
