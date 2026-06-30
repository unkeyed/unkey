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

	t.Run("add to empty environment creates all keys", func(t *testing.T) {
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
		require.Equal(t, "verbosity", raw["LOG_LEVEL"].description)
	})

	t.Run("adds new keys alongside unrelated existing ones", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVarFull(t, h, env, "EXISTING", "original", db.AppEnvironmentVariablesTypeWriteonly, "keep me")

		call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "NEW", Value: "fresh"},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 2)

		// EXISTING is fully unchanged: value, type, description.
		require.Equal(t, "original", raw["EXISTING"].value, "existing value must not be overwritten")
		require.Equal(t, db.AppEnvironmentVariablesTypeWriteonly, raw["EXISTING"].varType)
		require.Equal(t, "keep me", raw["EXISTING"].description)

		require.Equal(t, "fresh", decrypt(t, env.environmentID, raw["NEW"].value))
	})

	t.Run("kind defaults to writeonly when omitted", func(t *testing.T) {
		env := seedEnvironment(t, h)
		call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "PLAIN", Value: "v"},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Equal(t, db.AppEnvironmentVariablesTypeWriteonly, raw["PLAIN"].varType)
		require.Equal(t, "", raw["PLAIN"].description)
	})

	t.Run("emits a single audit event with the created key set", func(t *testing.T) {
		env := seedEnvironment(t, h)

		call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "ALPHA", Value: "a"},
			{Key: "BETA", Value: "b"},
		}))

		logs := h.FindAuditLogsByTargetID(ctx, t, env.environmentID)
		require.Len(t, logs, 1)
		require.Contains(t, logs[0].Description, "Added environment variables")

		require.Len(t, logs[0].Targets, 1)
		keys := fmt.Sprintf("%v", logs[0].Targets[0].Meta["keys"])
		require.Contains(t, keys, "ALPHA")
		require.Contains(t, keys, "BETA")
	})
}
