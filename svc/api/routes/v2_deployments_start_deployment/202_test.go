package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_start_deployment"
)

// startDeployment forwards to ctrl's WakeDeployment RPC; the public verb is
// "start" but the wire call must be the wake RPC with the same deployment id.
func TestStartDeployment(t *testing.T) {
	h := testutil.NewHarness(t)
	mock := &testutil.MockDeploymentClient{}
	route := newRoute(h, mock)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.start_deployment"},
	})

	preview := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   setup.Project.ID,
		AppID:       setup.App.ID,
		Slug:        "preview",
		Description: "preview environment",
	})

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: preview.ID,
		Status:        db.DeploymentsStatusStopped,
		GitBranch:     "KEBAP",
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		DeploymentId: dep.ID,
	})
	require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)

	require.Len(t, mock.WakeDeploymentCalls, 1)
	require.Equal(t, dep.ID, mock.WakeDeploymentCalls[0].GetDeploymentId())
}

// The environment-scoped permission must work as well as the wildcard.
func TestStartDeploymentScopedPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	mock := &testutil.MockDeploymentClient{}
	route := newRoute(h, mock)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()

	preview := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   setup.Project.ID,
		AppID:       setup.App.ID,
		Slug:        "preview",
		Description: "preview environment",
	})

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: preview.ID,
		Status:        db.DeploymentsStatusStopped,
	})

	rootKey := h.CreateRootKey(setup.Workspace.ID, "environment."+preview.ID+".start_deployment")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		DeploymentId: dep.ID,
	})
	require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)
	require.Len(t, mock.WakeDeploymentCalls, 1)
}
