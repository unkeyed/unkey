// Package ratelimitblocklistcleanup implements the Restate handler that
// prunes expired rows from ratelimit_blocklist. The table is the cross-region
// propagation channel for rate-limit denials; without periodic cleanup it
// would grow unbounded under sustained abuse and slow down BlocklistListActive
// scans on every node.
//
// The handler is intentionally minimal: a single DELETE under a Restate
// step. No fan-out, no state. Local in-memory state in the ratelimit service
// is cleaned by its own janitor, and the hot path filters on expires_at,
// so the lag between this cron firing and rows actually disappearing is
// only a storage concern, not a correctness one.
package ratelimitblocklistcleanup

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"

	rldb "github.com/unkeyed/unkey/internal/services/ratelimit/db"
)

// Service implements [hydrav1.RatelimitBlocklistCleanupServiceServer].
// It owns no state; each invocation is independent.
type Service struct {
	hydrav1.UnimplementedRatelimitBlocklistCleanupServiceServer
	db    *rldb.Database
	clock clock.Clock
}

var _ hydrav1.RatelimitBlocklistCleanupServiceServer = (*Service)(nil)

// Config holds dependencies for the cleanup service.
type Config struct {
	// DB is the wrapped ratelimit_blocklist database. Must not be nil.
	DB *rldb.Database
	// Clock provides the cutoff timestamp for expired rows. Optional; defaults
	// to a real clock. Tests should inject a fake clock to drive cutoffs.
	Clock clock.Clock
}

// New constructs a Service. Returns an error if any required field is missing.
func New(cfg Config) (*Service, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
	); err != nil {
		return nil, err
	}
	if cfg.Clock == nil {
		cfg.Clock = clock.New()
	}

	return &Service{
		UnimplementedRatelimitBlocklistCleanupServiceServer: hydrav1.UnimplementedRatelimitBlocklistCleanupServiceServer{},
		db:    cfg.DB,
		clock: cfg.Clock,
	}, nil
}
