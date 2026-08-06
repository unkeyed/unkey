package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_gateway_set_policies"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestSetPoliciesSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	ctx := context.Background()
	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "environment.*.set_policies")
	headers := authHeaders(rootKey)

	call := func(t *testing.T, req handler.Request) {
		t.Helper()
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	}

	t.Run("batch of all four variants stores dashboard-compatible wire JSON", func(t *testing.T) {
		env := seedEnvironment(t, h)
		api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})

		call(t, makeRequest(env, []openapi.Policy{
			{
				Name:    "keyauth",
				Enabled: true,
				Keyauth: &openapi.KeyauthPolicy{Keyspaces: []string{api.KeyAuthID.String}},
			},
			{
				Name:    "ratelimit",
				Enabled: true,
				Ratelimit: &openapi.RatelimitPolicy{
					Limit:      100,
					WindowMs:   60000,
					Identifier: &openapi.RatelimitIdentifier{RemoteIp: &openapi.RemoteIpKey{}},
				},
			},
			firewallPolicy("KEBAP", false),
			{
				Name:    "openapi",
				Enabled: true,
				Openapi: &openapi.OpenapiPolicy{},
			},
		}))

		stored := readStoredPolicies(t, h, env)
		require.Len(t, stored, 4)

		// The gateway must be able to parse the stored blob the way its
		// ParseMiddleware does.
		require.NoError(t, protojson.Unmarshal([]byte(readStoredBlob(t, h, env)), &frontlinev1.Config{}))

		// The dashboard reads the blob through a strict schema: enabled must be
		// present even when false, ids must exist, and no type field may appear.
		names := make([]string, 0, len(stored))
		byName := make(map[string]map[string]json.RawMessage)
		for _, raw := range stored {
			var keys map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(raw, &keys))
			require.NotContains(t, keys, "type")
			require.Contains(t, keys, "enabled")

			var id, name string
			require.NoError(t, json.Unmarshal(keys["id"], &id))
			require.NoError(t, json.Unmarshal(keys["name"], &name))
			require.NotEmpty(t, id)
			names = append(names, name)
			byName[name] = keys
		}

		require.Equal(t, []string{"keyauth", "ratelimit", "KEBAP", "openapi"}, names,
			"stored order must be the request order")
		require.JSONEq(t, `false`, string(byName["KEBAP"]["enabled"]))
		require.JSONEq(t, `{"action":"ACTION_DENY"}`, string(byName["KEBAP"]["firewall"]))
		require.JSONEq(t,
			fmt.Sprintf(`{"keySpaceIds":["%s"]}`, api.KeyAuthID.String),
			string(byName["keyauth"]["keyauth"]),
		)
		require.NotContains(t, readStoredBlob(t, h, env), `"keyspaces"`,
			"the public field name must never leak into the stored blob")
		require.JSONEq(t, `{}`, string(byName["openapi"]["openapi"]))
		require.JSONEq(
			t,
			`{"limit":"100","windowMs":"60000","identifier":{"remoteIp":{}}}`,
			string(byName["ratelimit"]["ratelimit"]),
		)

		logs := h.FindAuditLogsByTargetID(ctx, t, env.environmentID)
		require.Len(t, logs, 4)
		for _, l := range logs {
			require.Contains(t, l.Description, "Set policy")
			require.NotEmpty(t, l.Targets)

			// Each audit row must carry the policy document exactly as
			// stored, so the trail alone reconstructs past configurations.
			meta := l.Targets[0].Meta
			auditDoc, err := json.Marshal(meta["policy"])
			require.NoError(t, err)
			var docFields struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}
			require.NoError(t, json.Unmarshal(auditDoc, &docFields))
			require.Equal(t, docFields.ID, meta["policyId"])

			storedDoc, err := json.Marshal(byName[docFields.Name])
			require.NoError(t, err)
			require.JSONEq(t, string(storedDoc), string(auditDoc))
		}
	})

	t.Run("set replaces stored policies including variants this API cannot create", func(t *testing.T) {
		env := seedEnvironment(t, h)
		jwtauth := `{"id":"pol_jwt","name":"legacy jwt","enabled":true,"jwtauth":{}}`
		seedSentinelConfig(t, h, env, fmt.Sprintf(`{"policies":[%s]}`, jwtauth))

		call(t, makeRequest(env, []openapi.Policy{firewallPolicy("deny", true)}))

		stored := readStoredPolicies(t, h, env)
		require.Len(t, stored, 1)
		require.Contains(t, string(stored[0]), `"name":"deny"`)
		require.NotContains(t, readStoredBlob(t, h, env), "pol_jwt")
	})

	t.Run("second set replaces the first and regenerates ids", func(t *testing.T) {
		env := seedEnvironment(t, h)
		call(t, makeRequest(env, []openapi.Policy{
			firewallPolicy("first", true),
			firewallPolicy("second", true),
		}))
		before := storedPolicyIDs(t, h, env)
		require.Len(t, before, 2)

		call(t, makeRequest(env, []openapi.Policy{firewallPolicy("only", false)}))

		after := storedPolicyIDs(t, h, env)
		require.Len(t, after, 1)
		require.NotContains(t, before, after[0], "every set generates fresh ids")
		stored := readStoredPolicies(t, h, env)
		require.Contains(t, string(stored[0]), `"name":"only"`)
	})

	t.Run("empty list removes all policies", func(t *testing.T) {
		env := seedEnvironment(t, h)
		call(t, makeRequest(env, []openapi.Policy{firewallPolicy("doomed", true)}))

		call(t, makeRequest(env, []openapi.Policy{}))

		stored := readStoredPolicies(t, h, env)
		require.Empty(t, stored)
		// The dashboard's strict schema requires the `policies` key even when
		// empty (protojson would omit it, so the handler writes it literally).
		require.JSONEq(t, `{"policies":[]}`, readStoredBlob(t, h, env))

		logs := h.FindAuditLogsByTargetID(ctx, t, env.environmentID)
		var removed bool
		for _, l := range logs {
			if strings.Contains(l.Description, "Removed all policies") {
				removed = true
			}
		}
		require.True(t, removed, "clearing must leave a dedicated audit entry")
	})

	// Pins the stored wire bytes for every sub-shape a policy can carry, so
	// a marshalling or proto-name drift in any of them fails here instead of
	// in the gateway or the dashboard.
	t.Run("stores every identifier, matcher and keyauth sub-shape", func(t *testing.T) {
		env := seedEnvironment(t, h)
		apiA := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
		apiB := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})

		present := openapi.FieldMatchPresent(true)
		kitchenSink := openapi.Policy{
			Name:    "kitchen-sink",
			Enabled: true,
			Match: &[]openapi.MatchExpr{
				{Path: &openapi.PathMatch{Path: openapi.StringMatch{Exact: ptr.P("/v1/kebap"), IgnoreCase: ptr.P(false)}}},
				{Path: &openapi.PathMatch{Path: openapi.StringMatch{Prefix: ptr.P("/api/"), IgnoreCase: ptr.P(true)}}},
				{Path: &openapi.PathMatch{Path: openapi.StringMatch{Regex: ptr.P("^/a/[0-9]+$")}}},
				{Method: &openapi.MethodMatch{Methods: []openapi.MethodMatchMethods{"GET", "POST"}}},
				{Header: &openapi.FieldMatch{Name: "x-kebap", Present: &present}},
				{Header: &openapi.FieldMatch{Name: "x-token", Value: &openapi.StringMatch{Prefix: ptr.P("tok_")}}},
				{QueryParam: &openapi.FieldMatch{Name: "debug", Present: &present}},
				{QueryParam: &openapi.FieldMatch{Name: "v", Value: &openapi.StringMatch{Exact: ptr.P("1")}}},
			},
			Keyauth: &openapi.KeyauthPolicy{
				Keyspaces: []string{apiA.KeyAuthID.String, apiB.KeyAuthID.String},
				Locations: &[]openapi.KeyLocation{
					{Bearer: &openapi.BearerTokenLocation{}},
					{Header: &openapi.HeaderKeyLocation{Name: "x-api-key", StripPrefix: ptr.P("Key ")}},
					{Header: &openapi.HeaderKeyLocation{Name: "x-plain"}},
					{QueryParam: &openapi.QueryParamKeyLocation{Name: "api_key"}},
				},
				PermissionQuery: ptr.P("documents.read AND (billing.read OR billing.admin)"),
				Ratelimits: &[]openapi.KeyRatelimit{
					{Name: "requests"},
					{Name: "burst", Limit: ptr.P(int64(10)), Duration: ptr.P(int64(1000))},
					{Name: "heavy", Limit: ptr.P(int64(5)), Duration: ptr.P(int64(60000)), Cost: ptr.P(int64(2))},
					{Name: "kebap", Cost: ptr.P(int64(3))},
				},
			},
		}

		ratelimitPolicy := func(name string, id openapi.RatelimitIdentifier) openapi.Policy {
			return openapi.Policy{
				Name:      name,
				Enabled:   true,
				Ratelimit: &openapi.RatelimitPolicy{Limit: 100, WindowMs: 60000, Identifier: &id},
			}
		}

		policies := []openapi.Policy{
			kitchenSink,
			ratelimitPolicy("by-header", openapi.RatelimitIdentifier{Header: &openapi.HeaderKey{Name: "x-client-id"}}),
			ratelimitPolicy("by-subject", openapi.RatelimitIdentifier{AuthenticatedSubject: &openapi.AuthenticatedSubjectKey{}}),
			ratelimitPolicy("by-path", openapi.RatelimitIdentifier{Path: &openapi.PathKey{}}),
			ratelimitPolicy("by-principal", openapi.RatelimitIdentifier{PrincipalField: &openapi.PrincipalFieldKey{Path: "claims.org"}}),
			{
				Name:    "empty-optionals",
				Enabled: true,
				Keyauth: &openapi.KeyauthPolicy{
					Keyspaces:  []string{apiA.KeyAuthID.String},
					Locations:  &[]openapi.KeyLocation{},
					Ratelimits: &[]openapi.KeyRatelimit{},
				},
			},
		}
		call(t, makeRequest(env, policies))

		require.NoError(t, protojson.Unmarshal([]byte(readStoredBlob(t, h, env)), &frontlinev1.Config{}))

		// Guards the conversion against silently dropping a spec field: every
		// key this request serializes must reappear in the stored blob. The
		// request covers every sub-shape, so a field added to the spec but
		// missed in convert.go fails here mechanically. `keyspaces` is stored
		// as `keySpaceIds`; zero values carry no content and are normalized
		// away, so they are exempt.
		reqJSON, err := json.Marshal(policies)
		require.NoError(t, err)
		blobKeys := jsonContentKeys(t, []byte(readStoredBlob(t, h, env)))
		for key := range jsonContentKeys(t, reqJSON) {
			if key == "keyspaces" {
				key = "keySpaceIds"
			}
			require.Contains(t, blobKeys, key,
				"request field %q is missing from the stored blob; the conversion dropped it", key)
		}

		stored := readStoredPolicies(t, h, env)
		require.Len(t, stored, 6)
		byName := make(map[string]map[string]json.RawMessage, len(stored))
		for _, raw := range stored {
			var keys map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(raw, &keys))
			var name string
			require.NoError(t, json.Unmarshal(keys["name"], &name))
			byName[name] = keys
		}

		// protojson emits int64 as JSON strings and normalizes zero values
		// away (an explicit ignoreCase:false in the request is not stored).
		require.JSONEq(t, `[
			{"path":{"path":{"exact":"/v1/kebap"}}},
			{"path":{"path":{"prefix":"/api/","ignoreCase":true}}},
			{"path":{"path":{"regex":"^/a/[0-9]+$"}}},
			{"method":{"methods":["GET","POST"]}},
			{"header":{"name":"x-kebap","present":true}},
			{"header":{"name":"x-token","value":{"prefix":"tok_"}}},
			{"queryParam":{"name":"debug","present":true}},
			{"queryParam":{"name":"v","value":{"exact":"1"}}}
		]`, string(byName["kitchen-sink"]["match"]))

		require.JSONEq(t, fmt.Sprintf(`{
			"keySpaceIds":["%s","%s"],
			"locations":[
				{"bearer":{}},
				{"header":{"name":"x-api-key","stripPrefix":"Key "}},
				{"header":{"name":"x-plain"}},
				{"queryParam":{"name":"api_key"}}
			],
			"permissionQuery":"documents.read AND (billing.read OR billing.admin)",
			"ratelimits":[
				{"name":"requests"},
				{"name":"burst","limit":"10","duration":"1000"},
				{"name":"heavy","limit":"5","duration":"60000","cost":"2"},
				{"name":"kebap","cost":"3"}
			]
		}`, apiA.KeyAuthID.String, apiB.KeyAuthID.String), string(byName["kitchen-sink"]["keyauth"]))

		require.JSONEq(t, `{"limit":"100","windowMs":"60000","identifier":{"header":{"name":"x-client-id"}}}`,
			string(byName["by-header"]["ratelimit"]))
		require.JSONEq(t, `{"limit":"100","windowMs":"60000","identifier":{"authenticatedSubject":{}}}`,
			string(byName["by-subject"]["ratelimit"]))
		require.JSONEq(t, `{"limit":"100","windowMs":"60000","identifier":{"path":{}}}`,
			string(byName["by-path"]["ratelimit"]))
		require.JSONEq(t, `{"limit":"100","windowMs":"60000","identifier":{"principalField":{"path":"claims.org"}}}`,
			string(byName["by-principal"]["ratelimit"]))

		// Explicit empty arrays in the request are proto defaults, so protojson
		// normalizes them out of the stored blob.
		require.JSONEq(t, fmt.Sprintf(`{"keySpaceIds":["%s"]}`, apiA.KeyAuthID.String),
			string(byName["empty-optionals"]["keyauth"]))
	})

	t.Run("accepts exactly 50 policies", func(t *testing.T) {
		env := seedEnvironment(t, h)
		policies := make([]openapi.Policy, 50)
		for i := range policies {
			policies[i] = firewallPolicy(fmt.Sprintf("p%d", i), true)
		}
		call(t, makeRequest(env, policies))
		require.Len(t, readStoredPolicies(t, h, env), 50)
	})

	t.Run("concurrent sets end with exactly one intact list", func(t *testing.T) {
		env := seedEnvironment(t, h)

		const workers = 5
		var wg sync.WaitGroup
		statuses := make([]int, workers)
		for i := range workers {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers,
					makeRequest(env, []openapi.Policy{firewallPolicy(fmt.Sprintf("policy-%d", i), true)}))
				statuses[i] = res.Status
			}(i)
		}
		wg.Wait()

		for i, status := range statuses {
			require.Equal(t, 200, status, "worker %d", i)
		}

		// Smoke: concurrent sets all succeed and last writer wins.
		stored := readStoredPolicies(t, h, env)
		require.Len(t, stored, 1)
		require.Regexp(t, `"name":"policy-\d"`, string(stored[0]))
	})
}
