package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_gateway_update_policy"
)

func TestUpdatePolicyForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	env := seedEnvironment(t, h)
	ids := seedFirewallPolicies(t, h, env, 1)

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "wildcard permission", permissions: []string{"environment.*.update_policy"}, shouldPass: true},
		{name: "specific permission", permissions: []string{fmt.Sprintf("environment.%s.update_policy", env.environmentID)}, shouldPass: true},
		{name: "permission and more", permissions: []string{"some.other.permission", "environment.*.update_policy"}, shouldPass: true},
		{name: "read action is not enough", permissions: []string{"environment.*.read_policies"}, shouldPass: false},
		{name: "set_policies action is not enough", permissions: []string{"environment.*.set_policies"}, shouldPass: false},
		{name: "other environment id does not match", permissions: []string{fmt.Sprintf("environment.%s.update_policy", uid.New(uid.EnvironmentPrefix))}, shouldPass: false},
		{name: "unrelated permission", permissions: []string{"api.*.read_api"}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
		{name: "urn style wildcard gateway permission", permissions: []string{"unkey:v1:" + env.workspaceID + ":projects/*/apps/*/environments/*/gateway#write_policies"}, shouldPass: true},
		{name: "urn style specific gateway permission", permissions: []string{"unkey:v1:" + env.workspaceID + ":projects/" + env.projectID + "/apps/" + env.appID + "/environments/" + env.environmentID + "/gateway#write_policies"}, shouldPass: true},
		{name: "urn style wrong action", permissions: []string{"unkey:v1:" + env.workspaceID + ":projects/*/apps/*/environments/*/gateway#read_policies"}, shouldPass: false},
		{name: "urn style other environment gateway", permissions: []string{"unkey:v1:" + env.workspaceID + ":projects/" + env.projectID + "/apps/" + env.appID + "/environments/" + uid.New(uid.EnvironmentPrefix) + "/gateway#write_policies"}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(env.workspaceID, tc.permissions...)
			headers := authHeaders(rootKey)

			req := makeRequest(env, ids[0])
			req.Name = ptr.P("KEBAP")
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			if tc.shouldPass {
				require.Equal(t, 200, res.Status, "expected 200 for %v, got: %s", tc.permissions, res.RawBody)
				return
			}
			require.Equal(t, http.StatusForbidden, res.Status, "expected 403 for %v, got: %s", tc.permissions, res.RawBody)
		})
	}
}
