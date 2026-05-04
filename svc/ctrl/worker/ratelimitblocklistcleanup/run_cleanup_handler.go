package ratelimitblocklistcleanup

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
)

// RunCleanup deletes every ratelimit_blocklist row whose expires_at is in the
// past relative to s.clock. The DELETE is wrapped in restate.Run so a crash
// or retry replays cleanly: at-least-once delivery on a deterministic DELETE
// is safe.
func (s *Service) RunCleanup(
	ctx restate.Context,
	_ *hydrav1.RunCleanupRequest,
) (*hydrav1.RunCleanupResponse, error) {
	cutoff := s.clock.Now().UnixMilli()

	deleted, err := restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
		return s.db.RW().BlocklistDeleteExpired(rc, uint64(cutoff))
	}, restate.WithName("delete expired"))
	if err != nil {
		return nil, fmt.Errorf("delete expired blocklist rows: %w", err)
	}

	logger.Info("ratelimit blocklist cleanup complete",
		"rows_deleted", deleted,
		"cutoff_ms", cutoff,
	)

	return &hydrav1.RunCleanupResponse{
		RowsDeleted: deleted,
	}, nil
}
