package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_sessions"
)

func TestListSessionsRejectsInvalidBody(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, workspaceAdminPermission(h))
	workspace := h.Resources().UserWorkspace
	stored := seedPortal(t, h, workspace.ID, "list-invalid")

	testCases := map[string]handler.Request{
		"missing portal":  {Portal: "", Limit: nil, Cursor: nil, Search: nil},
		"limit zero":      {Portal: stored.ID, Limit: new(0), Cursor: nil, Search: nil},
		"limit too large": {Portal: stored.ID, Limit: new(101), Cursor: nil, Search: nil},
		"search too long": {Portal: stored.ID, Limit: nil, Cursor: nil, Search: new(strings.Repeat("a", 257))},
	}

	for name, req := range testCases {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		})
	}

	t.Run("unknown field", func(t *testing.T) {
		res := testutil.CallRoute[map[string]any, openapi.BadRequestErrorResponse](h, route, headers,
			map[string]any{"portal": stored.ID, "externalId": "user_1"})
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
	})
}

func TestListSessionsRejectsMalformedAuthorization(t *testing.T) {
	h := testutil.NewHarness(t)
	route := registerRoute(h)
	workspace := h.Resources().UserWorkspace
	stored := seedPortal(t, h, workspace.ID, "list-malformed")

	testCases := map[string]http.Header{
		"no authorization header": {"Content-Type": {"application/json"}},
		"missing bearer prefix":   {"Content-Type": {"application/json"}, "Authorization": {"unkey_notabearer"}},
		"empty after prefix":      {"Content-Type": {"application/json"}, "Authorization": {"Bearer "}},
	}

	for name, headers := range testCases {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, request(stored.ID))
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		})
	}
}
