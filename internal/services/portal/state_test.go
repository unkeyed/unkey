package portal

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
)

const nowMs int64 = 1_700_000_000_000

func nullInt(v int64) sql.NullInt64 {
	return sql.NullInt64{Int64: v, Valid: true}
}

func nullStr(v string) sql.NullString {
	return sql.NullString{String: v, Valid: true}
}

// TestStateOf covers every legal combination documented alongside the table
// definition in web/internal/db/src/schema/portal_sessions.ts, plus the two
// degenerate rows. Vitess cannot enforce those combinations, so this function is
// the only thing standing between a malformed row and an authenticated request.
func TestStateOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		row  db.PortalSession
		want State
	}{
		{
			name: "pending: code minted, not yet redeemed, not expired",
			row: db.PortalSession{
				ExchangeCodeHash:      "code-hash",
				ExchangeCodeExpiresAt: nowMs + 1,
			},
			want: StatePending,
		},
		{
			name: "code_expired: code expired exactly at now, never redeemed",
			row: db.PortalSession{
				ExchangeCodeHash:      "code-hash",
				ExchangeCodeExpiresAt: nowMs,
			},
			want: StateCodeExpired,
		},
		{
			name: "code_expired: code expired in the past, never redeemed",
			row: db.PortalSession{
				ExchangeCodeHash:      "code-hash",
				ExchangeCodeExpiresAt: nowMs - 1,
			},
			want: StateCodeExpired,
		},
		{
			name: "active: token issued and still valid",
			row: db.PortalSession{
				ExchangeCodeHash:      "code-hash",
				ExchangeCodeExpiresAt: nowMs - 1,
				AccessTokenHash:       nullStr("token-hash"),
				AccessTokenCreatedAt:  nullInt(nowMs - 1),
				AccessTokenExpiresAt:  nullInt(nowMs + 1),
			},
			want: StateActive,
		},
		{
			name: "expired: token expiry passed",
			row: db.PortalSession{
				ExchangeCodeHash:      "code-hash",
				ExchangeCodeExpiresAt: nowMs - 1,
				AccessTokenHash:       nullStr("token-hash"),
				AccessTokenCreatedAt:  nullInt(nowMs - 2),
				AccessTokenExpiresAt:  nullInt(nowMs - 1),
			},
			want: StateExpired,
		},
		{
			name: "expired: token expiry is exactly now",
			row: db.PortalSession{
				ExchangeCodeHash:      "code-hash",
				ExchangeCodeExpiresAt: nowMs - 1,
				AccessTokenHash:       nullStr("token-hash"),
				AccessTokenCreatedAt:  nullInt(nowMs - 2),
				AccessTokenExpiresAt:  nullInt(nowMs),
			},
			want: StateExpired,
		},
		{
			name: "revoked takes precedence over an otherwise active token",
			row: db.PortalSession{
				ExchangeCodeHash:      "code-hash",
				ExchangeCodeExpiresAt: nowMs - 1,
				AccessTokenHash:       nullStr("token-hash"),
				AccessTokenCreatedAt:  nullInt(nowMs - 1),
				AccessTokenExpiresAt:  nullInt(nowMs + 1),
				RevokedAt:             nullInt(nowMs - 1),
			},
			want: StateRevoked,
		},
		{
			name: "revoked takes precedence over expiry",
			row: db.PortalSession{
				ExchangeCodeHash:      "code-hash",
				ExchangeCodeExpiresAt: nowMs - 1,
				AccessTokenHash:       nullStr("token-hash"),
				AccessTokenCreatedAt:  nullInt(nowMs - 2),
				AccessTokenExpiresAt:  nullInt(nowMs - 1),
				RevokedAt:             nullInt(nowMs - 1),
			},
			want: StateRevoked,
		},
		{
			name: "revoked takes precedence over a pending code",
			row: db.PortalSession{
				ExchangeCodeHash:      "code-hash",
				ExchangeCodeExpiresAt: nowMs + 1,
				RevokedAt:             nullInt(nowMs - 1),
			},
			want: StateRevoked,
		},
		{
			// The degenerate row this function must never call active: a token
			// with no expiry would otherwise authenticate forever.
			name: "degenerate: token hash with NULL expiry fails closed",
			row: db.PortalSession{
				ExchangeCodeHash:      "code-hash",
				ExchangeCodeExpiresAt: nowMs - 1,
				AccessTokenHash:       nullStr("token-hash"),
				AccessTokenCreatedAt:  nullInt(nowMs - 1),
			},
			want: StateExpired,
		},
		{
			// The sibling degenerate row. GetSession reports it as corrupt, but
			// stateOf must not call it active either.
			name: "degenerate: token hash with NULL created-at and NULL expiry fails closed",
			row: db.PortalSession{
				ExchangeCodeHash:      "code-hash",
				ExchangeCodeExpiresAt: nowMs - 1,
				AccessTokenHash:       nullStr("token-hash"),
			},
			want: StateExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, stateOf(tt.row, nowMs))
		})
	}
}
