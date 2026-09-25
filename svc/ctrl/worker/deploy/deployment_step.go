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

// DeploymentStep records a step around fn and hands fn the caller's own
// context. Deploy passes a WorkflowContext and the shared Build handler a
// WorkflowSharedContext, so it is generic over both: narrowing the parameter
// to restate.Context would compile here but would leave Deploy's step bodies
// without the workflow context waitForDeployments needs.
//
// A function rather than a method because Go has no type parameters on
// methods
func DeploymentStep[C restate.Context](
	w *Workflow,
	ctx C,
	step db.DeploymentStepsStep,
	deploymentID string,
	fn func(stepCtx C) error,
) error {
	err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		now := time.Now().UnixMilli()
		deploymentStatus := mysqltype.DeploymentsStatusPending
		switch step {
		case db.DeploymentStepsStepQueued:
			deploymentStatus = mysqltype.DeploymentsStatusPending
		case db.DeploymentStepsStepBuilding:
			deploymentStatus = mysqltype.DeploymentsStatusBuilding
		case db.DeploymentStepsStepDeploying:
			deploymentStatus = mysqltype.DeploymentsStatusDeploying
		case db.DeploymentStepsStepNetwork:
			deploymentStatus = mysqltype.DeploymentsStatusNetwork
		case db.DeploymentStepsStepFinalizing:
			deploymentStatus = mysqltype.DeploymentsStatusFinalizing
		case db.DeploymentStepsStepStarting:
			// Build goes from queued straight to building. The enum value stays
			// for deployments that recorded this step before it was dropped
			return fmt.Errorf("deployment step %s is no longer written", step)
		default:
			return fmt.Errorf("unexpected deployment step: %s", step)
		}

		return db.Tx(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
			current, err := db.NewQueries(tx).FindDeploymentForStep(txCtx, deploymentID)
			if err != nil {
				return err
			}
			// A cancel between two steps has already ended the deployment. Reviving
			// it here would let the compensation stack later mark it failed
			if current.Status.IsTerminal() {
				return restate.ToTerminalError(
					fmt.Errorf("deployment %s is already %s, not starting step %s", deploymentID, current.Status, step),
					restate.WithErrorCode(409),
				)
			}

			if err := db.NewQueries(tx).InsertDeploymentStep(txCtx, db.InsertDeploymentStepParams{
				WorkspaceID:   current.WorkspaceID,
				ProjectID:     current.ProjectID,
				AppID:         current.AppID,
				EnvironmentID: current.EnvironmentID,
				DeploymentID:  deploymentID,
				Step:          step,
				StartedAt:     uint64(now),
			}); err != nil {
				return err
			}

			return db.NewQueries(tx).UpdateDeploymentStatusIfActive(txCtx, db.UpdateDeploymentStatusIfActiveParams{
				ID:                  deploymentID,
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
			DeploymentID: deploymentID,
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
