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

func TestUpdateConfigurationConflict(t *testing.T) {
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

	// Two configs: renaming the second onto the first's slug must conflict.
	require.NoError(t, db.Query.InsertPortalConfig(ctx, h.DB.RW(), db.InsertPortalConfigParams{
		ID:          uid.New(uid.PortalConfigPrefix),
		WorkspaceID: workspaceID,
		Slug:        "occupied",
		KeyAuthID:   sql.NullString{Valid: true, String: uid.New(uid.KeySpacePrefix)},
		Enabled:     true,
		CreatedAt:   now,
	}))

	otherKeyspace := uid.New(uid.KeySpacePrefix)
	otherID := uid.New(uid.PortalConfigPrefix)
	require.NoError(t, db.Query.InsertPortalConfig(ctx, h.DB.RW(), db.InsertPortalConfigParams{
		ID:          otherID,
		WorkspaceID: workspaceID,
		Slug:        "mover",
		KeyAuthID:   sql.NullString{Valid: true, String: otherKeyspace},
		Enabled:     true,
		CreatedAt:   now + 1000,
	}))

	req := handler.Request{ConfigId: otherID, Slug: "occupied", DisplayName: "Mover", KeyspaceId: &otherKeyspace}
	res := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](h, route, headers, req)
	require.Equal(t, 409, res.Status, "expected 409, received: %s", res.RawBody)
}
