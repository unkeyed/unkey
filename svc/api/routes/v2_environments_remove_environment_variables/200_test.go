package handler_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_remove_environment_variables"
)

func TestRemoveEnvironmentVariablesSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "environment.*.set_environment_variables")
	headers := authHeaders(rootKey)

	call := func(t *testing.T, req handler.Request) handler.Response {
		t.Helper()
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		return *res.Body
	}

	keysOf := func(data []openapi.EnvironmentVariableMetadata) []string {
		out := make([]string, 0, len(data))
		for _, d := range data {
			out = append(out, d.Key)
		}
		sort.Strings(out)
		return out
	}

	t.Run("remove existing keys deletes them and returns the remaining set", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "GONE", "v", db.AppEnvironmentVariablesTypeRecoverable, false)
		seedVar(t, h, env, "ALSO_GONE", "v", db.AppEnvironmentVariablesTypeRecoverable, false)
		seedVar(t, h, env, "KEEP", "v", db.AppEnvironmentVariablesTypeRecoverable, false)

		body := call(t, makeRequest(env, []string{"GONE", "ALSO_GONE"}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		_, ok := raw["KEEP"]
		require.True(t, ok)

		require.Equal(t, []string{"KEEP"}, keysOf(body.Data))
	})

	t.Run("removing a key that is not present is a noop", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "KEEP", "v", db.AppEnvironmentVariablesTypeRecoverable, false)

		body := call(t, makeRequest(env, []string{"MISSING"}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		require.Equal(t, []string{"KEEP"}, keysOf(body.Data))
	})

	t.Run("delete-protected keys are never removed", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "PROTECTED", "v", db.AppEnvironmentVariablesTypeRecoverable, true)

		body := call(t, makeRequest(env, []string{"PROTECTED"}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		require.True(t, raw["PROTECTED"].deleteProtection)

		require.Len(t, body.Data, 1)
		require.Equal(t, "PROTECTED", body.Data[0].Key)
		require.True(t, body.Data[0].DeleteProtection)
	})

	t.Run("mixed present, missing, and protected removes only unprotected present keys", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "GONE", "v", db.AppEnvironmentVariablesTypeRecoverable, false)
		seedVar(t, h, env, "PROTECTED", "v", db.AppEnvironmentVariablesTypeRecoverable, true)

		body := call(t, makeRequest(env, []string{"GONE", "PROTECTED", "MISSING"}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		_, ok := raw["PROTECTED"]
		require.True(t, ok)

		require.Equal(t, []string{"PROTECTED"}, keysOf(body.Data))
	})

	t.Run("metadata is preserved on remaining keys", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVarFull(t, h, env, "GONE", "v", db.AppEnvironmentVariablesTypeRecoverable, "", false)
		seedVarFull(t, h, env, "KEEP", "v", db.AppEnvironmentVariablesTypeWriteonly, "secret token", false)

		body := call(t, makeRequest(env, []string{"GONE"}))

		require.Len(t, body.Data, 1)
		require.Equal(t, "KEEP", body.Data[0].Key)
		require.True(t, body.Data[0].Sensitive)
		require.NotNil(t, body.Data[0].Description)
		require.Equal(t, "secret token", *body.Data[0].Description)
	})

	t.Run("duplicate keys in payload collapse to a single removal", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "DUP", "v", db.AppEnvironmentVariablesTypeRecoverable, false)

		body := call(t, makeRequest(env, []string{"DUP", "DUP"}))

		raw := listRawVars(t, h, env.environmentID)
		require.Empty(t, raw)
		require.Empty(t, body.Data)
	})

	t.Run("removing keys that are all missing or protected leaves state unchanged", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "PROTECTED", "v", db.AppEnvironmentVariablesTypeRecoverable, true)

		body := call(t, makeRequest(env, []string{"PROTECTED", "MISSING"}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		require.Equal(t, []string{"PROTECTED"}, keysOf(body.Data))
	})
}
