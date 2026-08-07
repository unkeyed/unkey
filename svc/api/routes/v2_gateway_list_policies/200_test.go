package handler_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_gateway_list_policies"
	setpolicies "github.com/unkeyed/unkey/svc/api/routes/v2_gateway_set_policies"
)

func TestListPoliciesSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "environment.*.read_policies")
	headers := authHeaders(rootKey)

	call := func(t *testing.T, req handler.Request) testutil.TestResponse[handler.Response] {
		t.Helper()
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		return res
	}

	t.Run("legacy empty blob returns empty list", func(t *testing.T) {
		// The environment seeder creates the runtime settings row with "{}".
		env := seedEnvironment(t, h)
		res := call(t, makeRequest(env))
		require.Empty(t, res.Body.Data)
		// data must serialize as [], not null.
		require.Contains(t, string(res.RawBody), `"data":[]`)
	})

	t.Run("explicit empty policies blob returns empty list", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedSentinelConfig(t, h, env, `{"policies":[]}`)
		res := call(t, makeRequest(env))
		require.Empty(t, res.Body.Data)
	})

	t.Run("missing runtime settings row returns empty list", func(t *testing.T) {
		env := seedEnvironment(t, h)
		_, err := h.DB.RW().ExecContext(context.Background(),
			"DELETE FROM app_runtime_settings WHERE app_id = ? AND environment_id = ?",
			env.appID, env.environmentID)
		require.NoError(t, err)

		res := call(t, makeRequest(env))
		require.Empty(t, res.Body.Data)
	})

	t.Run("all variants round out of storage with full fidelity", func(t *testing.T) {
		env := seedEnvironment(t, h)
		// protojson wire shape, including int64-as-string, exactly as the
		// write path stores it.
		seedSentinelConfig(t, h, env, `{"policies":[
			{"id":"pol_keyauth","name":"keyauth KEBAP","enabled":true,
				"match":[
					{"path":{"path":{"prefix":"/internal/","ignoreCase":true}}},
					{"method":{"methods":["GET","POST"]}},
					{"header":{"name":"X-Debug","present":true}},
					{"queryParam":{"name":"v","value":{"exact":"1"}}}
				],
				"keyauth":{
					"keySpaceIds":["ks_kebap"],
					"locations":[{"bearer":{}},{"header":{"name":"X-Api-Key","stripPrefix":"Key "}},{"queryParam":{"name":"api_key"}}],
					"permissionQuery":"documents.read",
					"ratelimits":[{"name":"tokens"},{"name":"burst","limit":"100","duration":"60000","cost":"2"}]
				}},
			{"id":"pol_ratelimit","name":"ratelimit","enabled":true,
				"ratelimit":{"limit":"100","windowMs":"60000","identifier":{"principalField":{"path":"sub"}}}},
			{"id":"pol_firewall","name":"firewall","enabled":false,
				"firewall":{"action":"ACTION_DENY"}},
			{"id":"pol_openapi","name":"openapi",
				"openapi":{}}
		]}`)

		res := call(t, makeRequest(env))
		require.Len(t, res.Body.Data, 4)

		keyauth := res.Body.Data[0]
		require.Equal(t, "pol_keyauth", keyauth.Id)
		require.Equal(t, "keyauth KEBAP", keyauth.Name)
		require.True(t, keyauth.Enabled)
		require.NotNil(t, keyauth.Keyauth)
		require.Nil(t, keyauth.Ratelimit)
		require.Nil(t, keyauth.Firewall)
		require.Nil(t, keyauth.Openapi)
		require.Equal(t, []string{"ks_kebap"}, keyauth.Keyauth.Keyspaces)
		require.Equal(t, ptr.P("documents.read"), keyauth.Keyauth.PermissionQuery)

		locations := ptr.SafeDeref(keyauth.Keyauth.Locations)
		require.Len(t, locations, 3)
		require.NotNil(t, locations[0].Bearer)
		require.NotNil(t, locations[1].Header)
		require.Equal(t, "X-Api-Key", locations[1].Header.Name)
		require.Equal(t, ptr.P("Key "), locations[1].Header.StripPrefix)
		require.NotNil(t, locations[2].QueryParam)
		require.Equal(t, "api_key", locations[2].QueryParam.Name)

		ratelimits := ptr.SafeDeref(keyauth.Keyauth.Ratelimits)
		require.Len(t, ratelimits, 2)
		require.Equal(t, "tokens", ratelimits[0].Name)
		require.Nil(t, ratelimits[0].Limit)
		require.Nil(t, ratelimits[0].Duration)
		require.Nil(t, ratelimits[0].Cost)
		require.Equal(t, "burst", ratelimits[1].Name)
		require.Equal(t, ptr.P(int64(100)), ratelimits[1].Limit)
		require.Equal(t, ptr.P(int64(60000)), ratelimits[1].Duration)
		require.Equal(t, ptr.P(int64(2)), ratelimits[1].Cost)

		match := ptr.SafeDeref(keyauth.Match)
		require.Len(t, match, 4)
		require.NotNil(t, match[0].Path)
		require.Equal(t, ptr.P("/internal/"), match[0].Path.Path.Prefix)
		require.Equal(t, ptr.P(true), match[0].Path.Path.IgnoreCase)
		require.NotNil(t, match[1].Method)
		require.Equal(t, []openapi.MethodMatchMethods{"GET", "POST"}, match[1].Method.Methods)
		require.NotNil(t, match[2].Header)
		require.Equal(t, "X-Debug", match[2].Header.Name)
		require.Equal(t, ptr.P(openapi.FieldMatchPresent(true)), match[2].Header.Present)
		require.Nil(t, match[2].Header.Value)
		require.NotNil(t, match[3].QueryParam)
		require.Equal(t, "v", match[3].QueryParam.Name)
		require.NotNil(t, match[3].QueryParam.Value)
		require.Equal(t, ptr.P("1"), match[3].QueryParam.Value.Exact)
		require.Nil(t, match[3].QueryParam.Value.IgnoreCase)

		ratelimit := res.Body.Data[1]
		require.Equal(t, "pol_ratelimit", ratelimit.Id)
		require.NotNil(t, ratelimit.Ratelimit)
		require.Equal(t, int64(100), ratelimit.Ratelimit.Limit)
		require.Equal(t, int64(60000), ratelimit.Ratelimit.WindowMs)
		// The blob stores the deprecated single identifier; the response
		// renders it as a one-entry identifiers array.
		require.Nil(t, ratelimit.Ratelimit.Identifier)
		identifiers := ptr.SafeDeref(ratelimit.Ratelimit.Identifiers)
		require.Len(t, identifiers, 1)
		require.NotNil(t, identifiers[0].PrincipalField)
		require.Equal(t, "sub", identifiers[0].PrincipalField.Path)

		firewall := res.Body.Data[2]
		require.Equal(t, "pol_firewall", firewall.Id)
		require.False(t, firewall.Enabled)
		require.NotNil(t, firewall.Firewall)
		require.Equal(t, openapi.FirewallPolicyAction("ACTION_DENY"), firewall.Firewall.Action)
		require.Nil(t, firewall.Match)

		oa := res.Body.Data[3]
		require.Equal(t, "pol_openapi", oa.Id)
		// enabled was absent in the blob: proto optional bool defaults to
		// false, matching frontline evaluation semantics.
		require.False(t, oa.Enabled)
		require.NotNil(t, oa.Openapi)
	})

	t.Run("every ratelimit identifier source maps", func(t *testing.T) {
		env := seedEnvironment(t, h)
		seedSentinelConfig(t, h, env, `{"policies":[
			{"id":"pol_ip","name":"ip","enabled":true,"ratelimit":{"limit":"1","windowMs":"1000","identifier":{"remoteIp":{}}}},
			{"id":"pol_hdr","name":"hdr","enabled":true,"ratelimit":{"limit":"1","windowMs":"1000","identifier":{"header":{"name":"X-Tenant"}}}},
			{"id":"pol_sub","name":"sub","enabled":true,"ratelimit":{"limit":"1","windowMs":"1000","identifier":{"authenticatedSubject":{}}}},
			{"id":"pol_path","name":"path","enabled":true,"ratelimit":{"limit":"1","windowMs":"1000","identifier":{"path":{}}}}
		]}`)

		res := call(t, makeRequest(env))
		require.Len(t, res.Body.Data, 4)
		// Legacy stored single identifiers render as one-entry arrays.
		identifierAt := func(i int) openapi.RatelimitIdentifier {
			t.Helper()
			require.Nil(t, res.Body.Data[i].Ratelimit.Identifier)
			identifiers := ptr.SafeDeref(res.Body.Data[i].Ratelimit.Identifiers)
			require.Len(t, identifiers, 1)
			return identifiers[0]
		}
		require.NotNil(t, identifierAt(0).RemoteIp)
		require.NotNil(t, identifierAt(1).Header)
		require.Equal(t, "X-Tenant", identifierAt(1).Header.Name)
		require.NotNil(t, identifierAt(2).AuthenticatedSubject)
		require.NotNil(t, identifierAt(3).Path)
	})

	t.Run("returns every policy in stored order", func(t *testing.T) {
		env := seedEnvironment(t, h)
		ids := seedFirewallPolicies(t, h, env, 50)

		res := call(t, makeRequest(env))
		require.Len(t, res.Body.Data, 50)
		for i, p := range res.Body.Data {
			require.Equal(t, ids[i], p.Id)
		}
	})

	t.Run("reads back what setPolicies wrote", func(t *testing.T) {
		writeRoute := &setpolicies.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
		h.Register(writeRoute)

		env := seedEnvironment(t, h)
		api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
		writeKey := h.CreateRootKey(workspace.ID, "environment.*.set_policies")

		writeRes := testutil.CallRoute[setpolicies.Request, setpolicies.Response](h, writeRoute, authHeaders(writeKey), setpolicies.Request{
			Project:     env.projectID,
			App:         env.appID,
			Environment: env.environmentID,
			Policies: []openapi.Policy{
				{
					Name:    "require key",
					Enabled: true,
					Keyauth: &openapi.KeyauthPolicy{Keyspaces: []string{api.KeyAuthID.String}},
				},
				{
					Name:     "deny",
					Enabled:  false,
					Firewall: &openapi.FirewallPolicy{Action: "ACTION_DENY"},
				},
			},
		})
		require.Equal(t, 200, writeRes.Status, "expected 200, received: %s", writeRes.RawBody)

		res := call(t, makeRequest(env))
		require.Len(t, res.Body.Data, 2)

		require.Equal(t, "require key", res.Body.Data[0].Name)
		require.True(t, res.Body.Data[0].Enabled)
		require.NotNil(t, res.Body.Data[0].Keyauth)
		require.Equal(t, []string{api.KeyAuthID.String}, res.Body.Data[0].Keyauth.Keyspaces)

		require.Equal(t, "deny", res.Body.Data[1].Name)
		require.False(t, res.Body.Data[1].Enabled)
		require.NotNil(t, res.Body.Data[1].Firewall)

		for _, p := range res.Body.Data {
			require.Regexp(t, `^pol_[a-zA-Z0-9]{9}$`, p.Id)
		}
	})
}
