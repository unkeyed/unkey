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
	unknownType := openapi.PortalMapping{Id: mapping.Id, Type: openapi.PortalMappingType("project")}

	testCases := map[string]handler.Request{
		"both portal and mapping": {Portal: ptr(stored.ID), Mapping: &mapping},
		"neither":                 {Portal: nil, Mapping: nil},
		"empty portal":            {Portal: ptr(""), Mapping: nil},
		"whitespace portal":       {Portal: ptr("   "), Mapping: nil},
		"unknown mapping type":    {Portal: nil, Mapping: &unknownType},
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
