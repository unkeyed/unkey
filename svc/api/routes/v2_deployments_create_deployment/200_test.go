package handler_test

import (
	"net/http"
	"testing"

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
	capture := &ctrlCapture{}
	route := newRoute(h, capture)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	seedDeployableRegion(t, h, setup)

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusCreated, res.Status, "expected 201, received: %s", res.RawBody)
	require.NotNil(t, res.Body)
	require.NotEmpty(t, res.Body.Data.DeploymentId)

	require.True(t, capture.called)
	require.Equal(t, "nginx:latest", capture.req.DockerImage)
	require.Equal(t, setup.Project.ID, capture.req.ProjectId)
	require.Equal(t, setup.App.ID, capture.req.AppId)
	require.Equal(t, setup.Environment.Slug, capture.req.EnvironmentSlug)
	require.Nil(t, capture.req.GetGitCommit(), "image source must not send git commit info")
	require.Equal(t, ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_API, capture.req.Trigger)
}

func TestImageSourceCliTrigger(t *testing.T) {
	h := testutil.NewHarness(t)
	capture := &ctrlCapture{}
	route := newRoute(h, capture)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	seedDeployableRegion(t, h, setup)

	headers := authHeaders(setup.RootKey)
	headers.Set("X-Unkey-Client", "unkey-cli/1.2.3")

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusCreated, res.Status, "expected 201, received: %s", res.RawBody)
	require.True(t, capture.called)
	require.Equal(t, ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_CLI, capture.req.Trigger)
}

func TestGitSource(t *testing.T) {
	h := testutil.NewHarness(t)
	capture := &ctrlCapture{}
	route := newRoute(h, capture)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	seedDeployableRegion(t, h, setup)
	connectRepo(t, h, setup.Workspace.ID, setup.Project.ID, setup.App.ID)

	req := gitRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, openapi.DeploymentSourceGit{
		Branch:    ptr.P("main"),
		CommitSha: ptr.P("abc123"),
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusCreated, res.Status, "expected 201, received: %s", res.RawBody)
	require.NotEmpty(t, res.Body.Data.DeploymentId)

	require.True(t, capture.called)
	require.NotNil(t, capture.req.GetGitCommit())
	require.Equal(t, "main", capture.req.GetGitCommit().GetBranch())
	require.Equal(t, "abc123", capture.req.GetGitCommit().GetCommitSha())
	require.Empty(t, capture.req.DockerImage)
}

func TestGitSourceWithFork(t *testing.T) {
	h := testutil.NewHarness(t)
	capture := &ctrlCapture{}
	route := newRoute(h, capture)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	seedDeployableRegion(t, h, setup)
	connectRepo(t, h, setup.Workspace.ID, setup.Project.ID, setup.App.ID)

	req := gitRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, openapi.DeploymentSourceGit{
		CommitSha:  ptr.P("9f2c1a7"),
		Repository: ptr.P("contributor/acme-api"),
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusCreated, res.Status, "expected 201, received: %s", res.RawBody)
	require.True(t, capture.called)
	require.NotNil(t, capture.req.GetGitCommit())
	require.Equal(t, "contributor/acme-api", capture.req.GetGitCommit().GetForkRepository())
	require.Equal(t, "9f2c1a7", capture.req.GetGitCommit().GetCommitSha())
}

func TestRedeployGitApp(t *testing.T) {
	h := testutil.NewHarness(t)
	capture := &ctrlCapture{}
	route := newRoute(h, capture)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	seedDeployableRegion(t, h, setup)
	connectRepo(t, h, setup.Workspace.ID, setup.Project.ID, setup.App.ID)

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
		GitBranch:     "main",
	})

	req := deploymentRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, dep.ID)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusCreated, res.Status, "expected 201, received: %s", res.RawBody)
	require.True(t, capture.called)
	require.NotNil(t, capture.req.GetGitCommit(), "git-connected app rebuilds from the recorded commit")
	require.Equal(t, "main", capture.req.GetGitCommit().GetBranch())
	require.Empty(t, capture.req.DockerImage)
}

func TestRedeployImageReuse(t *testing.T) {
	h := testutil.NewHarness(t)
	capture := &ctrlCapture{}
	route := newRoute(h, capture)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	seedDeployableRegion(t, h, setup)
	// No repo connection: redeploy reuses the recorded image rather than rebuilding.

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
	require.True(t, capture.called)
	require.Nil(t, capture.req.GetGitCommit(), "an app without a repo connection reuses the image instead of rebuilding")
}

// TestRedeployForkDeployment covers redeploying a deployment that was built from
// a fork. The fork repository and full commit metadata must be carried forward
// so ctrl resolves the commit against the fork, not the base repo.
func TestRedeployForkDeployment(t *testing.T) {
	h := testutil.NewHarness(t)
	capture := &ctrlCapture{}
	route := newRoute(h, capture)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	seedDeployableRegion(t, h, setup)
	connectRepo(t, h, setup.Workspace.ID, setup.Project.ID, setup.App.ID)

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:                     uid.New(uid.DeploymentPrefix),
		WorkspaceID:            setup.Workspace.ID,
		ProjectID:              setup.Project.ID,
		AppID:                  setup.App.ID,
		EnvironmentID:          setup.Environment.ID,
		GitBranch:              "feature",
		GitCommitSha:           "9f2c1a7",
		GitCommitMessage:       "add KEBAP endpoint",
		GitCommitAuthorHandle:  "contributor",
		GitCommitAuthorAvatar:  "https://example.com/avatar.png",
		GitCommitTimestamp:     1700000000,
		ForkRepositoryFullName: "contributor/acme-api",
	})

	req := deploymentRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, dep.ID)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusCreated, res.Status, "expected 201, received: %s", res.RawBody)
	require.True(t, capture.called)
	require.NotNil(t, capture.req.GetGitCommit())
	require.Equal(t, "contributor/acme-api", capture.req.GetGitCommit().GetForkRepository(), "fork must be carried forward")
	require.Equal(t, "9f2c1a7", capture.req.GetGitCommit().GetCommitSha())
	require.Equal(t, "feature", capture.req.GetGitCommit().GetBranch())
	require.Equal(t, "add KEBAP endpoint", capture.req.GetGitCommit().GetCommitMessage())
	require.Equal(t, "contributor", capture.req.GetGitCommit().GetAuthorHandle())
	require.Equal(t, int64(1700000000), capture.req.GetGitCommit().GetTimestamp())
	require.Empty(t, capture.req.DockerImage)
}

// TestRedeployImageDeploymentOnConnectedApp covers an image-origin deployment
// being redeployed after the app later gained a repo connection. The recorded
// image must be reused; the handler must not fabricate an empty git commit that
// ctrl would turn into a default-branch build.
func TestRedeployImageDeploymentOnConnectedApp(t *testing.T) {
	h := testutil.NewHarness(t)
	capture := &ctrlCapture{}
	route := newRoute(h, capture)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	seedDeployableRegion(t, h, setup)
	connectRepo(t, h, setup.Workspace.ID, setup.Project.ID, setup.App.ID)

	// Image-origin deployment: no git commit, but a built image on record.
	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})
	setDeploymentImage(t, h, dep.ID, "nginx:latest")

	req := deploymentRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, dep.ID)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusCreated, res.Status, "expected 201, received: %s", res.RawBody)
	require.True(t, capture.called)
	require.Nil(t, capture.req.GetGitCommit(), "an image-origin deployment has no commit to rebuild even on a connected app")
	require.Equal(t, "nginx:latest", capture.req.DockerImage, "must reuse the recorded image")
}

// TestRedeployDeploymentWithoutBuiltImage covers a deployment that never produced
// an image and has no git commit (e.g. a pending or failed build). It cannot be
// reproduced, so the handler must refuse rather than send an empty request.
func TestRedeployDeploymentWithoutBuiltImage(t *testing.T) {
	h := testutil.NewHarness(t)
	capture := &ctrlCapture{}
	route := newRoute(h, capture)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	seedDeployableRegion(t, h, setup)
	connectRepo(t, h, setup.Workspace.ID, setup.Project.ID, setup.App.ID)

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})

	req := deploymentRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, dep.ID)

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.False(t, capture.called, "ctrl must not be called for an unreproducible deployment")
}

func TestSpecificEnvironmentPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	capture := &ctrlCapture{}
	route := newRoute(h, capture)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	seedDeployableRegion(t, h, setup)
	rootKey := h.CreateRootKey(setup.Workspace.ID, "environment."+setup.Environment.ID+".create_deployment")

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), req)
	require.Equal(t, http.StatusCreated, res.Status, "expected 201, received: %s", res.RawBody)
	require.Equal(t, setup.Project.ID, capture.req.ProjectId)
}
