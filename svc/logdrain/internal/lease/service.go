package lease

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/repeat"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/logdrain/internal/db"
)

const (
	acquireInterval  = 5 * time.Second
	refreshInterval  = 30 * time.Second
	minimumTTL       = 90 * time.Second
	ttlJitter        = 30 * time.Second
	acquireBatchSize = 100
)

// Config provides the dependencies and routing identity required by [Service].
type Config struct {
	// DB persists lease ownership and fencing tokens.
	DB db.Database
	// LeaseID routes acquired leases to the poller in this process. It must be
	// unique among running processes.
	LeaseID string
	// Clock supplies loop cadence. Nil uses the system clock.
	Clock clock.Clock
}

// Service acquires and refreshes log drain leases for one process. Delivery
// runs separately and polls by lease ID.
type Service struct {
	db      db.Database
	leaseID string
	clock   clock.Clock
}

// acquireBatchResult carries committed acquisitions and controls bounded scans.
type acquireBatchResult struct {
	candidates int
	acquired   int
}

// New builds a lease service without starting its background loops.
func New(cfg Config) (*Service, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotEmpty(cfg.LeaseID, "lease ID must not be empty"),
	); err != nil {
		return nil, err
	}
	if cfg.Clock == nil {
		cfg.Clock = clock.New()
	}
	return &Service{
		db:      cfg.DB,
		leaseID: cfg.LeaseID,
		clock:   cfg.Clock,
	}, nil
}

// Run starts independent fixed-cadence acquisition and refresh loops.
// Refresh does not wait for an acquisition scan to finish.
func (s *Service) Run(ctx context.Context) error {
	stopAcquire := repeat.EveryClock(s.clock, acquireInterval, func() {
		acquired, err := s.acquire(ctx)
		if err != nil {
			if ctx.Err() == nil {
				logger.Error("logdrain lease acquisition failed", "error", err, "lease_id", s.leaseID)
			}
			return
		}
		if acquired > 0 {
			logger.Info("logdrain leases acquired", "lease_id", s.leaseID, "leases", acquired)
		}
	})
	defer stopAcquire()
	stopRefresh := repeat.EveryClock(s.clock, refreshInterval, func() {
		if err := s.refresh(ctx); err != nil && ctx.Err() == nil {
			logger.Error("logdrain lease refresh failed", "error", err, "lease_id", s.leaseID)
		}
	})
	defer stopRefresh()
	<-ctx.Done()
	return nil
}

// acquire claims expired leases in short, bounded transactions until no full
// candidate batch remains. Each acquisition gets a new fencing token.
func (s *Service) acquire(ctx context.Context) (int, error) {
	total := 0
	for {
		batch, err := mysql.TxWithResultRetry(ctx, s.db.Conn(), func(txCtx context.Context, tx mysql.DBTX) (acquireBatchResult, error) {
			queries := db.NewQueries(tx)
			ids, listErr := queries.ListExpiredLogdrainCandidates(txCtx, acquireBatchSize)
			if listErr != nil {
				return acquireBatchResult{}, fmt.Errorf("list expired logdrain leases: %w", listErr)
			}
			if len(ids) == 0 {
				return acquireBatchResult{candidates: 0, acquired: 0}, nil
			}
			lockedIDs, lockErr := queries.LockEnabledLogdrainsForUpdate(txCtx, ids)
			if lockErr != nil {
				return acquireBatchResult{}, fmt.Errorf("lock logdrain configs for lease acquisition: %w", lockErr)
			}
			result := acquireBatchResult{
				candidates: len(lockedIDs),
				acquired:   0,
			}
			for _, logdrainID := range lockedIDs {
				fencingToken := uid.New("")
				rows, acquireErr := queries.AcquireLogdrainLease(txCtx, db.AcquireLogdrainLeaseParams{
					LeaseID:      s.leaseID,
					FencingToken: fencingToken,
					TtlMillis:    leaseTTL().Milliseconds(),
					LogdrainID:   logdrainID,
				})
				if acquireErr != nil {
					return acquireBatchResult{}, fmt.Errorf("acquire logdrain lease %s: %w", logdrainID, acquireErr)
				}
				// One affected row means this transaction acquired the lease.
				if rows == 1 {
					result.acquired++
				}
			}
			return result, nil
		})
		if err != nil {
			return total, err
		}
		total += batch.acquired
		if batch.candidates < acquireBatchSize {
			// A partial batch reached the end of the leases that were eligible for this scan.
			return total, nil
		}
	}
}

// refresh replaces the expiry for all valid leases assigned to this process.
// The startup-unique lease ID is sufficient for renewal, while delivery state
// writes still require each drain's fencing token.
func (s *Service) refresh(ctx context.Context) error {
	_, err := s.db.RefreshLogdrainLeases(ctx, db.RefreshLogdrainLeasesParams{
		MinimumTtlMillis: minimumTTL.Milliseconds(),
		TtlJitterMillis:  ttlJitter.Milliseconds(),
		LeaseID:          s.leaseID,
	})
	if err != nil {
		return fmt.Errorf("refresh logdrain leases: %w", err)
	}
	return nil
}

// leaseTTL adds positive expiry jitter so leases do not churn in one batch.
func leaseTTL() time.Duration {
	return minimumTTL + time.Duration(rand.Int64N(int64(ttlJitter)+1))
}
