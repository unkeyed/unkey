package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_get_portal"
)

// A well-formed key that does not resolve is 401. A header that is missing or
// does not carry the Bearer prefix never reaches authentication at all and is
// reported as malformed, which is the second test below.
func TestGetPortalRequiresAuthentication(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	stored := h.SeedPortal(t, workspace.ID, "unauthenticated", "unauthenticated",
		keyspaceMapping(t, h, workspace.ID), nil, nil)
	req := handler.Request{Portal: ptr.P(stored.ID), KeyspaceId: nil, AppId: nil}

	testCases := map[string]string{
		"unknown key":   "Bearer unkey_thiskeydoesnotexist",
		"invalid token": "Bearer invalid_token",
	}

	for name, authorization := range testCases {
		t.Run(name, func(t *testing.T) {
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {authorization},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, http.StatusUnauthorized, res.Status,
				"expected 401, received: %s", res.RawBody)
			require.NotContains(t, res.RawBody, stored.Slug,
				"an unauthenticated response must not echo the portal")
		})
	}
}

// Reported as malformed rather than unauthorized: the header never gets far
// enough to be a credential.
func TestGetPortalRejectsMalformedAuthorization(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	stored := h.SeedPortal(t, workspace.ID, "malformed", "malformed",
		keyspaceMapping(t, h, workspace.ID), nil, nil)
	req := handler.Request{Portal: ptr.P(stored.ID), KeyspaceId: nil, AppId: nil}

	testCases := map[string]http.Header{
		"no authorization header": {"Content-Type": {"application/json"}},
		"missing bearer prefix":   {"Content-Type": {"application/json"}, "Authorization": {"unkey_notabearer"}},
		// An empty token after the prefix is malformed too, not an unknown key.
		"empty after prefix": {"Content-Type": {"application/json"}, "Authorization": {"Bearer "}},
	}

	for name, headers := range testCases {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, http.StatusBadRequest, res.Status,
				"expected 400, received: %s", res.RawBody)
		})
	}
}
