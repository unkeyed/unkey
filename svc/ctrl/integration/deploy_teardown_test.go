//go:build integration

package integration

import (
	"database/sql"
	"testing"
	"time"

	restatetest "github.com/restatedev/sdk-go/testing"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deployment"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deployteardown"
)

// startTeardown wires the DeploymentService (which applies the desired-state
// change) and the DeployTeardownService under a real Restate server, with a
// short drain poll/grace so tests don't wait on the production 5-minute timeout.
func startTeardown(t *testing.T, database db.Database) *restatetest.TestEnvironment {
	t.Helper()

	teardownSvc, err := deployteardown.New(deployteardown.Config{
		DB:                database,
		DrainPollInterval: 200 * time.Millisecond,
		DrainGraceTimeout: 2 * time.Second,
	})
	require.NoError(t, err)

	return restatetest.Start(t,
		hydrav1.NewDeploymentServiceServer(deployment.New(deployment.Config{DB: database})),
		hydrav1.NewDeployTeardownServiceServer(teardownSvc),
	)
}

// TestDeployTeardown_NoRunningDeployments verifies that tearing down a workspace
// with nothing running is a no-op that reports drained immediately.
func TestDeployTeardown_NoRunningDeployments(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	tEnv := startTeardown(t, h.DB)

	workspaceID := h.Resources().UserWorkspace.ID
	client := hydrav1.NewDeployTeardownServiceIngressClient(tEnv.Ingress(), workspaceID)
	resp, err := client.Teardown().Request(ctx, &hydrav1.TeardownRequest{
		Mode: hydrav1.TeardownMode_TEARDOWN_MODE_ARCHIVE,
	})
	require.NoError(t, err)

	require.True(t, resp.GetDrained())
	require.Equal(t, int32(0), resp.GetDeploymentsStopped())
}

// TestDeployTeardown_ClearsCurrentAndStops verifies the core teardown behavior:
// the running deployment is stopped even though it is the app's current
// deployment. Teardown clears apps.current_deployment_id first, so the
// DeploymentService guard (which refuses to touch the current deployment) passes
// on its own and the desired-state change applies.
//
// There is no krane in the test to flip the deployment's status to 'stopped', so
// the drain never completes and Teardown returns drained=false after the grace
// timeout. That is the expected, billing-safe behavior: it never blocks forever.
func TestDeployTeardown_ClearsCurrentAndStops(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	tEnv := startTeardown(t, h.DB)

	dep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       "us-east-1",
		DesiredState: db.DeploymentsDesiredStateRunning,
	}).Deployment

	// Make the deployment its app's current deployment so we exercise the
	// guard-bypass-by-clearing path.
	err := db.Query.UpdateAppDeployments(ctx, h.DB.RW(), db.UpdateAppDeploymentsParams{
		CurrentDeploymentID: sql.NullString{Valid: true, String: dep.ID},
		IsRolledBack:        false,
		UpdatedAt:           sql.NullInt64{Valid: true, Int64: h.Now()},
		AppID:               dep.AppID,
	})
	require.NoError(t, err)

	client := hydrav1.NewDeployTeardownServiceIngressClient(tEnv.Ingress(), dep.WorkspaceID)
	resp, err := client.Teardown().Request(ctx, &hydrav1.TeardownRequest{
		Mode: hydrav1.TeardownMode_TEARDOWN_MODE_ARCHIVE,
	})
	require.NoError(t, err)

	require.Equal(t, int32(1), resp.GetDeploymentsStopped())
	require.False(t, resp.GetDrained(), "no krane to drain, so the grace timeout forces completion")

	// current_deployment_id is cleared synchronously by teardown.
	app, err := db.Query.FindAppById(ctx, h.DB.RO(), dep.AppID)
	require.NoError(t, err)
	require.False(t, app.CurrentDeploymentID.Valid, "current_deployment_id should be cleared")

	// The desired-state change is applied asynchronously by the DeploymentService
	// VO, so poll for it. ARCHIVE maps to desired_state 'stopped'.
	require.Eventually(t, func() bool {
		got, getErr := db.Query.FindDeploymentById(ctx, h.DB.RO(), dep.ID)
		return getErr == nil && got.DesiredState == db.DeploymentsDesiredStateStopped
	}, 10*time.Second, 200*time.Millisecond, "desired_state should become stopped")
}

// TestDeployTeardown_SuspendThenResume verifies the resumable half of teardown:
// SUSPEND records the app's current deployment and stops it, then Resume brings
// it back to running and restores apps.current_deployment_id from that record.
func TestDeployTeardown_SuspendThenResume(t *testing.T) {
	h := New(t)
	ctx := h.Context()

	tEnv := startTeardown(t, h.DB)

	dep := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       "us-east-1",
		DesiredState: db.DeploymentsDesiredStateRunning,
	}).Deployment

	// Make the deployment its app's current deployment so SUSPEND records it.
	err := db.Query.UpdateAppDeployments(ctx, h.DB.RW(), db.UpdateAppDeploymentsParams{
		CurrentDeploymentID: sql.NullString{Valid: true, String: dep.ID},
		IsRolledBack:        false,
		UpdatedAt:           sql.NullInt64{Valid: true, Int64: h.Now()},
		AppID:               dep.AppID,
	})
	require.NoError(t, err)

	client := hydrav1.NewDeployTeardownServiceIngressClient(tEnv.Ingress(), dep.WorkspaceID)
	suspendResp, err := client.Teardown().Request(ctx, &hydrav1.TeardownRequest{
		Mode: hydrav1.TeardownMode_TEARDOWN_MODE_SUSPEND,
	})
	require.NoError(t, err)
	require.Equal(t, int32(1), suspendResp.GetDeploymentsStopped())

	// current_deployment_id is cleared synchronously by teardown.
	app, err := db.Query.FindAppById(ctx, h.DB.RO(), dep.AppID)
	require.NoError(t, err)
	require.False(t, app.CurrentDeploymentID.Valid, "current_deployment_id should be cleared on suspend")

	// SUSPEND maps to desired_state 'stopped', applied asynchronously.
	require.Eventually(t, func() bool {
		got, getErr := db.Query.FindDeploymentById(ctx, h.DB.RO(), dep.ID)
		return getErr == nil && got.DesiredState == db.DeploymentsDesiredStateStopped
	}, 10*time.Second, 200*time.Millisecond, "desired_state should become stopped")

	resumeResp, err := client.Resume().Request(ctx, &hydrav1.ResumeRequest{})
	require.NoError(t, err)
	require.Equal(t, int32(1), resumeResp.GetDeploymentsResumed())

	// Resume restores current_deployment_id synchronously.
	app, err = db.Query.FindAppById(ctx, h.DB.RO(), dep.AppID)
	require.NoError(t, err)
	require.True(t, app.CurrentDeploymentID.Valid, "current_deployment_id should be restored on resume")
	require.Equal(t, dep.ID, app.CurrentDeploymentID.String)

	// The deployment is back to desired_state 'running'.
	require.Eventually(t, func() bool {
		got, getErr := db.Query.FindDeploymentById(ctx, h.DB.RO(), dep.ID)
		return getErr == nil && got.DesiredState == db.DeploymentsDesiredStateRunning
	}, 10*time.Second, 200*time.Millisecond, "desired_state should become running again")
}
