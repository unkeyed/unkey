package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
)

func TestCreateSessionBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:            h.DB,
		Keys:          h.Keys,
		PortalBaseURL: "https://portal.unkey.com",
	}
	h.Register(route)

	// Seed a portal config + frontline route so we isolate validation errors.
	workspaceID := h.Resources().UserWorkspace.ID
	portalConfigID := uid.New(uid.PortalConfigPrefix)
	now := time.Now().UnixMilli()

	err := db.Query.InsertPortalConfig(ctx, h.DB.RW(), db.InsertPortalConfigParams{
		ID:          portalConfigID,
		WorkspaceID: workspaceID,
		KeyAuthID:   sql.NullString{Valid: true, String: uid.New(uid.KeySpacePrefix)},
		Enabled:     true,
		CreatedAt:   now,
	})
	require.NoError(t, err)

	err = db.Query.InsertPortalFrontlineRoute(ctx, h.DB.RW(), db.InsertPortalFrontlineRouteParams{
		ID:                       uid.New(uid.FrontlineRoutePrefix),
		PortalConfigID:           sql.NullString{Valid: true, String: portalConfigID},
		PathPrefix:               sql.NullString{Valid: true, String: "/portal"},
		FullyQualifiedDomainName: fmt.Sprintf("test-400-%s.unkey.com", uid.New(uid.TestPrefix)),
		CreatedAt:                now,
	})
	require.NoError(t, err)

	rootKey := h.CreateRootKey(workspaceID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("missing externalId", func(t *testing.T) {
		req := handler.Request{
			Permissions: []string{"keys:read"},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty externalId", func(t *testing.T) {
		req := handler.Request{
			ExternalID:  "",
			Permissions: []string{"keys:read"},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("missing permissions", func(t *testing.T) {
		req := handler.Request{
			ExternalID: "user_123",
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty permissions array", func(t *testing.T) {
		req := handler.Request{
			ExternalID:  "user_123",
			Permissions: []string{},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})
}
