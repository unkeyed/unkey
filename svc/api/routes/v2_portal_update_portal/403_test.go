package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_update_portal"
)

// Update masks a denial as a not-found, unlike create: the portal exists, and a
// 403 would confirm that to a caller that has no grant on it. So this matrix
// asserts 404 for every insufficient grant, and that nothing was written.
//
// The status is asserted explicitly rather than only the code, because the
// duplicate-and-not-found switch in the error middleware carries
// //nolint:exhaustive and a missing case would fall through to a 500.
func TestUpdatePortalAuthorizationMatrix(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	stored := seedPortal(t, h, workspace.ID, "gated", keyspaceMapping(t, h, workspace.ID),
		nullStringAbsent(), nullStringAbsent())
	otherPortalID := uid.New(uid.PortalPrefix)

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "no permissions", permissions: []string{}, shouldPass: false},
		{name: "wildcard update_portal", permissions: []string{"portal.*.update_portal"}, shouldPass: true},
		{name: "update_portal scoped to this portal", permissions: []string{fmt.Sprintf("portal.%s.update_portal", stored.ID)}, shouldPass: true},
		{name: "update_portal scoped to another portal", permissions: []string{fmt.Sprintf("portal.%s.update_portal", otherPortalID)}, shouldPass: false},
		{name: "read_portal only", permissions: []string{"portal.*.read_portal"}, shouldPass: false},
		{name: "create_portal only", permissions: []string{"portal.*.create_portal"}, shouldPass: false},
		{name: "delete_portal only", permissions: []string{"portal.*.delete_portal"}, shouldPass: false},
		{name: "session minting does not grant updates", permissions: []string{"portal.*.create_portal_session"}, shouldPass: false},
		{name: "unrelated api permission", permissions: []string{"api.*.read_api"}, shouldPass: false},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, tc.permissions...)

			// Counted per case rather than asserted as zero: the harness is shared
			// across subtests, so a passing case above has already written an entry.
			auditBefore := countAuditEntriesMentioning(t, h, workspace.ID, "portal.update")

			req := baseRequest(stored.ID)
			req.Slug = ptr(fmt.Sprintf("gated-%d", i))

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), req)

			if tc.shouldPass {
				require.Equal(t, http.StatusOK, res.Status,
					"expected 200 for %v, got: %s", tc.permissions, res.RawBody)
				return
			}

			require.Equal(t, http.StatusNotFound, res.Status,
				"expected a masked 404 for %v, got: %s", tc.permissions, res.RawBody)
			require.NotContains(t, res.RawBody, stored.ID,
				"a masked denial must not disclose the portal id")
			require.NotEqual(t, fmt.Sprintf("gated-%d", i), fetchPortal(t, h, workspace.ID, stored.ID).Slug,
				"a denied request must not write")
			require.Equal(t, auditBefore, countAuditEntriesMentioning(t, h, workspace.ID, "portal.update"),
				"a denied request must not write an audit entry")
		})
	}
}
