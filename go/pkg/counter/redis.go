package counter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
)

// redisCounter implements the Counter interface using Redis.
// It provides distributed counter functionality backed by Redis.
type redisCounter struct {
	// redis is the Redis client
	redis *redis.Client

	// logger for logging
	logger logging.Logger
}

var _ Counter = (*redisCounter)(nil)

// RedisConfig holds configuration options for the Redis counter.
type RedisConfig struct {
	// RedisURL is the connection URL for Redis.
	// Format: redis://[[username][:password]@][host][:port][/database]
	RedisURL string

	// Logger is the logging implementation to use.
	// Optional, but recommended for production use.
	Logger logging.Logger
}

// NewRedis creates a new Redis-backed counter implementation.
//
// Parameters:
//   - config: Configuration options for the Redis counter
//
// Returns:
//   - Counter: Redis implementation of the Counter interface
//   - error: Any errors during initialization
func NewRedis(config RedisConfig) (Counter, error) {
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

	return &redisCounter{
		redis:  rdb,
		logger: config.Logger,
	}, nil
}

// Increment increases the counter by the given value and returns the new count.
// If ttl is provided and the counter is newly created (new value is equal to the increment value),
// it also sets an expiration time for the counter.
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//   - key: Unique identifier for the counter
//   - value: Amount to increment the counter by
//   - ttl: Optional time-to-live duration. If provided and the key is new, sets this TTL.
//
// Returns:
//   - int64: The new counter value after incrementing
//   - error: Any errors that occurred during the operation
func (r *redisCounter) Increment(ctx context.Context, key string, value int64, ttl ...time.Duration) (int64, error) {
	ctx, span := tracing.Start(ctx, "RedisCounter.Increment")
	defer span.End()

	// Increment the counter
	newValue, err := r.redis.IncrBy(ctx, key, value).Result()
	if err != nil {
		return 0, err
	}

	// If TTL is provided and this is a new key (value == increment amount),
	// set the expiration time
	if len(ttl) > 0 && newValue == value {
		if err := r.redis.Expire(ctx, key, ttl[0]).Err(); err != nil {
			r.logger.Error("failed to set TTL on counter", "key", key, "error", err.Error())
			// We don't return the error since the increment operation was successful
		}
	}

	return newValue, nil
}

// Get retrieves the current value of a counter.
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//   - key: Unique identifier for the counter
//
// Returns:
//   - int64: The current counter value
//   - error: Any errors that occurred during the operation
func (r *redisCounter) Get(ctx context.Context, key string) (int64, error) {
	ctx, span := tracing.Start(ctx, "RedisCounter.Get")
	defer span.End()

	res, err := r.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		// Key doesn't exist, return 0 without error
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	return strconv.ParseInt(res, 10, 64)
}

// Decrement decreases the counter by the given value and returns the new count.
func (r *redisCounter) Decrement(ctx context.Context, key string, value int64, ttl ...time.Duration) (int64, error) {
	return r.Increment(ctx, key, -value, ttl...)
}

// DecrementIfExists decrements a counter only if it already exists.
func (r *redisCounter) DecrementIfExists(ctx context.Context, key string, value int64) (int64, bool, error) {
	return r.IncrementIfExists(ctx, key, -value)
}

// SetIfNotExists sets a counter to a specific value only if it doesn't already exist.
func (r *redisCounter) SetIfNotExists(ctx context.Context, key string, value int64, ttl ...time.Duration) (bool, error) {
	ctx, span := tracing.Start(ctx, "RedisCounter.SetIfNotExists")
	defer span.End()

	var duration time.Duration
	if len(ttl) > 0 {
		duration = ttl[0]
	}

	// Use Redis SETNX (SET if Not eXists) with optional TTL
	var result *redis.BoolCmd
	if duration > 0 {
		result = r.redis.SetNX(ctx, key, value, duration)
	} else {
		result = r.redis.SetNX(ctx, key, value, 0)
	}

	return result.Val(), result.Err()
}

// IncrementIfExists increments a counter only if it already exists.
// Uses a Lua script to atomically check existence and increment.
func (r *redisCounter) IncrementIfExists(ctx context.Context, key string, value int64) (int64, bool, error) {
	ctx, span := tracing.Start(ctx, "RedisCounter.IncrementIfExists")
	defer span.End()

	// Lua script to atomically check existence and increment
	script := `
		local key = KEYS[1]
		local increment = tonumber(ARGV[1])
		
		-- Check if key exists
		local exists = redis.call('EXISTS', key)
		if exists == 0 then
			return {0, 0}  -- {value, existed} where existed=0 means false
		end
		
		-- Key exists, increment it
		local newValue = redis.call('INCRBY', key, increment)
		return {newValue, 1}  -- {value, existed} where existed=1 means true
	`

	result, err := r.redis.Eval(ctx, script, []string{key}, value).Result()
	if err != nil {
		return 0, false, err
	}

	// Parse the result array [newValue, existed]
	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) != 2 {
		return 0, false, fmt.Errorf("unexpected result format from Lua script")
	}

	newValue, ok := resultSlice[0].(int64)
	if !ok {
		return 0, false, fmt.Errorf("invalid newValue type in result")
	}

	existedInt, ok := resultSlice[1].(int64)
	if !ok {
		return 0, false, fmt.Errorf("invalid existed type in result")
	}

	existed := existedInt == 1
	return newValue, existed, nil
}

// Close releases the Redis client connection.
//
// Returns:
//   - error: Any errors that occurred during shutdown
func (r *redisCounter) Close() error {
	return r.redis.Close()
}

// MultiGet retrieves the values of multiple counters in a single operation.
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//   - keys: Slice of unique identifiers for the counters
//
// Returns:
//   - map[string]int64: Map of counter keys to their current values
//   - error: Any errors that occurred during the operation
func (r *redisCounter) MultiGet(ctx context.Context, keys []string) (map[string]int64, error) {
	ctx, span := tracing.Start(ctx, "RedisCounter.MultiGet")
	defer span.End()

	if len(keys) == 0 {
		return make(map[string]int64), nil
	}

	values, err := r.redis.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string]int64, len(keys))
	for i, key := range keys {
		if i >= len(values) || values[i] == nil {
			// Key doesn't exist, set to 0
			result[key] = 0
			continue
		}

		s, ok := values[i].(string)
		if !ok {
			r.logger.Warn("unexpected type for counter value",
				"key", key,
				"type", fmt.Sprintf("%T", values[i]),
			)
			continue
		}
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			r.logger.Warn("failed to parse counter value",
				"key", key,
				"value", s,
				"error", err,
			)
			continue
		}

		result[key] = v
	}

	return result, nil
}
