package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_portal"
)

// Managing a portal as a resource accepts canonical URN grants. Acting as one --
// minting a session -- deliberately does not, and its own test pins that denial.
//
// This is what lets the dashboard reach the route: its proxy rewrites an admin
// grant into a workspace-wide URN, so a portal route that evaluated legacy tuples
// only would deny the single operator surface that exists.
func TestCreatePortalAuthorizesURNGrants(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	testCases := map[string]string{
		"portal-scoped wildcard URN": fmt.Sprintf("unkey:v1:%s:portals/*#create_portal", workspace.ID),
		// The form the dashboard proxy actually mints from admin:*.
		"workspace-wide admin URN": fmt.Sprintf("unkey:v1:%s:**#*", workspace.ID),
	}

	i := 0
	for name, grant := range testCases {
		i++
		t.Run(name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID,
				append([]string{grant}, targetReadGrants...)...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Slug:    fmt.Sprintf("urn-portal-%d", i),
				Mapping: keyspaceMapping(t, h, workspace.ID),
				Enabled: true,
			})
			require.Equal(t, http.StatusOK, res.Status,
				"a URN grant must authorize portal creation, got: %s", res.RawBody)
		})
	}
}
