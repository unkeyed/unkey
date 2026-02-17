package deploy

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/db"
)

// updateDeploymentStatus updates the status of a deployment in the database.
//
// This is a durable operation wrapped in restate.Run to ensure the status update
// is persisted even if the workflow is interrupted. Status updates are critical
// for tracking deployment progress and handling failures.
func (w *Workflow) updateDeploymentStatus(ctx restate.WorkflowSharedContext, deploymentID string, status db.DeploymentsStatus) error {
	_, err := restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		updateErr := db.Query.UpdateDeploymentStatus(stepCtx, w.db.RW(), db.UpdateDeploymentStatusParams{
			ID:        deploymentID,
			Status:    status,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if updateErr != nil {
			return restate.Void{}, fmt.Errorf("failed to update version status to building: %w", updateErr)
		}
		return restate.Void{}, nil
	}, restate.WithName(fmt.Sprintf("updating deployment status to %s", status)))
	return err
}

// startDeploymentStep records the start of a deployment step.
func (w *Workflow) startDeploymentStep(
	ctx restate.WorkflowSharedContext,
	deployment db.Deployment,
	step db.DeploymentStepsStep,
) error {
	return restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.InsertDeploymentStep(runCtx, w.db.RW(), db.InsertDeploymentStepParams{
			WorkspaceID:   deployment.WorkspaceID,
			ProjectID:     deployment.ProjectID,
			EnvironmentID: deployment.EnvironmentID,
			DeploymentID:  deployment.ID,
			Step:          step,
			StartedAt:     uint64(time.Now().UnixMilli()),
		})
	}, restate.WithName(fmt.Sprintf("start deployment step %s", step)))
}

// endDeploymentStep marks a deployment step as completed, optionally recording an error.
func (w *Workflow) endDeploymentStep(
	ctx restate.WorkflowSharedContext,
	deploymentID string,
	step db.DeploymentStepsStep,
	errorMessage *string,
) error {
	return restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		errStr := sql.NullString{}
		if errorMessage != nil {
			errStr = sql.NullString{Valid: true, String: *errorMessage}
		}
		return db.Query.EndDeploymentStep(runCtx, w.db.RW(), db.EndDeploymentStepParams{
			DeploymentID: deploymentID,
			Step:         step,
			EndedAt:      sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			Error:        errStr,
		})
	}, restate.WithName(fmt.Sprintf("end deployment step %s", step)))
}
