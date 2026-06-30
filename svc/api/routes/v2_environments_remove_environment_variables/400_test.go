package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_remove_environment_variables"
)

func TestRemoveEnvironmentVariablesBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.set_environment_variables")
	headers := authHeaders(rootKey)

	t.Run("invalid key names are rejected", func(t *testing.T) {
		for _, key := range []string{"foo-bar", "1leading", "has space", "a.b"} {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, makeRequest(env, []string{key}))
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400 for key %q, received: %s", key, res.RawBody)
		}
	})

	t.Run("empty variables array is rejected", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, makeRequest(env, []string{}))
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
	})

	t.Run("more than 50 variables are rejected", func(t *testing.T) {
		vars := make([]string, 51)
		for i := range vars {
			vars[i] = fmt.Sprintf("KEY_%d", i)
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, makeRequest(env, vars))
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
	})
}
