package deploy

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
)

func (w *Workflow) DeploymentStep(
	ctx restate.WorkflowSharedContext,
	step db.DeploymentStepsStep,
	deployment db.Deployment,
	fn func(innerCtx restate.WorkflowSharedContext) error) error {

	err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		now := time.Now().UnixMilli()
		deploymentStatus := db.DeploymentsStatusPending
		switch step {
		case db.DeploymentStepsStepQueued:
			deploymentStatus = db.DeploymentsStatusPending
		case db.DeploymentStepsStepStarting:
			deploymentStatus = db.DeploymentsStatusStarting
		case db.DeploymentStepsStepBuilding:
			deploymentStatus = db.DeploymentsStatusBuilding
		case db.DeploymentStepsStepDeploying:
			deploymentStatus = db.DeploymentsStatusDeploying
		case db.DeploymentStepsStepNetwork:
			deploymentStatus = db.DeploymentsStatusNetwork
		case db.DeploymentStepsStepFinalizing:
			deploymentStatus = db.DeploymentsStatusFinalizing
		default:
			return fmt.Errorf("unexpected deployment step: %s", step)
		}

		return db.Tx(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
			if err := db.Query.InsertDeploymentStep(txCtx, tx, db.InsertDeploymentStepParams{
				WorkspaceID:   deployment.WorkspaceID,
				ProjectID:     deployment.ProjectID,
				AppID:         deployment.AppID,
				EnvironmentID: deployment.EnvironmentID,
				DeploymentID:  deployment.ID,
				Step:          step,
				StartedAt:     uint64(now),
			}); err != nil {
				return err
			}

			if err := db.Query.UpdateDeploymentStatus(txCtx, tx, db.UpdateDeploymentStatusParams{
				ID:        deployment.ID,
				Status:    deploymentStatus,
				UpdatedAt: sql.NullInt64{Valid: true, Int64: now},
			}); err != nil {
				return err
			}
			return nil
		})

	}, restate.WithName(fmt.Sprintf("starting step: %s", step)))
	if err != nil {
		return err
	}

	stepErr := fn(ctx)

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.EndDeploymentStep(runCtx, w.db.RW(), db.EndDeploymentStepParams{
			DeploymentID: deployment.ID,
			Step:         step,
			EndedAt:      sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			Error:        sql.NullString{Valid: stepErr != nil, String: fault.UserFacingMessage(stepErr)},
		})
	}, restate.WithName(fmt.Sprintf("ending step: %s", step)))
	if err != nil {
		return err
	}

	return stepErr

}
