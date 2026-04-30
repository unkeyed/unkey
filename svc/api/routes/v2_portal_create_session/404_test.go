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

func TestCreateSessionNotFoundNonExistentPortalId(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		Keys:          h.Keys,
		PortalBaseURL: "https://portal.unkey.com",
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	rootKey := h.CreateRootKey(workspaceID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		PortalId:    uid.New(uid.PortalConfigPrefix),
		ExternalId:  "user_123",
		Permissions: []string{"api.*.read_key"},
	}

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.NotNil(t, res.Body)
}

func TestCreateSessionNotFoundWrongWorkspace(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		Keys:          h.Keys,
		PortalBaseURL: "https://portal.unkey.com",
	}
	h.Register(route)

	// Create a portal config in workspace A (the default user workspace).
	workspaceA := h.Resources().UserWorkspace.ID
	portalConfigID := uid.New(uid.PortalConfigPrefix)
	now := time.Now().UnixMilli()

	err := db.Query.InsertPortalConfig(ctx, h.DB.RW(), db.InsertPortalConfigParams{
		ID:          portalConfigID,
		WorkspaceID: workspaceA,
		KeyAuthID:   sql.NullString{Valid: true, String: uid.New(uid.KeySpacePrefix)},
		Enabled:     true,
		CreatedAt:   now,
	})
	require.NoError(t, err)

	// Authenticate as workspace B.
	workspaceB := h.CreateWorkspace()
	rootKeyB := h.CreateRootKey(workspaceB.ID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKeyB)},
	}

	// Use workspace A's portal config ID while authenticated as workspace B.
	req := handler.Request{
		PortalId:    portalConfigID,
		ExternalId:  "user_123",
		Permissions: []string{"api.*.read_key"},
	}

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.NotNil(t, res.Body)
}
