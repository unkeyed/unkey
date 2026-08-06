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

func TestCreateSessionForbiddenDisabledPortal(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	portalID := uid.New(uid.PortalPrefix)
	now := time.Now().UnixMilli()

	// Insert a disabled portal.
	err := db.Query.InsertPortal(ctx, h.DB.RW(), db.InsertPortalParams{
		ID:          portalID,
		WorkspaceID: workspaceID,
		Slug:        "disabled-portal",
		KeyspaceID:  sql.NullString{Valid: true, String: uid.New(uid.KeySpacePrefix)},
		Enabled:     false,
		CreatedAt:   now,
	})
	require.NoError(t, err)

	rootKey := h.CreateRootKey(workspaceID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		Portal:      "disabled-portal",
		ExternalId:  "user_123",
		Permissions: []openapi.V2PortalCreateSessionRequestBodyPermissions{"keys:read"},
	}

	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
	require.Equal(t, 403, res.Status)
	require.NotNil(t, res.Body)
}
