package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
)

func TestCreateSessionSuccess(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:   h.DB,
		Keys: h.Keys,
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	portalConfigID := uid.New(uid.PortalConfigPrefix)
	now := time.Now().UnixMilli()
	fqdn := "test-success.unkey.com"

	err := db.Query.InsertPortalConfig(ctx, h.DB.RW(), db.InsertPortalConfigParams{
		ID:          portalConfigID,
		WorkspaceID: workspaceID,
		KeyAuthID:   sql.NullString{Valid: true, String: "ks_test"},
		Enabled:     true,
		CreatedAt:   now,
	})
	require.NoError(t, err)

	err = db.Query.InsertPortalFrontlineRoute(ctx, h.DB.RW(), db.InsertPortalFrontlineRouteParams{
		ID:                       uid.New(uid.FrontlineRoutePrefix),
		PortalConfigID:           sql.NullString{Valid: true, String: portalConfigID},
		PathPrefix:               sql.NullString{Valid: true, String: "/portal"},
		FullyQualifiedDomainName: fqdn,
		CreatedAt:                now,
	})
	require.NoError(t, err)

	rootKey := h.CreateRootKey(workspaceID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("basic session creation", func(t *testing.T) {
		req := handler.Request{
			ExternalID:  "user_123",
			Permissions: []string{"keys:read"},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)

		require.NotEmpty(t, res.Body.Data.SessionID)
		require.NotEmpty(t, res.Body.Data.URL)
		require.NotZero(t, res.Body.Data.ExpiresAt)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// URL must contain the session ID and the correct FQDN.
		require.Contains(t, res.Body.Data.URL, fqdn)
		require.Contains(t, res.Body.Data.URL, res.Body.Data.SessionID)
		require.True(t, strings.HasPrefix(res.Body.Data.URL, "https://"))
	})

	t.Run("session token expiry is 15 minutes", func(t *testing.T) {
		before := time.Now()

		req := handler.Request{
			ExternalID:  "user_456",
			Permissions: []string{"keys:read", "analytics:read"},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)

		after := time.Now()

		expectedLow := before.Add(15 * time.Minute).UnixMilli()
		expectedHigh := after.Add(15 * time.Minute).UnixMilli()

		require.GreaterOrEqual(t, res.Body.Data.ExpiresAt, expectedLow)
		require.LessOrEqual(t, res.Body.Data.ExpiresAt, expectedHigh)
	})

	t.Run("with metadata and preview", func(t *testing.T) {
		req := handler.Request{
			ExternalID:  "user_789",
			Permissions: []string{"keys:read"},
			Metadata:    map[string]any{"name": "Test User", "email": "test@example.com"},
			Preview:     true,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.SessionID)

		// Verify the token was persisted with correct fields.
		token, err := db.Query.FindValidPortalSessionToken(ctx, h.DB.RO(), res.Body.Data.SessionID)
		require.NoError(t, err)
		require.Equal(t, "user_789", token.ExternalID)
		require.True(t, token.Preview)
		require.NotNil(t, token.Metadata)
	})

	t.Run("multiple sessions for same externalId", func(t *testing.T) {
		req := handler.Request{
			ExternalID:  "user_multi",
			Permissions: []string{"keys:read"},
		}

		res1 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res1.Status)

		res2 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res2.Status)

		// Each call must produce a unique session ID.
		require.NotEqual(t, res1.Body.Data.SessionID, res2.Body.Data.SessionID)
	})
}
