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

func TestUpdateConfigurationNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	rootKey := h.CreateRootKey(workspaceID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	keyspaceID := uid.New(uid.KeySpacePrefix)

	t.Run("nonexistent id", func(t *testing.T) {
		req := handler.Request{ConfigId: uid.New(uid.PortalConfigPrefix), Slug: "does-not-exist", KeyspaceId: &keyspaceID}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		require.Equal(t, "Portal configuration not found.", res.Body.Error.Detail)
	})
}

func TestUpdateConfigurationNotFoundWrongWorkspace(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	// Config in workspace A.
	workspaceA := h.Resources().UserWorkspace.ID
	configID := uid.New(uid.PortalConfigPrefix)
	require.NoError(t, db.Query.InsertPortalConfig(ctx, h.DB.RW(), db.InsertPortalConfigParams{
		ID:          configID,
		WorkspaceID: workspaceA,
		Slug:        "cross-ws-update",
		KeyAuthID:   sql.NullString{Valid: true, String: uid.New(uid.KeySpacePrefix)},
		Enabled:     true,
		CreatedAt:   time.Now().UnixMilli(),
	}))

	// Update as workspace B, targeting workspace A's config id.
	workspaceB := h.CreateWorkspace()
	rootKeyB := h.CreateRootKey(workspaceB.ID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKeyB)},
	}

	keyspaceID := uid.New(uid.KeySpacePrefix)
	req := handler.Request{ConfigId: configID, Slug: "cross-ws-update", KeyspaceId: &keyspaceID}
	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)

	// Workspace A's config must be untouched.
	stored, err := db.Query.FindPortalConfigByID(ctx, h.DB.RO(), db.FindPortalConfigByIDParams{ID: configID, WorkspaceID: workspaceA})
	require.NoError(t, err)
	require.Equal(t, "cross-ws-update", stored.Slug)
}
