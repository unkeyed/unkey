package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_gateway_list_policies"
)

func TestListPoliciesBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "environment.*.read_policies")
	headers := authHeaders(rootKey)
	env := seedEnvironment(t, h)
	seedFirewallPolicies(t, h, env, 3)

	callTyped := func(t *testing.T, req handler.Request) testutil.TestResponse[openapi.BadRequestErrorResponse] {
		t.Helper()
		return testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
	}

	t.Run("unknown cursor", func(t *testing.T) {
		req := makeRequest(env)
		req.Cursor = ptr.P("pol_does_not_exist")
		res := callTyped(t, req)
		require.Equal(t, http.StatusBadRequest, res.Status, "received: %s", res.RawBody)
		require.Contains(t, res.Body.Error.Type, "invalid_input")
		require.Contains(t, res.Body.Error.Detail, "cursor")
	})

	t.Run("cursor against empty policy list", func(t *testing.T) {
		empty := seedEnvironment(t, h)
		req := makeRequest(empty)
		req.Cursor = ptr.P("pol_000")
		res := callTyped(t, req)
		require.Equal(t, http.StatusBadRequest, res.Status, "received: %s", res.RawBody)
	})

	t.Run("limit zero", func(t *testing.T) {
		req := makeRequest(env)
		req.Limit = ptr.P(0)
		res := callTyped(t, req)
		require.Equal(t, http.StatusBadRequest, res.Status, "received: %s", res.RawBody)
	})

	t.Run("limit above maximum", func(t *testing.T) {
		req := makeRequest(env)
		req.Limit = ptr.P(101)
		res := callTyped(t, req)
		require.Equal(t, http.StatusBadRequest, res.Status, "received: %s", res.RawBody)
	})

	t.Run("missing identifiers", func(t *testing.T) {
		res := callTyped(t, handler.Request{
			Project:     "",
			App:         "",
			Environment: "",
			Cursor:      nil,
			Limit:       nil,
		})
		require.Equal(t, http.StatusBadRequest, res.Status, "received: %s", res.RawBody)
	})
}
