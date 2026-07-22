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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_delete_configuration"
)

func TestDeleteConfigurationSuccess(t *testing.T) {
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

	now := time.Now().UnixMilli()
	id := uid.New(uid.PortalConfigPrefix)
	require.NoError(t, db.Query.InsertPortalConfig(ctx, h.DB.RW(), db.InsertPortalConfigParams{
		ID:          id,
		WorkspaceID: workspaceID,
		Slug:        "delete-me",
		KeyAuthID:   sql.NullString{Valid: true, String: uid.New(uid.KeySpacePrefix)},
		Enabled:     true,
		CreatedAt:   now,
	}))
	require.NoError(t, db.Query.UpsertPortalBranding(ctx, h.DB.RW(), db.UpsertPortalBrandingParams{
		PortalConfigID: id,
		LogoUrl:        sql.NullString{Valid: true, String: "https://cdn.example.com/logo.png"},
		CreatedAt:      now,
	}))

	req := handler.Request{ConfigId: id}
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)

	// Config row is gone.
	_, err := db.Query.FindPortalConfigByID(ctx, h.DB.RO(), db.FindPortalConfigByIDParams{ID: id, WorkspaceID: workspaceID})
	require.True(t, db.IsNotFound(err), "config should be deleted, err=%v", err)

	// Branding row is gone too.
	_, err = db.Query.FindPortalBrandingByConfigID(ctx, h.DB.RO(), id)
	require.True(t, db.IsNotFound(err), "branding should be deleted, err=%v", err)
}
