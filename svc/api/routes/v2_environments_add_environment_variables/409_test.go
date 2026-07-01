package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_add_environment_variables"
)

func TestAddEnvironmentVariablesConflict(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.set_environment_variables")
	headers := authHeaders(rootKey)

	t.Run("adding an existing key is rejected and changes nothing", func(t *testing.T) {
		seedVar(t, h, env, "EXISTING", "original", db.AppEnvironmentVariablesTypeRecoverable)

		res := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](h, route, headers, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "EXISTING", Value: "should-not-apply"},
			{Key: "NEW", Value: "should-not-apply"},
		}))
		require.Equal(t, http.StatusConflict, res.Status, "expected 409, received: %s", res.RawBody)

		// Atomic: the conflicting request must not create the new key either.
		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		require.Equal(t, "original", raw["EXISTING"].value)
		_, created := raw["NEW"]
		require.False(t, created, "no variable may be created when the request conflicts")
	})
}
