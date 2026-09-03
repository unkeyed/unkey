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

	// CapKeysCreate would let the end user create new keys. Parked: no portal
	// route implements it and portal.createSession refuses the scope, so no
	// session can carry it.
	CapKeysCreate = "keys:create"

	// CapKeysReroll lets the end user rotate the secret of an existing key.
	CapKeysReroll = "keys:reroll"

	// CapAnalyticsRead lets the end user read their verification analytics.
	// Parked: v2_portal_get_verifications still authorizes it, but
	// portal.createSession refuses the scope, so only sessions minted before
	// that change can still satisfy it, until they expire.
	CapAnalyticsRead = "analytics:read"
)
