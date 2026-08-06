package handler_test

import (
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	listpolicies "github.com/unkeyed/unkey/svc/api/routes/v2_gateway_list_policies"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_gateway_update_policy"
)

func TestUpdatePolicySuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)
	listRoute := &listpolicies.Handler{DB: h.DB}
	h.Register(listRoute)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "environment.*.update_policy", "environment.*.read_policies")
	headers := authHeaders(rootKey)

	call := func(t *testing.T, req handler.Request) {
		t.Helper()
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	}

	list := func(t *testing.T, env seededEnv) []openapi.PolicyResponse {
		t.Helper()
		res := testutil.CallRoute[listpolicies.Request, listpolicies.Response](h, listRoute, headers, listpolicies.Request{
			Project:     env.projectID,
			App:         env.appID,
			Environment: env.environmentID,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		return res.Body.Data
	}

	t.Run("rename keeps id, rule and siblings", func(t *testing.T) {
		env := seedEnvironment(t, h)
		ids := seedFirewallPolicies(t, h, env, 3)

		req := makeRequest(env, ids[1])
		req.Name = ptr.P("KEBAP renamed")
		call(t, req)

		policies := list(t, env)
		require.Len(t, policies, 3)
		for i, p := range policies {
			require.Equal(t, ids[i], p.Id, "order and ids must be stable")
			require.NotNil(t, p.Firewall)
			require.True(t, p.Enabled)
		}
		require.Equal(t, "KEBAP 0", policies[0].Name)
		require.Equal(t, "KEBAP renamed", policies[1].Name)
		require.Equal(t, "KEBAP 2", policies[2].Name)
	})

	t.Run("disable survives storage roundtrip", func(t *testing.T) {
		env := seedEnvironment(t, h)
		ids := seedFirewallPolicies(t, h, env, 1)

		req := makeRequest(env, ids[0])
		req.Enabled = ptr.P(false)
		call(t, req)

		policies := list(t, env)
		require.Len(t, policies, 1)
		require.False(t, policies[0].Enabled)
		require.Equal(t, "KEBAP 0", policies[0].Name, "name must be untouched")
		// The dashboard's strict schema requires the key to be present.
		require.Contains(t, readStoredBlob(t, h, env), `"enabled":false`)
	})

	t.Run("replace match expressions", func(t *testing.T) {
		env := seedEnvironment(t, h)
		ids := seedFirewallPolicies(t, h, env, 1)

		req := makeRequest(env, ids[0])
		req.Match = nullable.NewNullableWithValue([]openapi.MatchExpr{
			{Path: &openapi.PathMatch{Path: openapi.StringMatch{Prefix: ptr.P("/internal/")}}},
			{Method: &openapi.MethodMatch{Methods: []openapi.MethodMatchMethods{"GET"}}},
		})
		call(t, req)

		policies := list(t, env)
		match := ptr.SafeDeref(policies[0].Match)
		require.Len(t, match, 2)
		require.Equal(t, ptr.P("/internal/"), match[0].Path.Path.Prefix)
		require.Equal(t, []openapi.MethodMatchMethods{"GET"}, match[1].Method.Methods)
	})

	t.Run("null match clears expressions", func(t *testing.T) {
		env := seedEnvironment(t, h)
		policy := firewallPolicy("KEBAP", pathPrefixMatch("/internal/"))
		seedSentinelConfig(t, h, env, policy)

		req := makeRequest(env, policy.GetId())
		req.Match = nullable.NewNullNullable[[]openapi.MatchExpr]()
		call(t, req)

		policies := list(t, env)
		require.Len(t, policies, 1)
		require.Nil(t, policies[0].Match)
		require.NotNil(t, policies[0].Firewall, "rule must be untouched")
	})

	t.Run("switch rule variant preserves id and match", func(t *testing.T) {
		env := seedEnvironment(t, h)
		policy := &frontlinev1.Policy{
			Id:      uid.New(uid.PolicyPrefix),
			Name:    "KEBAP",
			Enabled: ptr.P(true),
			Match:   []*frontlinev1.MatchExpr{pathPrefixMatch("/api/")},
			Config: &frontlinev1.Policy_Ratelimit{
				Ratelimit: &frontlinev1.RateLimit{
					Limit:    100,
					WindowMs: 60_000,
					Identifier: &frontlinev1.RateLimitIdentifier{
						Source: &frontlinev1.RateLimitIdentifier_RemoteIp{RemoteIp: &frontlinev1.RemoteIpKey{}},
					},
				},
			},
		}
		seedSentinelConfig(t, h, env, policy)

		req := makeRequest(env, policy.GetId())
		req.Firewall = &openapi.FirewallPolicy{Action: "ACTION_DENY"}
		call(t, req)

		policies := list(t, env)
		require.Len(t, policies, 1)
		require.Equal(t, policy.GetId(), policies[0].Id)
		require.NotNil(t, policies[0].Firewall)
		require.Nil(t, policies[0].Ratelimit, "old rule must be gone")
		require.Len(t, ptr.SafeDeref(policies[0].Match), 1)
	})

	t.Run("switch rule variant to logging", func(t *testing.T) {
		env := seedEnvironment(t, h)
		policy := firewallPolicy("KEBAP")
		seedSentinelConfig(t, h, env, policy)

		req := makeRequest(env, policy.GetId())
		req.Logging = &openapi.LoggingPolicy{Headers: ptr.P(true), Bodies: ptr.P(false)}
		call(t, req)

		policies := list(t, env)
		require.Len(t, policies, 1)
		require.Equal(t, policy.GetId(), policies[0].Id)
		require.NotNil(t, policies[0].Logging)
		require.Equal(t, ptr.P(true), policies[0].Logging.Headers, "capture flags must survive storage")
		require.Equal(t, ptr.P(false), policies[0].Logging.Bodies, "capture flags must survive storage")
		require.Nil(t, policies[0].Firewall, "old rule must be gone")
	})

	t.Run("update keyauth with owned keyspaces", func(t *testing.T) {
		env := seedEnvironment(t, h)
		ids := seedFirewallPolicies(t, h, env, 1)
		api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})

		req := makeRequest(env, ids[0])
		req.Keyauth = &openapi.KeyauthPolicy{
			Keyspaces:       []string{api.KeyAuthID.String},
			PermissionQuery: ptr.P("documents.read"),
		}
		call(t, req)

		policies := list(t, env)
		require.NotNil(t, policies[0].Keyauth)
		require.Equal(t, []string{api.KeyAuthID.String}, policies[0].Keyauth.Keyspaces)
		require.Equal(t, ptr.P("documents.read"), policies[0].Keyauth.PermissionQuery)
		require.Nil(t, policies[0].Firewall)
	})

	t.Run("combined patch in one call", func(t *testing.T) {
		env := seedEnvironment(t, h)
		ids := seedFirewallPolicies(t, h, env, 2)

		req := makeRequest(env, ids[0])
		req.Name = ptr.P("KEBAP combined")
		req.Enabled = ptr.P(false)
		req.Match = nullable.NewNullableWithValue([]openapi.MatchExpr{
			{Path: &openapi.PathMatch{Path: openapi.StringMatch{Exact: ptr.P("/health")}}},
		})
		req.Openapi = &openapi.OpenapiPolicy{}
		call(t, req)

		policies := list(t, env)
		require.Len(t, policies, 2)
		updated := policies[0]
		require.Equal(t, ids[0], updated.Id)
		require.Equal(t, "KEBAP combined", updated.Name)
		require.False(t, updated.Enabled)
		require.NotNil(t, updated.Openapi)
		require.Nil(t, updated.Firewall)
		require.Len(t, ptr.SafeDeref(updated.Match), 1)

		sibling := policies[1]
		require.Equal(t, ids[1], sibling.Id)
		require.Equal(t, "KEBAP 1", sibling.Name)
		require.True(t, sibling.Enabled)
		require.NotNil(t, sibling.Firewall)
		require.Nil(t, sibling.Match)
	})
}
