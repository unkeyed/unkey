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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_configuration"
)

func TestCreateConfigurationConflict(t *testing.T) {
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

	existingKeyspace := uid.New(uid.KeySpacePrefix)
	require.NoError(t, db.Query.InsertPortalConfig(ctx, h.DB.RW(), db.InsertPortalConfigParams{
		ID:          uid.New(uid.PortalConfigPrefix),
		WorkspaceID: workspaceID,
		Slug:        "taken-slug",
		KeyAuthID:   sql.NullString{Valid: true, String: existingKeyspace},
		Enabled:     true,
		CreatedAt:   time.Now().UnixMilli(),
	}))

	t.Run("duplicate slug", func(t *testing.T) {
		keyspaceID := uid.New(uid.KeySpacePrefix)
		req := handler.Request{Slug: "taken-slug", KeyspaceId: &keyspaceID}
		res := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](h, route, headers, req)
		require.Equal(t, 409, res.Status, "expected 409, received: %s", res.RawBody)
	})

	t.Run("duplicate keyspace", func(t *testing.T) {
		req := handler.Request{Slug: "fresh-slug", KeyspaceId: &existingKeyspace}
		res := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](h, route, headers, req)
		require.Equal(t, 409, res.Status, "expected 409, received: %s", res.RawBody)
	})
}
