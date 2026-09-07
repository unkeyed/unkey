package deploy

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/deploymentretention"
)

// GarbageCollect soft deletes expired deployments, then purges them after recovery expires.
func (w *Workflow) GarbageCollect(ctx restate.ObjectContext, req *hydrav1.GarbageCollectDeploymentRequest) (*hydrav1.GarbageCollectDeploymentResponse, error) {
	deploymentID := req.GetDeploymentId()
	if deploymentID == "" {
		return nil, restate.TerminalErrorf("deployment ID is required")
	}
	if restate.Key(ctx) != deploymentID {
		return nil, restate.TerminalErrorf("deployment ID must match the object key")
	}

	deleted, err := restate.Run(ctx, func(runCtx restate.RunContext) (bool, error) {
		return w.collectDeployment(runCtx, deploymentID, time.Now())
	}, restate.WithName("recheck and delete expired deployment"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fmt.Errorf("garbage collect deployment %s: %w", deploymentID, err)
	}

	return &hydrav1.GarbageCollectDeploymentResponse{Deleted: deleted}, nil
}

func (w *Workflow) collectDeployment(ctx context.Context, deploymentID string, now time.Time) (bool, error) {
	productionCutoff, previewCutoff := deploymentretention.Cutoffs(now)
	return db.TxWithResult(ctx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) (bool, error) {
		queries := db.NewQueries(tx)
		deployment, findErr := queries.FindDeploymentGCEligible(txCtx, db.FindDeploymentGCEligibleParams{
			DeploymentID:     deploymentID,
			PreviewCutoff:    previewCutoff,
			ProductionCutoff: productionCutoff,
			KeepSuccessful:   deploymentretention.Successful,
			RecoveryCutoff:   sql.NullInt64{Int64: now.Add(-deploymentretention.RecoveryAge).UnixMilli(), Valid: true},
		})
		if findErr != nil {
			if db.IsNotFound(findErr) {
				return false, nil
			}
			return false, findErr
		}

		if !deployment.DeletedAt.Valid {
			_, deleteErr := queries.SoftDeleteDeploymentForGC(txCtx, db.SoftDeleteDeploymentForGCParams{
				DeploymentID: deploymentID,
				DeletedAt:    sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
			})
			return false, deleteErr
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
}

// Restore makes a recoverable deployment visible without starting compute or changing routes.
func (w *Workflow) Restore(ctx restate.ObjectContext, req *hydrav1.RestoreDeploymentRequest) (*hydrav1.RestoreDeploymentResponse, error) {
	if req.GetDeploymentId() == "" || req.GetDeploymentId() != restate.Key(ctx) {
		return nil, restate.TerminalErrorf("deployment ID must match the object key")
	}
	rows, err := restate.Run(ctx, func(runCtx restate.RunContext) (int64, error) {
		now := time.Now()
		return w.db.RestoreDeployment(runCtx, db.RestoreDeploymentParams{
			DeploymentID:   req.GetDeploymentId(),
			RestoredAt:     sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
			RecoveryCutoff: sql.NullInt64{Int64: now.Add(-deploymentretention.RecoveryAge).UnixMilli(), Valid: true},
		})
	}, restate.WithName("restore deployment during recovery"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fmt.Errorf("restore deployment: %w", err)
	}
	return &hydrav1.RestoreDeploymentResponse{Restored: rows == 1}, nil
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
