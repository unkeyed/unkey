package handler_test

import (
	"context"
	"strings"
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
	})

	t.Run("duplicate keys dedup with last occurrence winning", func(t *testing.T) {
		env := seedEnvironment(t, h)
		body := call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "DUP", Value: "first", Sensitive: ptr(false)},
			{Key: "DUP", Value: "last", Sensitive: ptr(true)},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		require.Equal(t, "last", decrypt(t, env.environmentID, raw["DUP"].value))
		require.Equal(t, db.AppEnvironmentVariablesTypeWriteonly, raw["DUP"].varType)

		require.Len(t, body.Data, 1)
		require.Equal(t, "DUP", body.Data[0].Key)
		require.True(t, body.Data[0].Sensitive)
	})

	t.Run("response returns metadata without values", func(t *testing.T) {
		env := seedEnvironment(t, h)
		body := call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "API_KEY", Value: "shh", Sensitive: ptr(true)},
		}))

		require.Len(t, body.Data, 1)
		require.Equal(t, "API_KEY", body.Data[0].Key)
		require.True(t, body.Data[0].Sensitive)
	})

	t.Run("replace removes vars absent from payload", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "OLD_ONE", "x", db.AppEnvironmentVariablesTypeRecoverable)
		seedVar(t, h, env, "OLD_TWO", "y", db.AppEnvironmentVariablesTypeRecoverable)

		call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "NEW_ONE", Value: "z"},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		_, ok := raw["NEW_ONE"]
		require.True(t, ok)
	})

	t.Run("existing var in payload is updated in place", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "API_KEY", "old", db.AppEnvironmentVariablesTypeRecoverable)

		call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "API_KEY", Value: "new"},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		require.Equal(t, "new", decrypt(t, env.environmentID, raw["API_KEY"].value))
	})

	t.Run("omitted optional fields preserve existing values", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVarFull(t, h, env, "SECRET", "old", db.AppEnvironmentVariablesTypeWriteonly, "db password")

		body := call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "SECRET", Value: "rotated"},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		require.Equal(t, "rotated", decrypt(t, env.environmentID, raw["SECRET"].value))
		// sensitive and description are preserved when omitted.
		require.Equal(t, db.AppEnvironmentVariablesTypeWriteonly, raw["SECRET"].varType)
		require.Equal(t, "db password", raw["SECRET"].description)

		// Response reflects the merged result, not the raw payload.
		require.Len(t, body.Data, 1)
		require.True(t, body.Data[0].Sensitive)
		require.NotNil(t, body.Data[0].Description)
		require.Equal(t, "db password", *body.Data[0].Description)
	})

	t.Run("emits per-variable audit events", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "EXISTING", "old", db.AppEnvironmentVariablesTypeRecoverable)
		seedVar(t, h, env, "DROP", "x", db.AppEnvironmentVariablesTypeRecoverable)

		call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "EXISTING", Value: "new"},
			{Key: "NEW", Value: "v"},
		}))

		logs := h.FindAuditLogsByTargetID(ctx, t, env.environmentID)
		var created, updated, removed []string
		for _, ev := range logs {
			switch {
			case strings.HasPrefix(ev.Description, "Created environment variable"):
				created = append(created, ev.Description)
			case strings.HasPrefix(ev.Description, "Updated environment variable"):
				updated = append(updated, ev.Description)
			case strings.HasPrefix(ev.Description, "Removed environment variable"):
				removed = append(removed, ev.Description)
			}
		}

		require.Len(t, created, 1)
		require.Contains(t, created[0], "NEW")
		require.Len(t, updated, 1)
		require.Contains(t, updated[0], "EXISTING")
		require.Len(t, removed, 1)
		require.Contains(t, removed[0], "DROP")
	})

	t.Run("empty payload clears all vars", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "ONE", "a", db.AppEnvironmentVariablesTypeRecoverable)
		seedVar(t, h, env, "TWO", "b", db.AppEnvironmentVariablesTypeRecoverable)

		body := call(t, makeRequest(env, []openapi.EnvironmentVariableInput{}))

		raw := listRawVars(t, h, env.environmentID)
		require.Empty(t, raw)
		require.Empty(t, body.Data)
	})
}
