package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_gateway_set_policies"
)

func TestSetPoliciesBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	env := seedEnvironment(t, h)
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	rootKey := h.CreateRootKey(workspace.ID, "environment.*.set_policies")
	headers := authHeaders(rootKey)

	callTyped := func(t *testing.T, policies []openapi.Policy) testutil.TestResponse[openapi.BadRequestErrorResponse] {
		t.Helper()
		return testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, makeRequest(env, policies))
	}

	// One representative conversion failure proves mapPoliciesToProto errors
	// surface as 400 with a useful detail; the full validation matrix lives
	// in mapping_test.go.
	t.Run("no variant set", func(t *testing.T) {
		res := callTyped(t, []openapi.Policy{{Name: "empty", Enabled: true}})
		require.Equal(t, http.StatusBadRequest, res.Status, "received: %s", res.RawBody)
		require.Contains(t, res.Body.Error.Detail, "exactly one of")
	})

	t.Run("more than 50 policies in request", func(t *testing.T) {
		policies := make([]openapi.Policy, 51)
		for i := range policies {
			policies[i] = firewallPolicy(fmt.Sprintf("p%d", i), true)
		}
		res := callTyped(t, policies)
		require.Equal(t, http.StatusBadRequest, res.Status, "received: %s", res.RawBody)
	})

	t.Run("more than 10 match expressions", func(t *testing.T) {
		match := make([]openapi.MatchExpr, 11)
		for i := range match {
			match[i] = openapi.MatchExpr{Path: &openapi.PathMatch{Path: openapi.StringMatch{Prefix: ptr.P(fmt.Sprintf("/p%d", i))}}}
		}
		p := firewallPolicy("too many matches", true)
		p.Match = &match
		res := callTyped(t, []openapi.Policy{p})
		require.Equal(t, http.StatusBadRequest, res.Status, "received: %s", res.RawBody)
	})

	t.Run("empty keyspaces", func(t *testing.T) {
		res := callTyped(t, []openapi.Policy{{
			Name:    "k",
			Enabled: true,
			Keyauth: &openapi.KeyauthPolicy{Keyspaces: []string{}},
		}})
		require.Equal(t, http.StatusBadRequest, res.Status, "received: %s", res.RawBody)
	})

	t.Run("more than 5 keyspaces", func(t *testing.T) {
		ids := make([]string, 6)
		for i := range ids {
			ids[i] = api.KeyAuthID.String
		}
		res := callTyped(t, []openapi.Policy{{
			Name:    "k",
			Enabled: true,
			Keyauth: &openapi.KeyauthPolicy{Keyspaces: ids},
		}})
		require.Equal(t, http.StatusBadRequest, res.Status, "received: %s", res.RawBody)
	})

	t.Run("permissionQuery over 1000 chars", func(t *testing.T) {
		res := callTyped(t, []openapi.Policy{{
			Name:    "k",
			Enabled: true,
			Keyauth: &openapi.KeyauthPolicy{
				Keyspaces:       []string{api.KeyAuthID.String},
				PermissionQuery: ptr.P(strings.Repeat("a", 1001)),
			},
		}})
		require.Equal(t, http.StatusBadRequest, res.Status, "received: %s", res.RawBody)
	})

	// Raw payloads exercise schema-level rejections the typed request cannot
	// express: unknown fields must never reach the stored blob.
	rawPolicy := func(t *testing.T, policy map[string]any) testutil.TestResponse[openapi.BadRequestErrorResponse] {
		t.Helper()
		return testutil.CallRoute[map[string]any, openapi.BadRequestErrorResponse](h, route, headers, map[string]any{
			"project":     env.projectID,
			"app":         env.appID,
			"environment": env.environmentID,
			"policies":    []map[string]any{policy},
		})
	}

	t.Run("jwtauth variant is rejected by the schema", func(t *testing.T) {
		res := rawPolicy(t, map[string]any{"name": "jwt", "enabled": true, "jwtauth": map[string]any{}})
		require.Equal(t, http.StatusBadRequest, res.Status, "received: %s", res.RawBody)
	})

	t.Run("client-supplied id is rejected by the schema", func(t *testing.T) {
		res := rawPolicy(t, map[string]any{
			"name": "with id", "enabled": true, "id": uid.New(uid.PolicyPrefix),
			"firewall": map[string]any{"action": "ACTION_DENY"},
		})
		require.Equal(t, http.StatusBadRequest, res.Status, "received: %s", res.RawBody)
	})

	t.Run("unknown firewall action is rejected by the schema", func(t *testing.T) {
		res := rawPolicy(t, map[string]any{
			"name": "allow", "enabled": true,
			"firewall": map[string]any{"action": "ACTION_ALLOW"},
		})
		require.Equal(t, http.StatusBadRequest, res.Status, "received: %s", res.RawBody)
	})

	// The dashboard schema only accepts `present: true` (zod literal), so the
	// spec's enum must keep rejecting false or stored blobs break the reader.
	t.Run("present false is rejected by the schema", func(t *testing.T) {
		res := rawPolicy(t, map[string]any{
			"name": "m", "enabled": true,
			"firewall": map[string]any{"action": "ACTION_DENY"},
			"match":    []map[string]any{{"header": map[string]any{"name": "x-kebap", "present": false}}},
		})
		require.Equal(t, http.StatusBadRequest, res.Status, "received: %s", res.RawBody)
	})

	// The dashboard schema requires `enabled` on every stored policy, so the
	// spec must keep it required rather than defaulting it.
	t.Run("missing enabled is rejected by the schema", func(t *testing.T) {
		res := rawPolicy(t, map[string]any{
			"name":     "m",
			"firewall": map[string]any{"action": "ACTION_DENY"},
		})
		require.Equal(t, http.StatusBadRequest, res.Status, "received: %s", res.RawBody)
	})
}
