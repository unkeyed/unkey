package handler_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_start_deployment"
)

// Starting a deployment that is not stopped fails before ctrl is called.
func TestStartDeploymentNotStopped(t *testing.T) {
	h := testutil.NewHarness(t)
	mock := &testutil.MockDeploymentClient{}
	route := newRoute(h, mock)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.start_deployment"},
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
	require.Contains(t, res.Body.Error.Detail, "is not stopped")
	require.Contains(t, res.Body.Error.Type, "deployment_not_stopped")
	require.Empty(t, mock.WakeDeploymentCalls, "ctrl must not be called for a deployment that is not stopped")
}

// Production deployments are never stopped, so starting one is rejected before
// ctrl is called.
func TestStartDeploymentProduction(t *testing.T) {
	h := testutil.NewHarness(t)
	mock := &testutil.MockDeploymentClient{}
	route := newRoute(h, mock)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.start_deployment"},
	})

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
		Status:        db.DeploymentsStatusStopped,
		DesiredState:  db.DeploymentsDesiredStateStopped,
	})

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, "Production deployments cannot be started.")
	require.Contains(t, res.Body.Error.Type, "deployment_is_production")
	require.Empty(t, mock.WakeDeploymentCalls, "ctrl must not be called for production deployments")
}

// A ctrl precondition failure (e.g. desired state changed concurrently) must
// surface as a 412, not a 500, and must not leak ctrl's internal error text.
func TestStartDeploymentCtrlPreconditionFailed(t *testing.T) {
	h := testutil.NewHarness(t)
	mock := &testutil.MockDeploymentClient{
		WakeDeploymentFunc: func(ctx context.Context, req *ctrlv1.WakeDeploymentRequest) (*ctrlv1.WakeDeploymentResponse, error) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("deployment is not stopped"))
		},
	}
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
		DesiredState:  db.DeploymentsDesiredStateStopped,
	})

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Len(t, mock.WakeDeploymentCalls, 1)

	// Only the fixed public message may reach the caller; ctrl's internal error
	// text must stay in the logs.
	require.Contains(t, res.Body.Error.Detail, "The deployment could not be started.")
	require.NotContains(t, res.RawBody, "deployment is not stopped")
}

// When ctrl rejects with a deploygate message (a race: the deployment was
// started concurrently after the handler's own gate passed), the response
// carries the same specific code and message the local gate would have produced.
func TestStartDeploymentCtrlGateRejection(t *testing.T) {
	h := testutil.NewHarness(t)
	mock := &testutil.MockDeploymentClient{
		WakeDeploymentFunc: func(ctx context.Context, req *ctrlv1.WakeDeploymentRequest) (*ctrlv1.WakeDeploymentResponse, error) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New(deploygate.StartNotStopped.Message()))
		},
	}
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
		DesiredState:  db.DeploymentsDesiredStateStopped,
	})

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Len(t, mock.WakeDeploymentCalls, 1)
	require.Contains(t, res.Body.Error.Type, "deployment_not_stopped")
	require.Equal(t, deploygate.StartNotStopped.Message(), res.Body.Error.Detail)
}

// Starting resumes compute spend, so a workspace suspended by its Compute
// spend cap is rejected before ctrl is called, with the billing reason rather
// than a misleading lifecycle message.
func TestStartDeploymentSpendSuspended(t *testing.T) {
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
		DesiredState:  db.DeploymentsDesiredStateStopped,
	})

	_, err := h.DB.RW().ExecContext(context.Background(),
		"INSERT INTO workspace_billing (workspace_id, spend_suspended, created_at_m) VALUES (?, true, ?) ON DUPLICATE KEY UPDATE spend_suspended = true",
		setup.Workspace.ID, time.Now().UnixMilli(),
	)
	require.NoError(t, err)

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Type, "precondition_failed")
	require.Equal(t, deploygate.StartSpendSuspended.Message(), res.Body.Error.Detail)
	require.Empty(t, mock.WakeDeploymentCalls, "ctrl must not be called for a spend-suspended workspace")
}
