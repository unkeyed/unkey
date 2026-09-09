package handler_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_portal"
)

// TestCreatePortalAuthorizesAdminURNAndLegacyTuple guarantees the dashboard
// admin grant and existing root-key permission can create a portal.
func TestCreatePortalAuthorizesAdminURNAndLegacyTuple(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	testCases := map[string][]string{
		"workspace-wide admin URN": {
			fmt.Sprintf("unkey:v1:%s:**#*", workspace.ID),
		},
		"legacy tuples": append([]string{"portal.*.create_portal"}, targetReadGrants...),
	}

	i := 0
	for name, grants := range testCases {
		i++
		t.Run(name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, grants...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Slug:        fmt.Sprintf("urn-portal-%d", i),
				DisplayName: "Acme",
				KeyspaceId:  ksOf(keyspaceMapping(t, h, workspace.ID)),
				AppId:       appOf(keyspaceMapping(t, h, workspace.ID)),
				Enabled:     ptr.P(true),
			})
			require.Equal(t, http.StatusOK, res.Status, "the grant must authorize portal creation: %s", res.RawBody)
		})
	}
}

// keyspaceMappingWithProject seeds an api and returns its keyspace mapping
// beside the project that owns it, which is what a canonical portal grant names.
func keyspaceMappingWithProject(t *testing.T, h *testutil.Harness, workspaceID string) (portal.Mapping, string) {
	t.Helper()

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	return portal.Mapping{Type: portal.MappingTypeKeyspace, ID: api.KeyAuthID.String}, api.ProjectID
}

// TestCreatePortalAuthorizesCanonicalPortalURNs pins which canonical grants
// reach this route. The portal id is minted after the check, so the grant has to
// cover a wildcard portal under the project that owns the mapping: a grant
// naming one concrete portal is not enough.
func TestCreatePortalAuthorizesCanonicalPortalURNs(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	testCases := []struct {
		name       string
		resource   func(projectID string) string
		action     string
		shouldPass bool
	}{
		{
			name:       "every portal in the project",
			resource:   func(p string) string { return fmt.Sprintf("projects/%s/portals/*", p) },
			action:     "write",
			shouldPass: true,
		},
		{
			name:       "project subtree",
			resource:   func(p string) string { return fmt.Sprintf("projects/%s/**", p) },
			action:     "write",
			shouldPass: true,
		},
		{
			name:       "workspace-wide write",
			resource:   func(string) string { return "**" },
			action:     "write",
			shouldPass: true,
		},
		{
			name:       "workspace-wide admin",
			resource:   func(string) string { return "**" },
			action:     "*",
			shouldPass: true,
		},
		{
			name: "every portal in another project",
			resource: func(string) string {
				return fmt.Sprintf("projects/%s/portals/*", uid.New(uid.ProjectPrefix))
			},
			action:     "write",
			shouldPass: false,
		},
		{
			name: "one concrete portal",
			resource: func(p string) string {
				return fmt.Sprintf("projects/%s/portals/%s", p, uid.New(uid.PortalPrefix))
			},
			action:     "write",
			shouldPass: false,
		},
		{
			name:       "workspace-wide read",
			resource:   func(string) string { return "**" },
			action:     "read",
			shouldPass: false,
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// A mapping per case, so a passing case cannot make the next one report a
			// conflict instead of an authorization result.
			mapping, projectID := keyspaceMappingWithProject(t, h, workspace.ID)
			rootKey := h.CreateRootKey(workspace.ID, append([]string{
				fmt.Sprintf("unkey:v1:%s:%s#%s", workspace.ID, tc.resource(projectID), tc.action),
			}, targetReadGrants...)...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			before := countPortals(t, h, workspace.ID)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Slug:        fmt.Sprintf("urn-grants-%d", i),
				DisplayName: "Acme",
				KeyspaceId:  ksOf(mapping),
				AppId:       appOf(mapping),
				Enabled:     ptr.P(true),
			})

			if tc.shouldPass {
				require.Equal(t, http.StatusOK, res.Status,
					"%s must authorize the create: %s", tc.name, res.RawBody)
				return
			}

			require.Equal(t, http.StatusForbidden, res.Status,
				"expected 403 for %s, got: %s", tc.name, res.RawBody)
			// Unlike the masked routes, create names the grant the caller is short
			// of, project id included. That project is in the caller's own workspace
			// and behind the mapping the caller itself named, so the alternative is a
			// 403 an operator cannot act on.
			require.Contains(t, res.RawBody,
				fmt.Sprintf("projects/%s/portals/*#write", projectID),
				"a denial must name the grant the caller lacks")
			require.Equal(t, before, countPortals(t, h, workspace.ID),
				"a denied request must not write a portal")
		})
	}
}

// A portal for an app is authorized under the app's project, not the workspace's
// default one, so the grant a caller needs follows the app.
func TestCreatePortalAuthorizesAppMappingUnderItsProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "portal-app",
		Slug:        "portal-app",
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		Name:        "portal-app",
		Slug:        "portal-app",
	})
	mapping := portal.Mapping{Type: portal.MappingTypeApp, ID: app.ID}

	rootKey := h.CreateRootKey(workspace.ID, append([]string{
		fmt.Sprintf("unkey:v1:%s:projects/%s/portals/*#write", workspace.ID, project.ID),
	}, targetReadGrants...)...)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Slug:        "app-project-portal",
		DisplayName: "Acme",
		KeyspaceId:  ksOf(mapping),
		AppId:       appOf(mapping),
		Enabled:     ptr.P(true),
	})
	require.Equal(t, http.StatusOK, res.Status,
		"a grant on the app's project must authorize the create: %s", res.RawBody)
}

// The resource-claim check is unscoped by workspace and says that an app or
// keyspace already backs a portal, so it must sit behind the grant: otherwise
// any authenticated caller could probe another tenant's wiring with it.
func TestCreatePortalDeniesBeforeTheResourceClaimSignal(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	mapping, projectID := keyspaceMappingWithProject(t, h, workspace.ID)

	// Another workspace already holds the portal for this caller's keyspace. The
	// unique key on the mapping columns spans the table, so the claim is visible
	// to the check no matter who owns it.
	other := h.CreateWorkspace()
	h.CreatePortal(seed.CreatePortalRequest{
		WorkspaceID: other.ID,
		ProjectID:   h.CreateApi(seed.CreateApiRequest{WorkspaceID: other.ID}).ProjectID,
		Slug:        "squatted",
		DisplayName: "squatted",
		KeyAuthID:   sql.NullString{String: mapping.ID, Valid: true},
		Enabled:     true,
	})

	req := handler.Request{
		Slug:        "claim-probe",
		DisplayName: "Acme",
		KeyspaceId:  ksOf(mapping),
		AppId:       appOf(mapping),
		Enabled:     ptr.P(true),
	}

	unauthorized := h.CreateRootKey(workspace.ID, targetReadGrants...)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", unauthorized)},
	}, req)
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, got: %s", res.RawBody)
	require.NotContains(t, res.RawBody, "already has a portal",
		"a caller without a portal grant must not learn the keyspace is claimed")

	// The same request behind the grant reaches the check, which is what makes the
	// assertion above about ordering rather than about the fixture.
	authorized := h.CreateRootKey(workspace.ID, append([]string{
		fmt.Sprintf("unkey:v1:%s:projects/%s/portals/*#write", workspace.ID, projectID),
	}, targetReadGrants...)...)
	res = testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", authorized)},
	}, req)
	require.Equal(t, http.StatusConflict, res.Status, "expected 409, got: %s", res.RawBody)
	require.Contains(t, res.RawBody, "already has a portal")
}

// Authorization needs the project behind the mapping, so a caller naming a
// mapping it does not own is answered by the mapping lookup before its missing
// portal grant is ever considered. Pinned because it is the one denial this
// route reports as a 404.
func TestCreatePortalReportsAnUnownedMappingBeforeADeniedGrant(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	other := h.CreateWorkspace()
	foreign, _ := keyspaceMappingWithProject(t, h, other.ID)

	rootKey := h.CreateRootKey(workspace.ID, targetReadGrants...)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{
		Slug:        "foreign-mapping",
		DisplayName: "Acme",
		KeyspaceId:  ksOf(foreign),
		AppId:       appOf(foreign),
		Enabled:     ptr.P(true),
	})
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, got: %s", res.RawBody)
}
