package environment

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Restore clears the environment's deletion_id only if it equals
// req.deletion_id. Does not revive deployment status — the user is
// expected to trigger a fresh deployment after restore.
//
// Key: environment_id
func (s *Service) Restore(
	ctx restate.ObjectContext,
	req *hydrav1.RestoreEnvironmentRequest,
) (*hydrav1.RestoreEnvironmentResponse, error) {
	envID := restate.Key(ctx)
	expectedDeletionID := req.GetDeletionId()

	env, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Environment, error) {
		return db.Query.FindEnvironmentAnyById(runCtx, s.db.RO(), envID)
	}, restate.WithName("find environment"))
	if err != nil {
		return nil, fmt.Errorf("find environment: %w", err)
	}

	if !env.DeletionID.Valid {
		return &hydrav1.RestoreEnvironmentResponse{}, nil
	}

	if env.DeletionID.String != expectedDeletionID {
		logger.Info("environment restore skipped: deletion_id mismatch",
			"environment_id", envID,
			"expected", expectedDeletionID,
			"actual", env.DeletionID.String,
		)
		return &hydrav1.RestoreEnvironmentResponse{}, nil
	}

	logger.Info("environment restore",
		"environment_id", envID,
		"deletion_id", expectedDeletionID,
	)

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		// CAS: only clear if deletion_id still matches the cascade.
		_, err := db.Query.UpdateEnvironmentDeletionId(runCtx, s.db.RW(), db.UpdateEnvironmentDeletionIdParams{
			ID:                 envID,
			ExpectedDeletionID: sql.NullString{Valid: true, String: expectedDeletionID},
			DeletionID:         sql.NullString{Valid: false},
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		return err
	}, restate.WithName("clear environment deletion id")); err != nil {
		return nil, fmt.Errorf("clear environment deletion id: %w", err)
	}

	return &hydrav1.RestoreEnvironmentResponse{}, nil
}
