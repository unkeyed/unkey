package handler_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_set_environment_variables"
)

func TestSetEnvironmentVariablesSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs}
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

	decrypt := func(t *testing.T, environmentID, encrypted string) string {
		t.Helper()
		res, err := h.Vault.Decrypt(ctx, &vaultv1.DecryptRequest{Keyring: environmentID, Encrypted: encrypted})
		require.NoError(t, err)
		return res.GetPlaintext()
	}

	t.Run("set on empty environment encrypts and stores values", func(t *testing.T) {
		env := seedEnvironment(t, h)
		call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "DATABASE_URL", Value: "postgres://secret", Kind: ptr(openapi.Writeonly)},
			{Key: "LOG_LEVEL", Value: "debug", Kind: ptr(openapi.Recoverable), Description: ptr("verbosity")},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 2)

		require.Equal(t, db.AppEnvironmentVariablesTypeWriteonly, raw["DATABASE_URL"].varType)
		require.Equal(t, "postgres://secret", decrypt(t, env.environmentID, raw["DATABASE_URL"].value))
		require.NotEqual(t, "postgres://secret", raw["DATABASE_URL"].value, "value must be stored encrypted")

		require.Equal(t, db.AppEnvironmentVariablesTypeRecoverable, raw["LOG_LEVEL"].varType)
		require.Equal(t, "debug", decrypt(t, env.environmentID, raw["LOG_LEVEL"].value))
	})

	t.Run("kind defaults to writeonly when omitted", func(t *testing.T) {
		env := seedEnvironment(t, h)
		call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "PLAIN", Value: "v"},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Equal(t, db.AppEnvironmentVariablesTypeWriteonly, raw["PLAIN"].varType)
	})

	t.Run("upsert leaves vars absent from payload untouched", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "KEEP_ONE", "KEBAP", db.AppEnvironmentVariablesTypeRecoverable)
		seedVar(t, h, env, "KEEP_TWO", "y", db.AppEnvironmentVariablesTypeRecoverable)

		call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "KEEP_ONE", Value: "updated", Kind: ptr(openapi.Recoverable)},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 2)
		require.Equal(t, "updated", decrypt(t, env.environmentID, raw["KEEP_ONE"].value))
		_, ok := raw["KEEP_TWO"]
		require.True(t, ok, "KEEP_TWO must survive: it was not in the payload and prune is false")
	})

	t.Run("set creates a new key alongside existing vars", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "EXISTING", "x", db.AppEnvironmentVariablesTypeRecoverable)

		call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "NEW_ONE", Value: "z"},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 2)
		_, hasExisting := raw["EXISTING"]
		_, hasNew := raw["NEW_ONE"]
		require.True(t, hasExisting)
		require.True(t, hasNew)
	})

	t.Run("existing var in payload is overwritten", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "API_KEY", "old", db.AppEnvironmentVariablesTypeRecoverable)

		call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "API_KEY", Value: "new"},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		require.Equal(t, "new", decrypt(t, env.environmentID, raw["API_KEY"].value))
	})

	t.Run("overwrite is a hard overwrite: omitted optional fields fall back to defaults", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVarFull(t, h, env, "SECRET", "old", db.AppEnvironmentVariablesTypeRecoverable, "db password")

		call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "SECRET", Value: "rotated"},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		require.Equal(t, "rotated", decrypt(t, env.environmentID, raw["SECRET"].value))
		// Nothing merged: kind defaults to writeonly and description is cleared.
		require.Equal(t, db.AppEnvironmentVariablesTypeWriteonly, raw["SECRET"].varType)
		require.Empty(t, raw["SECRET"].description)
	})

	t.Run("prune removes vars absent from payload", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "OLD_ONE", "x", db.AppEnvironmentVariablesTypeRecoverable)
		seedVar(t, h, env, "OLD_TWO", "y", db.AppEnvironmentVariablesTypeRecoverable)

		req := makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "NEW_ONE", Value: "z"},
		})
		req.Prune = ptr(true)
		call(t, req)

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		_, ok := raw["NEW_ONE"]
		require.True(t, ok)
	})

	t.Run("prune replaces the set and overwrites a surviving key", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "KEEP", "old", db.AppEnvironmentVariablesTypeRecoverable)
		seedVar(t, h, env, "GONE", "x", db.AppEnvironmentVariablesTypeRecoverable)

		req := makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "KEEP", Value: "new", Kind: ptr(openapi.Recoverable)},
		})
		req.Prune = ptr(true)
		call(t, req)

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		require.Equal(t, "new", decrypt(t, env.environmentID, raw["KEEP"].value))
		_, gone := raw["GONE"]
		require.False(t, gone)

		logs := h.FindAuditLogsByTargetID(ctx, t, env.environmentID)
		require.Len(t, logs, 1)
		require.Equal(t, "KEEP", fmt.Sprintf("%v", logs[0].Targets[0].Meta["key"]))
		require.Equal(t, "true", fmt.Sprintf("%v", logs[0].Targets[0].Meta["prune"]))
	})

	t.Run("prune with empty payload clears all vars", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "ONE", "a", db.AppEnvironmentVariablesTypeRecoverable)
		seedVar(t, h, env, "TWO", "b", db.AppEnvironmentVariablesTypeRecoverable)

		req := makeRequest(env, []openapi.EnvironmentVariableInput{})
		req.Prune = ptr(true)
		call(t, req)

		raw := listRawVars(t, h, env.environmentID)
		require.Empty(t, raw)

		// The wipe has no keys to log per-var, so it emits one summary event.
		logs := h.FindAuditLogsByTargetID(ctx, t, env.environmentID)
		require.Len(t, logs, 1)
		require.Contains(t, logs[0].Description, "Pruned all environment variables")
		require.Equal(t, "true", fmt.Sprintf("%v", logs[0].Targets[0].Meta["prune"]))
	})

	t.Run("empty payload without prune is a no-op", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "ONE", "a", db.AppEnvironmentVariablesTypeRecoverable)
		seedVar(t, h, env, "TWO", "b", db.AppEnvironmentVariablesTypeRecoverable)

		call(t, makeRequest(env, []openapi.EnvironmentVariableInput{}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 2, "an empty payload without prune must not touch existing vars")
	})

	t.Run("emits one correlated audit event per applied key", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "EXISTING", "old", db.AppEnvironmentVariablesTypeRecoverable)
		seedVar(t, h, env, "IDLE", "z", db.AppEnvironmentVariablesTypeRecoverable)

		call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "EXISTING", Value: "new"},
			{Key: "NEW", Value: "v"},
		}))

		// Exactly one log per applied key: IDLE was untouched and gets none.
		logs := h.FindAuditLogsByTargetID(ctx, t, env.environmentID)
		require.Len(t, logs, 2)

		seen := make(map[string]bool)
		for _, l := range logs {
			require.Contains(t, l.Description, "Set environment variable")
			require.Len(t, l.Targets, 1)
			seen[fmt.Sprintf("%v", l.Targets[0].Meta["key"])] = true
			require.Equal(t, "false", fmt.Sprintf("%v", l.Targets[0].Meta["prune"]))
		}
		require.True(t, seen["EXISTING"])
		require.True(t, seen["NEW"])

		// Separate entries are auto-correlated so one set call is traceable as a unit.
		require.NotEmpty(t, logs[0].CorrelationID)
		require.Equal(t, logs[0].CorrelationID, logs[1].CorrelationID)
	})
}
