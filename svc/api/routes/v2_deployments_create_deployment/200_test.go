package handler_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_create_deployment"
)

func TestImageSource(t *testing.T) {
	h := testutil.NewHarness(t)
	restate, creates := testutil.RecordingDeployRestate(t)
	route := newRoute(h, restate)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusCreated, res.Status, "expected 201, received: %s", res.RawBody)
	require.NotNil(t, res.Body)
	require.NotEmpty(t, res.Body.Data.DeploymentId)

	observed := testutil.Receive(t, creates, 10*time.Second)
	require.Equal(t, res.Body.Data.DeploymentId, observed.DeploymentID,
		"the id in the response must be the id the create ran under")
	require.Equal(t, "nginx:latest", observed.Request.GetImage().GetImage())
	require.Equal(t, setup.Project.ID, observed.Request.GetProjectId())
	require.Equal(t, setup.App.ID, observed.Request.GetAppId())
	require.Equal(t, setup.Environment.ID, observed.Request.GetEnvironmentId())
	require.Nil(t, observed.Request.GetGit(), "image source must not send git commit info")
	require.Equal(t, ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_API, observed.Request.GetTrigger().GetSource())
}

func TestImageSourceCliTrigger(t *testing.T) {
	h := testutil.NewHarness(t)
	restate, creates := testutil.RecordingDeployRestate(t)
	route := newRoute(h, restate)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	headers := authHeaders(setup.RootKey)
	headers.Set("X-Unkey-Client", "unkey-cli/1.2.3")

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusCreated, res.Status, "expected 201, received: %s", res.RawBody)
	observed := testutil.Receive(t, creates, 10*time.Second)
	require.Equal(t, ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_CLI, observed.Request.GetTrigger().GetSource())
}

func TestGitSource(t *testing.T) {
	h := testutil.NewHarness(t)
	restate, creates := testutil.RecordingDeployRestate(t)
	route := newRoute(h, restate)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	req := gitRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, openapi.DeploymentSourceGit{
		Branch:    ptr.P("main"),
		CommitSha: ptr.P("abc123"),
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusCreated, res.Status, "expected 201, received: %s", res.RawBody)
	require.NotEmpty(t, res.Body.Data.DeploymentId)

	observed := testutil.Receive(t, creates, 10*time.Second)
	require.NotNil(t, observed.Request.GetGit().GetCommit())
	require.Equal(t, "main", observed.Request.GetGit().GetCommit().Branch)
	require.Equal(t, "abc123", observed.Request.GetGit().GetCommit().CommitSha)
	require.Nil(t, observed.Request.GetImage())
}

func TestGitSourceWithFork(t *testing.T) {
	h := testutil.NewHarness(t)
	restate, creates := testutil.RecordingDeployRestate(t)
	route := newRoute(h, restate)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	req := gitRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, openapi.DeploymentSourceGit{
		CommitSha:  ptr.P("9f2c1a7"),
		Repository: ptr.P("contributor/acme-api"),
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusCreated, res.Status, "expected 201, received: %s", res.RawBody)
	observed := testutil.Receive(t, creates, 10*time.Second)
	require.NotNil(t, observed.Request.GetGit().GetCommit())
	require.Equal(t, "contributor/acme-api", observed.Request.GetGit().GetCommit().ForkRepository)
	require.Equal(t, "9f2c1a7", observed.Request.GetGit().GetCommit().CommitSha)
}

// TestRedeploy pins that the handler names the source deployment by id and
// resolves nothing itself. What it rebuilds from is the worker's to decide.
func TestRedeploy(t *testing.T) {
	h := testutil.NewHarness(t)
	restate, creates := testutil.RecordingDeployRestate(t)
	route := newRoute(h, restate)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})

	req := deploymentRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, dep.ID)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusCreated, res.Status, "expected 201, received: %s", res.RawBody)
	observed := testutil.Receive(t, creates, 10*time.Second)
	require.Equal(t, dep.ID, observed.Request.GetExistingDeployment().GetDeploymentId())
	require.Nil(t, observed.Request.GetGit())
	require.Nil(t, observed.Request.GetImage())
}

func TestSpecificEnvironmentPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	restate, creates := testutil.RecordingDeployRestate(t)
	route := newRoute(h, restate)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	rootKey := h.CreateRootKey(setup.Workspace.ID, "environment."+setup.Environment.ID+".create_deployment")

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), req)
	require.Equal(t, http.StatusCreated, res.Status, "expected 201, received: %s", res.RawBody)
	observed := testutil.Receive(t, creates, 10*time.Second)
	require.Equal(t, setup.Project.ID, observed.Request.GetProjectId())
}
