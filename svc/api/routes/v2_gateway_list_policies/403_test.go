package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_gateway_list_policies"
)

func TestListPoliciesForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	env := seedEnvironment(t, h)

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "canonical urn grant", permissions: []string{fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s/environments/%s/gateway/policies/*#read", env.workspaceID, env.projectID, env.appID, env.environmentID)}, shouldPass: true},
		{name: "permission and more", permissions: []string{"some.other.permission", fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s/environments/%s/gateway/policies/*#read", env.workspaceID, env.projectID, env.appID, env.environmentID)}, shouldPass: true},
		{name: "urn missing the project and app segments", permissions: []string{fmt.Sprintf("unkey:v1:%s:environments/*#read", env.workspaceID)}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(env.workspaceID, tc.permissions...)
			headers := authHeaders(rootKey)

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, makeRequest(env))
			if tc.shouldPass {
				require.Equal(t, 200, res.Status, "expected 200 for %v, got: %s", tc.permissions, res.RawBody)
				return
			}
			require.Equal(t, http.StatusForbidden, res.Status, "expected 403 for %v, got: %s", tc.permissions, res.RawBody)
		})
	}
}
