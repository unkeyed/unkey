package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
)

func TestCreateSessionNotFoundNonExistentPortalId(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID

	// Granted deliberately: the 404 must come from the portal lookup, not from a
	// missing permission. A caller who could mint sessions still cannot learn
	// whether an unknown portal exists.
	rootKey := h.CreateRootKey(workspaceID,
		"portal.*.create_portal_session",
		"api.*.read_key",
		"api.*.read_api",
	)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		Portal:     "nonexistent-portal",
		ExternalId: "user_123",
		Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
	}

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.Equal(t, "Portal not found.", res.Body.Error.Detail)
	require.NotContains(t, res.RawBody, req.Portal)
	require.NotContains(t, res.RawBody, req.ExternalId)
}

func TestCreateSessionNotFoundWrongWorkspace(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	h.Register(route)

	// Create a portal in workspace A (the default user workspace).
	workspaceA := h.Resources().UserWorkspace.ID
	projectID := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceA}).ProjectID

	portalID := insertKeyspacePortal(t, h, workspaceA, projectID, "cross-workspace-portal", uid.New(uid.KeySpacePrefix))

	// Authenticate as workspace B, holding every permission the mint would need
	// in its own workspace. Workspace A's portal must still be indistinguishable
	// from one that does not exist.
	workspaceB := h.CreateWorkspace()
	rootKeyB := h.CreateRootKey(workspaceB.ID,
		"portal.*.create_portal_session",
		"api.*.read_key",
		"api.*.read_api",
	)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKeyB)},
	}

	// Use workspace A's portal slug while authenticated as workspace B.
	req := handler.Request{
		Portal:     "cross-workspace-portal",
		ExternalId: "user_123",
		Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{"keys:read"},
	}

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.Equal(t, "Portal not found.", res.Body.Error.Detail)
	require.NotContains(t, res.RawBody, portalID)
	require.NotContains(t, res.RawBody, workspaceA)
}
