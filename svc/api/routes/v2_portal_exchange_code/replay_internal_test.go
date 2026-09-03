package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

// TestExchangeAlreadyClaimed pins the check that keeps a retried exchange from
// rejecting a token it already persisted. TxRetry replays the whole closure on a
// transient connection error, so the second attempt sees access_token_hash
// already set; only the minted hash distinguishes "we committed" from "someone
// else redeemed this code".
//
// This covers the decision, not the wiring. Driving Handle itself down the
// replay branch needs a transaction that commits and then reports a transient
// failure, and db.TxWithResult takes a concrete *db.Replica, so there is no
// injection seam for that without restructuring production code. The residual
// gap is the HTTP-level assertion that a replayed call answers 200 with the
// minted token and writes no second audit log.
func TestExchangeAlreadyClaimed(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	workspaceID := h.Resources().UserWorkspace.ID
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	portalID := uid.New(uid.PortalPrefix)
	now := time.Now().UnixMilli()

	require.NoError(t, db.Query.InsertPortal(ctx, h.DB.RW(), db.InsertPortalParams{
		ID:          portalID,
		WorkspaceID: workspaceID,
		ProjectID:   api.ProjectID,
		Slug:        "replay-portal",
		KeyAuthID:   api.KeyAuthID,
		Enabled:     true,
		CreatedAt:   now,
	}))

	scopes, err := json.Marshal(map[string]any{
		"keyspaceIds": []string{uid.New(uid.KeySpacePrefix)},
		"scopes":      []string{"keys:read"},
	})
	require.NoError(t, err)

	// A session already redeemed with a known access token.
	code := string(uid.PortalExchangeCodePrefix) + "_" + uid.Secure()
	claimedToken := string(uid.PortalAccessTokenPrefix) + "_" + uid.Secure()

	require.NoError(t, db.Query.InsertPortalSession(ctx, h.DB.RW(), db.InsertPortalSessionParams{
		ID:                    uid.New(uid.PortalSessionPrefix),
		WorkspaceID:           workspaceID,
		PortalID:              portalID,
		ExternalID:            "user_replay",
		Scopes:                scopes,
		ExchangeCodeHash:      hash.Sha256(code),
		ExchangeCodeExpiresAt: now + int64(15*time.Minute/time.Millisecond),
		CreatedAt:             now,
	}))

	res, err := db.Query.ExchangePortalSessionCode(ctx, h.DB.RW(), db.ExchangePortalSessionCodeParams{
		AccessTokenHash:      sql.NullString{String: hash.Sha256(claimedToken), Valid: true},
		AccessTokenCreatedAt: sql.NullInt64{Int64: now, Valid: true},
		AccessTokenExpiresAt: sql.NullInt64{Int64: now + int64(24*time.Hour/time.Millisecond), Valid: true},
		ExchangeCodeHash:     hash.Sha256(code),
		Now:                  now,
	})
	require.NoError(t, err)
	affected, err := res.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)

	t.Run("our own committed attempt is recognized", func(t *testing.T) {
		claimed, err := exchangeAlreadyClaimed(ctx, h.DB.RW(), hash.Sha256(code), hash.Sha256(claimedToken))
		require.NoError(t, err)
		require.True(t, claimed, "a stored hash equal to ours means this call already committed")
	})

	t.Run("another request's redemption is not ours", func(t *testing.T) {
		otherToken := string(uid.PortalAccessTokenPrefix) + "_" + uid.Secure()

		claimed, err := exchangeAlreadyClaimed(ctx, h.DB.RW(), hash.Sha256(code), hash.Sha256(otherToken))
		require.NoError(t, err)
		require.False(t, claimed, "a code redeemed by someone else must still be rejected")
	})

	t.Run("unknown code is not ours", func(t *testing.T) {
		unknown := string(uid.PortalExchangeCodePrefix) + "_" + uid.Secure()

		claimed, err := exchangeAlreadyClaimed(ctx, h.DB.RW(), hash.Sha256(unknown), hash.Sha256(claimedToken))
		require.NoError(t, err)
		require.False(t, claimed, "an unknown code must not be reported as claimed")
	})

	t.Run("a pending session is not claimed by anyone", func(t *testing.T) {
		pendingCode := string(uid.PortalExchangeCodePrefix) + "_" + uid.Secure()

		require.NoError(t, db.Query.InsertPortalSession(ctx, h.DB.RW(), db.InsertPortalSessionParams{
			ID:                    uid.New(uid.PortalSessionPrefix),
			WorkspaceID:           workspaceID,
			PortalID:              portalID,
			ExternalID:            "user_pending",
			Scopes:                scopes,
			ExchangeCodeHash:      hash.Sha256(pendingCode),
			ExchangeCodeExpiresAt: now + int64(15*time.Minute/time.Millisecond),
			CreatedAt:             now,
		}))

		claimed, err := exchangeAlreadyClaimed(ctx, h.DB.RW(), hash.Sha256(pendingCode), hash.Sha256(claimedToken))
		require.NoError(t, err)
		require.False(t, claimed, "an unredeemed session has a NULL access_token_hash")
	})
}
