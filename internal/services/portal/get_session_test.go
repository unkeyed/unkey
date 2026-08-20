package portal

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/hash"
)

// newServiceWithCachedRow builds a service whose cache already holds row under
// accessToken's hash. SWR therefore serves a hit and never reaches the origin,
// which is why the nil db is safe: these tests exercise the row-interpretation
// path, not the query.
func newServiceWithCachedRow(t *testing.T, accessToken string, row db.PortalSession) *service {
	t.Helper()

	clk := clock.NewTestClock(time.UnixMilli(nowMs))
	sessionCache, err := cache.New(cache.Config[string, db.PortalSession]{
		Fresh:    time.Minute,
		Stale:    time.Minute,
		MaxSize:  10,
		Resource: "portal_session_test",
		Clock:    clk,
	})
	require.NoError(t, err)

	sessionCache.Set(context.Background(), hash.Sha256(accessToken), row)

	return &service{sessionCache: sessionCache, clock: clk}
}

// activeRow is a well-formed, currently-valid session row.
func activeRow(t *testing.T) db.PortalSession {
	t.Helper()

	scopes, err := json.Marshal(Grant{
		KeyspaceIDs: []string{"ks_1"},
		Scopes:      []string{"keys:read"},
	})
	require.NoError(t, err)

	return db.PortalSession{
		ID:                    "ps_123",
		WorkspaceID:           "ws_123",
		PortalID:              "pc_123",
		ExternalID:            "customer_123",
		Scopes:                scopes,
		ExchangeCodeHash:      "code-hash",
		ExchangeCodeExpiresAt: nowMs - 1,
		AccessTokenHash:       nullStr("token-hash"),
		AccessTokenCreatedAt:  nullInt(nowMs - 1),
		AccessTokenExpiresAt:  nullInt(nowMs + 1),
	}
}

func TestGetSession_EmptyToken_ReturnsError(t *testing.T) {
	t.Parallel()

	svc := &service{}
	ctx := context.Background()

	info, err := svc.GetSession(ctx, "")

	require.Error(t, err)
	require.Nil(t, info)

	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Portal.Session.TokenMissing.URN(), code)
}

func TestGetSession_ActiveRow_ReturnsSessionInfo(t *testing.T) {
	t.Parallel()

	svc := newServiceWithCachedRow(t, "pat_valid", activeRow(t))

	info, err := svc.GetSession(context.Background(), "pat_valid")

	require.NoError(t, err)
	require.Equal(t, "ps_123", info.SessionID)
	require.Equal(t, "ws_123", info.WorkspaceID)
	require.Equal(t, "customer_123", info.ExternalID)
	require.Equal(t, "pc_123", info.PortalID)
	require.Equal(t, []string{"ks_1"}, info.KeyspaceIDs)
	require.Equal(t, []string{"keys:read"}, info.Scopes)
}

// TestGetSession_CorruptRow covers the degenerate rows: an access token hash
// with either timestamp missing. All must surface as an internal error rather
// than authenticating or masquerading as an ordinary expired session.
//
// Only the missing-expires-at case is a regression pin: before this behavior
// existed, that row authenticated successfully. The missing-created-at cases
// already returned an internal error and are here to hold that ground.
func TestGetSession_CorruptRow_ReturnsInternalError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*db.PortalSession)
	}{
		{
			name:   "access token hash without created-at",
			mutate: func(r *db.PortalSession) { r.AccessTokenCreatedAt = sql.NullInt64{} },
		},
		{
			name:   "access token hash without expires-at",
			mutate: func(r *db.PortalSession) { r.AccessTokenExpiresAt = sql.NullInt64{} },
		},
		{
			name: "access token hash without either timestamp",
			mutate: func(r *db.PortalSession) {
				r.AccessTokenCreatedAt = sql.NullInt64{}
				r.AccessTokenExpiresAt = sql.NullInt64{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			row := activeRow(t)
			tt.mutate(&row)
			svc := newServiceWithCachedRow(t, "pat_corrupt", row)

			info, err := svc.GetSession(context.Background(), "pat_corrupt")

			require.Error(t, err)
			require.Nil(t, info)

			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.App.Internal.UnexpectedError.URN(), code)
		})
	}
}

// Revocation is a deliberate end state, so it outranks corruption: a revoked row
// is rejected as an invalid session however malformed its timestamps are, rather
// than paging someone with an internal error over a session shut off on purpose.
func TestGetSession_RevokedCorruptRow_PrefersRevoked(t *testing.T) {
	t.Parallel()

	row := activeRow(t)
	row.RevokedAt = nullInt(nowMs - 1)
	row.AccessTokenExpiresAt = sql.NullInt64{}
	row.AccessTokenCreatedAt = sql.NullInt64{}

	svc := newServiceWithCachedRow(t, "pat_revoked_corrupt", row)

	info, err := svc.GetSession(context.Background(), "pat_revoked_corrupt")

	require.Error(t, err)
	require.Nil(t, info)

	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Portal.Session.SessionNotFound.URN(), code,
		"a revoked row must reject as an invalid session, not an internal error")
}

// A pending row carries neither access-token timestamp legitimately, so it must
// be rejected as an invalid session rather than reported as corrupt.
func TestGetSession_PendingRow_IsNotReportedCorrupt(t *testing.T) {
	t.Parallel()

	row := activeRow(t)
	row.AccessTokenHash = sql.NullString{}
	row.AccessTokenCreatedAt = sql.NullInt64{}
	row.AccessTokenExpiresAt = sql.NullInt64{}
	row.ExchangeCodeExpiresAt = nowMs + 1

	svc := newServiceWithCachedRow(t, "pat_pending", row)

	info, err := svc.GetSession(context.Background(), "pat_pending")

	require.Error(t, err)
	require.Nil(t, info)

	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Portal.Session.SessionNotFound.URN(), code)
}

// Revocation and expiry must both reject, and must not leak which one applied.
func TestGetSession_RevokedAndExpiredRows_Reject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*db.PortalSession)
	}{
		{
			name:   "revoked",
			mutate: func(r *db.PortalSession) { r.RevokedAt = nullInt(nowMs - 1) },
		},
		{
			name:   "expired",
			mutate: func(r *db.PortalSession) { r.AccessTokenExpiresAt = nullInt(nowMs - 1) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			row := activeRow(t)
			tt.mutate(&row)
			svc := newServiceWithCachedRow(t, "pat_rejected", row)

			info, err := svc.GetSession(context.Background(), "pat_rejected")

			require.Error(t, err)
			require.Nil(t, info)

			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.Portal.Session.SessionNotFound.URN(), code)
			require.Equal(t, "The portal session is invalid or has expired.", fault.UserFacingMessage(err))
		})
	}
}

func TestSessionInfo_FieldsExist(t *testing.T) {
	t.Parallel()

	info := SessionInfo{
		SessionID:   "ps_001",
		WorkspaceID: "ws_123",
		ExternalID:  "user_456",
		PortalID:    "pc_789",
		Scopes:      []string{"keys:read", "analytics:read"},
		Preview:     true,
	}

	require.Equal(t, "ps_001", info.SessionID)
	require.Equal(t, "ws_123", info.WorkspaceID)
	require.Equal(t, "user_456", info.ExternalID)
	require.Equal(t, "pc_789", info.PortalID)
	require.Equal(t, []string{"keys:read", "analytics:read"}, info.Scopes)
	require.True(t, info.Preview)
}

func TestSessionInfo_NilScopes(t *testing.T) {
	t.Parallel()

	info := SessionInfo{
		WorkspaceID: "ws_123",
		ExternalID:  "user_456",
		PortalID:    "pc_789",
	}

	require.Nil(t, info.Scopes)
	require.False(t, info.Preview)
}
