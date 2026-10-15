package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
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
