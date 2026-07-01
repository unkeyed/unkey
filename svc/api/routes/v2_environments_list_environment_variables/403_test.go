package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_list_environment_variables"
)

func TestListEnvironmentVariablesForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Vault: h.Vault}
	h.Register(route)

	env := seedEnvironment(t, h)

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "wildcard permission", permissions: []string{"environment.*.read_environment_variables"}, shouldPass: true},
		{name: "specific permission", permissions: []string{fmt.Sprintf("environment.%s.read_environment_variables", env.environmentID)}, shouldPass: true},
		{name: "permission and more", permissions: []string{"some.other.permission", "environment.*.read_environment_variables"}, shouldPass: true},
		{name: "set action does not grant read", permissions: []string{"environment.*.set_environment_variables"}, shouldPass: false},
		{name: "read_environment action is not enough", permissions: []string{"environment.*.read_environment"}, shouldPass: false},
		{name: "other environment id does not match", permissions: []string{fmt.Sprintf("environment.%s.read_environment_variables", uid.New(uid.EnvironmentPrefix))}, shouldPass: false},
		{name: "unrelated permission", permissions: []string{"api.*.read_api"}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(env.workspaceID, tc.permissions...)
			headers := authHeaders(rootKey)

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, makeRequest(env, nil, nil))
			if tc.shouldPass {
				require.Equal(t, 200, res.Status, "expected 200 for %v, got: %s", tc.permissions, res.RawBody)
				return
			}
			require.Equal(t, http.StatusForbidden, res.Status, "expected 403 for %v, got: %s", tc.permissions, res.RawBody)
		})
	}
}
