package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
)

// sessionURNFixture seeds a keyspace-mapped portal, under the given slug, whose
// keyspace has an owning API. Stage 2 needs that API to express its API-scoped
// checks, so a portal without one would fail for the wrong reason. It returns
// the project and portal ids a canonical session grant names.
func sessionURNFixture(t *testing.T, h *testutil.Harness, slug string) (string, string) {
	t.Helper()

	workspaceID := h.Resources().UserWorkspace.ID
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	portalID := insertKeyspacePortal(t, h, workspaceID, slug, api.KeyAuthID.String)

	return api.ProjectID, portalID
}

// TestCreateSessionAuthorizesCanonicalSessionURNs pins which canonical grants
// reach stage 1. The requirement is write on the portal's session sub-resource
// with a wildcard session id, because the session does not exist at check time,
// so a grant that stops at the portal itself is not enough.
func TestCreateSessionAuthorizesCanonicalSessionURNs(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	projectID, portalID := sessionURNFixture(t, h, "urn-session-portal")
	otherProjectID := uid.New(uid.ProjectPrefix)

	// Every case carries the keys:read ceiling as legacy tuples so stage 2 is
	// never what refuses. Stage 1 is what is under test.
	keyGrants := []string{"api.*.read_key", "api.*.read_api"}

	testCases := []struct {
		name       string
		resource   string
		action     string
		shouldPass bool
	}{
		{name: "sessions of this portal", resource: fmt.Sprintf("projects/%s/portals/%s/sessions/*", projectID, portalID), action: "write", shouldPass: true},
		{name: "this portal subtree", resource: fmt.Sprintf("projects/%s/portals/%s/**", projectID, portalID), action: "write", shouldPass: true},
		{name: "project subtree", resource: fmt.Sprintf("projects/%s/**", projectID), action: "write", shouldPass: true},
		{name: "workspace-wide write", resource: "**", action: "write", shouldPass: true},
		{name: "workspace-wide admin", resource: "**", action: "*", shouldPass: true},

		// The portal resource governs administering the portal. Minting a
		// session is a write on its session sub-resource, so a grant that stops
		// at the portal must not reach here.
		{name: "the portal itself", resource: fmt.Sprintf("projects/%s/portals/%s", projectID, portalID), action: "write", shouldPass: false},
		{name: "every portal in the project", resource: fmt.Sprintf("projects/%s/portals/*", projectID), action: "write", shouldPass: false},

		{name: "sessions of this portal in another project", resource: fmt.Sprintf("projects/%s/portals/%s/sessions/*", otherProjectID, portalID), action: "write", shouldPass: false},
		{name: "one concrete session", resource: fmt.Sprintf("projects/%s/portals/%s/sessions/%s", projectID, portalID, uid.New(uid.PortalSessionPrefix)), action: "write", shouldPass: false},
		{name: "reading sessions", resource: fmt.Sprintf("projects/%s/portals/%s/sessions/*", projectID, portalID), action: "read", shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			grants := append([]string{fmt.Sprintf("unkey:v1:%s:%s#%s", workspaceID, tc.resource, tc.action)}, keyGrants...)
			rootKey := h.CreateRootKey(workspaceID, grants...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			externalID := uid.New(uid.TestPrefix)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Portal:     "urn-session-portal",
				ExternalId: externalID,
				Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.KeysRead},
			})

			if tc.shouldPass {
				require.Equal(t, http.StatusOK, res.Status, "the grant must authorize minting: %s", res.RawBody)
				return
			}

			// Stage-1 denials stay masked as 404 whichever vocabulary they were
			// evaluated in.
			require.Equal(t, http.StatusNotFound, res.Status, "expected a masked denial, got: %s", res.RawBody)
			require.NotContains(t, res.RawBody, portalID, "a denial must not disclose the resolved portal id")
			require.Zero(t, countPortalSessions(t, h, workspaceID, externalID), "a denial must write no session row")
			require.Zero(t, countAuditEntriesMentioning(t, h, workspaceID, externalID), "a denial must write no audit entry")
		})
	}
}
