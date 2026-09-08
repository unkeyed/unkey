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
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/logger"
)

// batchLimit bounds each DELETE. PlanetScale rejects a single DML statement
// that would affect more than 100,000 rows, so an unbounded DELETE fails
// outright once a backlog builds; the handler loops until a batch deletes fewer
// than this. Sized well under the ceiling so row locks stay short and
// replication lag stays bounded.
const batchLimit int32 = 25_000

// maxBatches caps how much one invocation drains. The handler is a
// singleton-keyed VO on an hourly schedule, so an unbounded loop over a large
// backlog would both grow the journal without limit and keep later ticks queued
// behind it. Leftover rows are only a storage concern: the hot path filters on
// expires_at, so they are never read as live counters, and the next tick
// continues where this one stopped.
const maxBatches = 40

// Config holds the handler's dependencies.
type Config struct {
	// DB is the ratelimit database. Must not be nil.
	DB *rldb.Database
	// Clock provides the cutoff timestamp. Must not be nil.
	Clock clock.Clock
	// Heartbeat is pinged after a successful sweep. Must not be nil; use
	// healthcheck.NewNoop() if monitoring is not configured.
	Heartbeat healthcheck.Heartbeat
}

// Handler executes RunRatelimitGlobalCountersCleanup.
type Handler struct {
	db        *rldb.Database
	clock     clock.Clock
	heartbeat healthcheck.Heartbeat
}

// New constructs a Handler.
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Clock, "Clock must not be nil"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &Handler{db: cfg.DB, clock: cfg.Clock, heartbeat: cfg.Heartbeat}, nil
}

// Handle deletes ratelimit_global_counters rows whose expires_at is in the
// past relative to h.clock, in bounded batches. Each batch DELETE is wrapped
// in restate.Run so a crash or retry replays cleanly: at-least-once delivery
// on a deterministic, cutoff-bounded DELETE is safe — re-running only removes
// rows that were already eligible.
//
// Stateless — the VO key is fixed at "ratelimit-global-counters-cleanup" so a
// wedged invocation cannot block other cron handlers. It does block later ticks
// of this handler, which is why the registered retry policy kills rather than
// pauses on exhaustion. Local in-memory state in the ratelimit service is
// cleaned by its own janitor, and the hot path filters on expires_at, so the
// lag between this cron firing and rows actually disappearing is only a storage
// concern, not a correctness one.
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunRatelimitGlobalCountersCleanupRequest,
) (*hydrav1.RunRatelimitGlobalCountersCleanupResponse, error) {
	cutoff := h.clock.Now().UnixMilli()

	var totalDeleted int64
	drained := false
	for batchNum := range maxBatches {
		deleted, err := restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
			return h.db.RW().GlobalCountersDeleteExpired(rc, rldb.GlobalCountersDeleteExpiredParams{
				Cutoff: uint64(cutoff),
				Limit:  batchLimit,
			})
		}, restate.WithName(fmt.Sprintf("delete batch-%d", batchNum)))
		if err != nil {
			return nil, fmt.Errorf("delete expired global counter rows batch %d: %w", batchNum, err)
		}

		totalDeleted += deleted

		if deleted < int64(batchLimit) {
			drained = true
			break
		}
	}

	if !drained {
		logger.Warn("ratelimit global counters cleanup hit batch cap; expired rows remain for the next tick",
			"rows_deleted", totalDeleted,
			"max_batches", maxBatches,
			"batch_limit", batchLimit,
			"cutoff_ms", cutoff,
		)
	}

	logger.Info("ratelimit global counters cleanup complete",
		"rows_deleted", totalDeleted,
		"drained", drained,
		"cutoff_ms", cutoff,
	)

	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	return &hydrav1.RunRatelimitGlobalCountersCleanupResponse{
		RowsDeleted: totalDeleted,
	}, nil
}
