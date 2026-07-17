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

	// CapKeysCreate lets the end user create new keys.
	CapKeysCreate = "keys:create"

	// CapKeysReroll lets the end user rotate the secret of an existing key.
	CapKeysReroll = "keys:reroll"

	// CapAnalyticsRead lets the end user read their verification analytics.
	CapAnalyticsRead = "analytics:read"
)
