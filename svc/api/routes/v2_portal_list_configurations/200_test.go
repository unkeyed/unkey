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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_configurations"
)

func TestListConfigurationsSuccess(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	rootKey := h.CreateRootKey(workspaceID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("empty list", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotNil(t, res.Body)
		require.Empty(t, res.Body.Data)
	})

	t.Run("lists configs with branding, newest first", func(t *testing.T) {
		now := time.Now().UnixMilli()

		olderID := uid.New(uid.PortalConfigPrefix)
		require.NoError(t, db.Query.InsertPortalConfig(ctx, h.DB.RW(), db.InsertPortalConfigParams{
			ID:          olderID,
			WorkspaceID: workspaceID,
			Slug:        "older-portal",
			KeyAuthID:   sql.NullString{Valid: true, String: uid.New(uid.KeySpacePrefix)},
			Enabled:     true,
			CreatedAt:   now,
		}))
		require.NoError(t, db.Query.UpsertPortalBranding(ctx, h.DB.RW(), db.UpsertPortalBrandingParams{
			PortalConfigID: olderID,
			LogoUrl:        sql.NullString{Valid: true, String: "https://cdn.example.com/logo.png"},
			PrimaryColor:   sql.NullString{Valid: true, String: "#4f46e5"},
			CreatedAt:      now,
		}))

		newerID := uid.New(uid.PortalConfigPrefix)
		require.NoError(t, db.Query.InsertPortalConfig(ctx, h.DB.RW(), db.InsertPortalConfigParams{
			ID:          newerID,
			WorkspaceID: workspaceID,
			Slug:        "newer-portal",
			AppID:       sql.NullString{Valid: true, String: uid.New(uid.AppPrefix)},
			Enabled:     false,
			CreatedAt:   now + 1000,
		}))

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Len(t, res.Body.Data, 2)

		// Ordered by created_at DESC.
		require.Equal(t, "newer-portal", res.Body.Data[0].Slug)
		require.Equal(t, "older-portal", res.Body.Data[1].Slug)

		// Newer is app-mapped, disabled, no branding.
		require.NotEmpty(t, res.Body.Data[0].AppId)
		require.False(t, res.Body.Data[0].Enabled)
		require.Nil(t, res.Body.Data[0].Branding)

		// Older is keyspace-mapped with branding.
		require.NotEmpty(t, res.Body.Data[1].KeyspaceId)
		require.NotNil(t, res.Body.Data[1].Branding)
		require.Equal(t, "https://cdn.example.com/logo.png", res.Body.Data[1].Branding.LogoUrl)
	})
}

func TestListConfigurationsWorkspaceIsolation(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	// Config in workspace A.
	workspaceA := h.Resources().UserWorkspace.ID
	require.NoError(t, db.Query.InsertPortalConfig(ctx, h.DB.RW(), db.InsertPortalConfigParams{
		ID:          uid.New(uid.PortalConfigPrefix),
		WorkspaceID: workspaceA,
		Slug:        "workspace-a-portal",
		KeyAuthID:   sql.NullString{Valid: true, String: uid.New(uid.KeySpacePrefix)},
		Enabled:     true,
		CreatedAt:   time.Now().UnixMilli(),
	}))

	// List as workspace B: must not see workspace A's config.
	workspaceB := h.CreateWorkspace()
	rootKeyB := h.CreateRootKey(workspaceB.ID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKeyB)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})
	require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
	require.Empty(t, res.Body.Data)
	require.NotContains(t, res.RawBody, "workspace-a-portal")
}
