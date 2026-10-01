package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_revoke_session"
)

// A session grant naming another project reaches nothing.
func TestRevokeSessionGrantIsScopedToProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := registerRoute(h)
	workspace := h.Resources().UserWorkspace

	stored, mapping := seedPortal(t, h, workspace.ID, "revoke-project")
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})

	otherProject := uid.New(uid.ProjectPrefix)
	rootKey := h.CreateRootKey(workspace.ID,
		fmt.Sprintf("unkey:v1:%s:projects/%s/portals/%s/sessions/*#write", workspace.ID, otherProject, stored.ID))

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, testutil.RootKeyHeaders(rootKey), request(stored.ID, "user_1"))
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.Equal(t, 1, h.CountLivePortalSessions(t, stored.ID, "user_1"))
}
