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
// environment deployment collection permission cannot expose sibling environments.
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

// TestListDeployments_URNDoesNotBypassAncestryOrWorkspace guarantees URN
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
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		})
	}
}

// TestListDeployments_URNRejectionsMatchLegacy guarantees a URN grant on every
// deployment gets the same 400 and 404 answers a legacy wildcard key gets
func TestListDeployments_URNRejectionsMatchLegacy(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	rootKey := h.CreateRootKey(setup.Workspace.ID, deploymentPermission(setup.Workspace.ID, "*", "*", "*", "*", "read"))

	for _, tc := range []struct {
		name   string
		req    handler.Request
		status int
	}{
		{name: "app without project", req: handler.Request{App: rid(setup.App.Slug)}, status: http.StatusBadRequest},
		{name: "environment without app", req: handler.Request{Project: rid(setup.Project.Slug), Environment: rid(setup.Environment.Slug)}, status: http.StatusBadRequest},
		{name: "branch without app", req: handler.Request{Project: rid(setup.Project.Slug), Branch: &[]string{"main"}}, status: http.StatusBadRequest},
		{name: "unknown project", req: handler.Request{Project: rid(uid.New(uid.ProjectPrefix))}, status: http.StatusNotFound},
		{name: "unknown app", req: handler.Request{Project: rid(setup.Project.Slug), App: rid(uid.New(uid.AppPrefix))}, status: http.StatusNotFound},
		{name: "unknown environment", req: handler.Request{Project: rid(setup.Project.Slug), App: rid(setup.App.Slug), Environment: rid(uid.New(uid.EnvironmentPrefix))}, status: http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, map[string]any](h, route, authHeaders(rootKey), tc.req)
			require.Equal(t, tc.status, res.Status, "received: %s", res.RawBody)
		})
	}
}

// TestListDeployments_ScopedURNGetsRequestErrors guarantees a key that can read
// the requested scope gets the request error, not a 403 for permissions it
// does not need
func TestListDeployments_ScopedURNGetsRequestErrors(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	rootKey := h.CreateRootKey(setup.Workspace.ID, deploymentPermission(setup.Workspace.ID, setup.Project.ID, "*", "*", "*", "read"))

	for name, req := range map[string]handler.Request{
		"app without project":     {App: rid(setup.App.ID)},
		"environment without app": {Project: rid(setup.Project.ID), Environment: rid(setup.Environment.ID)},
		"branch without app":      {Project: rid(setup.Project.ID), Branch: &[]string{"main"}},
		"empty time range":        {Project: rid(setup.Project.ID), StartTime: new(int64(2)), EndTime: new(int64(1))},
	} {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, map[string]any](h, route, authHeaders(rootKey), req)
			require.Equal(t, http.StatusBadRequest, res.Status, "received: %s", res.RawBody)
		})
	}
}

// TestListDeployments_ScopedURNCannotProbe guarantees a key scoped below the
// requested collection gets 403 for a missing resource and for an existing one
// it cannot read, so neither the status nor the body reveals existence
func TestListDeployments_ScopedURNCannotProbe(t *testing.T) {
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
		ProjectID:   setup.Project.ID,
		Name:        "other app",
		Slug:        "other-app",
	})
	otherEnvironment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   setup.Project.ID,
		AppID:       setup.App.ID,
		Slug:        "preview",
		Description: "preview environment",
	})
	rootKey := h.CreateRootKey(setup.Workspace.ID, deploymentPermission(
		setup.Workspace.ID, setup.Project.ID, setup.App.ID, setup.Environment.ID, "*", "read",
	))

	for _, tc := range []struct {
		name     string
		existing handler.Request
		missing  handler.Request
	}{
		{
			name:     "project",
			existing: handler.Request{Project: rid(otherProject.Slug)},
			missing:  handler.Request{Project: rid("missing-project")},
		},
		{
			name:     "app",
			existing: handler.Request{Project: rid(setup.Project.ID), App: rid(otherApp.Slug)},
			missing:  handler.Request{Project: rid(setup.Project.ID), App: rid("missing-app")},
		},
		{
			name:     "environment",
			existing: handler.Request{Project: rid(setup.Project.ID), App: rid(setup.App.ID), Environment: rid(otherEnvironment.Slug)},
			missing:  handler.Request{Project: rid(setup.Project.ID), App: rid(setup.App.ID), Environment: rid("missing-environment")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			existing := testutil.CallRoute[handler.Request, map[string]any](h, route, authHeaders(rootKey), tc.existing)
			missing := testutil.CallRoute[handler.Request, map[string]any](h, route, authHeaders(rootKey), tc.missing)
			require.Equal(t, http.StatusForbidden, existing.Status, "existing: %s", existing.RawBody)
			require.Equal(t, http.StatusForbidden, missing.Status, "missing: %s", missing.RawBody)
			require.Equal(t, (*existing.Body)["error"], (*missing.Body)["error"], "403 bodies must not differ by existence")
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

// deploymentPermission returns a URN deployment permission for a test permission.
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
