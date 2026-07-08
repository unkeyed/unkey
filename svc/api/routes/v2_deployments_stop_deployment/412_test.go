package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_stop_deployment"
)

// Stopping a deployment that is not running fails before ctrl is called.
func TestStopDeploymentNotRunning(t *testing.T) {
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
		Status:        db.DeploymentsStatusStopped,
	})

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, "is not running")
	require.Empty(t, mock.StopDeploymentCalls, "ctrl must not be called for a deployment that is not running")
}

// A deployment that is already draining (desired_state=stopped, status still
// ready until krane removes the last instance) is rejected before ctrl is
// called.
func TestStopDeploymentAlreadyStopping(t *testing.T) {
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

	err := db.Query.UpdateDeploymentDesiredState(t.Context(), h.DB.RW(), db.UpdateDeploymentDesiredStateParams{
		ID:           dep.ID,
		DesiredState: db.DeploymentsDesiredStateStopped,
	})
	require.NoError(t, err)

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, "already stopping")
	require.Empty(t, mock.StopDeploymentCalls, "ctrl must not be called for a deployment that is already stopping")
}

// Production deployments are never stopped; rejected before ctrl is called.
func TestStopDeploymentProduction(t *testing.T) {
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
		Status:        db.DeploymentsStatusReady,
	})

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, "Production deployments cannot be stopped.")
	require.Empty(t, mock.StopDeploymentCalls, "ctrl must not be called for production deployments")
}

// A ctrl precondition failure (e.g. desired state changed concurrently) must
// surface as a 412, not a 500, and must not leak ctrl's internal error text.
func TestStopDeploymentCtrlPreconditionFailed(t *testing.T) {
	h := testutil.NewHarness(t)
	mock := &testutil.MockDeploymentClient{
		StopDeploymentFunc: func(ctx context.Context, req *ctrlv1.StopDeploymentRequest) (*ctrlv1.StopDeploymentResponse, error) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("deployment is not running"))
		},
	}
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

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Len(t, mock.StopDeploymentCalls, 1)

	// Only the fixed public message may reach the caller; ctrl's internal error
	// text must stay in the logs.
	require.Contains(t, res.Body.Error.Detail, "The deployment could not be stopped.")
	require.NotContains(t, res.RawBody, "deployment is not running")
}
