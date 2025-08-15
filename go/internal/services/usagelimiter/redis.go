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
)

type CreditChange struct {
	KeyID  string
	Amount int32
}

// RedisConfig holds configuration options for the Redis usage limiter.
type RedisConfig struct {
	// DB is the database connection for fallback and replay operations
	DB db.Database

	// Logger is the logging implementation to use.
	// Optional, but recommended for production use.
	Logger logging.Logger

	// RedisURL is the connection URL for Redis.
	// Format: redis://[[username][:password]@][host][:port][/database]
	RedisURL string

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
	ttl := config.TTL
	if ttl == 0 {
		ttl = 10 * time.Minute
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
	for range 8 {
		go s.replayRequests()
	}

	return s, nil
}

// Limit processes a usage request and enforces credit limits using distributed Redis counters.
//
// ARCHITECTURE:
// This function implements a two-phase credit limiting system:
// 1. Fast path: Atomic decrement on existing Redis counters
// 2. Slow path: Database load + Redis initialization for new keys
//
// ALGORITHM:
// The system uses Redis DECRBY operations for atomic credit decrements, ensuring
// perfect accuracy under high concurrency. When a key doesn't exist in Redis,
// we load its current credit count from the database and initialize a Redis
// counter. Race conditions during initialization are handled by using Redis
// SETNX (SetIfNotExists) to ensure only one node initializes each key.
//
// CONCURRENCY SAFETY:
// - Redis DECRBY operations are atomic, preventing race conditions
// - SetIfNotExists prevents multiple nodes from initializing the same key
// - Negative credit detection with automatic reversion ensures accuracy
// - No distributed locks needed - Redis atomicity is sufficient
//
// PERFORMANCE CHARACTERISTICS:
// - Fast path: Single Redis DECRBY operation (~1ms latency)
// - Slow path: Database query + Redis initialization (~10-50ms latency)
// - Scales linearly with request volume due to Redis atomicity
// - Falls back to direct database operations on Redis failures
//
// ERROR HANDLING:
// - Redis failures trigger automatic fallback to direct DB operations
// - Failed credit reverts are logged but don't fail the request
// - Circuit breaker protects against database overload during fallbacks
//
// METRICS:
// - usagelimiter_decisions_total: Tracks allowed/denied decisions by backend
// - usagelimiter_credits_processed_total: Tracks total credits consumed
// - usagelimiter_fallback_operations_total: Tracks fallback frequency
//
// EXAMPLES:
//
//	// Successful credit deduction:
//	req := UsageRequest{KeyId: "key_123", Cost: 5}
//	resp, err := limiter.Limit(ctx, req)
//	// resp.Valid = true, resp.Remaining = 45 (if started with 50)
//
//	// Insufficient credits:
//	req := UsageRequest{KeyId: "key_123", Cost: 100}
//	resp, err := limiter.Limit(ctx, req)
//	// resp.Valid = false, resp.Remaining = 0
func (s *counterService) Limit(ctx context.Context, req UsageRequest) (UsageResponse, error) {
	ctx, span := tracing.Start(ctx, "usagelimiter.counter.Limit")
	defer span.End()

	redisKey := fmt.Sprintf("credits:%s", req.KeyId)

	// Try atomic decrement only if key usage is already stored
	remaining, exists, err := s.counter.DecrementIfExists(ctx, redisKey, int64(req.Cost))
	if err != nil {
		metrics.UsagelimiterFallbackOperations.Inc()
		return s.dbFallback.Limit(ctx, req)
	}

	// Key doesn't exist - load from db
	if !exists {
		return s.initializeFromDatabase(ctx, req, redisKey)
	}

	// Key exists, check if decrement was successful
	return s.handleDecrementResult(ctx, req, redisKey, remaining)
}

// handleDecrementResult processes the result of a decrement operation
func (s *counterService) handleDecrementResult(ctx context.Context, req UsageRequest, redisKey string, remaining int64) (UsageResponse, error) {
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

	// Insufficient credits (remaining < 0) - revert the decrement
	// originalValue := remaining + int64(req.Cost)
	_, revertErr := s.counter.Increment(ctx, redisKey, int64(req.Cost))
	if revertErr != nil {
		s.logger.Error("failed to revert insufficient credit decrement", "error", revertErr, "keyId", req.KeyId)
	}

	metrics.UsagelimiterDecisions.WithLabelValues("redis", "denied").Inc()
	// s.logger.Info("insufficient credits", "keyId", req.KeyId, "available", originalValue, "needed", req.Cost)

	return UsageResponse{Valid: false, Remaining: 0}, nil
}

// initializeFromDatabase loads credits from DB and initializes the counter
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

	// Try to initialize with the already decremented value
	wasSet, err := s.counter.SetIfNotExists(ctx, redisKey, int64(limit.Int32-req.Cost), s.ttl)
	if err != nil {
		s.logger.Debug("failed to initialize counter with SetIfNotExists, falling back to DB", "error", err, "keyId", req.KeyId)
		metrics.UsagelimiterFallbackOperations.Inc()
		return s.dbFallback.Limit(ctx, req)
	}

	if wasSet {
		// We won the initialization race - already set the decremented value, skip extra Redis call
		remaining := int64(limit.Int32) - int64(req.Cost)
		return s.handleDecrementResult(ctx, req, redisKey, remaining)
	}

	// Another node already initialized the key - do atomic decrement
	remaining, exists, err := s.counter.DecrementIfExists(ctx, redisKey, int64(req.Cost))
	if err != nil || !exists {
		s.logger.Debug("failed to decrement after initialization attempt", "error", err, "exists", exists, "keyId", req.KeyId, "wasSet", wasSet)
		return s.dbFallback.Limit(ctx, req)
	}

	// Process the decrement result
	return s.handleDecrementResult(ctx, req, redisKey, remaining)
}

// replayRequests processes buffered credit changes and updates the database
func (s *counterService) replayRequests() {
	for change := range s.replayBuffer.Consume() {
		err := s.syncWithDB(context.Background(), change)

		if err != nil {
			s.logger.Error("failed to replay credit change", "error", err.Error())
		}
	}
}

func (s *counterService) syncWithDB(ctx context.Context, change CreditChange) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := s.dbCircuitBreaker.Do(ctx, func(ctx context.Context) (any, error) {
		return nil, db.Query.UpdateKeyCredits(ctx, s.db.RW(), db.UpdateKeyCreditsParams{
			ID:        change.KeyID,
			Operation: "decrement",
			Credits:   sql.NullInt32{Int32: change.Amount, Valid: true},
		})
	})

	if err != nil {
		s.logger.Error("failed to sync credit change with DB", "keyId", change.KeyID, "amount", change.Amount, "error", err)
		return err
	}

	return nil
}

func (s *counterService) Close() error {
	return s.GracefulShutdown(context.Background())
}

// GracefulShutdown performs a graceful shutdown of the usage limiter service.
// It allows the replay buffer to drain pending changes before closing.
func (s *counterService) GracefulShutdown(ctx context.Context) error {
	s.logger.Debug("beginning graceful shutdown of usage limiter")

	// Set a reasonable timeout for draining if context doesn't have one
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 30*time.Second {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	// Stop accepting new buffer writes
	s.replayBuffer.Close()

	// Wait for the buffer to drain with periodic checks
	drainTicker := time.NewTicker(100 * time.Millisecond)
	defer drainTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Get remaining count for logging
			remaining := len(s.replayBuffer.Consume())
			if remaining > 0 {
				s.logger.Warn("usage limiter shutdown timeout reached, some changes may be lost",
					"timeout", ctx.Err(), "remaining_changes", remaining)
			} else {
				s.logger.Debug("usage limiter shutdown completed (timeout but buffer was empty)")
			}
			return s.counter.Close()

		case <-drainTicker.C:
			// Check if buffer is empty by looking at the underlying channel length
			remaining := len(s.replayBuffer.Consume())
			if remaining == 0 {
				s.logger.Debug("usage limiter replay buffer drained successfully")
				return s.counter.Close()
			}
			s.logger.Debug("waiting for replay buffer to drain", "remaining", remaining)
		}
	}
}
