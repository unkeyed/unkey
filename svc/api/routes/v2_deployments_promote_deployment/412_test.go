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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_promote_deployment"
)

// Promoting a deployment that never became ready fails before ctrl is called.
func TestPromoteDeploymentNotReady(t *testing.T) {
	h := testutil.NewHarness(t)
	mock := &testutil.MockDeploymentClient{}
	route := newRoute(h, mock)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.promote_deployment"},
	})

	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
		Status:        db.DeploymentsStatusFailed,
	})

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, "is not ready")
	require.Empty(t, mock.PromoteCalls, "ctrl must not be called for a non-ready deployment")
}

// Promotion swaps the app's production live pointer, so it is rejected for
// non-production environments before ctrl is called.
func TestPromoteDeploymentNonProduction(t *testing.T) {
	h := testutil.NewHarness(t)
	mock := &testutil.MockDeploymentClient{}
	route := newRoute(h, mock)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.promote_deployment"},
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
	require.Contains(t, res.Body.Error.Detail, "Only production deployments can be promoted.")
	require.Empty(t, mock.PromoteCalls, "ctrl must not be called for non-production deployments")
}

// Promoting when the app has no live deployment fails before ctrl is called.
func TestPromoteDeploymentNoLiveDeployment(t *testing.T) {
	h := testutil.NewHarness(t)
	mock := &testutil.MockDeploymentClient{}
	route := newRoute(h, mock)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.promote_deployment"},
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
	require.Contains(t, res.Body.Error.Detail, "no live deployment")
	require.Empty(t, mock.PromoteCalls, "ctrl must not be called when the app has no live deployment")
}

// Promoting the deployment that is already live (and not rolled back) fails
// before ctrl is called.
func TestPromoteDeploymentAlreadyLive(t *testing.T) {
	h := testutil.NewHarness(t)
	mock := &testutil.MockDeploymentClient{}
	route := newRoute(h, mock)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.promote_deployment"},
	})

	live := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
		Status:        db.DeploymentsStatusReady,
	})
	setCurrentDeployment(t, h, setup.App.ID, live.ID)

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: live.ID})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, "already live")
	require.Empty(t, mock.PromoteCalls, "ctrl must not be called when the deployment is already live")
}

// A ctrl precondition failure (e.g. a concurrent promotion) must surface as a
// 412, not a 500, and must not leak ctrl's internal error text.
func TestPromoteDeploymentCtrlPreconditionFailed(t *testing.T) {
	h := testutil.NewHarness(t)
	mock := &testutil.MockDeploymentClient{
		PromoteFunc: func(ctx context.Context, req *ctrlv1.PromoteRequest) (*ctrlv1.PromoteResponse, error) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("target deployment is already the live deployment"))
		},
	}
	route := newRoute(h, mock)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.promote_deployment"},
	})

	live := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
		Status:        db.DeploymentsStatusReady,
	})
	setCurrentDeployment(t, h, setup.App.ID, live.ID)

	target := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
		Status:        db.DeploymentsStatusReady,
	})

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: target.ID})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Len(t, mock.PromoteCalls, 1)

	// Only the fixed public message may reach the caller; ctrl's internal error
	// text must stay in the logs.
	require.Contains(t, res.Body.Error.Detail, "The deployment could not be promoted.")
	require.NotContains(t, res.RawBody, "target deployment is already the live deployment")
}
