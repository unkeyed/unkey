package deploy

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
)

// WakeDeployment is the public Restate entrypoint for waking a stopped
// deployment. It schedules desired_state=running, marks the visible phase as
// deploying, then polls instance health until the deployment can be marked
// ready.
func (w *Workflow) WakeDeployment(ctx restate.ObjectContext, req *hydrav1.WakeDeploymentRequest) (*hydrav1.WakeDeploymentResponse, error) {
	deploymentID := req.GetDeploymentId()
	if deploymentID == "" {
		return nil, restate.TerminalError(fmt.Errorf("deployment_id is required"), 400)
	}

	deployment, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Deployment, error) {
		return db.Query.FindDeploymentById(runCtx, w.db.RO(), deploymentID)
	}, restate.WithName("find deployment for wake"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.TerminalError(fmt.Errorf("deployment not found"), 404)
		}
		return nil, fmt.Errorf("failed to load deployment: %w", err)
	}

	if deployment.DesiredState != db.DeploymentsDesiredStateStopped {
		return nil, restate.TerminalError(fmt.Errorf("deployment is not stopped"), 400)
	}

	_, err = hydrav1.NewDeploymentServiceClient(ctx, deploymentID).
		ScheduleDesiredStateChange().
		Request(
			&hydrav1.ScheduleDesiredStateChangeRequest{
				DelayMillis: 0,
				State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_RUNNING,
			},
		)
	if err != nil {
		return nil, fmt.Errorf("wake deployment workflow failed: %w", err)
	}

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.UpdateDeploymentStatus(runCtx, w.db.RW(), db.UpdateDeploymentStatusParams{
			ID:        deploymentID,
			Status:    db.DeploymentsStatusDeploying,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("mark waking deployment as deploying"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fmt.Errorf("mark waking deployment as deploying: %w", err)
	}

	rows, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.FindDeploymentTopologyMinReplicasRow, error) {
		return db.Query.FindDeploymentTopologyMinReplicas(runCtx, w.db.RW(), deploymentID)
	}, restate.WithName("find wake topology min replicas"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fmt.Errorf("find wake topology min replicas: %w", err)
	}
	if len(rows) == 0 {
		return nil, restate.TerminalError(fmt.Errorf("deployment has no topology"), 400)
	}

	regionMinReplicas := make(map[string]uint32, len(rows))
	for _, row := range rows {
		regionMinReplicas[row.RegionID] = row.AutoscalingReplicasMin
	}
	requiredRegions := max(len(regionMinReplicas)-1, 1)
	attempts := int(regionReadyTimeout / wakeReadinessPollInterval)
	for attempt := 0; attempt <= attempts; attempt++ {
		healthy, err := restate.Run(ctx, func(runCtx restate.RunContext) (bool, error) {
			return w.checkInstancesHealthy(runCtx, deploymentID, regionMinReplicas, requiredRegions)
		}, restate.WithName(fmt.Sprintf("check wake instance readiness %d", attempt)), restate.WithMaxRetryAttempts(runMaxAttempts))
		if err != nil {
			return nil, fmt.Errorf("check wake instance readiness: %w", err)
		}
		if healthy {
			err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
				return db.Query.UpdateDeploymentStatus(runCtx, w.db.RW(), db.UpdateDeploymentStatusParams{
					ID:        deploymentID,
					Status:    db.DeploymentsStatusReady,
					UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				})
			}, restate.WithName("mark woken deployment ready"), restate.WithMaxRetryAttempts(runMaxAttempts))
			if err != nil {
				return nil, fmt.Errorf("mark woken deployment ready: %w", err)
			}
			return &hydrav1.WakeDeploymentResponse{}, nil
		}

		if attempt < attempts {
			if err := restate.Sleep(ctx, wakeReadinessPollInterval); err != nil {
				return nil, fmt.Errorf("sleep before wake readiness check: %w", err)
			}
		}
	}

	return nil, fault.Wrap(
		restate.TerminalErrorf("not enough regions became healthy in %v, required %d of %d", regionReadyTimeout, requiredRegions, len(regionMinReplicas)),
		fault.Public("Not enough regions became healthy in time."),
	)
}
