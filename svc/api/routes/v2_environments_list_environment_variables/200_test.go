package handler_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_list_environment_variables"
)

func TestListEnvironmentVariablesSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Vault: h.Vault}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "environment.*.read_environment_variables")
	headers := authHeaders(rootKey)

	call := func(t *testing.T, req handler.Request) handler.Response {
		t.Helper()
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		return *res.Body
	}

	byKey := func(data []openapi.EnvironmentVariable) map[string]openapi.EnvironmentVariable {
		out := make(map[string]openapi.EnvironmentVariable, len(data))
		for _, v := range data {
			out[v.Key] = v
		}
		return out
	}

	t.Run("recoverable returns decrypted value", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "DATABASE_URL", "postgres://kebap", db.AppEnvironmentVariablesTypeRecoverable, "primary db")

		res := call(t, makeRequest(env, nil, nil))
		require.Len(t, res.Data, 1)
		require.Equal(t, openapi.Recoverable, res.Data[0].Kind)
		require.Equal(t, "postgres://kebap", res.Data[0].Value)
		require.Equal(t, "primary db", res.Data[0].Description)
	})

	t.Run("writeonly omits value", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "SECRET_TOKEN", "KEBAP", db.AppEnvironmentVariablesTypeWriteonly, "")

		res := call(t, makeRequest(env, nil, nil))
		require.Len(t, res.Data, 1)
		require.Equal(t, openapi.Writeonly, res.Data[0].Kind)
		require.Empty(t, res.Data[0].Value)
	})

	t.Run("mixed types in one call", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "LOG_LEVEL", "debug", db.AppEnvironmentVariablesTypeRecoverable, "")
		seedVar(t, h, env, "API_KEY", "KEBAP", db.AppEnvironmentVariablesTypeWriteonly, "")

		res := call(t, makeRequest(env, nil, nil))
		require.Len(t, res.Data, 2)
		m := byKey(res.Data)
		require.Equal(t, openapi.Recoverable, m["LOG_LEVEL"].Kind)
		require.Equal(t, "debug", m["LOG_LEVEL"].Value)
		require.Equal(t, openapi.Writeonly, m["API_KEY"].Kind)
		require.Empty(t, m["API_KEY"].Value)
	})

	t.Run("empty environment returns empty list", func(t *testing.T) {
		env := seedEnvironment(t, h)

		res := call(t, makeRequest(env, nil, nil))
		require.Empty(t, res.Data)
		require.NotNil(t, res.Pagination)
		require.False(t, res.Pagination.HasMore)
		require.Nil(t, res.Pagination.Cursor)
	})

	t.Run("pagination walks every variable", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedVar(t, h, env, "ALPHA", "1", db.AppEnvironmentVariablesTypeRecoverable, "")
		seedVar(t, h, env, "BETA", "2", db.AppEnvironmentVariablesTypeRecoverable, "")
		seedVar(t, h, env, "KEBAP", "3", db.AppEnvironmentVariablesTypeRecoverable, "")
		seedVar(t, h, env, "XYZ", "4", db.AppEnvironmentVariablesTypeWriteonly, "")

		first := call(t, makeRequest(env, ptr(2), nil))
		require.Len(t, first.Data, 2)
		require.True(t, first.Pagination.HasMore)
		require.NotNil(t, first.Pagination.Cursor)

		second := call(t, makeRequest(env, ptr(2), first.Pagination.Cursor))
		require.Len(t, second.Data, 2)
		require.False(t, second.Pagination.HasMore)

		seen := map[string]bool{}
		for _, v := range append(first.Data, second.Data...) {
			seen[v.Key] = true
		}
		require.Len(t, seen, 4)
	})
}
