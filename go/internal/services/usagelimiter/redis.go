package usagelimiter

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/counter"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
)

const (
	// UnlimitedKeyMarker is a very large positive number used to mark unlimited keys in Redis.
	// This value is much larger than any realistic credit amount (1 billion).
	// When a key is unlimited, we store this value in Redis to avoid database lookups.
	// Even after decrementing costs, the value will remain >= (UnlimitedKeyMarker - max_cost).
	UnlimitedKeyMarker = 1_000_000_000
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

	// Redis client for coordination operations (SETNX etc)
	Redis *redis.Client

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

	s := &counterService{
		db:      config.DB,
		logger:  config.Logger,
		counter: config.Counter,
		ttl:     ttl,
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
		return s.fallbackToDirectDB(ctx, req)
	}

	// Case 1: Key didn't exist - load from database with coordination
	if !existed {
		s.logger.Debug("key doesn't exist in Redis, loading from database", "keyId", req.KeyId)
		return s.loadFromDatabaseWithCounter(ctx, req, redisKey)
	}

	// Case 2: Key existed - check for special cases and successful decrements

	// Special case: Unlimited key marker (remains very large even after decrement)
	// For unlimited keys, we store UnlimitedKeyMarker in Redis as a marker
	if remaining >= UnlimitedKeyMarker-int64(req.Cost) {
		s.logger.Debug("unlimited key detected from Redis cache", "keyId", req.KeyId, "remaining", remaining)
		return UsageResponse{Valid: true, Remaining: -1}, nil
	}

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

	// Unlimited key - cache it in Redis with the unlimited marker
	if !limit.Valid {
		// Set the unlimited marker value in Redis to prevent future database lookups
		_, err = s.counter.Increment(ctx, redisKey, UnlimitedKeyMarker, s.ttl)
		if err != nil {
			s.logger.Debug("failed to cache unlimited key marker, continuing anyway", "error", err, "keyId", req.KeyId)
		}
		s.logger.Debug("unlimited key detected and cached", "keyId", req.KeyId)
		return UsageResponse{Valid: true, Remaining: -1}, nil
	}

	// Check if we have enough credits
	if limit.Int32 < req.Cost {
		// Set counter to current DB value for future requests
		// Use positive increment from 0 to set initial value
		_, err = s.counter.Increment(ctx, redisKey, int64(limit.Int32), s.ttl)
		if err != nil {
			s.logger.Debug("failed to initialize counter, falling back to DB", "error", err, "keyId", req.KeyId)
			return s.fallbackToDirectDB(ctx, req)
		}

		metrics.UsagelimiterDecisions.WithLabelValues("redis", "denied").Inc()
		s.logger.Debug("insufficient credits from DB", "keyId", req.KeyId, "available", limit.Int32, "needed", req.Cost)
		return UsageResponse{Valid: false, Remaining: 0}, nil
	}

	// We have enough credits - initialize counter with decremented value
	newValue := limit.Int32 - req.Cost

	// Atomically set the counter to the decremented value
	// This uses the fact that if counter is 0, incrementing by newValue sets it to newValue
	finalValue, err := s.counter.Increment(ctx, redisKey, int64(newValue), s.ttl)
	if err != nil {
		s.logger.Debug("failed to initialize counter with decremented value, falling back to DB", "error", err, "keyId", req.KeyId)
		return s.fallbackToDirectDB(ctx, req)
	}

	// Check for race condition - if finalValue != newValue, another process initialized the key
	if finalValue != int64(newValue) {
		s.logger.Debug("race condition detected during initialization, retrying", "keyId", req.KeyId, "expected", newValue, "actual", finalValue)
		// Retry the original operation now that key exists
		time.Sleep(1 * time.Millisecond)
		return s.Limit(ctx, req)
	}

	// Success - buffer the change for async DB update
	s.replayBuffer.Buffer(CreditChange{
		KeyID:  req.KeyId,
		Amount: req.Cost,
	})

	metrics.UsagelimiterDecisions.WithLabelValues("redis", "allowed").Inc()
	metrics.UsagelimiterCreditsProcessed.Add(float64(req.Cost))
	s.logger.Debug("database load and counter initialization successful", "keyId", req.KeyId, "remaining", newValue)
	return UsageResponse{Valid: true, Remaining: newValue}, nil
}

func (s *counterService) fallbackToDirectDB(ctx context.Context, req UsageRequest) (UsageResponse, error) {
	ctx, span := tracing.Start(ctx, "usagelimiter.counter.fallbackToDirectDB")
	defer span.End()

	metrics.UsagelimiterFallbackOperations.Inc()

	limit, err := db.Query.FindKeyCredits(ctx, s.db.RW(), req.KeyId)
	if err != nil {
		if db.IsNotFound(err) {
			return UsageResponse{Valid: false, Remaining: 0}, nil
		}
		return UsageResponse{Valid: false, Remaining: 0}, err
	}

	if !limit.Valid {
		return UsageResponse{Valid: true, Remaining: -1}, nil
	}

	remaining := limit.Int32
	if remaining <= 0 && req.Cost != 0 || remaining-req.Cost < 0 {
		metrics.UsagelimiterDecisions.WithLabelValues("db", "denied").Inc()
		return UsageResponse{Valid: false, Remaining: 0}, nil
	}

	err = db.Query.UpdateKeyCredits(ctx, s.db.RW(), db.UpdateKeyCreditsParams{
		ID:        req.KeyId,
		Operation: "decrement",
		Credits:   sql.NullInt32{Int32: req.Cost, Valid: true},
	})
	if err != nil {
		return UsageResponse{}, err
	}

	metrics.UsagelimiterDecisions.WithLabelValues("db", "allowed").Inc()
	metrics.UsagelimiterCreditsProcessed.Add(float64(req.Cost))
	return UsageResponse{Valid: true, Remaining: max(0, remaining-req.Cost)}, nil
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
