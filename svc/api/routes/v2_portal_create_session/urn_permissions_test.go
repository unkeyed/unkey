package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	createportal "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_portal"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
	deleteportal "github.com/unkeyed/unkey/svc/api/routes/v2_portal_delete_portal"
	getportal "github.com/unkeyed/unkey/svc/api/routes/v2_portal_get_portal"
	updateportal "github.com/unkeyed/unkey/svc/api/routes/v2_portal_update_portal"
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

// TestCreateSessionAuthorizesCanonicalSessionURNs guarantees stage 1 requires
// write on the portal's session sub-resource with a wildcard session id, so a
// grant stopping at the portal itself is not enough. Every case authenticates
// with a root key; credential_test.go covers the other credential types.
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

	// Legacy tuples for the keys:read ceiling, so stage 2 never refuses and
	// stage 1 is what is under test.
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

		// A grant on the portal itself administers it; minting is a write on the
		// session sub-resource.
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

			// Stage-1 denials stay masked as 404 in either vocabulary.
			require.Equal(t, http.StatusNotFound, res.Status, "expected a masked denial, got: %s", res.RawBody)
			require.NotContains(t, res.RawBody, portalID, "a denial must not disclose the resolved portal id")
			require.Zero(t, countPortalSessions(t, h, workspaceID, externalID), "a denial must write no session row")
			require.Zero(t, countAuditEntriesMentioning(t, h, workspaceID, externalID), "a denial must write no audit entry")
		})
	}
}

// projectWideGrants is a trailing-wildcard grant at the project level. Actions
// are flat with no implication between them, so the set carries one per action
// the portal routes name.
func projectWideGrants(workspaceID, projectID string) []string {
	return []string{
		fmt.Sprintf("unkey:v1:%s:projects/%s/**#read", workspaceID, projectID),
		fmt.Sprintf("unkey:v1:%s:projects/%s/**#write", workspaceID, projectID),
		fmt.Sprintf("unkey:v1:%s:projects/%s/**#delete", workspaceID, projectID),
	}
}

// TestProjectWideGrantsDeliberatelyReachPortals accepts a consequence of putting
// portals in the URN catalog: projects/{id}/** now covers portal resources and,
// through the session sub-resource, minting. Nothing had to name portals for
// that, since V1.Covers strips a trailing "**" and prefix-matches.
//
// Pinned so that narrowing Covers or excluding portals later fails here and says
// it is revoking grants customers hold, rather than doing it silently.
func TestProjectWideGrantsDeliberatelyReachPortals(t *testing.T) {
	h := testutil.NewHarness(t)

	create := &createportal.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	get := &getportal.Handler{DB: h.DB}
	update := &updateportal.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	remove := &deleteportal.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	mint := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
		Clock:         h.Clock,
	}
	for _, route := range []zen.Route{create, get, update, remove, mint} {
		h.Register(route)
	}

	workspaceID := h.Resources().UserWorkspace.ID
	projectID := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID}).ProjectID

	testCases := []struct {
		name           string
		grantedProject string
		// Create names the grant the caller is short of instead of masking the
		// denial as a 404.
		createStatus int
		otherStatus  int
	}{
		{
			name:           "the project owning the portals",
			grantedProject: projectID,
			createStatus:   http.StatusOK,
			otherStatus:    http.StatusOK,
		},
		{
			name:           "another project",
			grantedProject: uid.New(uid.ProjectPrefix),
			createStatus:   http.StatusForbidden,
			otherStatus:    http.StatusNotFound,
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// A keyspace backs at most one portal, so the create leg needs its
			// own separate from the one already behind a seeded portal.
			target := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
			existing := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
			slug := fmt.Sprintf("project-wide-%d", i)
			stored := insertKeyspacePortal(t, h, workspaceID, slug, existing.KeyAuthID.String)

			rootKey := h.CreateRootKey(workspaceID, projectWideGrants(workspaceID, tc.grantedProject)...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			createRes := testutil.CallRoute[createportal.Request, createportal.Response](h, create, headers, createportal.Request{
				Slug:        fmt.Sprintf("project-wide-created-%d", i),
				DisplayName: "Acme",
				KeyspaceId:  ptr.P(openapi.PortalKeyspaceId(target.KeyAuthID.String)),
				Enabled:     ptr.P(true),
			})
			require.Equal(t, tc.createStatus, createRes.Status, "create: %s", createRes.RawBody)

			getRes := testutil.CallRoute[getportal.Request, getportal.Response](h, get, headers, getportal.Request{
				Portal: ptr.P(openapi.ResourceIdentifier(stored)),
			})
			require.Equal(t, tc.otherStatus, getRes.Status, "get: %s", getRes.RawBody)

			updateRes := testutil.CallRoute[updateportal.Request, updateportal.Response](h, update, headers, updateportal.Request{
				Portal:      openapi.ResourceIdentifier(stored),
				DisplayName: ptr.P("Renamed"),
			})
			require.Equal(t, tc.otherStatus, updateRes.Status, "update: %s", updateRes.RawBody)

			// The credential restriction refuses every other type before any
			// grant is read.
			externalID := uid.New(uid.TestPrefix)
			mintRes := testutil.CallRoute[handler.Request, handler.Response](h, mint, headers, handler.Request{
				Portal:     slug,
				ExternalId: externalID,
				Scopes:     []openapi.V2PortalCreateSessionRequestBodyScopes{openapi.KeysRead},
			})
			require.Equal(t, tc.otherStatus, mintRes.Status, "mint: %s", mintRes.RawBody)

			deleteRes := testutil.CallRoute[deleteportal.Request, deleteportal.Response](h, remove, headers, deleteportal.Request{
				Portal: openapi.ResourceIdentifier(stored),
			})
			require.Equal(t, tc.otherStatus, deleteRes.Status, "delete: %s", deleteRes.RawBody)
		})
	}
}
