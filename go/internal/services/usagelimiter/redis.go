package usagelimiter

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/counter"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
	"github.com/unkeyed/unkey/go/pkg/repeat"
)

const (
	defaultTTL          = 10 * time.Minute
	defaultReplyWorkers = 8
)

// CreditChange represents a change to a credit balance that needs to be
// replayed to the database for eventual consistency with Redis counters.
type CreditChange struct {
	// KeyID is the unique identifier of the key whose credits changed
	KeyID string
	// Amount is the number of credits that were deducted
	Amount int32
}

// RedisConfig holds configuration options for the Redis usage limiter.
type RedisConfig struct {
	// DB is the database connection for fallback and replay operations
	DB db.Database

	// Logger is the logging implementation to use.
	// Optional, but recommended for production use.
	Logger logging.Logger

	// Counter is the counter implementation to use.
	Counter counter.Counter

	// TTL for Redis keys, defaults to 10 minutes if not specified
	TTL time.Duration
}

// counterService implements usage limiting using the counter interface
// This provides truly atomic operations via Redis INCRBY commands
type counterService struct {
	db      db.Database
	logger  logging.Logger
	counter counter.Counter

	// Fallback to direct DB implementation when Redis fails
	dbFallback Service

	// Replay buffer for async DB updates
	replayBuffer *buffer.Buffer[CreditChange]

	// Circuit breaker for DB writes
	dbCircuitBreaker circuitbreaker.CircuitBreaker[any]

	// TTL for Redis keys
	ttl time.Duration
}

var _ Service = (*counterService)(nil)

// CounterConfig holds configuration options for the counter-based usage limiter
type CounterConfig struct {
	// DB is the database connection for fallback and replay operations
	DB db.Database

	// Logger is the logging implementation to use
	Logger logging.Logger

	// Counter is the distributed counter implementation to use
	Counter counter.Counter

	// TTL for Redis keys, defaults to 10 minutes if not specified
	TTL time.Duration

	// ReplayWorkers is the number of goroutines processing replay requests
	// Defaults to 8 if not specified
	ReplayWorkers int
}

// NewCounter creates a new counter-based usage limiter implementation.
//
// This implementation uses the counter interface for truly atomic operations.
// It provides the best performance and simplicity by leveraging Redis INCRBY
// for atomic credit decrements.
//
// Characteristics:
//   - Uses Redis INCRBY for atomic decrements (single operation)
//   - No race conditions - Redis handles atomicity
//   - Simple, fast code with minimal complexity
//   - Excellent performance under high contention
//   - Falls back to direct DB on Redis failures
//
// Parameters:
//   - config: Configuration options including counter implementation
//
// Returns:
//   - Service: Counter-based implementation of the Service interface
//   - error: Any initialization errors
func NewCounter(config CounterConfig) (Service, error) {
	// Validate required fields
	if config.Counter == nil {
		return nil, fmt.Errorf("counter cannot be nil")
	}
	if config.DB == nil {
		return nil, fmt.Errorf("database cannot be nil")
	}

	ttl := config.TTL
	if ttl == 0 {
		ttl = defaultTTL
	}

	// Create the direct DB fallback service
	dbFallback, err := New(Config{
		DB:     config.DB,
		Logger: config.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create DB fallback: %w", err)
	}

	s := &counterService{
		db:         config.DB,
		logger:     config.Logger,
		counter:    config.Counter,
		dbFallback: dbFallback,
		ttl:        ttl,
		replayBuffer: buffer.New[CreditChange](buffer.Config{
			Name:     "usagelimiter_replays",
			Capacity: 10_000,
			Drop:     false, // Do NOT drop credits
		}),
		dbCircuitBreaker: circuitbreaker.New[any](
			"usagelimiter_db_writes",
			circuitbreaker.WithLogger(config.Logger),
		),
	}

	// Start replay workers
	replayWorkers := config.ReplayWorkers
	if replayWorkers <= 0 {
		replayWorkers = defaultReplyWorkers
	}

	for range replayWorkers {
		go s.replayRequests()
	}

	return s, nil
}

// Limit processes a usage request and enforces credit limits using distributed Redis counters.
//
// This function implements a two-phase credit limiting system:
// 1. When key exists in redis: Atomic decrement on existing Redis counters
// 2. When key does not exist in redis: Database load + Redis initialization for new keys
// ERROR HANDLING:
// - Redis failures trigger automatic fallback to direct DB operations
// - Lua script prevents negative counters, insufficient credit attempts are denied atomically
// - Circuit breaker protects against database overload during fallbacks
//
// METRICS:
// - usagelimiter_decisions_total: Tracks allowed/denied decisions by backend
// - usagelimiter_credits_processed_total: Tracks total credits consumed
// - usagelimiter_fallback_operations_total: Tracks fallback frequency
func (s *counterService) Limit(ctx context.Context, req UsageRequest) (UsageResponse, error) {
	ctx, span := tracing.Start(ctx, "usagelimiter.counter.Limit")
	defer span.End()

	redisKey := fmt.Sprintf("credits:%s", req.KeyId)

	// Try decrement only if key usage is already stored
	remaining, exists, err := s.counter.DecrementIfExists(ctx, redisKey, int64(req.Cost))
	if err != nil {
		metrics.UsagelimiterFallbackOperations.Inc()
		return s.dbFallback.Limit(ctx, req)
	}

	// Key doesn't exist. load from db
	if !exists {
		return s.initializeFromDatabase(ctx, req, redisKey)
	}

	// Key exists, check if decrement was successful
	return s.handleDecrementResult(req, remaining)
}

// handleDecrementResult processes the result of a decrement operation
func (s *counterService) handleDecrementResult(req UsageRequest, remaining int64) (UsageResponse, error) {
	if remaining >= 0 {
		// Success - buffer the credit change for async DB update
		s.replayBuffer.Buffer(CreditChange{
			KeyID:  req.KeyId,
			Amount: req.Cost,
		})

		metrics.UsagelimiterDecisions.WithLabelValues("redis", "allowed").Inc()
		metrics.UsagelimiterCreditsProcessed.Add(float64(req.Cost))

		return UsageResponse{Valid: true, Remaining: int32(remaining)}, nil
	}

	// Insufficient credits - the Lua script already handled this case
	// and returned the current value without decrementing
	metrics.UsagelimiterDecisions.WithLabelValues("redis", "denied").Inc()

	return UsageResponse{Valid: false, Remaining: int32(remaining)}, nil
}

// initializeFromDatabase loads credits from DB and initializes the counter.
// It ensures Redis is never initialized with negative values, which could skew
// subsequent decisions from other nodes in a distributed environment.
func (s *counterService) initializeFromDatabase(ctx context.Context, req UsageRequest, redisKey string) (UsageResponse, error) {
	ctx, span := tracing.Start(ctx, "usagelimiter.counter.initializeFromDatabase")
	defer span.End()

	limit, err := db.Query.FindKeyCredits(ctx, s.db.RO(), req.KeyId)
	if err != nil {
		if db.IsNotFound(err) {
			return UsageResponse{Valid: false, Remaining: 0}, nil
		}

		return UsageResponse{Valid: false, Remaining: 0}, err
	}

	// This usage limiter should only be called for keys with limits
	// Unlimited keys (limit.Valid == false) should be handled at the key validator level
	if !limit.Valid {
		// Return valid anyway to not break existing behavior, but this is a logic error
		return UsageResponse{Valid: true, Remaining: -1}, nil
	}

	// Determine what value to initialize Redis with - never use negative values
	currentCredits := int64(limit.Int32)
	hasSufficientCredits := currentCredits >= int64(req.Cost)

	// Initialize Redis with appropriate value (current credits if insufficient, decremented if sufficient)
	var initValue int64
	if hasSufficientCredits {
		initValue = currentCredits - int64(req.Cost)
	} else {
		initValue = currentCredits // Don't initialize with negative values
	}

	wasSet, err := s.counter.SetIfNotExists(ctx, redisKey, initValue, s.ttl)
	if err != nil {
		metrics.UsagelimiterFallbackOperations.Inc()
		s.logger.Debug("failed to initialize counter with SetIfNotExists, falling back to DB", "error", err, "keyId", req.KeyId)
		return s.dbFallback.Limit(ctx, req)
	}

	// If we successfully initialized the key, return the appropriate response
	if wasSet {
		if hasSufficientCredits {
			// Successful decrement - credit change already applied in initValue
			return s.handleDecrementResult(req, initValue)
		} else {
			// Insufficient credits - return denial without decrementing
			metrics.UsagelimiterDecisions.WithLabelValues("redis", "denied").Inc()
			return UsageResponse{Valid: false, Remaining: int32(currentCredits)}, nil
		}
	}

	// Another node already initialized the key, check if we have enough after decrement
	remaining, exists, err := s.counter.DecrementIfExists(ctx, redisKey, int64(req.Cost))
	if err != nil || !exists {
		metrics.UsagelimiterFallbackOperations.Inc()
		s.logger.Debug("failed to decrement after initialization attempt", "error", err, "exists", exists, "keyId", req.KeyId)
		return s.dbFallback.Limit(ctx, req)
	}

	// Process the decrement result
	return s.handleDecrementResult(req, remaining)
}

// replayRequests processes buffered credit changes and updates the database
func (s *counterService) replayRequests() {
	for change := range s.replayBuffer.Consume() {
		err := s.syncWithDB(context.Background(), change)

		if err != nil {
			s.logger.Error("failed to replay credit change", "error", err)
		}
	}
}

func (s *counterService) syncWithDB(ctx context.Context, change CreditChange) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	start := time.Now()
	defer func() {
		metrics.UsagelimiterReplayLatency.Observe(time.Since(start).Seconds())
	}()

	_, err := s.dbCircuitBreaker.Do(ctx, func(ctx context.Context) (any, error) {
		return nil, db.Query.UpdateKeyCreditsDecrement(ctx, s.db.RW(), db.UpdateKeyCreditsDecrementParams{
			ID:      change.KeyID,
			Credits: sql.NullInt32{Int32: change.Amount, Valid: true},
		})
	})

	if err != nil {
		metrics.UsagelimiterReplayOperations.WithLabelValues("error").Inc()
		return err
	}

	metrics.UsagelimiterReplayOperations.WithLabelValues("success").Inc()
	return nil
}

// Close performs a graceful shutdown of the usage limiter service.
// It allows the replay buffer to drain pending changes before closing.
func (s *counterService) Close() error {
	ctx := context.Background()

	s.logger.Debug("beginning graceful shutdown of usage limiter")

	// Set a reasonable timeout for draining if context doesn't have one
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 30*time.Second {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	// Stop accepting new buffer writes
	s.replayBuffer.Close()

	// Wait for the buffer to drain with periodic checks using repeat package
	done := make(chan struct{})

	stopRepeater := repeat.Every(100*time.Millisecond, func() {
		remaining := s.replayBuffer.Size()
		if remaining == 0 {
			s.logger.Debug("usage limiter replay buffer drained successfully")
			close(done)
			return
		}
		s.logger.Debug("waiting for replay buffer to drain", "remaining", remaining)
	})
	defer stopRepeater()

	select {
	case <-ctx.Done():
		s.logger.Warn("shutdown timeout reached, actively draining remaining buffer items")
		// Actively drain any remaining items to avoid data loss
		for {
			remaining := s.replayBuffer.Size()
			if remaining == 0 {
				s.logger.Debug("successfully drained all remaining buffer items")
				break
			}

			// Process remaining items directly
			select {
			case change := <-s.replayBuffer.Consume():
				err := s.syncWithDB(context.Background(), change)
				if err != nil {
					s.logger.Error("failed to sync credit change during shutdown", "error", err)
				}
			default:
				// Channel is closed and empty
			}
		}
		return nil

	case <-done:
		s.logger.Debug("usage limiter replay buffer drained successfully")
		return nil
	}

}
