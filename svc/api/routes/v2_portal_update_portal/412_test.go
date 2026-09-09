package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_update_portal"
)

// Re-pointing a portal at a resource in another project is refused, and the
// refusal says so rather than masking as not-found. The caller has proved update
// rights on this portal and the resource is in their own workspace, so there is
// nothing to conceal, and the request carries no project field they could
// correct. Both cases below cross projects in ordinary use: keyspaces land in
// the workspace default project while apps take a caller-named one.
func TestUpdatePortalRejectsMappingInAnotherProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.update_portal")
	workspace := h.Resources().UserWorkspace

	homeProject, _, homeKeyspace := mappingsInOneProject(t, h, workspace.ID, "home")
	stored := h.SeedPortal(t, workspace.ID, "homebound", "homebound", homeKeyspace, nil, nil)
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{homeKeyspace.ID}, []string{"keys.read"})

	_, elsewhereApp, elsewhereKeyspace := mappingsInOneProject(t, h, workspace.ID, "elsewhere")

	testCases := map[string]portal.Mapping{
		"keyspace in another project": elsewhereKeyspace,
		"app in another project":      elsewhereApp,
	}

	for name, requested := range testCases {
		t.Run(name, func(t *testing.T) {
			req := baseRequest(stored.ID)
			req.KeyspaceId = ksOf(requested)
			req.AppId = appOf(requested)

			res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, headers, req)
			require.Equal(t, http.StatusPreconditionFailed, res.Status,
				"expected 412, received: %s", res.RawBody)
			require.Contains(t, res.Body.Error.Detail, "different project",
				"the caller can see this resource, so the refusal must say what is wrong")
			require.NotContains(t, res.Body.Error.Detail, "not found",
				"answering not-found for a visible resource is a false answer")

			row := fetchPortal(t, h, workspace.ID, stored.ID)
			require.Equal(t, homeKeyspace.ID, row.KeyAuthID.String, "the association must not change")
			require.False(t, row.AppID.Valid)
			require.Equal(t, homeProject, row.ProjectID, "the portal must stay in its own project")
			require.Equal(t, 1, liveSessions(t, h, stored.ID),
				"a rejected re-point must not revoke sessions")
		})
	}
}
