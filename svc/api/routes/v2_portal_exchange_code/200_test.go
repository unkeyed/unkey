package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_exchange_code"
)

func TestExchangeCodeSuccess(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route, h.PublicMiddleware()...)

	workspaceID := h.Resources().UserWorkspace.ID
	portalID := uid.New(uid.PortalPrefix)
	now := time.Now().UnixMilli()

	err := db.Query.InsertPortal(ctx, h.DB.RW(), db.InsertPortalParams{
		ID:          portalID,
		WorkspaceID: workspaceID,
		Slug:        "test-portal",
		KeyspaceID:  sql.NullString{Valid: true, String: uid.New(uid.KeySpacePrefix)},
		Enabled:     true,
		CreatedAt:   now,
	})
	require.NoError(t, err)

	headers := http.Header{
		"Content-Type": {"application/json"},
	}

	t.Run("valid exchange", func(t *testing.T) {
		sessionID := uid.New(uid.PortalSessionPrefix)
		exchangeCode := uid.Secure()
		perms, _ := json.Marshal([]string{"api.*.read_key", "api.*.read_analytics"})

		err := db.Query.InsertPortalSession(ctx, h.DB.RW(), db.InsertPortalSessionParams{
			ID:                    sessionID,
			WorkspaceID:           workspaceID,
			PortalID:              portalID,
			ExternalID:            "user_valid",
			Permissions:           perms,
			ExchangeCodeHash:      sql.NullString{String: hash.Sha256(exchangeCode), Valid: true},
			ExchangeCodeExpiresAt: now + int64(15*time.Minute/time.Millisecond),
			CreatedAt:             now,
		})
		require.NoError(t, err)

		before := time.Now()

		req := handler.Request{Code: exchangeCode}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		require.NotEmpty(t, res.Body.Data.AccessToken)
		require.NotZero(t, res.Body.Data.ExpiresAt)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// Browser session expiry must be ~24 hours from now.
		after := time.Now()
		expectedLow := before.Add(24 * time.Hour).UnixMilli()
		expectedHigh := after.Add(24 * time.Hour).UnixMilli()
		require.GreaterOrEqual(t, res.Body.Data.ExpiresAt, expectedLow)
		require.LessOrEqual(t, res.Body.Data.ExpiresAt, expectedHigh)

		// Verify the browser session was persisted.
		session, err := db.Query.FindValidPortalSession(ctx, h.DB.RO(), db.FindValidPortalSessionParams{
			AccessTokenHash: sql.NullString{String: hash.Sha256(res.Body.Data.AccessToken), Valid: true},
			Now:             sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		})
		require.NoError(t, err)
		require.Equal(t, sessionID, session.ID)
		require.Equal(t, workspaceID, session.WorkspaceID)
		require.Equal(t, "user_valid", session.ExternalID)
		require.Equal(t, portalID, session.PortalID)
		require.False(t, session.ExchangeCodeHash.Valid)
		require.True(t, session.AccessTokenCreatedAt.Valid)
		require.Equal(t, hash.Sha256(res.Body.Data.AccessToken), session.AccessTokenHash.String)
		require.NotEqual(t, res.Body.Data.AccessToken, session.AccessTokenHash.String)
	})

	t.Run("single-use enforcement", func(t *testing.T) {
		sessionID := uid.New(uid.PortalSessionPrefix)
		exchangeCode := uid.Secure()
		perms, _ := json.Marshal([]string{"api.*.read_key"})

		err := db.Query.InsertPortalSession(ctx, h.DB.RW(), db.InsertPortalSessionParams{
			ID:                    sessionID,
			WorkspaceID:           workspaceID,
			PortalID:              portalID,
			ExternalID:            "user_single_use",
			Permissions:           perms,
			ExchangeCodeHash:      sql.NullString{String: hash.Sha256(exchangeCode), Valid: true},
			ExchangeCodeExpiresAt: now + int64(15*time.Minute/time.Millisecond),
			CreatedAt:             now,
		})
		require.NoError(t, err)

		req := handler.Request{Code: exchangeCode}

		// First exchange succeeds.
		res1 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res1.Status)

		// Second exchange must fail.
		res2 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 401, res2.Status)
	})
}
