package handler_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_remove_environment_variables"
)

func TestRemoveEnvironmentVariablesSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	ctx := context.Background()
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

	t.Run("remove existing keys deletes them", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "GONE", "v", db.AppEnvironmentVariablesTypeRecoverable, false)
		seedVar(t, h, env, "ALSO_GONE", "v", db.AppEnvironmentVariablesTypeRecoverable, false)
		seedVar(t, h, env, "KEEP", "v", db.AppEnvironmentVariablesTypeRecoverable, false)

		call(t, makeRequest(env, []string{"GONE", "ALSO_GONE"}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		_, ok := raw["KEEP"]
		require.True(t, ok)
	})

	t.Run("removing a key that is not present is a noop", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "KEEP", "v", db.AppEnvironmentVariablesTypeRecoverable, false)

		call(t, makeRequest(env, []string{"MISSING"}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		_, ok := raw["KEEP"]
		require.True(t, ok)
	})

	t.Run("delete protection no longer blocks removal", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "PROTECTED", "v", db.AppEnvironmentVariablesTypeRecoverable, true)

		call(t, makeRequest(env, []string{"PROTECTED"}))

		raw := listRawVars(t, h, env.environmentID)
		require.Empty(t, raw)
	})

	t.Run("duplicate keys in payload collapse to a single removal", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "DUP", "v", db.AppEnvironmentVariablesTypeRecoverable, false)

		call(t, makeRequest(env, []string{"DUP", "DUP"}))

		raw := listRawVars(t, h, env.environmentID)
		require.Empty(t, raw)
	})

	t.Run("emits a single audit event with only the keys that existed", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "ALPHA", "v", db.AppEnvironmentVariablesTypeRecoverable, false)
		seedVar(t, h, env, "BETA", "v", db.AppEnvironmentVariablesTypeRecoverable, false)

		// Duplicates and a non-existent key (KEBAP) in the request: the audit log
		// records only the keys that were actually present and removed.
		call(t, makeRequest(env, []string{"BETA", "ALPHA", "BETA", "ALPHA", "KEBAP"}))

		logs := h.FindAuditLogsByTargetID(ctx, t, env.environmentID)
		require.Len(t, logs, 1)
		require.Contains(t, logs[0].Description, "Removed environment variables")

		require.Len(t, logs[0].Targets, 1)
		keys := fmt.Sprintf("%v", logs[0].Targets[0].Meta["keys"])
		require.Contains(t, keys, "ALPHA")
		require.Contains(t, keys, "BETA")
		require.NotContains(t, keys, "KEBAP")
	})
}
