package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_delete_portal"
)

// Every way a delete can fail to reach a portal returns the same bytes. If
// any of these diverged the response would answer whether an id exists in a
// workspace the caller cannot see.
func TestDeletePortalMasksEveryMiss(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.delete_portal")
	workspace := h.Resources().UserWorkspace

	// A control case, so the misses below cannot be masking a broken handler. Its
	// id is reused afterwards as the already-deleted case.
	visible := h.SeedPortal(t, workspace.ID, "visible", "visible", keyspaceMapping(t, h, workspace.ID), nil, nil)
	ok := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(visible.ID))
	require.Equal(t, http.StatusOK, ok.Status, "the control case must succeed: %s", ok.RawBody)

	other := h.CreateWorkspace()
	otherKeyspace := keyspaceMapping(t, h, other.ID)
	otherPortal := h.SeedPortal(t, other.ID, "theirs", "theirs", otherKeyspace, nil, nil)

	testCases := map[string]string{
		"unknown id":                  uid.New(uid.PortalPrefix),
		"unknown slug":                "no-such-portal",
		"already deleted id":          visible.ID,
		"already deleted slug":        visible.Slug,
		"portal in another workspace": otherPortal.ID,
		"slug in another workspace":   otherPortal.Slug,
	}

	bodies := map[string]string{}
	for name, target := range testCases {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(target))
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
	require.True(t, portalExists(t, h, other.ID, otherPortal.ID))
	require.Equal(t, 1, countPortals(t, h, other.ID))
}

// Parity across callers: a caller lacking the grant and a caller naming a
// portal that does not exist must receive the same bytes.
func TestDeletePortalDenialMatchesAbsence(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	mapping := keyspaceMapping(t, h, workspace.ID)
	stored := h.SeedPortal(t, workspace.ID, "parity", "parity", mapping, nil, nil)
	h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})
	require.Equal(t, 1, liveSessions(t, h, stored.ID))

	deniedKey := h.CreateRootKey(workspace.ID, "portal.*.read_portal")
	denied := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(deniedKey), request(stored.ID))
	require.Equal(t, http.StatusNotFound, denied.Status,
		"a denial must be masked, received: %s", denied.RawBody)
	require.True(t, portalExists(t, h, workspace.ID, stored.ID), "a denied delete must not delete")
	require.Equal(t, 1, liveSessions(t, h, stored.ID), "a denied delete must not revoke")
	require.Equal(t, 0, countAuditEntriesMentioning(t, h, workspace.ID, "portal.delete"),
		"a denied delete must not write an audit entry")

	allowedKey := h.CreateRootKey(workspace.ID, "portal.*.delete_portal")
	absent := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(allowedKey),
		request(uid.New(uid.PortalPrefix)))
	require.Equal(t, http.StatusNotFound, absent.Status,
		"an absent portal must be 404, received: %s", absent.RawBody)

	require.Equal(t, normalizeRequestID(absent.RawBody), normalizeRequestID(denied.RawBody),
		"a denied delete and an absent portal must be indistinguishable")
}
