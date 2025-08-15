package usagelimiter

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
)

type CreditChange struct {
	KeyID  string
	Amount int32
}

type redisService struct {
	db     db.Database
	logger logging.Logger
	redis  *redis.Client

	// Replay buffer for async DB updates
	replayBuffer *buffer.Buffer[CreditChange]

	// Circuit breaker for DB writes
	dbCircuitBreaker circuitbreaker.CircuitBreaker[any]

	// TTL for Redis keys
	ttl time.Duration
}

var _ Service = (*redisService)(nil)

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

// NewRedis creates a new Redis-backed usage limiter implementation.
//
// Parameters:
//   - config: Configuration options for the Redis usage limiter
//
// Returns:
//   - Service: Redis implementation of the Service interface
//   - error: Any errors during initialization
func NewRedis(config RedisConfig) (Service, error) {
	err := assert.All(
		assert.NotEmpty(config.RedisURL, "Redis URL must not be empty"),
	)
	if err != nil {
		return nil, err
	}

	opts, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	rdb := redis.NewClient(opts)
	config.Logger.Debug("pinging redis")

	// Test connection
	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	ttl := config.TTL
	if ttl == 0 {
		ttl = 10 * time.Minute
	}

	s := &redisService{
		db:     config.DB,
		logger: config.Logger,
		redis:  rdb,
		ttl:    ttl,
		replayBuffer: buffer.New[CreditChange](buffer.Config{
			Name:     "usagelimiter_replays",
			Capacity: 10_000,
			Drop:     false, // NEVER drop credit changes - accuracy is critical
		}),
		dbCircuitBreaker: circuitbreaker.New[any]("usagelimiter_db_writes"),
	}

	// Start replay workers like ratelimiter does
	for range 8 {
		go s.replayRequests()
	}

	return s, nil
}

// Lua script for atomic check-and-decrement
// Returns the new value if successful, or -1 if insufficient credits
const checkAndDecrementScript = `
local key = KEYS[1]
local cost = tonumber(ARGV[1])
local current = redis.call('GET', key)

if current == false then
	return -2  -- Key doesn't exist
end

current = tonumber(current)

if current >= cost then
	local new_value = redis.call('DECRBY', key, cost)
	return new_value
else
	return -1  -- Insufficient credits
end
`

// Enhanced atomic script with integrated locking
// Returns: new_value (>=0), -1 (insufficient), -2 (acquired lock), -3 (wait)
const atomicCheckDecrementWithLockScript = `
local key = KEYS[1]
local lock_key = KEYS[2]
local cost = tonumber(ARGV[1])
local lock_value = ARGV[2]
local lock_ttl = tonumber(ARGV[3])

-- First try normal check-and-decrement
local current = redis.call('GET', key)

if current ~= false then
    current = tonumber(current)
    if current >= cost then
        local new_value = redis.call('DECRBY', key, cost)
        return new_value
    else
        return -1  -- Insufficient credits
    end
end

-- Key doesn't exist, try to acquire lock atomically
local lock_acquired = redis.call('SET', lock_key, lock_value, 'NX', 'EX', lock_ttl)
if lock_acquired then
    return -2  -- Lock acquired, caller should load from DB
else
    return -3  -- Lock not acquired, caller should wait and retry
end
`

func (s *redisService) Limit(ctx context.Context, req UsageRequest) (UsageResponse, error) {
	ctx, span := tracing.Start(ctx, "usagelimiter.redis.Limit")
	defer span.End()

	// Build Redis keys
	redisKey := fmt.Sprintf("credits:%s", req.KeyId)
	lockKey := fmt.Sprintf("lock:credits:%s", req.KeyId)
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())
	lockTTL := 5 // seconds

	// Use atomic script that includes locking
	result := s.redis.Eval(ctx, atomicCheckDecrementWithLockScript,
		[]string{redisKey, lockKey}, req.Cost, lockValue, lockTTL)
	if result.Err() != nil {
		s.logger.Error("redis eval failed", "error", result.Err().Error())
		return s.fallbackToDirectDB(ctx, req)
	}

	val, err := result.Int64()
	if err != nil {
		return UsageResponse{Valid: false, Remaining: 0}, err
	}

	if val == -2 {
		// Lock acquired, we need to load from database
		s.logger.Info("lock acquired in script, loading from database", "keyId", req.KeyId)
		return s.loadFromDatabaseWithAcquiredLock(ctx, req, lockKey)
	}

	if val == -3 {
		// Lock not acquired, wait and retry
		s.logger.Info("lock not acquired, retrying", "keyId", req.KeyId)
		time.Sleep(10 * time.Millisecond)
		// Retry with simple script since someone else is loading
		result := s.redis.Eval(ctx, checkAndDecrementScript, []string{redisKey}, req.Cost)
		if result.Err() != nil {
			return s.fallbackToDirectDB(ctx, req)
		}
		val, err := result.Int64()
		if err != nil {
			return s.fallbackToDirectDB(ctx, req)
		}
		if val == -2 {
			// Still not loaded, fallback to DB
			s.logger.Info("key still not loaded after retry, falling back to DB", "keyId", req.KeyId)
			return s.fallbackToDirectDB(ctx, req)
		}
		if val == -1 {
			metrics.UsagelimiterDecisions.WithLabelValues("redis", "denied").Inc()
			return UsageResponse{Valid: false, Remaining: 0}, nil
		}
		// Success after retry
		s.replayBuffer.Buffer(CreditChange{KeyID: req.KeyId, Amount: req.Cost})
		metrics.UsagelimiterDecisions.WithLabelValues("redis", "allowed").Inc()
		metrics.UsagelimiterCreditsProcessed.Add(float64(req.Cost))
		return UsageResponse{Valid: true, Remaining: int32(val)}, nil
	}

	if val == -1 {
		// Insufficient credits
		metrics.UsagelimiterDecisions.WithLabelValues("redis", "denied").Inc()
		return UsageResponse{Valid: false, Remaining: 0}, nil
	}

	// Success - buffer the change for async DB update
	s.replayBuffer.Buffer(CreditChange{
		KeyID:  req.KeyId,
		Amount: req.Cost,
	})

	metrics.UsagelimiterDecisions.WithLabelValues("redis", "allowed").Inc()
	metrics.UsagelimiterCreditsProcessed.Add(float64(req.Cost))
	return UsageResponse{Valid: true, Remaining: int32(val)}, nil
}

func (s *redisService) fallbackToDirectDB(ctx context.Context, req UsageRequest) (UsageResponse, error) {
	ctx, span := tracing.Start(ctx, "usagelimiter.redis.fallbackToDirectDB")
	defer span.End()

	metrics.UsagelimiterFallbackOperations.Inc()

	// Use the original service implementation
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

// loadFromDatabaseWithAcquiredLock handles loading when we already have the lock
func (s *redisService) loadFromDatabaseWithAcquiredLock(ctx context.Context, req UsageRequest, lockKey string) (UsageResponse, error) {
	// We already have the lock, release it when done
	defer func() {
		s.redis.Del(ctx, lockKey)
	}()

	s.logger.Info("loading from database with acquired lock", "keyId", req.KeyId)

	limit, err := db.Query.FindKeyCredits(ctx, s.db.RO(), req.KeyId)
	if err != nil {
		if db.IsNotFound(err) {
			s.logger.Info("key not found in database", "keyId", req.KeyId)
			return UsageResponse{Valid: false, Remaining: 0}, nil
		}
		return UsageResponse{Valid: false, Remaining: 0}, err
	}

	s.logger.Info("loaded key from database with acquired lock", "keyId", req.KeyId, "limit", limit)

	// Unlimited key
	if !limit.Valid {
		s.logger.Info("unlimited key detected", "keyId", req.KeyId)
		return UsageResponse{Valid: true, Remaining: -1}, nil
	}

	// Check if we have enough credits
	if limit.Int32 < req.Cost {
		// Not enough credits, set Redis to current DB value for future requests
		redisKey := fmt.Sprintf("credits:%s", req.KeyId)
		s.redis.Set(ctx, redisKey, limit.Int32, s.ttl)
		metrics.UsagelimiterDecisions.WithLabelValues("redis", "denied").Inc()
		s.logger.Info("insufficient credits", "keyId", req.KeyId, "available", limit.Int32, "needed", req.Cost)
		return UsageResponse{Valid: false, Remaining: 0}, nil
	}

	// We have enough credits - set Redis with decremented value atomically
	redisKey := fmt.Sprintf("credits:%s", req.KeyId)
	newValue := limit.Int32 - req.Cost

	// Use SET to initialize Redis with the decremented value
	err = s.redis.Set(ctx, redisKey, newValue, s.ttl).Err()
	if err != nil {
		s.logger.Error("failed to set redis after DB load", "error", err, "keyId", req.KeyId)
		return s.fallbackToDirectDB(ctx, req)
	}

	// Buffer the change for async DB update
	s.replayBuffer.Buffer(CreditChange{
		KeyID:  req.KeyId,
		Amount: req.Cost,
	})

	metrics.UsagelimiterDecisions.WithLabelValues("redis", "allowed").Inc()
	metrics.UsagelimiterCreditsProcessed.Add(float64(req.Cost))
	s.logger.Info("successfully loaded and decremented with acquired lock", "keyId", req.KeyId, "remaining", newValue)
	return UsageResponse{Valid: true, Remaining: newValue}, nil
}

func (s *redisService) Close() error {
	s.replayBuffer.Close()
	return s.redis.Close()
}
