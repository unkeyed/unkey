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
			Drop:     false,
		}),
		dbCircuitBreaker: circuitbreaker.New[any]("usagelimiter_db_writes"),
	}

	// Start replay workers
	for range 8 {
		go s.replayRequests()
	}

	return s, nil
}

func (s *counterService) Limit(ctx context.Context, req UsageRequest) (UsageResponse, error) {
	ctx, span := tracing.Start(ctx, "usagelimiter.counter.Limit")
	defer span.End()

	redisKey := fmt.Sprintf("credits:%s", req.KeyId)

	// Try atomic decrement only if key exists - no guesswork needed!
	remaining, existed, err := s.counter.DecrementIfExists(ctx, redisKey, int64(req.Cost))
	if err != nil {
		s.logger.Debug("counter DecrementIfExists failed, falling back to DB", "error", err, "keyId", req.KeyId)
		metrics.UsagelimiterFallbackOperations.Inc()
		return s.dbFallback.Limit(ctx, req)
	}

	s.logger.Debug("DecrementIfExists result", "keyId", req.KeyId, "remaining", remaining, "existed", existed, "cost", req.Cost)

	// Key doesn't exist - load from database
	if !existed {
		s.logger.Debug("key doesn't exist in Redis, loading from database", "keyId", req.KeyId)
		return s.loadFromDatabaseWithCounter(ctx, req, redisKey)
	}

	// Key exists - check if decrement was successful

	// Normal case: Decrement was successful (remaining >= 0)
	if remaining >= 0 {
		s.replayBuffer.Buffer(CreditChange{
			KeyID:  req.KeyId,
			Amount: req.Cost,
		})
		metrics.UsagelimiterDecisions.WithLabelValues("redis", "allowed").Inc()
		metrics.UsagelimiterCreditsProcessed.Add(float64(req.Cost))
		s.logger.Debug("atomic decrement successful", "keyId", req.KeyId, "remaining", remaining)
		return UsageResponse{Valid: true, Remaining: int32(remaining)}, nil
	}

	// Case 3: Key existed but insufficient credits (remaining < 0)
	// Revert the decrement since we can't fulfill the request
	originalValue := remaining + int64(req.Cost)
	_, revertErr := s.counter.Increment(ctx, redisKey, int64(req.Cost))
	if revertErr != nil {
		s.logger.Error("failed to revert insufficient credit decrement", "error", revertErr, "keyId", req.KeyId)
	}

	metrics.UsagelimiterDecisions.WithLabelValues("redis", "denied").Inc()
	s.logger.Debug("insufficient credits", "keyId", req.KeyId, "available", originalValue, "needed", req.Cost)
	return UsageResponse{Valid: false, Remaining: 0}, nil
}

// loadFromDatabaseWithCounter loads credits from DB and initializes the counter
func (s *counterService) loadFromDatabaseWithCounter(ctx context.Context, req UsageRequest, redisKey string) (UsageResponse, error) {
	ctx, span := tracing.Start(ctx, "usagelimiter.counter.loadFromDatabaseWithCounter")
	defer span.End()

	limit, err := db.Query.FindKeyCredits(ctx, s.db.RO(), req.KeyId)
	if err != nil {
		// This shouldn't happen
		if db.IsNotFound(err) {
			return UsageResponse{Valid: false, Remaining: 0}, nil
		}

		return UsageResponse{Valid: false, Remaining: 0}, err
	}

	// This usage limiter should only be called for keys with limits
	// Unlimited keys (limit.Valid == false) should be handled at the key validator level
	if !limit.Valid {
		s.logger.Error("usage limiter called for unlimited key - this should be handled at key validator level", "keyId", req.KeyId)
		// Return valid anyway to not break existing behavior, but this is a logic error
		return UsageResponse{Valid: true, Remaining: -1}, nil
	}

	// Check if we have enough credits
	if limit.Int32 < req.Cost {
		// Set counter to current DB value for future requests
		// Use positive increment from 0 to set initial value
		_, err = s.counter.Increment(ctx, redisKey, int64(limit.Int32), s.ttl)
		if err != nil {
			s.logger.Debug("failed to initialize counter, falling back to DB", "error", err, "keyId", req.KeyId)
			metrics.UsagelimiterFallbackOperations.Inc()
			return s.dbFallback.Limit(ctx, req)
		}

		metrics.UsagelimiterDecisions.WithLabelValues("redis", "denied").Inc()
		s.logger.Debug("insufficient credits from DB", "keyId", req.KeyId, "available", limit.Int32, "needed", req.Cost)
		return UsageResponse{Valid: false, Remaining: 0}, nil
	}

	// We have enough credits - initialize counter with DB value, then decrement
	s.logger.Debug("initializing counter with DB value", "keyId", req.KeyId, "dbCredits", limit.Int32)

	// Use SetIfNotExists to avoid race conditions - only one node can initialize the key
	wasSet, err := s.counter.SetIfNotExists(ctx, redisKey, int64(limit.Int32), s.ttl)
	if err != nil {
		s.logger.Debug("failed to initialize counter with SetIfNotExists, falling back to DB", "error", err, "keyId", req.KeyId)
		metrics.UsagelimiterFallbackOperations.Inc()
		return s.dbFallback.Limit(ctx, req)
	}

	// If SetIfNotExists returned false, another node already initialized the key
	if !wasSet {
		s.logger.Debug("another node already initialized the key, retrying", "keyId", req.KeyId)
		// Retry the original operation now that key exists
		time.Sleep(1 * time.Millisecond)
		return s.Limit(ctx, req)
	}

	// We successfully initialized the key, now decrement it
	remaining, existed, err := s.counter.DecrementIfExists(ctx, redisKey, int64(req.Cost))
	if err != nil || !existed {
		s.logger.Debug("failed to decrement after initialization, falling back to DB", "error", err, "existed", existed, "keyId", req.KeyId)
		return s.dbFallback.Limit(ctx, req)
	}

	// Check if the decrement resulted in negative credits
	if remaining < 0 {
		// We went negative - this means we didn't have enough credits
		// Revert the decrement to restore the original value
		originalValue := remaining + int64(req.Cost)
		_, revertErr := s.counter.Increment(ctx, redisKey, int64(req.Cost))
		if revertErr != nil {
			s.logger.Error("failed to revert negative decrement after init", "error", revertErr, "keyId", req.KeyId)
		}

		metrics.UsagelimiterDecisions.WithLabelValues("redis", "denied").Inc()
		s.logger.Debug("init decrement denied - insufficient credits", "keyId", req.KeyId, "available", originalValue, "needed", req.Cost)
		return UsageResponse{Valid: false, Remaining: 0}, nil
	}

	// Success - buffer the change for async DB update
	s.replayBuffer.Buffer(CreditChange{
		KeyID:  req.KeyId,
		Amount: req.Cost,
	})

	metrics.UsagelimiterDecisions.WithLabelValues("redis", "allowed").Inc()
	metrics.UsagelimiterCreditsProcessed.Add(float64(req.Cost))
	s.logger.Debug("database load and counter initialization successful", "keyId", req.KeyId, "remaining", remaining)
	return UsageResponse{Valid: true, Remaining: int32(remaining)}, nil
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

	s.logger.Debug("synced credit change with DB", "keyId", change.KeyID, "amount", change.Amount)

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
