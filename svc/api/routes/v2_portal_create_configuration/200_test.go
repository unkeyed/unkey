package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_configuration"
)

func TestCreateConfigurationSuccess(t *testing.T) {
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

	t.Run("keyspace-mapped with branding", func(t *testing.T) {
		keyspaceID := uid.New(uid.KeySpacePrefix)
		logo := "https://cdn.example.com/logo.png"
		color := "#4f46e5"
		req := handler.Request{
			Slug:       "portal-keyspace",
			KeyspaceId: &keyspaceID,
			Branding:   &openapi.PortalBranding{LogoUrl: logo, PrimaryColor: color},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotNil(t, res.Body)

		require.NotEmpty(t, res.Body.Data.Id)
		require.Equal(t, "portal-keyspace", res.Body.Data.Slug)
		require.Equal(t, keyspaceID, res.Body.Data.KeyspaceId)
		require.Empty(t, res.Body.Data.AppId)
		require.True(t, res.Body.Data.Enabled)
		require.NotNil(t, res.Body.Data.Branding)
		require.Equal(t, logo, res.Body.Data.Branding.LogoUrl)
		require.Equal(t, color, res.Body.Data.Branding.PrimaryColor)

		// Config + branding rows must be persisted.
		stored, err := db.Query.FindPortalConfigByID(ctx, h.DB.RO(), db.FindPortalConfigByIDParams{
			ID:          res.Body.Data.Id,
			WorkspaceID: workspaceID,
		})
		require.NoError(t, err)
		require.Equal(t, keyspaceID, stored.KeyAuthID.String)
		require.False(t, stored.AppID.Valid)

		branding, err := db.Query.FindPortalBrandingByConfigID(ctx, h.DB.RO(), res.Body.Data.Id)
		require.NoError(t, err)
		require.Equal(t, logo, branding.LogoUrl.String)
		require.Equal(t, color, branding.PrimaryColor.String)
	})

	t.Run("app-mapped without branding, disabled", func(t *testing.T) {
		appID := uid.New(uid.AppPrefix)
		enabled := false
		req := handler.Request{
			Slug:    "portal-app",
			AppId:   &appID,
			Enabled: &enabled,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, appID, res.Body.Data.AppId)
		require.Empty(t, res.Body.Data.KeyspaceId)
		require.False(t, res.Body.Data.Enabled)
		require.Nil(t, res.Body.Data.Branding)

		branding, err := db.Query.FindPortalBrandingByConfigID(ctx, h.DB.RO(), res.Body.Data.Id)
		require.True(t, db.IsNotFound(err), "expected no branding row, got branding=%+v err=%v", branding, err)
	})
}
