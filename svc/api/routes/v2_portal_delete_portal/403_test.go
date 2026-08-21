package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_delete_portal"
)

// Delete masks a denial as a not-found, unlike create: the portal exists, and a
// 403 would confirm that to a caller that has no grant on it. So this matrix
// asserts 404 for every insufficient grant, and that nothing was deleted or
// revoked.
//
// The status is asserted explicitly rather than only the code, because the
// duplicate-and-not-found switch in the error middleware carries
// //nolint:exhaustive and a missing case would fall through to a 500.
func TestDeletePortalAuthorizationMatrix(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	otherPortalID := uid.New(uid.PortalPrefix)

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "no permissions", permissions: []string{}, shouldPass: false},
		{name: "wildcard delete_portal", permissions: []string{"portal.*.delete_portal"}, shouldPass: true},
		{name: "read_portal only", permissions: []string{"portal.*.read_portal"}, shouldPass: false},
		{name: "create_portal only", permissions: []string{"portal.*.create_portal"}, shouldPass: false},
		{name: "update_portal only", permissions: []string{"portal.*.update_portal"}, shouldPass: false},
		{name: "session minting does not grant deletes", permissions: []string{"portal.*.create_portal_session"}, shouldPass: false},
		{name: "unrelated api permission", permissions: []string{"api.*.read_api"}, shouldPass: false},
		{name: "delete_portal scoped to another portal", permissions: []string{fmt.Sprintf("portal.%s.delete_portal", otherPortalID)}, shouldPass: false},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// A portal per case: a delete is not idempotent, so the cases cannot share
			// a fixture the way an update matrix can.
			mapping := keyspaceMapping(t, h, workspace.ID)
			stored := seedPortal(t, h, workspace.ID, fmt.Sprintf("gated-%d", i), mapping)
			h.CreatePortalSessionForPortal(stored.ID, workspace.ID,
				fmt.Sprintf("user_%d", i), []string{mapping.Id}, []string{"keys:read"})

			rootKey := h.CreateRootKey(workspace.ID, tc.permissions...)

			// Counted per case rather than asserted as zero: the harness is shared
			// across subtests, so a passing case above has already written an entry.
			auditBefore := countAuditEntriesMentioning(t, h, workspace.ID, "portal.delete")

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), request(stored.ID))

			if tc.shouldPass {
				require.Equal(t, http.StatusOK, res.Status,
					"expected 200 for %v, got: %s", tc.permissions, res.RawBody)
				require.False(t, portalExists(t, h, workspace.ID, stored.ID))
				return
			}

			require.Equal(t, http.StatusNotFound, res.Status,
				"expected a masked 404 for %v, got: %s", tc.permissions, res.RawBody)
			require.NotContains(t, res.RawBody, stored.ID,
				"a masked denial must not disclose the portal id")
			require.NotContains(t, res.RawBody, stored.Slug,
				"a masked denial must not disclose the portal slug")
			require.True(t, portalExists(t, h, workspace.ID, stored.ID),
				"a denied request must not delete")
			require.Equal(t, 1, liveSessions(t, h, stored.ID),
				"a denied request must not revoke a session")
			require.Equal(t, auditBefore, countAuditEntriesMentioning(t, h, workspace.ID, "portal.delete"),
				"a denied request must not write an audit entry")
		})
	}
}

// The id-scoped grant is the arm the second `rbac.T` exists for. A grant naming
// this portal is enough; one naming another portal is not, and is asserted in the
// matrix above.
func TestDeletePortalAllowsGrantScopedToThisPortal(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	stored := seedPortal(t, h, workspace.ID, "scoped", keyspaceMapping(t, h, workspace.ID))

	rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("portal.%s.delete_portal", stored.ID))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), request(stored.ID))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.False(t, portalExists(t, h, workspace.ID, stored.ID))
}
