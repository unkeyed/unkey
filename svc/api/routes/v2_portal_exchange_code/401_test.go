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
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_exchange_code"
)

func TestExchangeCodeUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route, h.PublicMiddleware()...)

	workspaceID := h.Resources().UserWorkspace.ID
	portalID := uid.New(uid.PortalConfigPrefix)
	now := time.Now().UnixMilli()

	// Seed the portal the sessions belong to.
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

	scopes, err := json.Marshal(map[string]any{
		"keyspaceIds": []string{uid.New(uid.KeySpacePrefix)},
		"scopes":      []string{"keys:read"},
	})
	require.NoError(t, err)

	t.Run("unknown code", func(t *testing.T) {
		req := handler.Request{Code: "nonexistent_code"}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 401, res.Status)
		require.Equal(t, "Session is invalid, expired, or has already been used.", res.Body.Error.Detail)
		require.NotContains(t, res.RawBody, req.Code)
	})

	t.Run("expired code", func(t *testing.T) {
		code := uid.New(uid.PortalExchangeCodePrefix)

		// A session whose exchange code expired an hour ago.
		require.NoError(t, db.Query.InsertPortalSession(ctx, h.DB.RW(), db.InsertPortalSessionParams{
			ID:                    uid.New(uid.PortalSessionPrefix),
			WorkspaceID:           workspaceID,
			PortalID:              portalID,
			ExternalID:            "user_expired",
			Scopes:                scopes,
			ExchangeCodeHash:      hash.Sha256(code),
			ExchangeCodeExpiresAt: now - int64(time.Hour/time.Millisecond),
			CreatedAt:             now - int64(2*time.Hour/time.Millisecond),
		}))

		req := handler.Request{Code: code}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 401, res.Status)
		require.Equal(t, "Session is invalid, expired, or has already been used.", res.Body.Error.Detail)
		require.NotContains(t, res.RawBody, code)
	})

	t.Run("already redeemed code", func(t *testing.T) {
		code := uid.New(uid.PortalExchangeCodePrefix)

		require.NoError(t, db.Query.InsertPortalSession(ctx, h.DB.RW(), db.InsertPortalSessionParams{
			ID:                    uid.New(uid.PortalSessionPrefix),
			WorkspaceID:           workspaceID,
			PortalID:              portalID,
			ExternalID:            "user_redeemed",
			Scopes:                scopes,
			ExchangeCodeHash:      hash.Sha256(code),
			ExchangeCodeExpiresAt: now + int64(15*time.Minute/time.Millisecond),
			CreatedAt:             now,
		}))

		// Redeem it out of band.
		redeemed, err := db.Query.ExchangePortalSessionCode(ctx, h.DB.RW(), db.ExchangePortalSessionCodeParams{
			AccessTokenHash:      sql.NullString{String: hash.Sha256(uid.New(uid.PortalSessionPrefix)), Valid: true},
			AccessTokenCreatedAt: sql.NullInt64{Int64: now, Valid: true},
			AccessTokenExpiresAt: sql.NullInt64{Int64: now + int64(24*time.Hour/time.Millisecond), Valid: true},
			ExchangeCodeHash:     hash.Sha256(code),
			Now:                  now,
		})
		require.NoError(t, err)
		affected, err := redeemed.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(1), affected)

		req := handler.Request{Code: code}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 401, res.Status)
		require.Equal(t, "Session is invalid, expired, or has already been used.", res.Body.Error.Detail)
		require.NotContains(t, res.RawBody, code)
	})

	t.Run("empty code", func(t *testing.T) {
		req := handler.Request{Code: ""}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})
}
