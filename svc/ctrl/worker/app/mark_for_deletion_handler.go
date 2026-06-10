package app

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// MarkForDeletion points the app at the cascade's shared deletion_id
// and Sends MarkForDeletion to every live environment carrying the
// same id. The deletions row was already written by the cascade root
// (ProjectService.MarkForDeletion) — this handler only sets the
// per-resource reference.
//
// Idempotent: if the app already has a deletion_id, skip writes and
// cascade (a re-issued cascade is a no-op).
//
// Key: app_id
func (s *Service) MarkForDeletion(
	ctx restate.ObjectContext,
	req *hydrav1.MarkAppForDeletionRequest,
) (*hydrav1.MarkAppForDeletionResponse, error) {
	appID := restate.Key(ctx)
	deletionID := req.GetDeletionId()

	logger.Info("app mark for deletion",
		"app_id", appID,
		"deletion_id", deletionID,
	)

	app, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.App, error) {
		return db.Query.FindAppAnyById(runCtx, s.db.RO(), appID)
	}, restate.WithName("find app"))
	if err != nil {
		return nil, fmt.Errorf("find app: %w", err)
	}

	if app.DeletionID.Valid {
		return &hydrav1.MarkAppForDeletionResponse{}, nil
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		res, err := db.Query.UpdateAppDeletionId(runCtx, s.db.RW(), db.UpdateAppDeletionIdParams{
			ID:                 appID,
			ExpectedDeletionID: sql.NullString{Valid: false},
			DeletionID:         sql.NullString{Valid: true, String: deletionID},
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return fmt.Errorf("app %s: CAS lost (concurrently scheduled)", appID)
		}
		return nil
	}, restate.WithName("set app deletion id")); err != nil {
		return nil, fmt.Errorf("set app deletion id: %w", err)
	}

	envIDs, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]string, error) {
		return db.Query.ListLiveEnvironmentIdsByApp(runCtx, s.db.RO(), appID)
	}, restate.WithName("list live environments"))
	if err != nil {
		return nil, fmt.Errorf("list live environments: %w", err)
	}

	for _, envID := range envIDs {
		hydrav1.NewEnvironmentServiceClient(ctx, envID).
			MarkForDeletion().
			Send(&hydrav1.MarkEnvironmentForDeletionRequest{DeletionId: deletionID})
	}

	logger.Info("app mark for deletion cascade dispatched",
		"app_id", appID,
		"environments_cascaded", len(envIDs),
	)

	return &hydrav1.MarkAppForDeletionResponse{}, nil
}
