package counter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
)

const (
	// decrementIfExistsScript is the Lua script for atomic decrement operations.
	// This script checks if the key exists and if there are sufficient credits
	// before decrementing, avoiding negative values that would need reverting.
	//
	// Decrement Logic:
	// - If key doesn't exist: return {0, 0, 0} (value=0, existed=false, success=false)
	// - If insufficient credits: return {current, 1, 0} (unchanged value, existed=true, success=false)
	// - If sufficient credits: return {new_value, 1, 1} (decremented value, existed=true, success=true)
	//
	// The third return value (success flag) provides unambiguous indication of whether
	// the decrement operation succeeded, eliminating the need to infer success from values.
	decrementIfExistsScript = `
		local key = KEYS[1]
		local decrement = tonumber(ARGV[1])

		-- Check if key exists
		local current = redis.call('GET', key)
		if current == false then
			return {0, 0, 0}  -- {value=0, existed=false, success=false}
		end

		current = tonumber(current)
		-- Check if we have sufficient credits before decrementing
		if current < decrement then
			return {current, 1, 0}  -- {current_unchanged, existed=true, success=false}
		end

		-- Sufficient credits, perform atomic decrement preserving TTL
		local newValue = redis.call('DECRBY', key, decrement)
		return {newValue, 1, 1}  -- {new_decremented_value, existed=true, success=true}
	`
)

var (
	// decrementIfExistsScriptCached is the cached script for atomic decrement operations
	decrementIfExistsScriptCached = redis.NewScript(decrementIfExistsScript)
)

// redisCounter implements the Counter interface using Redis.
// It provides distributed counter functionality backed by Redis.
type redisCounter struct {
	// redis is the Redis client
	redis *redis.Client
}

var _ Counter = (*redisCounter)(nil)

// RedisConfig holds configuration options for the Redis counter.
type RedisConfig struct {
	// RedisURL is the connection URL for Redis.
	// Format: redis://[[username][:password]@][host][:port][/database]
	RedisURL string
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
	logger.Debug("pinging redis")

	// Test connection
	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &redisCounter{
		redis: rdb,
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
			logger.Error("failed to set TTL on counter", "key", key, "error", err.Error())
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
	ctx, span := tracing.Start(ctx, "RedisCounter.Decrement")
	defer span.End()

	// Decrement the counter
	newValue, err := r.redis.DecrBy(ctx, key, value).Result()
	if err != nil {
		return 0, err
	}

	// If TTL is provided and this appears to be a new key (negative value after decrement),
	// set the expiration time
	if len(ttl) > 0 && newValue == -value {
		if err := r.redis.Expire(ctx, key, ttl[0]).Err(); err != nil {
			logger.Error("failed to set TTL on counter", "key", key, "error", err.Error())
			// We don't return the error since the decrement operation was successful
		}
	}

	return newValue, nil
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
	result := r.redis.SetNX(ctx, key, value, duration)

	return result.Val(), result.Err()
}

// DecrementIfExists performs an atomic decrement operation.
// Uses a cached Lua script (EVALSHA) to atomically check existence, verify sufficient
// credits, and conditionally decrement. Returns the actual current counter value and
// explicit success/failure indication.
//
// The decrement logic ensures that:
// 1. Counters never go negative (preserves data integrity)
// 2. Returns actual current values (no sentinel values like -1)
// 3. Provides unambiguous success indication via separate flag
func (r *redisCounter) DecrementIfExists(ctx context.Context, key string, value int64) (int64, bool, bool, error) {
	ctx, span := tracing.Start(ctx, "RedisCounter.DecrementIfExists")
	defer span.End()

	result, err := decrementIfExistsScriptCached.Run(ctx, r.redis, []string{key}, value).Result()
	if err != nil {
		return 0, false, false, err
	}

	// Parse the result array [actualValue, existedFlag, successFlag]
	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) != 3 {
		return 0, false, false, fmt.Errorf("unexpected result format from Lua script")
	}

	// Defensively parse actualValue from multiple numeric types
	actualValue, err := parseNumericValue(resultSlice[0])
	if err != nil {
		return 0, false, false, fmt.Errorf("invalid actualValue in result: %w", err)
	}

	// Defensively parse existed flag from multiple numeric types
	existedFlag, err := parseNumericValue(resultSlice[1])
	if err != nil {
		return 0, false, false, fmt.Errorf("invalid existed flag in result: %w", err)
	}

	// Defensively parse success flag from multiple numeric types
	successFlag, err := parseNumericValue(resultSlice[2])
	if err != nil {
		return 0, false, false, fmt.Errorf("invalid success flag in result: %w", err)
	}

	return actualValue, existedFlag == 1, successFlag == 1, nil
}

// parseNumericValue safely converts various numeric types to int64
func parseNumericValue(v interface{}) (int64, error) {
	switch val := v.(type) {
	case int64:
		return val, nil
	case float64:
		// Safely convert float64 to int64, checking for overflow
		if val > float64(9223372036854775807) || val < float64(-9223372036854775808) {
			return 0, fmt.Errorf("float64 value %v overflows int64", val)
		}
		return int64(val), nil
	case string:
		parsed, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse string '%s' as int64: %w", val, err)
		}
		return parsed, nil
	case int:
		return int64(val), nil
	case int32:
		return int64(val), nil
	default:
		return 0, fmt.Errorf("unsupported type %T for numeric value", v)
	}
}

// Delete removes a counter key from Redis.
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//   - key: Unique identifier for the counter to delete
//
// Returns:
//   - error: Any errors that occurred during the operation
func (r *redisCounter) Delete(ctx context.Context, key string) error {
	ctx, span := tracing.Start(ctx, "RedisCounter.Delete")
	defer span.End()

	return r.redis.Del(ctx, key).Err()
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
			logger.Warn("unexpected type for counter value",
				"key", key,
				"type", fmt.Sprintf("%T", values[i]),
			)
			continue
		}
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			logger.Warn("failed to parse counter value",
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
