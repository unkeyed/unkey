package deploy

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
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
		return w.db.FindDeploymentById(runCtx, deploymentID)
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

	environment, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Environment, error) {
		return w.db.FindEnvironmentById(runCtx, deployment.EnvironmentID)
	}, restate.WithName("find environment for wake"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.TerminalError(fmt.Errorf("environment not found"), 404)
		}
		return nil, fmt.Errorf("failed to load environment: %w", err)
	}
	if environment.Slug == "production" {
		return nil, restate.TerminalError(fmt.Errorf("production deployments cannot be woken"), 400)
	}

	_, err = hydrav1.NewDeploymentServiceClient(ctx, deploymentID).
		ScheduleDesiredStateChange().
		Request(
			&hydrav1.ScheduleDesiredStateChangeRequest{
				DelayMillis: 0,
				State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_RUNNING,
				Overwrite:   true,
			},
		)
	if err != nil {
		return nil, fmt.Errorf("wake deployment workflow failed: %w", err)
	}

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return w.db.UpdateDeploymentStatus(runCtx, db.UpdateDeploymentStatusParams{
			ID:        deploymentID,
			Status:    db.DeploymentsStatusDeploying,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("mark waking deployment as deploying"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fmt.Errorf("mark waking deployment as deploying: %w", err)
	}

	rows, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.FindDeploymentTopologyMinReplicasRow, error) {
		return w.db.FindDeploymentTopologyMinReplicas(runCtx, deploymentID)
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

	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, err
	}
	// The deadline is anchored to the journaled wall clock so it stays stable if
	// this side effect is retried. The poll below runs as a single blocking
	// restate.Run rather than a durable restate.Sleep loop: it does not journal
	// per-check progress, so a worker restart mid-wake restarts the poll from the
	// start (bounded by deadline).
	deadline := now.Add(regionReadyTimeout)

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		for {
			healthy, checkErr := w.checkInstancesHealthy(runCtx, deploymentID, regionMinReplicas, requiredRegions)
			if checkErr != nil {
				return fmt.Errorf("check wake instance readiness: %w", checkErr)
			}
			if healthy {
				return nil
			}

			if time.Now().After(deadline) {
				return fault.Wrap(
					restate.TerminalErrorf("not enough regions became healthy in %v, required %d of %d", regionReadyTimeout, requiredRegions, len(regionMinReplicas)),
					fault.Public("Not enough regions became healthy in time."),
				)
			}
			time.Sleep(time.Second)
		}
	})
	if err != nil {
		return nil, err
	}

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return w.db.UpdateDeploymentStatus(runCtx, db.UpdateDeploymentStatusParams{
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
