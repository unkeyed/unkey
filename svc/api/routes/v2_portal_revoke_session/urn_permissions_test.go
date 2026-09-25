package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_revoke_session"
)

// Pins which grants reach this route: the same ones that mint a session for the
// portal. The workspace admin grant is the dashboard's, so this is also what lets
// a dashboard caller revoke.
func TestRevokeSessionAuthorizesMintingGrants(t *testing.T) {
	h := testutil.NewHarness(t)
	route := registerRoute(h)
	workspace := h.Resources().UserWorkspace

	testCases := []struct {
		name       string
		permission func(projectID, portalID string) string
		shouldPass bool
	}{
		{
			name:       "legacy wildcard tuple",
			permission: func(_, _ string) string { return "portal.*.create_portal_session" },
			shouldPass: true,
		},
		{
			name:       "legacy tuple for this portal",
			permission: func(_, id string) string { return fmt.Sprintf("portal.%s.create_portal_session", id) },
			shouldPass: true,
		},
		{
			name: "sessions of this portal",
			permission: func(p, id string) string {
				return fmt.Sprintf("unkey:v1:%s:projects/%s/portals/%s/sessions/*#write", workspace.ID, p, id)
			},
			shouldPass: true,
		},
		{
			name:       "workspace admin",
			permission: func(_, _ string) string { return fmt.Sprintf("unkey:v1:%s:**#*", workspace.ID) },
			shouldPass: true,
		},
		{
			name: "this portal itself",
			permission: func(p, id string) string {
				return fmt.Sprintf("unkey:v1:%s:projects/%s/portals/%s#*", workspace.ID, p, id)
			},
			shouldPass: false,
		},
		{
			name: "read on this portal's sessions",
			permission: func(p, id string) string {
				return fmt.Sprintf("unkey:v1:%s:projects/%s/portals/%s/sessions/*#read", workspace.ID, p, id)
			},
			shouldPass: false,
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mapping, projectID := keyspaceMapping(t, h, workspace.ID)
			slug := fmt.Sprintf("revoke-urn-%d", i)
			stored := h.SeedPortal(t, workspace.ID, slug, slug, mapping, nil, nil)
			h.CreatePortalSessionForPortal(stored.ID, workspace.ID, "user_1", []string{mapping.ID}, []string{"keys:read"})

			rootKey := h.CreateRootKey(workspace.ID, tc.permission(projectID, stored.ID))

			if tc.shouldPass {
				res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), request(stored.Slug, "user_1"))
				require.Equal(t, http.StatusOK, res.Status, "%s must authorize the revoke: %s", tc.name, res.RawBody)
				require.Equal(t, int64(1), res.Body.Data.SessionsRevoked)
				return
			}

			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headersFor(rootKey), request(stored.Slug, "user_1"))
			require.Equal(t, http.StatusNotFound, res.Status, "expected a masked 404 for %s, got: %s", tc.name, res.RawBody)
			require.Equal(t, 1, sessionsFor(t, h, stored.ID, "user_1", "revoked_at IS NULL"),
				"a denied request must not revoke")
		})
	}
}
