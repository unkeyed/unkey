package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_list_deployments"
)

func TestListWorkspaceWide(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})

	want := map[string]bool{}
	for range 3 {
		dep := h.CreateDeployment(seed.CreateDeploymentRequest{
			ID:            uid.New(uid.DeploymentPrefix),
			WorkspaceID:   setup.Workspace.ID,
			ProjectID:     setup.Project.ID,
			AppID:         setup.App.ID,
			EnvironmentID: setup.Environment.ID,
			GitBranch:     "main",
		})
		want[dep.ID] = true
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.NotNil(t, res.Body)
	require.Len(t, res.Body.Data, 3)
	require.False(t, res.Body.Pagination.HasMore)

	for _, d := range res.Body.Data {
		require.True(t, want[d.Id], "unexpected deployment %s", d.Id)
		require.Equal(t, openapi.DeploymentStatusPending, d.Status)
	}

	// Internal fields must never appear in the response body.
	for _, leaked := range []string{"k8s_name", "k8sName", "workspace_id", "workspaceId", "sentinel", "encrypted", "build_id", "buildId", "invocation", "github_deployment", "githubDeployment", "\"pk\""} {
		require.False(t, strings.Contains(res.RawBody, leaked), "response leaked internal field %q: %s", leaked, res.RawBody)
	}
}

func TestListFilterByEnvironment(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})

	// A second environment in the same app whose deployments must be excluded.
	otherEnv := h.CreateEnvironment(seed.CreateEnvironmentRequest{
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
		EnvironmentID: otherEnv.ID,
	})

	req := handler.Request{
		Project:     rid(setup.Project.Slug),
		App:         rid(setup.App.Slug),
		Environment: rid(setup.Environment.Slug),
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, 1)
	require.Equal(t, target.ID, res.Body.Data[0].Id)
}

func TestListFilterByProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})

	target := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})

	// A second project in the same workspace whose deployments must be excluded.
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: setup.Workspace.ID,
		Name:        "other",
		Slug:        "other-project",
	})
	otherApp := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     otherProject.ID,
		Name:          "other",
		Slug:          "other-app",
		DefaultBranch: "main",
	})
	otherEnv := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   otherProject.ID,
		AppID:       otherApp.ID,
		Slug:        "production",
		Description: "other project production",
	})
	h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     otherProject.ID,
		AppID:         otherApp.ID,
		EnvironmentID: otherEnv.ID,
	})

	// Resolve the project by id to exercise the id-seek branch of the resolver.
	req := handler.Request{Project: rid(setup.Project.ID)}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, 1)
	require.Equal(t, target.ID, res.Body.Data[0].Id)
}

func TestListFilterByApp(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})

	// A second app in the same project whose deployments must be excluded.
	otherApp := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		Name:          "other",
		Slug:          "other-app",
		DefaultBranch: "main",
	})
	otherEnv := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   setup.Project.ID,
		AppID:       otherApp.ID,
		Slug:        "production",
		Description: "other app production",
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
		AppID:         otherApp.ID,
		EnvironmentID: otherEnv.ID,
	})

	// project + app filter, no environment: app_id resolves, environment_id is NULL.
	req := handler.Request{
		Project: rid(setup.Project.Slug),
		App:     rid(setup.App.Slug),
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, 1)
	require.Equal(t, target.ID, res.Body.Data[0].Id)
}

func TestListFilterByStatus(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})

	// Seeded deployments are all in status "pending".
	h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})

	pending := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Status: ptr.P([]openapi.DeploymentStatus{openapi.DeploymentStatusPending}),
	})
	require.Equal(t, http.StatusOK, pending.Status, "expected 200, received: %s", pending.RawBody)
	require.Len(t, pending.Body.Data, 1)

	failed := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Status: ptr.P([]openapi.DeploymentStatus{openapi.DeploymentStatusFailed}),
	})
	require.Equal(t, http.StatusOK, failed.Status, "expected 200, received: %s", failed.RawBody)
	require.Empty(t, failed.Body.Data)
}

func TestListEmptyWorkspace(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Empty(t, res.Body.Data)
	require.False(t, res.Body.Pagination.HasMore)
}
