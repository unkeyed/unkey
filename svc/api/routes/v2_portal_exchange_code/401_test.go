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
	portalID := uid.New(uid.PortalPrefix)
	now := time.Now().UnixMilli()

	// Seed a portal for session insertion.
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

	t.Run("invalid exchange code", func(t *testing.T) {
		req := handler.Request{Code: "nonexistent_exchange_code"}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 401, res.Status)
		require.Equal(t, "Session is invalid, expired, or has already been used.", res.Body.Error.Detail)
		require.NotContains(t, res.RawBody, req.Code)
	})

	t.Run("expired exchange code", func(t *testing.T) {
		sessionID := uid.New(uid.PortalSessionPrefix)
		exchangeCode := uid.Secure()
		perms, _ := json.Marshal([]string{"api.*.read_key"})

		// Insert an exchange code that expired 1 hour ago.
		err := db.Query.InsertPortalSession(ctx, h.DB.RW(), db.InsertPortalSessionParams{
			ID:                    sessionID,
			WorkspaceID:           workspaceID,
			PortalID:              portalID,
			ExternalID:            "user_expired",
			Permissions:           perms,
			ExchangeCodeHash:      sql.NullString{String: hash.Sha256(exchangeCode), Valid: true},
			ExchangeCodeExpiresAt: now - int64(time.Hour/time.Millisecond),
			CreatedAt:             now - int64(2*time.Hour/time.Millisecond),
		})
		require.NoError(t, err)

		req := handler.Request{Code: exchangeCode}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 401, res.Status)
		require.Equal(t, "Session is invalid, expired, or has already been used.", res.Body.Error.Detail)
		require.NotContains(t, res.RawBody, exchangeCode)
	})

	t.Run("already consumed exchange code", func(t *testing.T) {
		sessionID := uid.New(uid.PortalSessionPrefix)
		exchangeCode := uid.Secure()
		accessToken := uid.Secure()
		perms, _ := json.Marshal([]string{"api.*.read_key"})

		// Insert a session whose exchange code has already been consumed.
		err := db.Query.InsertPortalSession(ctx, h.DB.RW(), db.InsertPortalSessionParams{
			ID:                    sessionID,
			WorkspaceID:           workspaceID,
			PortalID:              portalID,
			ExternalID:            "user_exchanged",
			Permissions:           perms,
			ExchangeCodeExpiresAt: now + int64(15*time.Minute/time.Millisecond),
			AccessTokenCreatedAt:  sql.NullInt64{Int64: now, Valid: true},
			AccessTokenHash:       sql.NullString{String: hash.Sha256(accessToken), Valid: true},
			AccessTokenExpiresAt:  sql.NullInt64{Int64: now + int64(24*time.Hour/time.Millisecond), Valid: true},
			CreatedAt:             now,
		})
		require.NoError(t, err)

		req := handler.Request{Code: exchangeCode}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 401, res.Status)
		require.Equal(t, "Session is invalid, expired, or has already been used.", res.Body.Error.Detail)
		require.NotContains(t, res.RawBody, exchangeCode)
	})

	t.Run("empty code", func(t *testing.T) {
		req := handler.Request{Code: ""}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})
}
