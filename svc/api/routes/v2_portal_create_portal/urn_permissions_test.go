package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_portal"
)

// TestCreatePortalAuthorizesAdminURNAndLegacyTuple guarantees the dashboard
// admin grant and existing root-key permission can create a portal.
func TestCreatePortalAuthorizesAdminURNAndLegacyTuple(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	testCases := map[string][]string{
		"workspace-wide admin URN": {
			fmt.Sprintf("unkey:v1:%s:**#*", workspace.ID),
		},
		"legacy tuples": append([]string{"portal.*.create_portal"}, targetReadGrants...),
	}

	i := 0
	for name, grants := range testCases {
		i++
		t.Run(name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, grants...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Slug:        fmt.Sprintf("urn-portal-%d", i),
				DisplayName: "Acme",
				KeyspaceId:  ksOf(keyspaceMapping(t, h, workspace.ID)),
				AppId:       appOf(keyspaceMapping(t, h, workspace.ID)),
				Enabled:     ptr.P(true),
			})
			require.Equal(t, http.StatusOK, res.Status, "the grant must authorize portal creation: %s", res.RawBody)
		})
	}
}
