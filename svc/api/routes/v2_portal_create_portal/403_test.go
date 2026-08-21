package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_portal"
)

// Create denies with 403 rather than the masked 404 the other portal routes use.
// There is no portal yet, so a denial cannot disclose one's existence, and every
// other v2 create in the repo answers a permission gap with 403. Masking here
// would only leave an operator unable to tell a missing grant from a bad request.
func TestCreatePortalAuthorizationMatrix(t *testing.T) {
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
		{name: "wildcard create_portal", permissions: []string{"portal.*.create_portal"}, shouldPass: true},
		{name: "create_portal among others", permissions: []string{"some.other.permission", "portal.*.create_portal"}, shouldPass: true},
		// The id does not exist when the request is authorized, so only a
		// wildcard can carry this. An id-scoped grant naming some other portal is
		// the closest a caller can get, and it must not be enough.
		{name: "create_portal scoped to a portal id", permissions: []string{fmt.Sprintf("portal.%s.create_portal", otherPortalID)}, shouldPass: false},
		{name: "read_portal only", permissions: []string{"portal.*.read_portal"}, shouldPass: false},
		{name: "update_portal only", permissions: []string{"portal.*.update_portal"}, shouldPass: false},
		{name: "delete_portal only", permissions: []string{"portal.*.delete_portal"}, shouldPass: false},
		{name: "session minting does not grant creation", permissions: []string{"portal.*.create_portal_session"}, shouldPass: false},
		{name: "unrelated api permission", permissions: []string{"api.*.read_api"}, shouldPass: false},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// The target-read grants are constant across cases: this matrix is about
			// the portal action, and without them every pass case would fail on the
			// mapping-target check instead.
			rootKey := h.CreateRootKey(workspace.ID,
				append(append([]string{}, tc.permissions...), targetReadGrants...)...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			before := countPortals(t, h, workspace.ID)
			// Counted per case rather than asserted as zero: the harness is shared
			// across subtests, so the passing cases above have already written
			// entries of their own.
			auditBefore := countAuditEntriesMentioning(t, h, workspace.ID, "portal.create")
			// A distinct slug and mapping per case, so a pass cannot collide with
			// an earlier one and report a conflict instead of success.
			mapping := keyspaceMapping(t, h, workspace.ID)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Slug:    fmt.Sprintf("matrix-%d", i),
				Mapping: mapping,
				Enabled: ptr(true),
			})

			if tc.shouldPass {
				require.Equal(t, http.StatusOK, res.Status,
					"expected 200 for %v, got: %s", tc.permissions, res.RawBody)
				return
			}

			require.Equal(t, http.StatusForbidden, res.Status,
				"expected 403 for %v, got: %s", tc.permissions, res.RawBody)
			require.Equal(t, before, countPortals(t, h, workspace.ID),
				"a denied request must not write a portal")
			require.Equal(t, auditBefore, countAuditEntriesMentioning(t, h, workspace.ID, "portal.create"),
				"a denied request must not write an audit entry")
		})
	}
}
