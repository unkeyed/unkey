package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_sessions"
)

// A portal read grant naming another project reaches nothing.
func TestListSessionsGrantIsScopedToProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := registerRoute(h)
	workspace := h.Resources().UserWorkspace
	stored := seedPortal(t, h, workspace.ID, "list-project")
	insertSession(t, h, stored.ID, workspace.ID, active(h, "user_1"))

	otherProject := uid.New(uid.ProjectPrefix)
	rootKey := h.CreateRootKey(workspace.ID,
		fmt.Sprintf("unkey:v1:%s:projects/%s/portals/*#read", workspace.ID, otherProject))

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headersFor(rootKey), request(stored.ID))
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
}
