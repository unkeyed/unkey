package lease

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
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

// acquire claims expired leases in bounded batches until no full candidate
// batch remains. Each atomic update gets a new fencing token.
func (s *Service) acquire(ctx context.Context) (int, error) {
	total := 0
	for {
		ids, err := s.db.ListExpiredLogdrainCandidates(ctx, acquireBatchSize)
		if err != nil {
			return total, fmt.Errorf("list expired logdrain leases: %w", err)
		}
		for _, logdrainID := range ids {
			rows, acquireErr := s.db.AcquireLogdrainLease(ctx, db.AcquireLogdrainLeaseParams{
				LeaseID:      s.leaseID,
				FencingToken: uid.New(""),
				TtlMillis:    leaseTTL().Milliseconds(),
				LogdrainID:   logdrainID,
			})
			if acquireErr != nil {
				return total, fmt.Errorf("acquire logdrain lease %s: %w", logdrainID, acquireErr)
			}
			if rows == 1 {
				total++
			}
		}
		if len(ids) < acquireBatchSize {
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
