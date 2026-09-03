package handler_test

import (
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

// TestCreatePortalAuthorizesProjectPortalURN guarantees portal creation uses
// the project that owns the requested mapping.
func TestCreatePortalAuthorizesProjectPortalURN(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspaceID,
	})
	mapping := portal.Mapping{Type: portal.MappingTypeKeyspace, ID: api.KeyAuthID.String}
	rootKey := h.CreateRootKey(workspaceID,
		append([]string{
			fmt.Sprintf("unkey:v1:%s:projects/%s/portals/*#write", workspaceID, api.ProjectID),
		}, targetReadGrants...)...,
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Slug:        "project-urn-portal",
		DisplayName: "Acme",
		KeyspaceId:  ksOf(mapping),
		AppId:       appOf(mapping),
		Enabled:     ptr.P(true),
	})
	require.Equal(t, http.StatusOK, res.Status, "the project portal grant must authorize creation: %s", res.RawBody)

	wrongProjectKey := h.CreateRootKey(workspaceID,
		append([]string{
			fmt.Sprintf("unkey:v1:%s:projects/%s/portals/*#write", workspaceID, uid.New(uid.ProjectPrefix)),
		}, targetReadGrants...)...,
	)
	res = testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", wrongProjectKey)},
	}, handler.Request{
		Slug:        "wrong-project-urn-portal",
		DisplayName: "Acme",
		KeyspaceId:  ksOf(mapping),
		AppId:       appOf(mapping),
	})
	require.Equal(t, http.StatusForbidden, res.Status, "a grant for another project must not authorize creation: %s", res.RawBody)
}

// TestCreatePortalAuthorizesProjectPortalURNForApp guarantees app mappings use
// the app project and accept the canonical app read grant.
func TestCreatePortalAuthorizesProjectPortalURNForApp(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, Clock: h.Clock}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspaceID,
		Name:        "portal-app",
		Slug:        "portal-app",
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		Name:          "portal-app",
		Slug:          "portal-app",
		DefaultBranch: "main",
	})
	mapping := portal.Mapping{Type: portal.MappingTypeApp, ID: app.ID}
	rootKey := h.CreateRootKey(workspaceID,
		fmt.Sprintf("unkey:v1:%s:projects/%s/portals/*#write", workspaceID, project.ID),
		fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s#read", workspaceID, project.ID, app.ID),
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Slug:        "project-app-urn-portal",
		DisplayName: "Acme",
		KeyspaceId:  ksOf(mapping),
		AppId:       appOf(mapping),
	})
	require.Equal(t, http.StatusOK, res.Status, "the project portal grant must authorize app portal creation: %s", res.RawBody)
}
