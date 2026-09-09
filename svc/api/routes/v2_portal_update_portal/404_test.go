package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_update_portal"
)

// Every way an update can fail to reach a portal returns the same bytes. If
// any of these diverged the response would answer whether an id exists in a
// workspace the caller cannot see.
func TestUpdatePortalMasksEveryMiss(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.update_portal")
	workspace := h.Resources().UserWorkspace

	// A control case, so the misses below cannot be masking a broken handler.
	visible := h.SeedPortal(t, workspace.ID, "visible", "visible", keyspaceMapping(t, h, workspace.ID),
		nil, nil)
	control := baseRequest(visible.ID)
	control.Enabled = ptr.P(false)
	ok := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, control)
	require.Equal(t, http.StatusOK, ok.Status, "the control case must succeed: %s", ok.RawBody)

	other := h.CreateWorkspace()
	otherKeyspace := keyspaceMapping(t, h, other.ID)
	otherPortal := h.SeedPortal(t, other.ID, "theirs", "theirs", otherKeyspace, nil, nil)

	testCases := map[string]string{
		"unknown id":                  uid.New(uid.PortalPrefix),
		"unknown slug":                "no-such-portal",
		"portal in another workspace": otherPortal.ID,
		"slug in another workspace":   otherPortal.Slug,
	}

	bodies := map[string]string{}
	for name, target := range testCases {
		t.Run(name, func(t *testing.T) {
			req := baseRequest(target)
			req.Enabled = ptr.P(false)

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			bodies[name] = normalizeRequestID(res.RawBody)
		})
	}

	for name, body := range bodies {
		require.Equal(t, bodies["unknown id"], body,
			"%q must be byte-identical to an unknown id after normalizing the request id", name)
	}

	// The foreign portal is untouched, which is the consequence the
	// workspace-scoped resolve exists to guarantee.
	require.True(t, fetchPortal(t, h, other.ID, otherPortal.ID).Enabled)
}

// Parity across callers: a caller lacking the grant and a caller naming a
// portal that does not exist must receive the same bytes.
func TestUpdatePortalDenialMatchesAbsence(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	stored := h.SeedPortal(t, workspace.ID, "parity", "parity", keyspaceMapping(t, h, workspace.ID),
		nil, nil)

	req := baseRequest(stored.ID)
	req.Enabled = ptr.P(false)

	deniedKey := h.CreateRootKey(workspace.ID, "portal.*.read_portal")
	denied := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(deniedKey), req)
	require.Equal(t, http.StatusNotFound, denied.Status,
		"a denial must be masked, received: %s", denied.RawBody)
	require.True(t, fetchPortal(t, h, workspace.ID, stored.ID).Enabled,
		"a denied update must not write")
	require.Equal(t, 0, countAuditEntriesMentioning(t, h, workspace.ID, "portal.update"),
		"a denied update must not write an audit entry")

	absentReq := baseRequest(uid.New(uid.PortalPrefix))
	absentReq.Enabled = ptr.P(false)
	allowedKey := h.CreateRootKey(workspace.ID, "portal.*.update_portal")
	absent := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(allowedKey), absentReq)
	require.Equal(t, http.StatusNotFound, absent.Status,
		"an absent portal must be 404, received: %s", absent.RawBody)

	require.Equal(t, normalizeRequestID(absent.RawBody), normalizeRequestID(denied.RawBody),
		"a denied update and an absent portal must be indistinguishable")
}

// A mapping the caller does not own is a not-found identical to one that
// exists nowhere, and the row it tried to re-point is untouched. Those unique
// keys span the whole table, so allowing this would be a permanent global claim
// on another tenant's app or keyspace.
func TestUpdatePortalRejectsMappingsItDoesNotOwn(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.update_portal")
	workspace := h.Resources().UserWorkspace

	mapping := keyspaceMapping(t, h, workspace.ID)
	stored := h.SeedPortal(t, workspace.ID, "mine", "mine", mapping, nil, nil)
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys.read"})

	other := h.CreateWorkspace()
	foreignKeyspace := keyspaceMapping(t, h, other.ID)
	foreignApp := appMapping(t, h, other.ID, "theirs")

	testCases := map[string]portal.Mapping{
		"keyspace owned by another workspace": foreignKeyspace,
		"app owned by another workspace":      foreignApp,
		"keyspace that exists nowhere":        {ID: "ks_doesnotexist", Type: portal.MappingTypeKeyspace},
		"app that exists nowhere":             {ID: "app_doesnotexist", Type: portal.MappingTypeApp},
	}

	bodies := map[string]string{}
	for name, requested := range testCases {
		t.Run(name, func(t *testing.T) {
			req := baseRequest(stored.ID)
			req.KeyspaceId = ksOf(requested)
			req.AppId = appOf(requested)

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.NotContains(t, res.RawBody, other.ID,
				"the response must not name the owning workspace")

			row := fetchPortal(t, h, workspace.ID, stored.ID)
			require.Equal(t, mapping.ID, row.KeyAuthID.String, "the association must not change")
			require.False(t, row.AppID.Valid)
			require.Equal(t, 1, liveSessions(t, h, stored.ID),
				"a rejected re-point must not revoke sessions")

			bodies[name] = normalizeRequestID(res.RawBody)
		})
	}

	require.Equal(t, bodies["keyspace that exists nowhere"], bodies["keyspace owned by another workspace"],
		"a foreign keyspace must look identical to an absent one")
	require.Equal(t, bodies["app that exists nowhere"], bodies["app owned by another workspace"],
		"a foreign app must look identical to an absent one")
}
