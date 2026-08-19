package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_stop_deployment"
)

func TestStopDeploymentUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)
	mock := &testutil.MockDeploymentClient{}
	route := newRoute(h, mock)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.stop_deployment"},
	})

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer invalid_token"},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, received: %s", res.RawBody)
	require.Empty(t, mock.StopDeploymentCalls, "ctrl must not be called without authentication")
}
