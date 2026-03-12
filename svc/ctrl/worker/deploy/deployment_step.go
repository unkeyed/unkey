package deploy

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/ctrl/worker/internal/db"
)

func (w *Workflow) DeploymentStep(
	ctx restate.ObjectContext,
	step db.DeploymentStepsStep,
	deployment db.Deployment,
	fn func(innerCtx restate.ObjectContext) error) error {

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

		return db.Tx(runCtx, w.db, func(txCtx context.Context, tx db.Querier) error {
			if err := tx.InsertDeploymentStep(txCtx, db.InsertDeploymentStepParams{
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

			if err := tx.UpdateDeploymentStatus(txCtx, db.UpdateDeploymentStatusParams{
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
		return w.db.EndDeploymentStep(runCtx, db.EndDeploymentStepParams{
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
