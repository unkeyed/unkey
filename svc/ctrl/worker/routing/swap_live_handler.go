package routing

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type swapLiveDeploymentResult struct {
	PreviousDeploymentID         string                               `json:"previous_deployment_id"`
	CurrentDeploymentID          string                               `json:"current_deployment_id"`
	AutomaticPromotionSkipReason hydrav1.AutomaticPromotionSkipReason `json:"automatic_promotion_skip_reason"`
	AlreadyCurrent               bool                                 `json:"already_current"`
}

// SwapLiveDeployment atomically performs the operations that make a deployment
// the live one for its environment:
//
//  1. Reassign the given frontline routes to the target deployment.
//  2. Update apps.current_deployment_id to the target deployment.
//  3. Set apps.is_rolled_back per the request flag.
//
// Because the RoutingService VO is keyed by env_id, concurrent swaps on the
// same environment serialize here. Automatic deployments can require a
// newest-wins check so an older build that finishes last cannot overwrite a
// newer live deployment. Manual promotion and rollback bypass that check.
func (s *Service) SwapLiveDeployment(
	ctx restate.ObjectContext,
	req *hydrav1.SwapLiveDeploymentRequest,
) (*hydrav1.SwapLiveDeploymentResponse, error) {
	deploymentID := req.GetDeploymentId()

	// Journal the current deployment before the mutating transaction. If the
	// transaction commits but its result is lost, Restate replays this value so
	// the real previous deployment can still be scheduled for standby.
	previousDeploymentID := ""
	if req.GetAutomaticPromotion() {
		var err error
		previousDeploymentID, err = restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
			deployment, findErr := s.db.FindDeploymentById(runCtx, deploymentID)
			if findErr != nil {
				return "", fmt.Errorf("find target deployment: %w", findErr)
			}
			app, findErr := s.db.FindAppById(runCtx, deployment.AppID)
			if findErr != nil {
				return "", fmt.Errorf("find app: %w", findErr)
			}
			return app.CurrentDeploymentID.String, nil
		}, restate.WithName("record current deployment before automatic promotion"))
		if err != nil {
			return nil, err
		}
	}

	result, err := restate.Run(ctx, func(runCtx restate.RunContext) (swapLiveDeploymentResult, error) {
		return db.TxWithResult(runCtx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) (swapLiveDeploymentResult, error) {
			deployment, findErr := db.NewQueries(tx).FindDeploymentById(txCtx, deploymentID)
			if findErr != nil {
				return swapLiveDeploymentResult{}, fmt.Errorf("find target deployment: %w", findErr)
			}
			currentApp, findErr := db.NewQueries(tx).FindAppById(txCtx, deployment.AppID)
			if findErr != nil {
				return swapLiveDeploymentResult{}, fmt.Errorf("find app: %w", findErr)
			}

			if req.GetAutomaticPromotion() &&
				currentApp.CurrentDeploymentID.Valid &&
				currentApp.CurrentDeploymentID.String == deploymentID {
				// The transaction may have committed before Restate persisted its
				// result. A replay recovers the journaled previous deployment. A
				// separate duplicate invocation journals the target and returns no
				// previous deployment, so it cannot schedule the target for standby.
				recoveredPreviousDeploymentID := previousDeploymentID
				if recoveredPreviousDeploymentID == deploymentID {
					recoveredPreviousDeploymentID = ""
				}
				return swapLiveDeploymentResult{
					PreviousDeploymentID:         recoveredPreviousDeploymentID,
					CurrentDeploymentID:          deploymentID,
					AutomaticPromotionSkipReason: hydrav1.AutomaticPromotionSkipReason_AUTOMATIC_PROMOTION_SKIP_REASON_UNSPECIFIED,
					AlreadyCurrent:               true,
				}, nil
			}

			if req.GetAutomaticPromotion() && currentApp.IsRolledBack {
				return swapLiveDeploymentResult{
					PreviousDeploymentID:         "",
					CurrentDeploymentID:          currentApp.CurrentDeploymentID.String,
					AutomaticPromotionSkipReason: hydrav1.AutomaticPromotionSkipReason_AUTOMATIC_PROMOTION_SKIP_REASON_ROLLED_BACK,
					AlreadyCurrent:               false,
				}, nil
			}

			if req.GetAutomaticPromotion() && currentApp.CurrentDeploymentID.Valid {
				currentDeployment, currentErr := db.NewQueries(tx).FindDeploymentById(txCtx, currentApp.CurrentDeploymentID.String)
				if currentErr != nil {
					return swapLiveDeploymentResult{}, fmt.Errorf("find current deployment: %w", currentErr)
				}
				// pk is the database insertion order. Unlike millisecond timestamps,
				// it provides a strict order when deployments are created together.
				if currentDeployment.Pk > deployment.Pk {
					return swapLiveDeploymentResult{
						PreviousDeploymentID:         "",
						CurrentDeploymentID:          currentDeployment.ID,
						AutomaticPromotionSkipReason: hydrav1.AutomaticPromotionSkipReason_AUTOMATIC_PROMOTION_SKIP_REASON_NEWER_DEPLOYMENT,
						AlreadyCurrent:               false,
					}, nil
				}
			}

			now := sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()}
			for _, frontlineRouteID := range req.GetFrontlineRouteIds() {
				if reassignErr := db.NewQueries(tx).ReassignFrontlineRoute(txCtx, db.ReassignFrontlineRouteParams{
					ID:           frontlineRouteID,
					DeploymentID: deploymentID,
					UpdatedAt:    now,
				}); reassignErr != nil {
					return swapLiveDeploymentResult{}, fmt.Errorf("reassign frontline route %q: %w", frontlineRouteID, reassignErr)
				}
			}

			updateErr := db.NewQueries(tx).UpdateAppDeployments(txCtx, db.UpdateAppDeploymentsParams{
				AppID:               deployment.AppID,
				CurrentDeploymentID: sql.NullString{Valid: true, String: deploymentID},
				IsRolledBack:        req.GetSetRollbackFlag(),
				UpdatedAt:           now,
			})
			if updateErr != nil {
				return swapLiveDeploymentResult{}, fmt.Errorf("update app deployments: %w", updateErr)
			}

			return swapLiveDeploymentResult{
				PreviousDeploymentID:         currentApp.CurrentDeploymentID.String,
				CurrentDeploymentID:          deploymentID,
				AutomaticPromotionSkipReason: hydrav1.AutomaticPromotionSkipReason_AUTOMATIC_PROMOTION_SKIP_REASON_UNSPECIFIED,
				AlreadyCurrent:               false,
			}, nil
		})
	}, restate.WithName("swap live deployment"))
	if err != nil {
		return nil, err
	}

	if result.AutomaticPromotionSkipReason != hydrav1.AutomaticPromotionSkipReason_AUTOMATIC_PROMOTION_SKIP_REASON_UNSPECIFIED {
		logger.Info("skipped automatic live deployment swap",
			"env_id", restate.Key(ctx),
			"deployment_id", deploymentID,
			"current_deployment_id", result.CurrentDeploymentID,
			"reason", result.AutomaticPromotionSkipReason.String(),
		)
		return &hydrav1.SwapLiveDeploymentResponse{
			AutomaticPromotionSkipReason: result.AutomaticPromotionSkipReason,
		}, nil
	}
	if result.AlreadyCurrent {
		logger.Info("automatic deployment is already live",
			"env_id", restate.Key(ctx),
			"deployment_id", deploymentID,
			"previous_deployment_id", result.PreviousDeploymentID,
		)
		return &hydrav1.SwapLiveDeploymentResponse{
			PreviousDeploymentId: result.PreviousDeploymentID,
		}, nil
	}

	logger.Info("swapped live deployment",
		"env_id", restate.Key(ctx),
		"new_deployment_id", deploymentID,
		"previous_deployment_id", result.PreviousDeploymentID,
		"is_rolled_back", req.GetSetRollbackFlag(),
	)

	return &hydrav1.SwapLiveDeploymentResponse{
		PreviousDeploymentId: result.PreviousDeploymentID,
	}, nil
}
