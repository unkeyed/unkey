package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
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
	visible := h.SeedPortal(t, workspace.ID, "visible", "visible", keyspaceMapping(t, h, workspace.ID),
		nil, nil)
	ok := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Portal:     ptr.P(visible.ID),
		KeyspaceId: nil,
		AppId:      nil,
	})
	require.Equal(t, http.StatusOK, ok.Status, "the control case must succeed: %s", ok.RawBody)

	// An association in this workspace with no portal behind it.
	unmappedKeyspace := keyspaceMapping(t, h, workspace.ID)
	unmappedApp := appMapping(t, h, workspace.ID, "unmapped")

	// Another workspace with a portal of its own, addressed both by id and through
	// the keyspace it maps.
	other := h.CreateWorkspace()
	otherKeyspace := keyspaceMapping(t, h, other.ID)
	otherPortal := h.SeedPortal(t, other.ID, "theirs", "theirs", otherKeyspace, nil, nil)

	unknownKeyspace := portal.Mapping{Type: portal.MappingTypeKeyspace, ID: "ks_doesnotexist"}
	unknownApp := portal.Mapping{Type: portal.MappingTypeApp, ID: "app_doesnotexist"}

	testCases := map[string]handler.Request{
		"unknown id":                    {Portal: ptr.P("pc_doesnotexist"), KeyspaceId: nil, AppId: nil},
		"unknown slug":                  {Portal: ptr.P("no-such-portal"), KeyspaceId: nil, AppId: nil},
		"portal in another workspace":   {Portal: ptr.P(otherPortal.ID), KeyspaceId: nil, AppId: nil},
		"slug in another workspace":     {Portal: ptr.P(otherPortal.Slug), KeyspaceId: nil, AppId: nil},
		"keyspace with no portal":       {Portal: nil, KeyspaceId: ksOf(unmappedKeyspace), AppId: appOf(unmappedKeyspace)},
		"app with no portal":            {Portal: nil, KeyspaceId: ksOf(unmappedApp), AppId: appOf(unmappedApp)},
		"keyspace in another workspace": {Portal: nil, KeyspaceId: ksOf(otherKeyspace), AppId: appOf(otherKeyspace)},
		"keyspace that exists nowhere":  {Portal: nil, KeyspaceId: ksOf(unknownKeyspace), AppId: appOf(unknownKeyspace)},
		"app that exists nowhere":       {Portal: nil, KeyspaceId: ksOf(unknownApp), AppId: appOf(unknownApp)},
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
	stored := h.SeedPortal(t, workspace.ID, "parity", "parity", keyspaceMapping(t, h, workspace.ID),
		nil, nil)

	deniedKey := h.CreateRootKey(workspace.ID, "portal.*.create_portal")
	denied := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(deniedKey), handler.Request{
		Portal:     ptr.P(stored.ID),
		KeyspaceId: nil,
		AppId:      nil,
	})
	require.Equal(t, http.StatusNotFound, denied.Status,
		"a denial must be masked, received: %s", denied.RawBody)

	allowedKey := h.CreateRootKey(workspace.ID, "portal.*.read_portal")
	absent := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(allowedKey), handler.Request{
		Portal:     ptr.P("pc_doesnotexist"),
		KeyspaceId: nil,
		AppId:      nil,
	})
	require.Equal(t, http.StatusNotFound, absent.Status,
		"an absent portal must be 404, received: %s", absent.RawBody)

	require.Equal(t, normalizeRequestID(absent.RawBody), normalizeRequestID(denied.RawBody),
		"a denied read and an absent portal must be indistinguishable")
}
