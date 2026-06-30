package handler_test

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_add_environment_variables"
)

func TestAddEnvironmentVariablesSuccessfully(t *testing.T) {
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

	keysOf := func(data []openapi.EnvironmentVariableMetadata) []string {
		out := make([]string, 0, len(data))
		for _, d := range data {
			out = append(out, d.Key)
		}
		sort.Strings(out)
		return out
	}

	t.Run("add to empty environment creates all keys", func(t *testing.T) {
		env := seedEnvironment(t, h)
		body := call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "DATABASE_URL", Value: "postgres://secret", Sensitive: ptr(true)},
			{Key: "LOG_LEVEL", Value: "debug", Description: ptr("verbosity")},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 2)

		require.Equal(t, db.AppEnvironmentVariablesTypeWriteonly, raw["DATABASE_URL"].varType)
		require.Equal(t, "postgres://secret", decrypt(t, env.environmentID, raw["DATABASE_URL"].value))
		require.NotEqual(t, "postgres://secret", raw["DATABASE_URL"].value, "value must be stored encrypted")

		require.Equal(t, db.AppEnvironmentVariablesTypeRecoverable, raw["LOG_LEVEL"].varType)
		require.Equal(t, "debug", decrypt(t, env.environmentID, raw["LOG_LEVEL"].value))
		require.Equal(t, "verbosity", raw["LOG_LEVEL"].description)

		require.ElementsMatch(t, []string{"DATABASE_URL", "LOG_LEVEL"}, keysOf(body.Data))
	})

	t.Run("existing keys are left untouched and new keys created", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVarFull(t, h, env, "EXISTING", "original", db.AppEnvironmentVariablesTypeWriteonly, "keep me", true)

		body := call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "EXISTING", Value: "should-be-ignored", Sensitive: ptr(false), Description: ptr("changed"), DeleteProtection: ptr(false)},
			{Key: "NEW", Value: "fresh"},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 2)

		// EXISTING is fully unchanged: value, type, description, delete_protection.
		require.Equal(t, "original", raw["EXISTING"].value, "existing value must not be overwritten")
		require.Equal(t, db.AppEnvironmentVariablesTypeWriteonly, raw["EXISTING"].varType)
		require.Equal(t, "keep me", raw["EXISTING"].description)
		require.True(t, raw["EXISTING"].deleteProtection)

		require.Equal(t, "fresh", decrypt(t, env.environmentID, raw["NEW"].value))

		// Response echoes the full resulting set.
		require.ElementsMatch(t, []string{"EXISTING", "NEW"}, keysOf(body.Data))
	})

	t.Run("adding only existing keys is a noop", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "ONLY", "original", db.AppEnvironmentVariablesTypeRecoverable, false)

		body := call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "ONLY", Value: "ignored"},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		require.Equal(t, "original", raw["ONLY"].value)
		require.Equal(t, []string{"ONLY"}, keysOf(body.Data))
	})

	t.Run("new key field defaults when omitted", func(t *testing.T) {
		env := seedEnvironment(t, h)
		body := call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "PLAIN", Value: "v"},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Equal(t, db.AppEnvironmentVariablesTypeRecoverable, raw["PLAIN"].varType)
		require.Equal(t, "", raw["PLAIN"].description)
		require.False(t, raw["PLAIN"].deleteProtection)

		require.Len(t, body.Data, 1)
		require.Equal(t, "PLAIN", body.Data[0].Key)
		require.False(t, body.Data[0].Sensitive)
		require.False(t, body.Data[0].DeleteProtection)
	})

	t.Run("duplicate keys dedup with last occurrence winning", func(t *testing.T) {
		env := seedEnvironment(t, h)
		call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "DUP", Value: "first", Sensitive: ptr(false)},
			{Key: "DUP", Value: "last", Sensitive: ptr(true)},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		require.Equal(t, "last", decrypt(t, env.environmentID, raw["DUP"].value))
		require.Equal(t, db.AppEnvironmentVariablesTypeWriteonly, raw["DUP"].varType)
	})
}
