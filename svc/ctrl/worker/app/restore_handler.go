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

// Restore clears the app's deletion_id only if it equals
// req.deletion_id. The match guards against restoring an app that was
// independently deleted (it would carry a different deletion_id from
// its own restore tree). Cascades to environments with the same id.
//
// Key: app_id
func (s *Service) Restore(
	ctx restate.ObjectContext,
	req *hydrav1.RestoreAppRequest,
) (*hydrav1.RestoreAppResponse, error) {
	appID := restate.Key(ctx)
	expectedDeletionID := req.GetDeletionId()

	app, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.App, error) {
		return db.Query.FindAppAnyById(runCtx, s.db.RO(), appID)
	}, restate.WithName("find app"))
	if err != nil {
		return nil, fmt.Errorf("find app: %w", err)
	}

	if !app.DeletionID.Valid {
		// Already live (or already restored as part of this cascade).
		return &hydrav1.RestoreAppResponse{}, nil
	}

	if app.DeletionID.String != expectedDeletionID {
		// Independently deleted at a different id. Leave it alone.
		logger.Info("app restore skipped: deletion_id mismatch",
			"app_id", appID,
			"expected", expectedDeletionID,
			"actual", app.DeletionID.String,
		)
		return &hydrav1.RestoreAppResponse{}, nil
	}

	logger.Info("app restore",
		"app_id", appID,
		"deletion_id", expectedDeletionID,
	)

	envIDs, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]string, error) {
		return db.Query.ListEnvironmentIdsByApp(runCtx, s.db.RO(), appID)
	}, restate.WithName("list all environments"))
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}

	for _, envID := range envIDs {
		hydrav1.NewEnvironmentServiceClient(ctx, envID).
			Restore().
			Send(&hydrav1.RestoreEnvironmentRequest{DeletionId: expectedDeletionID})
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		// CAS guards against a concurrent re-delete: only clear the
		// deletion_id if it still matches the value the cascade is
		// restoring. A 0-affected result means the app was re-deleted
		// at a different id; leave it alone.
		_, err := db.Query.UpdateAppDeletionId(runCtx, s.db.RW(), db.UpdateAppDeletionIdParams{
			ID:                 appID,
			ExpectedDeletionID: sql.NullString{Valid: true, String: expectedDeletionID},
			DeletionID:         sql.NullString{Valid: false},
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		return err
	}, restate.WithName("clear app deletion id")); err != nil {
		return nil, fmt.Errorf("clear app deletion id: %w", err)
	}

	return &hydrav1.RestoreAppResponse{}, nil
}
