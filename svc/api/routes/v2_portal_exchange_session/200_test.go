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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_exchange_session"
)

func TestExchangeSessionSuccess(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	portalConfigID := uid.New(uid.PortalConfigPrefix)
	now := time.Now().UnixMilli()

	err := db.Query.InsertPortalConfig(ctx, h.DB.RW(), db.InsertPortalConfigParams{
		ID:          portalConfigID,
		WorkspaceID: workspaceID,
		KeyAuthID:   sql.NullString{Valid: true, String: "ks_test"},
		Enabled:     true,
		CreatedAt:   now,
	})
	require.NoError(t, err)

	headers := http.Header{
		"Content-Type": {"application/json"},
	}

	t.Run("valid exchange", func(t *testing.T) {
		tokenID := uid.New(uid.PortalSessionTokenPrefix)
		perms, _ := json.Marshal([]string{"keys:read", "analytics:read"})
		meta, _ := json.Marshal(map[string]any{"name": "Test User"})

		err := db.Query.InsertPortalSessionToken(ctx, h.DB.RW(), db.InsertPortalSessionTokenParams{
			ID:             tokenID,
			WorkspaceID:    workspaceID,
			PortalConfigID: portalConfigID,
			ExternalID:     "user_valid",
			Metadata:       meta,
			Permissions:    perms,
			ExpiresAt:      now + int64(15*time.Minute/time.Millisecond),
			CreatedAt:      now,
		})
		require.NoError(t, err)

		before := time.Now()

		req := handler.Request{SessionID: tokenID}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		require.NotEmpty(t, res.Body.Data.Token)
		require.NotZero(t, res.Body.Data.ExpiresAt)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// Browser session expiry must be ~24 hours from now.
		after := time.Now()
		expectedLow := before.Add(24 * time.Hour).UnixMilli()
		expectedHigh := after.Add(24 * time.Hour).UnixMilli()
		require.GreaterOrEqual(t, res.Body.Data.ExpiresAt, expectedLow)
		require.LessOrEqual(t, res.Body.Data.ExpiresAt, expectedHigh)

		// Verify the browser session was persisted.
		session, err := db.Query.FindValidPortalSession(ctx, h.DB.RO(), res.Body.Data.Token)
		require.NoError(t, err)
		require.Equal(t, workspaceID, session.WorkspaceID)
		require.Equal(t, "user_valid", session.ExternalID)
		require.Equal(t, portalConfigID, session.PortalConfigID)
	})

	t.Run("single-use enforcement", func(t *testing.T) {
		tokenID := uid.New(uid.PortalSessionTokenPrefix)
		perms, _ := json.Marshal([]string{"keys:read"})

		err := db.Query.InsertPortalSessionToken(ctx, h.DB.RW(), db.InsertPortalSessionTokenParams{
			ID:             tokenID,
			WorkspaceID:    workspaceID,
			PortalConfigID: portalConfigID,
			ExternalID:     "user_single_use",
			Permissions:    perms,
			ExpiresAt:      now + int64(15*time.Minute/time.Millisecond),
			CreatedAt:      now,
		})
		require.NoError(t, err)

		req := handler.Request{SessionID: tokenID}

		// First exchange succeeds.
		res1 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res1.Status)

		// Second exchange must fail.
		res2 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 401, res2.Status)
	})
}
