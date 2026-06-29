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
		require.False(t, body.Data[0].DeleteProtection)
	})

	t.Run("replace removes unprotected vars absent from payload", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "OLD_ONE", "x", db.AppEnvironmentVariablesTypeRecoverable, false)
		seedVar(t, h, env, "OLD_TWO", "y", db.AppEnvironmentVariablesTypeRecoverable, false)

		call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "NEW_ONE", Value: "z"},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		_, ok := raw["NEW_ONE"]
		require.True(t, ok)
	})

	t.Run("protected var omitted from payload survives", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "PROTECTED", "keep", db.AppEnvironmentVariablesTypeRecoverable, true)
		seedVar(t, h, env, "PLAIN", "drop", db.AppEnvironmentVariablesTypeRecoverable, false)

		call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "FRESH", Value: "v"},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 2)
		require.Contains(t, raw, "PROTECTED")
		require.Contains(t, raw, "FRESH")
		require.NotContains(t, raw, "PLAIN")
		require.True(t, raw["PROTECTED"].deleteProtection)
	})

	t.Run("protected var in payload is updated in place", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "PROTECTED", "old", db.AppEnvironmentVariablesTypeRecoverable, true)

		call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "PROTECTED", Value: "new", DeleteProtection: ptr(true)},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		require.Equal(t, "new", decrypt(t, env.environmentID, raw["PROTECTED"].value))
		require.True(t, raw["PROTECTED"].deleteProtection)
	})

	t.Run("omitted optional fields preserve existing values", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVarFull(t, h, env, "SECRET", "old", db.AppEnvironmentVariablesTypeWriteonly, "db password", true)

		body := call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "SECRET", Value: "rotated"},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		require.Equal(t, "rotated", decrypt(t, env.environmentID, raw["SECRET"].value))
		// sensitive, description, and protection are preserved when omitted.
		require.Equal(t, db.AppEnvironmentVariablesTypeWriteonly, raw["SECRET"].varType)
		require.Equal(t, "db password", raw["SECRET"].description)
		require.True(t, raw["SECRET"].deleteProtection)

		// Response reflects the merged result, not the raw payload.
		require.Len(t, body.Data, 1)
		require.True(t, body.Data[0].Sensitive)
		require.True(t, body.Data[0].DeleteProtection)
		require.NotNil(t, body.Data[0].Description)
		require.Equal(t, "db password", *body.Data[0].Description)
	})

	t.Run("explicit false overrides preserved protection", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "WAS_PROTECTED", "v", db.AppEnvironmentVariablesTypeRecoverable, true)

		call(t, makeRequest(env, []openapi.EnvironmentVariableInput{
			{Key: "WAS_PROTECTED", Value: "v2", DeleteProtection: ptr(false)},
		}))

		raw := listRawVars(t, h, env.environmentID)
		require.False(t, raw["WAS_PROTECTED"].deleteProtection)
	})

	t.Run("emits per-variable audit events", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "EXISTING", "old", db.AppEnvironmentVariablesTypeRecoverable, false)
		seedVar(t, h, env, "DROP", "x", db.AppEnvironmentVariablesTypeRecoverable, false)
		seedVar(t, h, env, "PROTECTED", "keep", db.AppEnvironmentVariablesTypeRecoverable, true)

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
		// PROTECTED was preserved, so it must not produce a removal event.
		require.NotContains(t, removed[0], "PROTECTED")
	})

	t.Run("empty payload clears unprotected and keeps protected", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "PROTECTED", "keep", db.AppEnvironmentVariablesTypeRecoverable, true)
		seedVar(t, h, env, "PLAIN", "drop", db.AppEnvironmentVariablesTypeRecoverable, false)

		body := call(t, makeRequest(env, []openapi.EnvironmentVariableInput{}))

		raw := listRawVars(t, h, env.environmentID)
		require.Len(t, raw, 1)
		require.Contains(t, raw, "PROTECTED")
		// Response echoes only what was set; preserved protected vars are not listed.
		require.Empty(t, body.Data)
	})
}
