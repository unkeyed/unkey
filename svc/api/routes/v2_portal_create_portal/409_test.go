package handler_test

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_portal"
)

// Both collisions return 409 with the same code, but the messages must differ:
// a caller told only "already exists" cannot tell whether to pick another slug
// or accept that the resource is taken. The status is asserted explicitly because
// the middleware switch that maps this code carries //nolint:exhaustive, so a
// missing case would fall through to a 500 still carrying the right code.
func TestCreatePortalConflicts(t *testing.T) {
	h := testutil.NewHarness(t)
	route, headers := newRoute(t, h, "portal.*.create_portal")
	workspace := h.Resources().UserWorkspace

	taken := keyspaceMapping(t, h, workspace.ID)
	h.CreatePortal(seed.CreatePortalRequest{
		WorkspaceID: workspace.ID,
		Slug:        "taken-slug",
		KeyAuthID:   sql.NullString{String: taken.ID, Valid: true},
		Enabled:     true,
	})

	t.Run("duplicate slug", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Slug:        "taken-slug",
			DisplayName: "Acme",
			KeyspaceId:  ksOf(keyspaceMapping(t, h, workspace.ID)),
			AppId:       appOf(keyspaceMapping(t, h, workspace.ID)),
			Enabled:     ptr.P(true),
		})
		require.Equal(t, http.StatusConflict, res.Status, "expected 409, received: %s", res.RawBody)
		require.Contains(t, res.RawBody, "portal_already_exists")
		require.Contains(t, res.RawBody, "slug", "the message names which input to change")
		// Echoing the caller's own slug is harmless here, but the mapping case
		// below must not name anything about the other tenant.
	})

	t.Run("mapping already backs a portal", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Slug:        "fresh-slug",
			DisplayName: "Acme",
			KeyspaceId:  ksOf(taken),
			AppId:       appOf(taken),
			Enabled:     ptr.P(true),
		})
		require.Equal(t, http.StatusConflict, res.Status, "expected 409, received: %s", res.RawBody)
		require.Contains(t, res.RawBody, "portal_already_exists")
		require.Contains(t, res.RawBody, "keyspace", "the message names the mapping, not the slug")
	})

	// The association unique keys are table-wide, so a caller can collide with a
	// portal in a workspace it cannot see. That must still be a conflict rather
	// than an internal error, and it must not name the owning workspace.
	t.Run("mapping claimed by another workspace", func(t *testing.T) {
		other := h.CreateWorkspace()
		sharedApi := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   other.ID,
			IpWhitelist:   "",
			EncryptedKeys: false,
			Name:          nil,
			CreatedAt:     nil,
			DefaultPrefix: nil,
			DefaultBytes:  nil,
		})
		h.CreatePortal(seed.CreatePortalRequest{
			WorkspaceID: other.ID,
			Slug:        "theirs",
			KeyAuthID:   sql.NullString{String: sharedApi.KeyAuthID.String, Valid: true},
			Enabled:     true,
		})

		// The caller does not own this keyspace, so ownership is checked first and
		// reports not-found. The conflict path for a foreign claim is only
		// reachable for a resource the caller does own, which cannot be
		// constructed here -- ownership and the global unique key disagree only in
		// data written outside these routes.
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Slug:        "another-slug",
			DisplayName: "Acme",
			KeyspaceId:  ksOf(portal.Mapping{Type: portal.MappingTypeKeyspace, ID: sharedApi.KeyAuthID.String}),
			AppId:       appOf(portal.Mapping{Type: portal.MappingTypeKeyspace, ID: sharedApi.KeyAuthID.String}),
			Enabled:     ptr.P(true),
		})
		require.Equal(t, http.StatusNotFound, res.Status,
			"ownership is checked before availability, so this is a 404: %s", res.RawBody)
		require.NotContains(t, res.RawBody, other.ID,
			"the response must not name the owning workspace")
	})
}
