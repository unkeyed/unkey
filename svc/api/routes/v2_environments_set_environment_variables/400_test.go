package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_set_environment_variables"
)

func TestSetEnvironmentVariablesBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.set_environment_variables")
	headers := authHeaders(rootKey)

	t.Run("invalid key names are rejected", func(t *testing.T) {
		for _, key := range []string{"foo-bar", "1leading", "has space", "a.b"} {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, makeRequest(env, []openapi.EnvironmentVariableInput{
				{Key: key, Value: "v"},
			}))
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400 for key %q, received: %s", key, res.RawBody)
		}
	})

	t.Run("more than 50 variables are rejected", func(t *testing.T) {
		vars := make([]openapi.EnvironmentVariableInput, 51)
		for i := range vars {
			vars[i] = openapi.EnvironmentVariableInput{Key: fmt.Sprintf("KEY_%d", i), Value: "v"}
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, makeRequest(env, vars))
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
	})

	t.Run("duplicate keys are rejected", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "DUP", Value: "first"},
			{Key: "DUP", Value: "second"},
		}))
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
	})

	t.Run("values over the byte cap are rejected even when under the code-point limit", func(t *testing.T) {
		// 5462 three-byte runes is 16386 UTF-8 bytes but only 5462 code points,
		// so it passes the spec maxLength (16384 code points) and must be caught
		// by the handler's server-side byte check.
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "TOO_BIG", Value: strings.Repeat("あ", 5462)},
		}))
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
	})
}
