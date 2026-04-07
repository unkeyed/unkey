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
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_exchange_session"
)

func TestExchangeSessionUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	portalConfigID := uid.New(uid.PortalConfigPrefix)
	now := time.Now().UnixMilli()

	// Seed portal config for token insertion.
	err := db.Query.InsertPortalConfig(ctx, h.DB.RW(), db.InsertPortalConfigParams{
		ID:          portalConfigID,
		WorkspaceID: workspaceID,
		KeyAuthID:   sql.NullString{Valid: true, String: uid.New(uid.KeySpacePrefix)},
		Enabled:     true,
		CreatedAt:   now,
	})
	require.NoError(t, err)

	headers := http.Header{
		"Content-Type": {"application/json"},
	}

	t.Run("invalid session_id", func(t *testing.T) {
		req := handler.Request{SessionID: "nonexistent_session"}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("expired session token", func(t *testing.T) {
		tokenID := uid.New(uid.PortalSessionTokenPrefix)
		perms, _ := json.Marshal([]string{"keys:read"})

		// Insert a token that expired 1 hour ago.
		err := db.Query.InsertPortalSessionToken(ctx, h.DB.RW(), db.InsertPortalSessionTokenParams{
			ID:             tokenID,
			WorkspaceID:    workspaceID,
			PortalConfigID: portalConfigID,
			ExternalID:     "user_expired",
			Permissions:    perms,
			ExpiresAt:      now - int64(time.Hour/time.Millisecond),
			CreatedAt:      now - int64(2*time.Hour/time.Millisecond),
		})
		require.NoError(t, err)

		req := handler.Request{SessionID: tokenID}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("already exchanged session token", func(t *testing.T) {
		tokenID := uid.New(uid.PortalSessionTokenPrefix)
		perms, _ := json.Marshal([]string{"keys:read"})

		// Insert a valid token that is already exchanged.
		err := db.Query.InsertPortalSessionToken(ctx, h.DB.RW(), db.InsertPortalSessionTokenParams{
			ID:             tokenID,
			WorkspaceID:    workspaceID,
			PortalConfigID: portalConfigID,
			ExternalID:     "user_exchanged",
			Permissions:    perms,
			ExpiresAt:      now + int64(15*time.Minute/time.Millisecond),
			CreatedAt:      now,
		})
		require.NoError(t, err)

		// Mark as exchanged.
		_, err = db.Query.ExchangePortalSessionToken(ctx, h.DB.RW(), db.ExchangePortalSessionTokenParams{
			ExchangedAt: sql.NullInt64{Int64: now, Valid: true},
			ID:          tokenID,
		})
		require.NoError(t, err)

		req := handler.Request{SessionID: tokenID}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty sessionId", func(t *testing.T) {
		req := handler.Request{SessionID: ""}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})
}
