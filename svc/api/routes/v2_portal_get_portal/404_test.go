package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_get_portal"
)

// Every way a read can fail to produce a portal returns the same body. If any of
// these diverged the response would answer a question the caller is not entitled
// to ask: whether an id exists in another workspace, or whether an app it does
// not own has a portal wired up.
func TestGetPortalMasksEveryMiss(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.read_portal")
	workspace := h.Resources().UserWorkspace

	// A portal the caller can see, so the passing path is known to work and the
	// misses below cannot be masking a broken handler.
	visible := seedPortal(t, h, workspace.ID, "visible", keyspaceMapping(t, h, workspace.ID),
		nullStringAbsent(), nullStringAbsent())
	ok := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Portal:  ptr(visible.ID),
		Mapping: nil,
	})
	require.Equal(t, http.StatusOK, ok.Status, "the control case must succeed: %s", ok.RawBody)

	// An association in this workspace with no portal behind it.
	unmappedKeyspace := keyspaceMapping(t, h, workspace.ID)
	unmappedApp := appMapping(t, h, workspace.ID, "unmapped")

	// Another workspace with a portal of its own, addressed both by id and through
	// the keyspace it maps.
	other := h.CreateWorkspace()
	otherKeyspace := keyspaceMapping(t, h, other.ID)
	otherPortal := h.CreatePortal(seed.CreatePortalRequest{
		ID:           "",
		WorkspaceID:  other.ID,
		Slug:         "theirs",
		AppID:        nullStringAbsent(),
		KeyAuthID:    nullString(otherKeyspace.Id),
		Enabled:      true,
		LogoUrl:      nullStringAbsent(),
		PrimaryColor: nullStringAbsent(),
	})

	unknownKeyspace := openapi.PortalMapping{Id: "ks_doesnotexist", Type: openapi.PortalMappingTypeKeyspace}
	unknownApp := openapi.PortalMapping{Id: "app_doesnotexist", Type: openapi.PortalMappingTypeApp}

	testCases := map[string]handler.Request{
		"unknown id":                    {Portal: ptr("pc_doesnotexist"), Mapping: nil},
		"unknown slug":                  {Portal: ptr("no-such-portal"), Mapping: nil},
		"portal in another workspace":   {Portal: ptr(otherPortal.ID), Mapping: nil},
		"slug in another workspace":     {Portal: ptr(otherPortal.Slug), Mapping: nil},
		"keyspace with no portal":       {Portal: nil, Mapping: &unmappedKeyspace},
		"app with no portal":            {Portal: nil, Mapping: &unmappedApp},
		"keyspace in another workspace": {Portal: nil, Mapping: &otherKeyspace},
		"keyspace that exists nowhere":  {Portal: nil, Mapping: &unknownKeyspace},
		"app that exists nowhere":       {Portal: nil, Mapping: &unknownApp},
	}

	bodies := map[string]string{}
	for name, req := range testCases {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, http.StatusNotFound, res.Status,
				"expected 404, received: %s", res.RawBody)
			require.NotContains(t, res.RawBody, otherPortal.ID,
				"a miss must not disclose another workspace's portal id")
			bodies[name] = normalizeRequestID(res.RawBody)
		})
	}

	for name, body := range bodies {
		require.Equal(t, bodies["unknown id"], body,
			"%q must be byte-identical to an unknown id after normalizing the request id", name)
	}
}

// The parity the masking depends on, asserted across the two callers rather than
// within one: a caller lacking the grant and a caller naming a portal that does
// not exist must receive the same bytes.
func TestGetPortalDenialMatchesAbsence(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	stored := seedPortal(t, h, workspace.ID, "parity", keyspaceMapping(t, h, workspace.ID),
		nullStringAbsent(), nullStringAbsent())

	deniedKey := h.CreateRootKey(workspace.ID, "portal.*.create_portal")
	denied := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(deniedKey), handler.Request{
		Portal:  ptr(stored.ID),
		Mapping: nil,
	})
	require.Equal(t, http.StatusNotFound, denied.Status,
		"a denial must be masked, received: %s", denied.RawBody)

	allowedKey := h.CreateRootKey(workspace.ID, "portal.*.read_portal")
	absent := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(allowedKey), handler.Request{
		Portal:  ptr("pc_doesnotexist"),
		Mapping: nil,
	})
	require.Equal(t, http.StatusNotFound, absent.Status,
		"an absent portal must be 404, received: %s", absent.RawBody)

	require.Equal(t, normalizeRequestID(absent.RawBody), normalizeRequestID(denied.RawBody),
		"a denied read and an absent portal must be indistinguishable")
}
