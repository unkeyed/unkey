package routing

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// SwapLiveDeployment atomically performs the three operations that make a
// deployment the live one for its environment:
//
//  1. Reassign the given frontline routes to the target deployment.
//  2. Update apps.current_deployment_id to the target deployment.
//  3. Set apps.is_rolled_back per the request flag.
//
// Because the RoutingService VO is keyed by env_id, concurrent swaps on the
// same environment serialize here. The handler returns the previous live
// deployment ID so the caller can schedule it for standby outside the
// atomic section (ScheduleDesiredStateChange is itself idempotent).
func (s *Service) SwapLiveDeployment(
	ctx restate.ObjectContext,
	req *hydrav1.SwapLiveDeploymentRequest,
) (*hydrav1.SwapLiveDeploymentResponse, error) {
	deploymentID := req.GetDeploymentId()

	// Reassign routes first — if the update fails, the live-deployment
	// marker stays pointing at the previous deployment so traffic is
	// unaffected.
	for _, frontlineRouteID := range req.GetFrontlineRouteIds() {
		_, err := restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
			return restate.Void{}, db.Query.ReassignFrontlineRoute(stepCtx, s.db.RW(), db.ReassignFrontlineRouteParams{
				ID:           frontlineRouteID,
				DeploymentID: deploymentID,
				UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
		}, restate.WithName(fmt.Sprintf("reassign-%s", frontlineRouteID)))
		if err != nil {
			return nil, err
		}
	}

	// Swap the live-deployment pointer. Reads + writes the apps row in a
	// single transaction so no other call can observe a half-applied state
	// (which matters if this handler is ever extended to touch multiple
	// rows).
	previous, err := restate.Run(ctx, func(runCtx restate.RunContext) (sql.NullString, error) {
		return db.TxWithResult(runCtx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) (sql.NullString, error) {
			deployment, findErr := db.Query.FindDeploymentById(txCtx, tx, deploymentID)
			if findErr != nil {
				return sql.NullString{}, fmt.Errorf("find target deployment: %w", findErr)
			}
			currentApp, findErr := db.Query.FindAppById(txCtx, tx, deployment.AppID)
			if findErr != nil {
				return sql.NullString{}, fmt.Errorf("find app: %w", findErr)
			}

			updateErr := db.Query.UpdateAppDeployments(txCtx, tx, db.UpdateAppDeploymentsParams{
				AppID:               deployment.AppID,
				CurrentDeploymentID: sql.NullString{Valid: true, String: deploymentID},
				IsRolledBack:        req.GetSetRollbackFlag(),
				UpdatedAt:           sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
			if updateErr != nil {
				return sql.NullString{}, fmt.Errorf("update app deployments: %w", updateErr)
			}

			return currentApp.CurrentDeploymentID, nil
		})
	}, restate.WithName("swap live deployment pointer"))
	if err != nil {
		return nil, err
	}

	logger.Info("swapped live deployment",
		"env_id", restate.Key(ctx),
		"new_deployment_id", deploymentID,
		"previous_deployment_id", previous.String,
		"is_rolled_back", req.GetSetRollbackFlag(),
	)

	return &hydrav1.SwapLiveDeploymentResponse{
		PreviousDeploymentId: previous.String,
	}, nil
}
