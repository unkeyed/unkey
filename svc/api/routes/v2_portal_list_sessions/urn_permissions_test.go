package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_list_sessions"
)

// Listing takes portal read. The session write grant that revokes does not
// imply it.
func TestListSessionsAuthorizesPortalRead(t *testing.T) {
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
			permission: func(_, _ string) string { return "portal.*.read_portal" },
			shouldPass: true,
		},
		{
			name:       "legacy tuple for this portal",
			permission: func(_, id string) string { return fmt.Sprintf("portal.%s.read_portal", id) },
			shouldPass: true,
		},
		{
			name: "read on this portal",
			permission: func(p, id string) string {
				return fmt.Sprintf("unkey:v1:%s:projects/%s/portals/%s#read", workspace.ID, p, id)
			},
			shouldPass: true,
		},
		{
			name: "read on every portal",
			permission: func(_, _ string) string {
				return fmt.Sprintf("unkey:v1:%s:projects/*/portals/*#read", workspace.ID)
			},
			shouldPass: true,
		},
		{
			name:       "workspace admin",
			permission: func(_, _ string) string { return fmt.Sprintf("unkey:v1:%s:**#*", workspace.ID) },
			shouldPass: true,
		},
		{
			name: "write on this portal's sessions",
			permission: func(p, id string) string {
				return fmt.Sprintf("unkey:v1:%s:projects/%s/portals/%s/sessions/*#write", workspace.ID, p, id)
			},
			shouldPass: false,
		},
		{
			name: "read on another portal",
			permission: func(p, _ string) string {
				return fmt.Sprintf("unkey:v1:%s:projects/%s/portals/%s#read", workspace.ID, p, uid.New(uid.PortalPrefix))
			},
			shouldPass: false,
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mapping, projectID := keyspaceMapping(t, h, workspace.ID)
			slug := fmt.Sprintf("list-urn-%d", i)
			stored := h.SeedPortal(t, workspace.ID, slug, slug, mapping, nil, nil)
			insertSession(t, h, stored.ID, workspace.ID, active(h, "user_1"))

			rootKey := h.CreateRootKey(workspace.ID, tc.permission(projectID, stored.ID))

			if tc.shouldPass {
				res := testutil.CallRoute[handler.Request, handler.Response](h, route, headersFor(rootKey), request(stored.Slug))
				require.Equal(t, http.StatusOK, res.Status, "%s must authorize the list: %s", tc.name, res.RawBody)
				require.Equal(t, []string{"user_1"}, externalIDs(res.Body))
				return
			}

			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headersFor(rootKey), request(stored.Slug))
			require.Equal(t, http.StatusNotFound, res.Status, "expected a masked 404 for %s, got: %s", tc.name, res.RawBody)
		})
	}
}
