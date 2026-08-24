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
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
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

// Re-pointing a portal decides which keyspace its end users reach, so it takes
// more than `update_portal`: the caller must also be able to read the target.
// Without this a narrow ops key could redirect a portal at a keyspace it has no
// rights over, and the customer's own backend would then mint sessions against
// it.
func TestUpdatePortalRequiresPermissionOnTheRemapTarget(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	from := keyspaceMapping(t, h, workspace.ID)
	to := keyspaceMapping(t, h, workspace.ID)
	stored := seedPortal(t, h, workspace.ID, "remap-authz", from, nullStringAbsent(), nullStringAbsent())

	t.Run("update_portal alone cannot remap", func(t *testing.T) {
		rootKey := h.CreateRootKey(workspace.ID, "portal.*.update_portal")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer " + rootKey},
		}

		req := baseRequest(stored.ID)
		req.KeyspaceId = ksOf(to)
		req.AppId = appOf(to)
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status,
			"expected 403, received: %s", res.RawBody)

		after := fetchPortal(t, h, workspace.ID, stored.ID)
		require.Equal(t, from.ID, after.KeyAuthID.String, "the mapping must be unchanged")
	})

	t.Run("update_portal plus read on the target succeeds", func(t *testing.T) {
		rootKey := h.CreateRootKey(workspace.ID, "portal.*.update_portal", "api.*.read_api")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer " + rootKey},
		}

		req := baseRequest(stored.ID)
		req.KeyspaceId = ksOf(to)
		req.AppId = appOf(to)
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

		after := fetchPortal(t, h, workspace.ID, stored.ID)
		require.Equal(t, to.ID, after.KeyAuthID.String)
	})

	// An update that does not name a mapping is unaffected: this gates the remap,
	// not every write.
	t.Run("a non-remap update needs no target permission", func(t *testing.T) {
		rootKey := h.CreateRootKey(workspace.ID, "portal.*.update_portal")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer " + rootKey},
		}

		req := baseRequest(stored.ID)
		req.Enabled = ptr(false)
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	})
}
