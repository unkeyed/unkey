package handler_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_sessions"
)

func TestListSessionsReturnsOnlyRevocableSessions(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, workspaceAdminPermission(h))
	workspace := h.Resources().UserWorkspace
	stored := seedPortal(t, h, workspace.ID, "list-revocable")

	activeID := insertSession(t, h, stored.ID, workspace.ID, active(h, "user_1"))
	pendingID := insertSession(t, h, stored.ID, workspace.ID, pending(h, "user_1"))

	expired := active(h, "user_1")
	expired.tokenExpiresAt = new(h.Clock.Now().Add(-time.Minute))
	insertSession(t, h, stored.ID, workspace.ID, expired)

	expiredCode := pending(h, "user_1")
	expiredCode.codeExpiresAt = h.Clock.Now().Add(-time.Minute)
	insertSession(t, h, stored.ID, workspace.ID, expiredCode)

	revoked := active(h, "user_1")
	revoked.revoked = true
	insertSession(t, h, stored.ID, workspace.ID, revoked)

	onlyExpired := active(h, "user_2")
	onlyExpired.tokenExpiresAt = new(h.Clock.Now().Add(-time.Minute))
	insertSession(t, h, stored.ID, workspace.ID, onlyExpired)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, []string{"user_1"}, externalIDs(res.Body), "an end user with no revocable session is not listed")

	sessions := res.Body.Data[0].Sessions
	require.Len(t, sessions, 2)
	byID := map[string]openapi.V2PortalListSessionsSession{}
	for _, s := range sessions {
		byID[s.Id] = s
	}

	require.Equal(t, openapi.Active, byID[activeID].Status)
	require.Equal(t, h.Clock.Now().Add(24*time.Hour).UnixMilli(), byID[activeID].ExpiresAt, "an active session expires with its token")
	require.Equal(t, []string{"keys:read", "keys:reroll"}, byID[activeID].Scopes)

	require.Equal(t, openapi.Pending, byID[pendingID].Status)
	require.Equal(t, h.Clock.Now().Add(15*time.Minute).UnixMilli(), byID[pendingID].ExpiresAt, "a pending session expires with its code")
	require.Equal(t, h.Clock.Now().UnixMilli(), byID[pendingID].CreatedAt)
}

func TestListSessionsOrdersSessionsNewestFirst(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, workspaceAdminPermission(h))
	workspace := h.Resources().UserWorkspace
	stored := seedPortal(t, h, workspace.ID, "list-order")

	first := insertSession(t, h, stored.ID, workspace.ID, active(h, "user_1"))
	h.Clock.Tick(time.Second)
	second := insertSession(t, h, stored.ID, workspace.ID, pending(h, "user_1"))
	h.Clock.Tick(time.Second)
	third := insertSession(t, h, stored.ID, workspace.ID, active(h, "user_1"))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, 1)

	ids := []string{}
	for _, s := range res.Body.Data[0].Sessions {
		ids = append(ids, s.Id)
	}
	require.Equal(t, []string{third, second, first}, ids)
}

func TestListSessionsIsScopedToPortalAndWorkspace(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, workspaceAdminPermission(h))
	workspace := h.Resources().UserWorkspace
	stored := seedPortal(t, h, workspace.ID, "list-scoped")
	sibling := seedPortal(t, h, workspace.ID, "list-sibling")

	other := h.CreateWorkspace()
	theirs := seedPortal(t, h, other.ID, "list-theirs")

	insertSession(t, h, stored.ID, workspace.ID, active(h, "user_1"))
	insertSession(t, h, sibling.ID, workspace.ID, active(h, "user_2"))
	insertSession(t, h, theirs.ID, other.ID, active(h, "user_3"))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, []string{"user_1"}, externalIDs(res.Body))
}

func TestListSessionsPaginatesByEndUser(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, workspaceAdminPermission(h))
	workspace := h.Resources().UserWorkspace
	stored := seedPortal(t, h, workspace.ID, "list-pages")

	for _, externalID := range []string{"c", "a", "b"} {
		insertSession(t, h, stored.ID, workspace.ID, active(h, externalID))
		insertSession(t, h, stored.ID, workspace.ID, pending(h, externalID))
	}

	req := request(stored.ID)
	req.Limit = new(2)
	first := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusOK, first.Status, "expected 200, received: %s", first.RawBody)
	require.Equal(t, []string{"a", "b"}, externalIDs(first.Body))
	require.True(t, first.Body.Pagination.HasMore)
	require.NotNil(t, first.Body.Pagination.Cursor)
	for _, group := range first.Body.Data {
		require.Len(t, group.Sessions, 2, "a page limit counts end users, not sessions")
	}

	req.Cursor = first.Body.Pagination.Cursor
	second := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusOK, second.Status, "expected 200, received: %s", second.RawBody)
	require.Equal(t, []string{"c"}, externalIDs(second.Body))
	require.False(t, second.Body.Pagination.HasMore)
	require.Nil(t, second.Body.Pagination.Cursor)
}

func TestListSessionsSearchesByExternalIDPrefix(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, workspaceAdminPermission(h))
	workspace := h.Resources().UserWorkspace
	stored := seedPortal(t, h, workspace.ID, "list-search")

	for _, externalID := range []string{"user_1", "user_10", "admin_1", "userx1", "User_1"} {
		insertSession(t, h, stored.ID, workspace.ID, active(h, externalID))
	}

	testCases := map[string]struct {
		search   string
		expected []string
	}{
		"prefix":                    {search: "user_1", expected: []string{"user_1", "user_10"}},
		"underscore matches itself": {search: "user_", expected: []string{"user_1", "user_10"}},
		"percent matches itself":    {search: "user%", expected: []string{}},
		"case-sensitive":            {search: "User", expected: []string{"User_1"}},
		"not a substring match":     {search: "_1", expected: []string{}},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			req := request(stored.ID)
			req.Search = new(tc.search)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
			require.Equal(t, tc.expected, externalIDs(res.Body))
		})
	}
}

func TestListSessionsEmptyPortal(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, workspaceAdminPermission(h))
	workspace := h.Resources().UserWorkspace
	stored := seedPortal(t, h, workspace.ID, "list-empty")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Empty(t, res.Body.Data)
	require.Contains(t, res.RawBody, `"data":[]`)
	require.False(t, res.Body.Pagination.HasMore)
}

func TestListSessionsBySlug(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, workspaceAdminPermission(h))
	workspace := h.Resources().UserWorkspace
	stored := seedPortal(t, h, workspace.ID, "list-by-slug")
	insertSession(t, h, stored.ID, workspace.ID, active(h, "user_1"))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.Slug))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, []string{"user_1"}, externalIDs(res.Body))
}

// Hashes, the return URL, and the keyspaces a session reaches stay server-side.
func TestListSessionsOmitsSecretsAndKeyspaces(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, workspaceAdminPermission(h))
	workspace := h.Resources().UserWorkspace
	stored := seedPortal(t, h, workspace.ID, "list-redacted")
	insertSession(t, h, stored.ID, workspace.ID, active(h, "user_1"))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request(stored.ID))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	for _, field := range []string{"hash", "Hash", "returnUrl", "keyspaceIds", "ks_1"} {
		require.NotContains(t, res.RawBody, field)
	}
}
