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
