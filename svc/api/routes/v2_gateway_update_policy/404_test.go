package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_gateway_update_policy"
)

func TestUpdatePolicyNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	env := seedEnvironment(t, h)
	ids := seedFirewallPolicies(t, h, env, 1)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.update_policy")
	headers := authHeaders(rootKey)

	call := func(t *testing.T, req handler.Request) testutil.TestResponse[openapi.NotFoundErrorResponse] {
		t.Helper()
		if req.Name == nil {
			req.Name = ptr.P("KEBAP")
		}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		return res
	}

	t.Run("nonexistent environment", func(t *testing.T) {
		req := makeRequest(env, ids[0])
		req.Environment = uid.New(uid.EnvironmentPrefix)
		call(t, req)
	})

	t.Run("nonexistent project", func(t *testing.T) {
		req := makeRequest(env, ids[0])
		req.Project = uid.New(uid.ProjectPrefix)
		call(t, req)
	})

	t.Run("nonexistent app", func(t *testing.T) {
		req := makeRequest(env, ids[0])
		req.App = uid.New(uid.AppPrefix)
		call(t, req)
	})

	t.Run("unknown policy id", func(t *testing.T) {
		res := call(t, makeRequest(env, uid.New(uid.PolicyPrefix)))
		require.Contains(t, res.Body.Error.Type, "policy_not_found")
		require.Contains(t, res.Body.Error.Detail, "policy ids change")
	})

	t.Run("policy id from another environment", func(t *testing.T) {
		other := seedEnvironment(t, h)
		foreign := firewallPolicy("KEBAP")
		seedSentinelConfig(t, h, other, foreign)
		call(t, makeRequest(env, foreign.GetId()))
	})

	t.Run("missing runtime settings row", func(t *testing.T) {
		bare := seedEnvironment(t, h)
		_, err := h.DB.RW().ExecContext(context.Background(),
			"DELETE FROM app_runtime_settings WHERE app_id = ? AND environment_id = ?",
			bare.appID, bare.environmentID)
		require.NoError(t, err)

		res := call(t, makeRequest(bare, uid.New(uid.PolicyPrefix)))
		require.Contains(t, res.Body.Error.Type, "policy_not_found")
	})

	t.Run("unowned keyspace", func(t *testing.T) {
		req := makeRequest(env, ids[0])
		req.Keyauth = &openapi.KeyauthPolicy{Keyspaces: []string{"ks_kebap_unowned"}}
		res := call(t, req)
		require.Contains(t, res.Body.Error.Type, "key_space_not_found")
	})

	t.Run("another workspace's environment", func(t *testing.T) {
		other := h.CreateWorkspace()
		foreignKey := h.CreateRootKey(other.ID, "environment.*.update_policy")
		req := makeRequest(env, ids[0])
		req.Name = ptr.P("KEBAP")
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(foreignKey), req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	})
}
