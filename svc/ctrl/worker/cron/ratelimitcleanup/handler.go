// Package ratelimitcleanup implements the
// CronService.RunRatelimitGlobalCountersCleanup handler. The handler
// deletes expired rows from ratelimit_global_counters so the cross-region
// propagation table stays bounded.
package ratelimitcleanup

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	rldb "github.com/unkeyed/unkey/internal/services/ratelimit/db"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Config holds the handler's dependencies.
type Config struct {
	// DB is the ratelimit database. Must not be nil.
	DB *rldb.Database
	// Clock provides the cutoff timestamp. Must not be nil.
	Clock clock.Clock
}

// Handler executes RunRatelimitGlobalCountersCleanup.
type Handler struct {
	db    *rldb.Database
	clock clock.Clock
}

// New constructs a Handler.
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Clock, "Clock must not be nil"),
	); err != nil {
		return nil, err
	}
	return &Handler{db: cfg.DB, clock: cfg.Clock}, nil
}

// Handle deletes every ratelimit_global_counters row whose expires_at is
// in the past relative to h.clock. The DELETE is wrapped in restate.Run
// so a crash or retry replays cleanly: at-least-once delivery on a
// deterministic DELETE is safe.
//
// Stateless — the VO key is fixed at "ratelimit-global-counters-cleanup"
// so a paused/wedged invocation cannot block other cron handlers. Local
// in-memory state in the ratelimit service is cleaned by its own janitor,
// and the hot path filters on expires_at, so the lag between this cron
// firing and rows actually disappearing is only a storage concern, not
// a correctness one.
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunRatelimitGlobalCountersCleanupRequest,
) (*hydrav1.RunRatelimitGlobalCountersCleanupResponse, error) {
	cutoff := h.clock.Now().UnixMilli()

	deleted, err := restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
		return h.db.RW().GlobalCountersDeleteExpired(rc, uint64(cutoff))
	}, restate.WithName("delete expired"))
	if err != nil {
		return nil, fmt.Errorf("delete expired global counter rows: %w", err)
	}

	logger.Info("ratelimit global counters cleanup complete",
		"rows_deleted", deleted,
		"cutoff_ms", cutoff,
	)

	return &hydrav1.RunRatelimitGlobalCountersCleanupResponse{
		RowsDeleted: deleted,
	}, nil
}
