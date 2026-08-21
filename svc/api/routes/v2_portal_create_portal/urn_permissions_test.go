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
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	// Grants are spelled out per case rather than topped up with a shared set.
	// Appending targetReadGrants to every case is what previously hid a real
	// denial: those are legacy tuples, and the dashboard principal this test
	// exists to represent holds none — the WorkOS translator and the local proxy
	// both emit URNs only. A case that needs a legacy tuple to pass is not
	// testing the caller it claims to.
	testCases := map[string][]string{
		// Exactly what a dashboard operator carries, and nothing else.
		"workspace-wide admin URN": {
			fmt.Sprintf("unkey:v1:%s:**#*", workspace.ID),
		},
		// A scoped operator still needs read on the target it points at, but can
		// satisfy it entirely with URNs.
		"portal-scoped URN with URN target read": {
			fmt.Sprintf("unkey:v1:%s:portals/*#create_portal", workspace.ID),
			fmt.Sprintf("unkey:v1:%s:keyspaces/*#read_keyspace", workspace.ID),
		},
		// The root-key shape, which legacy tuples must keep serving.
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
				Slug:    fmt.Sprintf("urn-portal-%d", i),
				Mapping: keyspaceMapping(t, h, workspace.ID),
				Enabled: ptr(true),
			})
			require.Equal(t, http.StatusOK, res.Status,
				"a URN grant must authorize portal creation, got: %s", res.RawBody)
		})
	}
}
