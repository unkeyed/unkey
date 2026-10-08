package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_sessions"
)

// A stored grant that no longer parses fails the request rather than listing the
// session with no scopes.
func TestListSessionsFailsOnMalformedScopes(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	stored := seedPortal(t, h, workspace.ID, "list-malformed-scopes")
	route, headers := newRoute(t, h, rbac.U(urn.New().Workspace(workspace.ID).Project(stored.ProjectID).Portal(stored.ID).Session("*"), permissions.Read).Value)

	malformed := active(h, "user_1")
	malformed.scopes = `["keys:read"]`
	insertSession(t, h, stored.ID, workspace.ID, malformed)

	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, headers, request(stored.ID))
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
	require.Equal(t, "We're unable to list the portal sessions.", res.Body.Error.Detail)
}
