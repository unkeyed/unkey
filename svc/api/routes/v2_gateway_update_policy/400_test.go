package handler_test

import (
	"net/http"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_gateway_update_policy"
)

func TestUpdatePolicyBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "environment.*.update_policy")
	headers := authHeaders(rootKey)
	env := seedEnvironment(t, h)
	ids := seedFirewallPolicies(t, h, env, 1)

	callTyped := func(t *testing.T, req handler.Request) testutil.TestResponse[openapi.BadRequestErrorResponse] {
		t.Helper()
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		return res
	}

	t.Run("no updatable fields", func(t *testing.T) {
		res := callTyped(t, makeRequest(env, ids[0]))
		require.Contains(t, res.Body.Error.Type, "invalid_input")
		require.Contains(t, res.Body.Error.Detail, "at least one field")
	})

	t.Run("more than one rule variant", func(t *testing.T) {
		req := makeRequest(env, ids[0])
		req.Firewall = &openapi.FirewallPolicy{Action: "ACTION_DENY"}
		req.Openapi = &openapi.OpenapiPolicy{}
		res := callTyped(t, req)
		require.Contains(t, res.Body.Error.Type, "invalid_input")
		require.Contains(t, res.Body.Error.Detail, "exactly one of keyauth, ratelimit, firewall or openapi; 2 are set")
	})

	t.Run("invalid regex in match", func(t *testing.T) {
		req := makeRequest(env, ids[0])
		req.Match = nullable.NewNullableWithValue([]openapi.MatchExpr{
			{Path: &openapi.PathMatch{Path: openapi.StringMatch{Regex: ptr.P("(unclosed")}}},
		})
		res := callTyped(t, req)
		require.Contains(t, res.Body.Error.Detail, "regular expression")
	})

	t.Run("match expression with no variant", func(t *testing.T) {
		req := makeRequest(env, ids[0])
		req.Match = nullable.NewNullableWithValue([]openapi.MatchExpr{{}})
		res := callTyped(t, req)
		require.Contains(t, res.Body.Error.Detail, "exactly one")
	})

	t.Run("invalid permission query", func(t *testing.T) {
		req := makeRequest(env, ids[0])
		req.Keyauth = &openapi.KeyauthPolicy{
			Keyspaces:       []string{uid.New(uid.KeySpacePrefix)},
			PermissionQuery: ptr.P("documents.read AND NOT other"),
		}
		res := callTyped(t, req)
		require.Contains(t, res.Body.Error.Detail, "permission query")
	})

	t.Run("keyauth ratelimit limit without duration", func(t *testing.T) {
		req := makeRequest(env, ids[0])
		req.Keyauth = &openapi.KeyauthPolicy{
			Keyspaces:  []string{uid.New(uid.KeySpacePrefix)},
			Ratelimits: ptr.P([]openapi.KeyRatelimit{{Name: "burst", Limit: ptr.P(int64(10))}}),
		}
		res := callTyped(t, req)
		require.Contains(t, res.Body.Error.Detail, "limit and duration together")
	})

	t.Run("missing identifiers", func(t *testing.T) {
		req := makeRequest(seededEnv{}, "")
		req.Enabled = ptr.P(false)
		callTyped(t, req)
	})

	t.Run("stored policies stay untouched after rejected update", func(t *testing.T) {
		blob := readStoredBlob(t, h, env)
		req := makeRequest(env, ids[0])
		req.Firewall = &openapi.FirewallPolicy{Action: "ACTION_UNKNOWN"}
		callTyped(t, req)
		require.Equal(t, blob, readStoredBlob(t, h, env))
	})
}
