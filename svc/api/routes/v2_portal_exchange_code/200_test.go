package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_exchange_code"
)

// seedPendingSession inserts a session in the pending state and returns the
// plaintext exchange code, which exists nowhere else.
func seedPendingSession(t *testing.T, h *testutil.Harness, workspaceID, portalID, externalID string) string {
	t.Helper()

	code := string(uid.PortalExchangeCodePrefix) + "_" + uid.Secure()
	scopes, err := json.Marshal(map[string]any{
		"keyspaceIds": []string{uid.New(uid.KeySpacePrefix)},
		"scopes":      []string{"keys:read"},
	})
	require.NoError(t, err)

	now := time.Now().UnixMilli()
	require.NoError(t, db.Query.InsertPortalSession(context.Background(), h.DB.RW(), db.InsertPortalSessionParams{
		ID:                    uid.New(uid.PortalSessionPrefix),
		WorkspaceID:           workspaceID,
		PortalID:              portalID,
		ExternalID:            externalID,
		Scopes:                scopes,
		ExchangeCodeHash:      hash.Sha256(code),
		ExchangeCodeExpiresAt: now + int64(15*time.Minute/time.Millisecond),
		CreatedAt:             now,
	}))

	return code
}

func TestExchangeCodeSuccess(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route, h.PublicMiddleware()...)

	workspaceID := h.Resources().UserWorkspace.ID
	portalID := uid.New(uid.PortalConfigPrefix)
	now := time.Now().UnixMilli()

	err := db.Query.InsertPortal(ctx, h.DB.RW(), db.InsertPortalParams{
		ID:          portalID,
		WorkspaceID: workspaceID,
		Slug:        "test-portal",
		KeyAuthID:   sql.NullString{Valid: true, String: uid.New(uid.KeySpacePrefix)},
		Enabled:     true,
		CreatedAt:   now,
	})
	require.NoError(t, err)

	headers := http.Header{
		"Content-Type": {"application/json"},
	}

	t.Run("valid exchange", func(t *testing.T) {
		code := seedPendingSession(t, h, workspaceID, portalID, "user_valid")

		before := time.Now()

		req := handler.Request{Code: code}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		require.NotEmpty(t, res.Body.Data.AccessToken)
		require.NotZero(t, res.Body.Data.ExpiresAt)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// Access token expiry must be ~24 hours from now.
		after := time.Now()
		require.GreaterOrEqual(t, res.Body.Data.ExpiresAt, before.Add(24*time.Hour).UnixMilli())
		require.LessOrEqual(t, res.Body.Data.ExpiresAt, after.Add(24*time.Hour).UnixMilli())

		// The exchange updates the session in place rather than creating a
		// second row: same row, now carrying an access token.
		session, err := db.Query.FindPortalSessionByAccessTokenHash(ctx, h.DB.RO(),
			sql.NullString{String: hash.Sha256(res.Body.Data.AccessToken), Valid: true})
		require.NoError(t, err)
		require.Equal(t, workspaceID, session.WorkspaceID)
		require.Equal(t, "user_valid", session.ExternalID)
		require.Equal(t, portalID, session.PortalID)
		require.Equal(t, hash.Sha256(code), session.ExchangeCodeHash)
		require.True(t, session.AccessTokenCreatedAt.Valid)
	})

	t.Run("stores the access token only as a hash", func(t *testing.T) {
		code := seedPendingSession(t, h, workspaceID, portalID, "user_hashed")

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Code: code})
		require.Equal(t, 200, res.Status)

		accessToken := res.Body.Data.AccessToken

		// Neither credential is queryable in plaintext.
		_, err := db.Query.FindPortalSessionByAccessTokenHash(ctx, h.DB.RO(),
			sql.NullString{String: accessToken, Valid: true})
		require.True(t, db.IsNotFound(err), "plaintext access token must not be stored")

		_, err = db.Query.FindPortalSessionByExchangeCodeHash(ctx, h.DB.RO(), code)
		require.True(t, db.IsNotFound(err), "plaintext exchange code must not be stored")
	})

	t.Run("single-use enforcement", func(t *testing.T) {
		code := seedPendingSession(t, h, workspaceID, portalID, "user_single_use")

		// First exchange succeeds.
		res1 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Code: code})
		require.Equal(t, 200, res1.Status)

		// Second exchange must fail: access_token_hash is no longer NULL.
		res2 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Code: code})
		require.Equal(t, 401, res2.Status)
	})

	t.Run("concurrent exchange yields exactly one winner", func(t *testing.T) {
		code := seedPendingSession(t, h, workspaceID, portalID, "user_concurrent")

		const attempts = 4
		statuses := make([]int, attempts)

		var wg sync.WaitGroup
		for i := range attempts {
			wg.Go(func() {
				res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Code: code})
				statuses[i] = res.Status
			})
		}
		wg.Wait()

		won := 0
		for _, status := range statuses {
			if status == 200 {
				won++
				continue
			}
			require.Equal(t, 401, status, "losers must be rejected, not errored")
		}
		require.Equal(t, 1, won, "exactly one redemption may succeed")
	})
}
