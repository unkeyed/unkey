package handler_test

import (
	"net/http"
	"testing"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_start_deployment"
)

// Starting a deployment that is not stopped fails before Restate is called.
func TestStartDeploymentNotStopped(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
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
		Status:        mysqltype.DeploymentsStatusReady,
	})

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, "is not stopped")
	require.Contains(t, res.Body.Error.Type, "deployment_not_stopped")
}

// Production deployments are never stopped, so starting one is rejected before
// Restate is called.
func TestStartDeploymentProduction(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
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
		Status:        mysqltype.DeploymentsStatusStopped,
		DesiredState:  mysqltype.DeploymentsDesiredStateStopped,
	})

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Detail, "Production deployments cannot be started.")
	require.Contains(t, res.Body.Error.Type, "deployment_is_production")
}

// Starting resumes compute spend, so a workspace suspended by its Compute
// spend cap is rejected before Restate is called, with the billing reason rather
// than a misleading lifecycle message.
func TestStartDeploymentSpendSuspended(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
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
		Status:        mysqltype.DeploymentsStatusStopped,
		DesiredState:  mysqltype.DeploymentsDesiredStateStopped,
	})

	err := db.Query.UpsertWorkspaceBillingSpendSuspended(t.Context(), h.DB.RW(), db.UpsertWorkspaceBillingSpendSuspendedParams{
		WorkspaceID:    setup.Workspace.ID,
		SpendSuspended: true,
		CreatedAtM:     time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Contains(t, res.Body.Error.Type, "precondition_failed")
	require.Equal(t, deploygate.StartSpendSuspended.Message(), res.Body.Error.Detail)
}

func TestStartDeploymentRequiresComputePlan(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.start_deployment"},
	})
	h.ClearComputePlanOverride(setup.Workspace.ID)
	preview := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   setup.Project.ID,
		AppID:       setup.App.ID,
		Slug:        "preview",
	})
	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: preview.ID,
		Status:        mysqltype.DeploymentsStatusStopped,
		DesiredState:  mysqltype.DeploymentsDesiredStateStopped,
	})

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Equal(t, "The workspace has no active Compute plan.", res.Body.Error.Detail)
}
