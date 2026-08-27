package handler_test

import (
	"net/http"
	"testing"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_stop_deployment"
)

// Stopping a deployment that is not running fails before Restate submission.
func TestStopDeploymentNotRunning(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
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
		Status:        mysqltype.DeploymentsStatusStopped,
	})

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, "is not running")
	require.Contains(t, res.Body.Error.Type, "deployment_not_running")
}

// A deployment that is already draining (desired_state=stopped, status still
// ready until krane removes the last instance) is rejected before Restate
// submission.
func TestStopDeploymentAlreadyStopping(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
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
		Status:        mysqltype.DeploymentsStatusReady,
	})

	err := db.Query.UpdateDeploymentDesiredState(t.Context(), h.DB.RW(), db.UpdateDeploymentDesiredStateParams{
		ID:           dep.ID,
		DesiredState: mysqltype.DeploymentsDesiredStateStopped,
	})
	require.NoError(t, err)

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, "already stopping")
	require.Contains(t, res.Body.Error.Type, "deployment_is_stopping")
}

// Production deployments are never stopped; reject them before Restate submission.
func TestStopDeploymentProduction(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
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
		Status:        mysqltype.DeploymentsStatusReady,
	})

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, "Production deployments cannot be stopped.")
	require.Contains(t, res.Body.Error.Type, "deployment_is_production")
}
