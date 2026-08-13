package portal

import (
	"github.com/unkeyed/unkey/pkg/db"
)

// State is the lifecycle position of a portal session, derived from the row
// rather than stored on it. Two of the five states are clock-driven, so a
// persisted status column would go stale with no write to trigger the update.
type State string

const (
	// StatePending means an exchange code was minted and has not been redeemed.
	StatePending State = "pending"

	// StateCodeExpired means the exchange code expired without ever being
	// redeemed. Terminal: re-auth mints a fresh code on a fresh row.
	StateCodeExpired State = "code_expired"

	// StateActive means an access token was issued and is still usable.
	StateActive State = "active"

	// StateExpired means the access token passed its natural expiry.
	StateExpired State = "expired"

	// StateRevoked means the session was explicitly invalidated. Takes
	// precedence over expiry, which is why it is checked first.
	StateRevoked State = "revoked"
)

// stateOf maps a session row to its state at time nowMs.
//
// The legal combinations are documented alongside the table definition in
// web/internal/db/src/schema/portal_sessions.ts; Vitess cannot enforce them, so
// this is the single place that interprets them. A row with an access token but
// no created-at timestamp is corrupt rather than a sixth state — the caller
// asserts on that separately instead of this function inventing a meaning.
func stateOf(row db.PortalSession, nowMs int64) State {
	if row.RevokedAt.Valid {
		return StateRevoked
	}

	if !row.AccessTokenHash.Valid {
		if row.ExchangeCodeExpiresAt > nowMs {
			return StatePending
		}
		return StateCodeExpired
	}

	if row.AccessTokenExpiresAt.Valid && row.AccessTokenExpiresAt.Int64 <= nowMs {
		return StateExpired
	}

	return StateActive
}
