package usagelimiter

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/assert"
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
	defaultTTL           = 10 * time.Minute
	defaultReplayWorkers = 8
)

// CreditChange represents a change to a credit balance that needs to be
// replayed to the database for eventual consistency with Redis counters.
type CreditChange struct {
	// KeyID for legacy keys.remaining_requests system
	KeyID string

	// CreditID for new credits table system (takes precedence)
	CreditID string

	// Cost is the number of credits that we should deduct
	Cost int32
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
	if err := assert.All(
		assert.NotNil(config.Counter),
		assert.NotNil(config.DB),
	); err != nil {
		return nil, err
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
		replayWorkers = defaultReplayWorkers
	}

	for range replayWorkers {
		go s.replayRequests()
	}

	return s, nil
}

// Limit processes a usage request and enforces credit limits using distributed Redis counters
// with decrement logic that prevents negative values and eliminates revert operations.
//
// This function implements a two-phase credit limiting system:
// 1. When key exists in Redis: Atomic decrement on existing Redis counters
// 2. When key does not exist in Redis: Database load + Redis initialization for new keys
//
// Decrement Benefits:
// - Prevents counters from ever going negative (maintains data integrity)
// - Eliminates need for revert operations when credits are insufficient
// - Always returns actual current credit counts (no sentinel values)
// - Atomic operation ensures consistency under high concurrency
//
// ERROR HANDLING:
// - Redis failures trigger automatic fallback to direct DB operations
// - Decrement Lua script handles insufficient credit scenarios atomically
// - Circuit breaker protects against database overload during fallbacks
//
// METRICS:
// - usagelimiter_decisions_total: Tracks allowed/denied decisions by backend
// - usagelimiter_credits_processed_total: Tracks total credits consumed
// - usagelimiter_fallback_operations_total: Tracks fallback frequency
func (s *counterService) Limit(ctx context.Context, req UsageRequest) (UsageResponse, error) {
	ctx, span := tracing.Start(ctx, "usagelimiter.counter.Limit")
	defer span.End()

	// Determine which system to use and generate appropriate Redis key
	var redisKey string
	if req.CreditID != "" {
		redisKey = s.creditRedisKey(req.CreditID)
	} else if req.KeyID != "" {
		redisKey = s.keyRedisKey(req.KeyID)
	} else {
		return UsageResponse{Valid: false, Remaining: 0}, nil
	}

	// Attempt decrement if key already exists in Redis
	// The Lua script handles cost=0 by returning current value without decrementing
	remaining, exists, success, err := s.counter.DecrementIfExists(ctx, redisKey, int64(req.Cost))
	if err != nil {
		return s.dbFallback.Limit(ctx, req)
	}

	// Key doesn't exist in Redis - initialize from database
	if !exists {
		return s.initializeFromDatabase(ctx, req, redisKey)
	}

	// Key exists in Redis - use the explicit success flag from decrement
	return s.handleResult(req, remaining, success), nil
}

func (s *counterService) Invalidate(ctx context.Context, identifier string) error {
	// Try both possible key formats since we don't know which system this is
	_ = s.counter.Delete(ctx, s.creditRedisKey(identifier))
	_ = s.counter.Delete(ctx, s.keyRedisKey(identifier))
	return nil
}

func (s *counterService) creditRedisKey(creditID string) string {
	return fmt.Sprintf("credits:%s", creditID)
}

func (s *counterService) keyRedisKey(keyID string) string {
	return fmt.Sprintf("key_remaining:%s", keyID)
}

// handleResult processes the result of a decrement operation using an explicit success flag.
// This eliminates ambiguity in determining whether the operation succeeded or failed.
func (s *counterService) handleResult(req UsageRequest, remaining int64, success bool) UsageResponse {
	if success {
		// Only buffer the change for async database sync if cost > 0
		// Zero cost requests don't need to be replayed to the database
		if req.Cost > 0 {
			s.replayBuffer.Buffer(CreditChange{
				KeyID:    req.KeyID,
				CreditID: req.CreditID,
				Cost:     req.Cost,
			})
		}

		metrics.UsagelimiterDecisions.WithLabelValues("redis", "allowed").Inc()
		return UsageResponse{Valid: true, Remaining: int32(remaining)} //nolint: gosec

	}

	// Insufficient credits - return actual current count for accurate response
	metrics.UsagelimiterDecisions.WithLabelValues("redis", "denied").Inc()
	return UsageResponse{Valid: false, Remaining: int32(remaining)} //nolint: gosec
}

// initializeFromDatabase loads credits from DB and initializes the counter.
// It ensures Redis is never initialized with negative values, which could skew
// subsequent decisions from other nodes in a distributed environment.
func (s *counterService) initializeFromDatabase(ctx context.Context, req UsageRequest, redisKey string) (UsageResponse, error) {
	ctx, span := tracing.Start(ctx, "usagelimiter.counter.initializeFromDatabase")
	defer span.End()

	// Load from appropriate database table
	var limit int32
	var err error

	if req.CreditID != "" {
		// New credits system
		limit, err = db.WithRetryContext(ctx, func() (int32, error) {
			return db.Query.FindRemainingCredits(ctx, s.db.RO(), req.CreditID)
		})
	} else if req.KeyID != "" {
		// Legacy key.remaining_requests system
		limit, err = db.WithRetryContext(ctx, func() (int32, error) {
			remaining, err := db.Query.FindRemainingKey(ctx, s.db.RO(), req.KeyID)
			if err != nil {
				return 0, err
			}

			if !remaining.Valid {
				// No usage limit configured - this shouldn't be called
				return 0, sql.ErrNoRows
			}

			return remaining.Int32, nil
		})
	} else {
		return UsageResponse{Valid: false, Remaining: 0}, nil
	}

	if err != nil {
		if db.IsNotFound(err) {
			return UsageResponse{Valid: false, Remaining: 0}, nil
		}

		return UsageResponse{Valid: false, Remaining: 0}, err
	}

	// Determine what value to initialize Redis with - never use negative values
	currentCredits := int64(limit)
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
		s.logger.Debug("failed to initialize counter with SetIfNotExists, falling back to DB", "error", err)
		return s.dbFallback.Limit(ctx, req)
	}

	// If we successfully initialized the key, return the appropriate response
	if wasSet {
		if hasSufficientCredits {
			// Successful decrement - return the decremented value
			return s.handleResult(req, initValue, true), nil
		} else {
			// Insufficient credits - return the current unchanged value
			return s.handleResult(req, currentCredits, false), nil
		}
	}

	// Another node already initialized the key, check if we have enough after decrement
	remaining, exists, success, err := s.counter.DecrementIfExists(ctx, redisKey, int64(req.Cost))
	if err != nil || !exists {
		s.logger.Debug("failed to decrement after initialization attempt", "error", err, "exists", exists)
		return s.dbFallback.Limit(ctx, req)
	}

	// Process the decrement result using explicit success flag
	return s.handleResult(req, remaining, success), nil
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

	var err error

	if change.CreditID != "" {
		// New credits system
		_, err = s.dbCircuitBreaker.Do(ctx, func(ctx context.Context) (any, error) {
			return nil, db.Query.UpdateCreditDecrement(ctx, s.db.RW(), db.UpdateCreditDecrementParams{
				ID:      change.CreditID,
				Credits: change.Cost,
			})
		})
	} else if change.KeyID != "" {
		// Legacy key.remaining_requests system
		_, err = s.dbCircuitBreaker.Do(ctx, func(ctx context.Context) (any, error) {
			return nil, db.Query.UpdateKeyCreditsDecrement(ctx, s.db.RW(), db.UpdateKeyCreditsDecrementParams{
				ID:      change.KeyID,
				Credits: sql.NullInt32{Int32: change.Cost, Valid: true},
			})
		})
	} else {
		metrics.UsagelimiterReplayOperations.WithLabelValues("error").Inc()
		return fmt.Errorf("neither CreditID nor KeyID provided")
	}

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
