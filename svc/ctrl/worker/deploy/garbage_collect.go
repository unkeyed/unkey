package deploy

import (
	"context"
	"database/sql"
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/deploymentretention"
)

// GarbageCollect deletes an expired terminal deployment. The DeployService is
// keyed by deployment ID, so this recheck and deletion serialize with its
// deploy, stop, and wake lifecycle operations.
func (w *Workflow) GarbageCollect(
	ctx restate.ObjectContext,
	req *hydrav1.GarbageCollectDeploymentRequest,
) (*hydrav1.GarbageCollectDeploymentResponse, error) {
	deploymentID := req.GetDeploymentId()
	if deploymentID == "" {
		return nil, restate.TerminalErrorf("deployment ID is required")
	}
	if restate.Key(ctx) != deploymentID {
		return nil, restate.TerminalErrorf("deployment ID must match the object key")
	}

	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get garbage collection time: %w", err)
	}
	productionCutoff, previewCutoff := deploymentretention.Cutoffs(now)

	deleted, err := restate.Run(ctx, func(runCtx restate.RunContext) (bool, error) {
		return db.TxWithResult(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) (bool, error) {
			queries := db.NewQueries(tx)
			_, findErr := queries.FindDeploymentGCEligible(txCtx, db.FindDeploymentGCEligibleParams{
				DeploymentID:     deploymentID,
				PreviewCutoff:    previewCutoff,
				ProductionCutoff: productionCutoff,
				KeepSuccessful:   deploymentretention.Successful,
			})
			if findErr != nil {
				if db.IsNotFound(findErr) {
					return false, nil
				}
				return false, findErr
			}

			if deleteErr := deleteDeploymentChildren(txCtx, queries, deploymentID); deleteErr != nil {
				return false, deleteErr
			}
			rows, deleteErr := queries.DeleteDeploymentByIDForGC(txCtx, deploymentID)
			if deleteErr != nil {
				return false, deleteErr
			}
			return rows == 1, nil
		})
	}, restate.WithName("recheck and delete expired deployment"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fmt.Errorf("garbage collect deployment %s: %w", deploymentID, err)
	}

	return &hydrav1.GarbageCollectDeploymentResponse{Deleted: deleted}, nil
}

func deleteDeploymentChildren(ctx context.Context, queries *db.Queries, deploymentID string) error {
	deleteQueries := []func(context.Context, string) error{
		queries.DeleteFrontlineRoutesByDeploymentID,
		queries.DeleteCiliumNetworkPoliciesByDeploymentID,
		queries.DeleteDeploymentTopologiesByDeploymentID,
		queries.DeleteDeploymentStepsByDeploymentID,
		queries.DeleteDeploymentChangesByDeploymentID,
	}
	for _, deleteQuery := range deleteQueries {
		if err := deleteQuery(ctx, deploymentID); err != nil {
			return err
		}
	}
	return queries.DeleteOpenAPISpecsByDeploymentID(ctx, sql.NullString{String: deploymentID, Valid: true})
}
