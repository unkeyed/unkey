package deploy

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (w *Workflow) DeploymentStep(
	ctx restate.WorkflowContext,
	step db.DeploymentStepsStep,
	deployment db.Deployment,
	fn func(innerCtx restate.WorkflowContext) error,
) error {
	err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		now := time.Now().UnixMilli()
		deploymentStatus := mysqltype.DeploymentsStatusPending
		switch step {
		case db.DeploymentStepsStepQueued:
			deploymentStatus = mysqltype.DeploymentsStatusPending
		case db.DeploymentStepsStepStarting:
			deploymentStatus = mysqltype.DeploymentsStatusStarting
		case db.DeploymentStepsStepBuilding:
			deploymentStatus = mysqltype.DeploymentsStatusBuilding
		case db.DeploymentStepsStepDeploying:
			deploymentStatus = mysqltype.DeploymentsStatusDeploying
		case db.DeploymentStepsStepNetwork:
			deploymentStatus = mysqltype.DeploymentsStatusNetwork
		case db.DeploymentStepsStepFinalizing:
			deploymentStatus = mysqltype.DeploymentsStatusFinalizing
		default:
			return fmt.Errorf("unexpected deployment step: %s", step)
		}

		return db.Tx(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
			current, err := db.NewQueries(tx).FindDeploymentById(txCtx, deployment.ID)
			if err != nil {
				return err
			}
			// A cancel between two steps has already ended the deployment. Reviving
			// it here would let the compensation stack later mark it failed
			if current.Status.IsTerminal() {
				return restate.TerminalError(
					fmt.Errorf("deployment %s is already %s, not starting step %s", deployment.ID, current.Status, step),
					409,
				)
			}

			if err := db.NewQueries(tx).InsertDeploymentStep(txCtx, db.InsertDeploymentStepParams{
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

			return db.NewQueries(tx).UpdateDeploymentStatusIfActive(txCtx, db.UpdateDeploymentStatusIfActiveParams{
				ID:                  deployment.ID,
				Status:              deploymentStatus,
				UpdatedAt:           sql.NullInt64{Valid: true, Int64: now},
				ProgressingStatuses: mysqltype.ProgressingDeploymentStatuses,
			})
		})
	}, restate.WithName(fmt.Sprintf("starting step: %s", step)), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return err
	}

	stepErr := fn(ctx)

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return w.db.EndDeploymentStep(runCtx, db.EndDeploymentStepParams{
			DeploymentID: deployment.ID,
			Step:         step,
			EndedAt:      sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			Error:        sql.NullString{Valid: stepErr != nil, String: truncateString(fault.UserFacingMessage(stepErr), 512)},
		})
	}, restate.WithName(fmt.Sprintf("ending step: %s", step)), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return err
	}

	return stepErr
}

// truncateString returns s unchanged if it fits within maxLen bytes, otherwise
// truncates it and appends "..." so the result is exactly maxLen bytes.
// Depot/BuildKit errors can be really lengthy so capping
// the length keeps Restate journal entries and stored error messages manageable.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
