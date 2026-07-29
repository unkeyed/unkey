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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_update_configuration"
)

func TestUpdateConfigurationSuccess(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	rootKey := h.CreateRootKey(workspaceID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	seed := func(t *testing.T, slug string, keyspaceID string) string {
		t.Helper()
		id := uid.New(uid.PortalConfigPrefix)
		require.NoError(t, db.Query.InsertPortalConfig(ctx, h.DB.RW(), db.InsertPortalConfigParams{
			ID:          id,
			WorkspaceID: workspaceID,
			Slug:        slug,
			KeyAuthID:   sql.NullString{Valid: true, String: keyspaceID},
			Enabled:     true,
			CreatedAt:   time.Now().UnixMilli(),
		}))
		return id
	}

	t.Run("rename, disable, add branding", func(t *testing.T) {
		keyspaceID := uid.New(uid.KeySpacePrefix)
		id := seed(t, "before-rename", keyspaceID)

		enabled := false
		req := handler.Request{
			ConfigId:   id,
			Slug:       "after-rename",
			KeyspaceId: &keyspaceID,
			Enabled:    &enabled,
			Branding:   &openapi.PortalBranding{LogoUrl: "https://cdn.example.com/l.png", PrimaryColor: "#010203"},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, "after-rename", res.Body.Data.Slug)
		require.False(t, res.Body.Data.Enabled)
		require.NotZero(t, res.Body.Data.UpdatedAt)
		require.NotNil(t, res.Body.Data.Branding)
		require.Equal(t, "#010203", res.Body.Data.Branding.PrimaryColor)

		stored, err := db.Query.FindPortalConfigByID(ctx, h.DB.RO(), db.FindPortalConfigByIDParams{ID: id, WorkspaceID: workspaceID})
		require.NoError(t, err)
		require.Equal(t, "after-rename", stored.Slug)
		require.False(t, stored.Enabled)
	})

	t.Run("remap keyspace to app", func(t *testing.T) {
		keyspaceID := uid.New(uid.KeySpacePrefix)
		id := seed(t, "remap-me", keyspaceID)

		appID := uid.New(uid.AppPrefix)
		req := handler.Request{
			ConfigId: id,
			Slug:     "remap-me",
			AppId:    &appID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, appID, res.Body.Data.AppId)
		require.Empty(t, res.Body.Data.KeyspaceId)

		stored, err := db.Query.FindPortalConfigByID(ctx, h.DB.RO(), db.FindPortalConfigByIDParams{ID: id, WorkspaceID: workspaceID})
		require.NoError(t, err)
		require.False(t, stored.KeyAuthID.Valid)
		require.Equal(t, appID, stored.AppID.String)
	})
}
