package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_stop_deployment"
)

func TestStopDeployment(t *testing.T) {
	h := testutil.NewHarness(t)
	mock := &testutil.MockDeploymentClient{}
	route := newRoute(h, mock)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.stop_deployment"},
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
		Status:        db.DeploymentsStatusReady,
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		DeploymentId: dep.ID,
	})
	require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)

	require.Len(t, mock.StopDeploymentCalls, 1)
	require.Equal(t, dep.ID, mock.StopDeploymentCalls[0].GetDeploymentId())
}
