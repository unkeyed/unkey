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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_list_deployments"
)

// TestListDeployments_AuthorizesEnvironmentCollectionURN guarantees an exact
// environment deployment collection grant cannot expose sibling environments.
func TestListDeployments_AuthorizesEnvironmentCollectionURN(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	otherEnvironment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   setup.Project.ID,
		AppID:       setup.App.ID,
		Slug:        "preview",
		Description: "preview environment",
	})
	target := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})
	h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: otherEnvironment.ID,
	})
	rootKey := h.CreateRootKey(setup.Workspace.ID, fmt.Sprintf(
		"unkey:v1:%s:projects/%s/apps/%s/environments/%s/deployments/*#read",
		setup.Workspace.ID,
		setup.Project.ID,
		setup.App.ID,
		setup.Environment.ID,
	))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Project:     rid(setup.Project.Slug),
		App:         rid(setup.App.Slug),
		Environment: rid(setup.Environment.Slug),
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, 1)
	require.Equal(t, target.ID, res.Body.Data[0].Id)
}

// TestListDeployments_AuthorizesFilteredCollectionURNs guarantees project and
// app filters require only their corresponding deployment collections.
func TestListDeployments_AuthorizesFilteredCollectionURNs(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	otherApp := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   setup.Project.ID,
		Name:        "other app",
		Slug:        "other-app",
	})
	otherEnvironment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   setup.Project.ID,
		AppID:       otherApp.ID,
		Slug:        "production",
		Description: "other app environment",
	})
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: setup.Workspace.ID,
		Name:        "other project",
		Slug:        "other-project",
	})
	otherProjectApp := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   otherProject.ID,
		Name:        "other project app",
		Slug:        "other-project-app",
	})
	otherProjectEnvironment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   otherProject.ID,
		AppID:       otherProjectApp.ID,
		Slug:        "production",
		Description: "other project environment",
	})

	appDeployment := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})
	otherAppDeployment := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         otherApp.ID,
		EnvironmentID: otherEnvironment.ID,
	})
	h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     otherProject.ID,
		AppID:         otherProjectApp.ID,
		EnvironmentID: otherProjectEnvironment.ID,
	})

	t.Run("project", func(t *testing.T) {
		rootKey := h.CreateRootKey(setup.Workspace.ID, deploymentPermission(
			setup.Workspace.ID,
			setup.Project.ID,
			"*",
			"*",
			"*",
			"read",
		))
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
			Project: rid(setup.Project.Slug),
		})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.ElementsMatch(t, []string{appDeployment.ID, otherAppDeployment.ID}, deploymentIDs(res.Body.Data))
	})

	t.Run("app", func(t *testing.T) {
		rootKey := h.CreateRootKey(setup.Workspace.ID, deploymentPermission(
			setup.Workspace.ID,
			setup.Project.ID,
			setup.App.ID,
			"*",
			"*",
			"read",
		))
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
			Project: rid(setup.Project.Slug),
			App:     rid(setup.App.Slug),
		})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.Equal(t, []string{appDeployment.ID}, deploymentIDs(res.Body.Data))
	})
}

// TestListDeployments_AuthorizesWorkspaceCollectionURN guarantees an
// unfiltered request requires coverage across every deployment hierarchy.
func TestListDeployments_AuthorizesWorkspaceCollectionURN(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: setup.Workspace.ID,
		Name:        "other project",
		Slug:        "other-project",
	})
	otherApp := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   otherProject.ID,
		Name:        "other app",
		Slug:        "other-app",
	})
	otherEnvironment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   otherProject.ID,
		AppID:       otherApp.ID,
		Slug:        "production",
		Description: "other environment",
	})
	first := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})
	second := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     otherProject.ID,
		AppID:         otherApp.ID,
		EnvironmentID: otherEnvironment.ID,
	})
	rootKey := h.CreateRootKey(setup.Workspace.ID, deploymentPermission(
		setup.Workspace.ID,
		"*",
		"*",
		"*",
		"*",
		"read",
	))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.ElementsMatch(t, []string{first.ID, second.ID}, deploymentIDs(res.Body.Data))
}

// TestListDeployments_AuthorizesEmptyURNCollection guarantees authorization is
// evaluated against the requested collection even when it contains no rows.
func TestListDeployments_AuthorizesEmptyURNCollection(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	rootKey := h.CreateRootKey(setup.Workspace.ID, deploymentPermission(
		setup.Workspace.ID,
		setup.Project.ID,
		setup.App.ID,
		setup.Environment.ID,
		"*",
		"read",
	))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Project:     rid(setup.Project.ID),
		App:         rid(setup.App.ID),
		Environment: rid(setup.Environment.ID),
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Empty(t, res.Body.Data)
}

// TestListDeployments_URNDoesNotBypassAncestryOrWorkspace guarantees canonical
// permissions are built only from resources resolved inside the request scope.
func TestListDeployments_URNDoesNotBypassAncestryOrWorkspace(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: setup.Workspace.ID,
		Name:        "other project",
		Slug:        "other-project",
	})
	otherApp := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   otherProject.ID,
		Name:        "other app",
		Slug:        "other-app",
	})
	foreign := h.CreateTestDeploymentSetup()
	rootKey := h.CreateRootKey(setup.Workspace.ID, deploymentPermission(
		setup.Workspace.ID,
		"*",
		"*",
		"*",
		"*",
		"read",
	))

	requests := map[string]handler.Request{
		"app outside project": {
			Project: rid(setup.Project.ID),
			App:     rid(otherApp.ID),
		},
		"project outside workspace": {
			Project: rid(foreign.Project.ID),
		},
	}
	for name, req := range requests {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), req)
			require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
		})
	}
}

// TestListDeployments_RejectsWrongURNAction guarantees deployment collection
// writes cannot authorize reads.
func TestListDeployments_RejectsWrongURNAction(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	rootKey := h.CreateRootKey(setup.Workspace.ID, deploymentPermission(
		setup.Workspace.ID,
		setup.Project.ID,
		setup.App.ID,
		setup.Environment.ID,
		"*",
		"write",
	))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Project:     rid(setup.Project.ID),
		App:         rid(setup.App.ID),
		Environment: rid(setup.Environment.ID),
	})
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
}

// TestListDeployments_NarrowURNsCannotAuthorizeWiderCollections guarantees the
// permission must cover the complete collection selected by the filters.
func TestListDeployments_NarrowURNsCannotAuthorizeWiderCollections(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	deployment := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})
	tests := []struct {
		name       string
		permission string
		req        handler.Request
	}{
		{
			name: "concrete deployment cannot list environment",
			permission: deploymentPermission(setup.Workspace.ID, setup.Project.ID, setup.App.ID, setup.Environment.ID,
				deployment.ID, "read"),
			req: handler.Request{
				Project:     rid(setup.Project.ID),
				App:         rid(setup.App.ID),
				Environment: rid(setup.Environment.ID),
			},
		},
		{
			name: "environment cannot list app",
			permission: deploymentPermission(setup.Workspace.ID, setup.Project.ID, setup.App.ID, setup.Environment.ID,
				"*", "read"),
			req: handler.Request{
				Project: rid(setup.Project.ID),
				App:     rid(setup.App.ID),
			},
		},
		{
			name: "app cannot list project",
			permission: deploymentPermission(setup.Workspace.ID, setup.Project.ID, setup.App.ID, "*",
				"*", "read"),
			req: handler.Request{Project: rid(setup.Project.ID)},
		},
		{
			name: "project cannot list workspace",
			permission: deploymentPermission(setup.Workspace.ID, setup.Project.ID, "*", "*",
				"*", "read"),
			req: handler.Request{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(setup.Workspace.ID, test.permission)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), test.req)
			require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
		})
	}
}

// deploymentPermission returns a canonical deployment permission for a test grant.
func deploymentPermission(workspaceID, projectID, appID, environmentID, deploymentID, action string) string {
	return fmt.Sprintf(
		"unkey:v1:%s:projects/%s/apps/%s/environments/%s/deployments/%s#%s",
		workspaceID,
		projectID,
		appID,
		environmentID,
		deploymentID,
		action,
	)
}

// deploymentIDs extracts response IDs for collection comparisons.
func deploymentIDs(deployments []openapi.Deployment) []string {
	ids := make([]string, len(deployments))
	for i, deployment := range deployments {
		ids[i] = deployment.Id
	}
	return ids
}
