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

func TestUpdateConfigurationBadRequest(t *testing.T) {
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

	keyspaceID := uid.New(uid.KeySpacePrefix)
	appID := uid.New(uid.AppPrefix)
	id := uid.New(uid.PortalConfigPrefix)
	require.NoError(t, db.Query.InsertPortalConfig(ctx, h.DB.RW(), db.InsertPortalConfigParams{
		ID:          id,
		WorkspaceID: workspaceID,
		Slug:        "update-bad-request",
		KeyAuthID:   sql.NullString{Valid: true, String: keyspaceID},
		Enabled:     true,
		CreatedAt:   time.Now().UnixMilli(),
	}))

	t.Run("invalid slug", func(t *testing.T) {
		req := handler.Request{ConfigId: id, Slug: "Bad__Slug", KeyspaceId: &keyspaceID}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
	})

	t.Run("neither keyspace nor app", func(t *testing.T) {
		req := handler.Request{ConfigId: id, Slug: "update-bad-request"}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, received: %s", res.RawBody)
	})

	t.Run("both keyspace and app", func(t *testing.T) {
		req := handler.Request{ConfigId: id, Slug: "update-bad-request", KeyspaceId: &keyspaceID, AppId: &appID}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, received: %s", res.RawBody)
	})
}
