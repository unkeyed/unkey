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
	rootKey := h.CreateRootKey(workspace.ID, "environment.*.remove_environment_variables")
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

	t.Run("emits one audit event per removed key, grouped by correlation id", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "ALPHA", "v", db.AppEnvironmentVariablesTypeRecoverable, false)
		seedVar(t, h, env, "BETA", "v", db.AppEnvironmentVariablesTypeRecoverable, false)

		// Duplicates and a non-existent key (KEBAP) in the request: only the keys
		// that were actually present and removed get an audit log.
		call(t, makeRequest(env, []string{"BETA", "ALPHA", "BETA", "ALPHA", "KEBAP"}))

		logs := h.FindAuditLogsByTargetID(ctx, t, env.environmentID)
		require.Len(t, logs, 2)

		keys := make([]string, 0, len(logs))
		for _, l := range logs {
			require.Contains(t, l.Description, "Removed environment variable")
			require.Len(t, l.Targets, 1)
			require.NotEmpty(t, l.CorrelationID)
			require.Equal(t, logs[0].CorrelationID, l.CorrelationID)
			keys = append(keys, fmt.Sprintf("%v", l.Targets[0].Meta["key"]))
		}
		require.Contains(t, keys, "ALPHA")
		require.Contains(t, keys, "BETA")
		require.NotContains(t, keys, "KEBAP")
	})
}
