package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_update_portal"
)

// Both collisions carry the same code, so the messages have to differ: a caller
// told only "already exists" cannot tell whether to pick another slug or accept
// that the resource is taken. The status is asserted explicitly because the
// middleware switch that maps this code carries //nolint:exhaustive, so a missing
// case would fall through to a 500 still carrying the right code.
func TestUpdatePortalConflicts(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.update_portal")
	workspace := h.Resources().UserWorkspace

	mine := seedPortal(t, h, workspace.ID, "mine", keyspaceMapping(t, h, workspace.ID),
		nullStringAbsent(), nullStringAbsent())
	siblingMapping := keyspaceMapping(t, h, workspace.ID)
	sibling := seedPortal(t, h, workspace.ID, "sibling", siblingMapping,
		nullStringAbsent(), nullStringAbsent())

	t.Run("slug held by a sibling", func(t *testing.T) {
		req := baseRequest(mine.ID)
		req.Slug = ptr(sibling.Slug)

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusConflict, res.Status, "expected 409, received: %s", res.RawBody)
		require.Contains(t, res.RawBody, "portal_already_exists")
		require.Contains(t, res.RawBody, "slug", "the message names which input to change")

		require.Equal(t, "mine", fetchPortal(t, h, workspace.ID, mine.ID).Slug,
			"a conflicting update must not write")
	})

	t.Run("mapping held by a sibling", func(t *testing.T) {
		req := baseRequest(mine.ID)
		req.KeyspaceId = ksOf(siblingMapping)
		req.AppId = appOf(siblingMapping)

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusConflict, res.Status, "expected 409, received: %s", res.RawBody)
		require.Contains(t, res.RawBody, "portal_already_exists")
		require.Contains(t, res.RawBody, "keyspace", "the message names the mapping, not the slug")

		row := fetchPortal(t, h, workspace.ID, mine.ID)
		require.NotEqual(t, siblingMapping.ID, row.KeyAuthID.String, "a conflicting update must not write")
	})
}

// Re-sending what the portal already holds is not a conflict. Without excluding
// the row from its own collision check, every idempotent retry -- and every
// dashboard save that round-trips the current values -- would 409.
func TestUpdatePortalAcceptsItsOwnCurrentValues(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.update_portal")
	workspace := h.Resources().UserWorkspace

	mapping := keyspaceMapping(t, h, workspace.ID)
	stored := seedPortal(t, h, workspace.ID, "idempotent", mapping, nullStringAbsent(), nullStringAbsent())

	req := baseRequest(stored.ID)
	req.Slug = ptr(stored.Slug)
	req.KeyspaceId = ksOf(mapping)
	req.AppId = appOf(mapping)
	req.Enabled = ptr(true)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusOK, res.Status,
		"a portal's own slug and mapping must not collide with itself: %s", res.RawBody)
	require.Equal(t, "idempotent", res.Body.Data.Slug)
	require.NotNil(t, res.Body.Data.KeyspaceId)
	require.Equal(t, mapping.ID, string(*res.Body.Data.KeyspaceId))
}

// The slug unique key is (workspace_id, slug), so a slug another workspace holds
// is not a collision here.
func TestUpdatePortalAllowsSlugHeldByAnotherWorkspace(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.update_portal")
	workspace := h.Resources().UserWorkspace

	stored := seedPortal(t, h, workspace.ID, "local", keyspaceMapping(t, h, workspace.ID),
		nullStringAbsent(), nullStringAbsent())

	other := h.CreateWorkspace()
	seedPortal(t, h, other.ID, "shared-slug", keyspaceMapping(t, h, other.ID),
		nullStringAbsent(), nullStringAbsent())

	req := baseRequest(stored.ID)
	req.Slug = ptr("shared-slug")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusOK, res.Status,
		"a slug held by another workspace must not block this one: %s", res.RawBody)
	require.Equal(t, "shared-slug", fetchPortal(t, h, workspace.ID, stored.ID).Slug)
}
