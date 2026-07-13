package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_get_deployment"
)

func TestGetDeployment(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:                    uid.New(uid.DeploymentPrefix),
		WorkspaceID:           setup.Workspace.ID,
		ProjectID:             setup.Project.ID,
		AppID:                 setup.App.ID,
		EnvironmentID:         setup.Environment.ID,
		GitBranch:             "main",
		GitCommitSha:          "9f2c1a7",
		GitCommitMessage:      "add KEBAP endpoint",
		GitCommitAuthorHandle: "octocat",
	})

	req := handler.Request{DeploymentId: dep.ID}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.NotNil(t, res.Body)

	d := res.Body.Data
	require.Equal(t, dep.ID, d.Id)
	require.Equal(t, openapi.DeploymentStatusPending, d.Status)
	require.Equal(t, 8080, d.Runtime.Port)
	require.Equal(t, openapi.SIGTERM, d.Runtime.ShutdownSignal)
	require.Equal(t, openapi.Http1, d.Runtime.UpstreamProtocol)
	require.NotNil(t, d.Runtime.Command)
	require.Nil(t, d.Runtime.Healthcheck)

	// Internal fields must never appear in the response body.
	for _, leaked := range []string{"k8s_name", "k8sName", "workspace_id", "workspaceId", "sentinel", "encrypted", "build_id", "buildId", "invocation", "github_deployment", "githubDeployment", "\"pk\""} {
		require.False(t, strings.Contains(res.RawBody, leaked), "response leaked internal field %q: %s", leaked, res.RawBody)
	}
}

func TestGetDeploymentSpecificEnvironmentPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	rootKey := h.CreateRootKey(setup.Workspace.ID, "environment."+setup.Environment.ID+".read_deployment")

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})

	req := handler.Request{DeploymentId: dep.ID}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), req)
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, dep.ID, res.Body.Data.Id)
}
