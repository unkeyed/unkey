// Package clickhouseuserreconcile keeps existing workspace ClickHouse users
// aligned with the allowed-table configuration compiled into control-worker.
package clickhouseuserreconcile

import (
	"errors"
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const (
	stateKeyAllowedTablesFingerprint = "allowed_tables_fingerprint"
	reconfigureBatchSize             = 10
)

// Config holds the handler's dependencies.
type Config struct {
	// DB lists workspaces that already have ClickHouse users.
	DB db.Database
}

// Handler executes RunClickhouseUserReconcile.
type Handler struct {
	db db.Database
}

// New constructs a Handler.
func New(cfg Config) (*Handler, error) {
	if err := assert.NotNil(cfg.DB, "DB must not be nil"); err != nil {
		return nil, err
	}

	return &Handler{db: cfg.DB}, nil
}

// Handle reconfigures all existing users once for each distinct allowed-table
// fingerprint. Successful runs store the fingerprint in Restate state; later
// ticks become a state read until the desired grants change again.
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunClickhouseUserReconcileRequest,
) (*hydrav1.RunClickhouseUserReconcileResponse, error) {
	fingerprint := clickhouse.DefaultAllowedTablesFingerprint()
	appliedFingerprint, err := restate.Get[string](ctx, stateKeyAllowedTablesFingerprint)
	if err != nil {
		return nil, fmt.Errorf("get applied allowed-table fingerprint: %w", err)
	}
	if appliedFingerprint == fingerprint {
		return &hydrav1.RunClickhouseUserReconcileResponse{UsersReconfigured: 0}, nil
	}

	workspaceIDs, err := restate.Run(ctx, func(rc restate.RunContext) ([]string, error) {
		return h.db.ListClickhouseWorkspaceIDs(rc)
	}, restate.WithName("list clickhouse workspace users"))
	if err != nil {
		return nil, fmt.Errorf("list clickhouse workspace users: %w", err)
	}

	logger.Info("reconfiguring clickhouse workspace users",
		"users", len(workspaceIDs),
		"previous_fingerprint", appliedFingerprint,
		"desired_fingerprint", fingerprint,
	)

	for batchStart := 0; batchStart < len(workspaceIDs); batchStart += reconfigureBatchSize {
		batchEnd := min(batchStart+reconfigureBatchSize, len(workspaceIDs))
		futures := make([]restate.ResponseFuture[*hydrav1.ReconcileUserResponse], 0, batchEnd-batchStart)
		for _, workspaceID := range workspaceIDs[batchStart:batchEnd] {
			future := hydrav1.NewClickhouseUserServiceClient(ctx, workspaceID).
				ReconcileUser().
				RequestFuture(&hydrav1.ReconcileUserRequest{})
			futures = append(futures, future)
		}

		var batchErr error
		for i, future := range futures {
			if _, responseErr := future.Response(); responseErr != nil {
				workspaceID := workspaceIDs[batchStart+i]
				batchErr = errors.Join(batchErr, fmt.Errorf("reconfigure workspace %s: %w", workspaceID, responseErr))
			}
		}
		if batchErr != nil {
			return nil, batchErr
		}
	}

	restate.Set(ctx, stateKeyAllowedTablesFingerprint, fingerprint)
	logger.Info("reconfigured clickhouse workspace users", "users", len(workspaceIDs), "fingerprint", fingerprint)

	return &hydrav1.RunClickhouseUserReconcileResponse{
		UsersReconfigured: int32(len(workspaceIDs)),
	}, nil
}
