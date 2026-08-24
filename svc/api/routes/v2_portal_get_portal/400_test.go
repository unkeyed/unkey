package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_get_portal"
)

// Exactly one of `portal` and `mapping` addresses the read. Both together would
// need a precedence rule that silently ignores half the request, and neither
// names nothing at all, so both are rejected before anything is resolved.
func TestGetPortalRejectsInvalidInput(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.read_portal")
	workspace := h.Resources().UserWorkspace

	mapping := keyspaceMapping(t, h, workspace.ID)
	stored := seedPortal(t, h, workspace.ID, "both-and-neither", mapping,
		nullStringAbsent(), nullStringAbsent())
	someAppID := openapi.PortalAppId("app_1234abcd")
	blankKeyspaceID := openapi.PortalKeyspaceId("   ")

	// Exactly one of the three addresses is required. The flat shape makes more
	// combinations expressible than the nested one did, so each is refused here.
	testCases := map[string]handler.Request{
		"portal and keyspace id":    {Portal: ptr(stored.ID), KeyspaceId: ksOf(mapping), AppId: nil},
		"portal and app id":         {Portal: ptr(stored.ID), KeyspaceId: nil, AppId: &someAppID},
		"keyspace id and app id":    {Portal: nil, KeyspaceId: ksOf(mapping), AppId: &someAppID},
		"all three":                 {Portal: ptr(stored.ID), KeyspaceId: ksOf(mapping), AppId: &someAppID},
		"neither":                   {Portal: nil, KeyspaceId: nil, AppId: nil},
		"empty portal":              {Portal: ptr(""), KeyspaceId: nil, AppId: nil},
		"whitespace portal":         {Portal: ptr("   "), KeyspaceId: nil, AppId: nil},
		"whitespace keyspace id":    {Portal: nil, KeyspaceId: &blankKeyspaceID, AppId: nil},
		"whitespace id with portal": {Portal: ptr(stored.ID), KeyspaceId: &blankKeyspaceID, AppId: nil},
	}

	for name, req := range testCases {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, http.StatusBadRequest, res.Status,
				"expected 400, received: %s", res.RawBody)
			require.NotContains(t, res.RawBody, stored.ID,
				"a validation error must not echo a resolved portal id")
		})
	}
}
