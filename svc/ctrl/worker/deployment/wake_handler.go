package deployment

import (
	"database/sql"
	"fmt"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/audit"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/gatefault"
	"github.com/unkeyed/unkey/svc/ctrl/internal/readiness"
)

func (v *VirtualObject) WakeDeployment(ctx restate.ObjectContext, req *hydrav1.WakeDeploymentRequest) (*hydrav1.WakeDeploymentResponse, error) {
	deploymentID := restate.Key(ctx)
	if id := req.GetDeploymentId(); id != "" && id != deploymentID {
		return nil, restate.ToTerminalError(fmt.Errorf("deployment_id %q does not match object key %q", id, deploymentID), restate.WithErrorCode(400))
	}

	deployment, err := v.loadDeployment(ctx, deploymentID, "wake")
	if err != nil {
		return nil, err
	}

	if err := deploygate.CheckStartTarget(deploygate.StartInput{
		DesiredState:    deployment.DesiredState,
		EnvironmentKind: deployment.EnvironmentKind,
		SpendSuspended:  false,
	}); err != nil {
		return nil, gatefault.Terminal(err)
	}

	if err := v.setDesiredState(ctx, deploymentID, hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_RUNNING); err != nil {
		return nil, err
	}

	restate.Clear(ctx, transitionKey)

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return v.db.UpdateDeploymentStatus(runCtx, db.UpdateDeploymentStatusParams{
			ID:        deploymentID,
			Status:    mysqltype.DeploymentsStatusDeploying,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("mark waking deployment as deploying"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fmt.Errorf("mark waking deployment as deploying: %w", err)
	}

	if err := v.waitUntilHealthy(ctx, deploymentID); err != nil {
		return nil, err
	}

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return v.db.UpdateDeploymentStatus(runCtx, db.UpdateDeploymentStatusParams{
			ID:        deploymentID,
			Status:    mysqltype.DeploymentsStatusReady,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("mark woken deployment ready"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fmt.Errorf("mark woken deployment ready: %w", err)
	}

	if err := audit.InsertDeploymentLifecycle(
		ctx,
		v.auditlogs,
		req.GetActor(),
		req.GetCorrelationId(),
		audit.DeploymentRef{
			ID:            deployment.ID,
			WorkspaceID:   deployment.WorkspaceID,
			ProjectID:     deployment.ProjectID,
			AppID:         deployment.AppID,
			EnvironmentID: deployment.EnvironmentID,
		},
		auditlog.DeploymentWakeEvent,
		fmt.Sprintf("Woke deployment %s", deploymentID),
	); err != nil {
		return nil, fmt.Errorf("insert wake deployment audit log: %w", err)
	}

	return &hydrav1.WakeDeploymentResponse{}, nil
}

// waitUntilHealthy tolerates one unhealthy region. The deadline is journaled
// so a retry keeps it
func (v *VirtualObject) waitUntilHealthy(ctx restate.ObjectContext, deploymentID string) error {
	now, err := restateutil.Now(ctx)
	if err != nil {
		return err
	}
	deadline := now.Add(regionReadyTimeout)

	return restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		rows, err := v.db.FindDeploymentTopologyMinReplicas(runCtx, deploymentID)
		if err != nil {
			return fmt.Errorf("load required instances per region: %w", err)
		}
		if len(rows) == 0 {
			return restate.ToTerminalError(fmt.Errorf("deployment has no topology"), restate.WithErrorCode(400))
		}
		requiredPerRegion := make(map[string]uint32, len(rows))
		for _, row := range rows {
			requiredPerRegion[row.RegionID] = row.AutoscalingReplicasMin
		}
		requiredRegions := max(len(requiredPerRegion)-1, 1)

		for {
			healthy, checkErr := readiness.InstancesHealthy(runCtx, v.db, deploymentID, requiredPerRegion, requiredRegions)
			if checkErr != nil {
				return fmt.Errorf("check wake instance readiness: %w", checkErr)
			}
			if healthy {
				return nil
			}

			if time.Now().After(deadline) {
				return fault.Wrap(
					restate.TerminalErrorf("not enough regions became healthy in %v, required %d of %d", regionReadyTimeout, requiredRegions, len(requiredPerRegion)),
					fault.Public("Not enough regions became healthy in time."),
				)
			}
			time.Sleep(time.Second)
		}
	}, restate.WithName("wait until healthy"))
}
