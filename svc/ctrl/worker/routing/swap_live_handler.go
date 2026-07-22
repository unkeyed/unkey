package routing

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
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

	// Lock the target and update routes plus the live pointer in one transaction.
	// Cleanup locks the same deployment row before pruning, so either routing
	// observes a valid ready target or cleanup completes first and routing fails.
	previous, err := restate.Run(ctx, func(runCtx restate.RunContext) (sql.NullString, error) {
		return db.TxWithResult(runCtx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) (sql.NullString, error) {
			queries := db.NewQueries(tx)
			deployment, findErr := queries.FindDeploymentByIdForUpdate(txCtx, deploymentID)
			if findErr != nil {
				return sql.NullString{}, fmt.Errorf("find target deployment: %w", findErr)
			}
			if deployment.Status != mysqltype.DeploymentsStatusReady || deployment.DesiredState != mysqltype.DeploymentsDesiredStateRunning {
				return sql.NullString{}, restate.TerminalError(fmt.Errorf(
					"target deployment must be ready and running, got status=%s desired_state=%s",
					deployment.Status,
					deployment.DesiredState,
				), 400)
			}
			currentApp, findErr := queries.FindAppById(txCtx, deployment.AppID)
			if findErr != nil {
				return sql.NullString{}, fmt.Errorf("find app: %w", findErr)
			}

			now := sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()}
			for _, frontlineRouteID := range req.GetFrontlineRouteIds() {
				if routeErr := queries.ReassignFrontlineRoute(txCtx, db.ReassignFrontlineRouteParams{
					ID:           frontlineRouteID,
					DeploymentID: deploymentID,
					UpdatedAt:    now,
				}); routeErr != nil {
					return sql.NullString{}, fmt.Errorf("reassign frontline route %s: %w", frontlineRouteID, routeErr)
				}
			}

			updateErr := queries.UpdateAppDeployments(txCtx, db.UpdateAppDeploymentsParams{
				AppID:               deployment.AppID,
				CurrentDeploymentID: sql.NullString{Valid: true, String: deploymentID},
				IsRolledBack:        req.GetSetRollbackFlag(),
				UpdatedAt:           now,
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
