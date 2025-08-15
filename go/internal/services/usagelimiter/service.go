package usagelimiter

import (
	"fmt"

	"github.com/unkeyed/unkey/go/pkg/counter"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// service implements the direct DB-based usage limiter
type service struct {
	db     db.Database
	logger logging.Logger
}

var _ Service = (*service)(nil)

// Config for the direct DB implementation
type Config struct {
	DB     db.Database
	Logger logging.Logger
}

// New creates a new direct DB-based usage limiter service.
// This implementation queries the database on every request.
// For higher performance, use NewRedis or NewRedisWithTransactions instead.
func New(config Config) (*service, error) {
	return &service{
		db:     config.DB,
		logger: config.Logger,
	}, nil
}

// NewRedisWithCounter creates a Redis usage limiter using the counter interface.
//
// This is the recommended Redis implementation as it provides truly atomic operations
// using Redis INCRBY commands. It's the simplest, fastest, and most reliable approach.
//
// Characteristics:
//   - Uses Redis INCRBY for atomic decrements (single operation, no race conditions)
//   - Excellent performance under high contention
//   - Simple, clean code with minimal complexity
//   - No distributed locking or transactions needed
//   - Falls back to direct DB on Redis failures
//
// Parameters:
//   - config: Redis configuration options
//
// Returns:
//   - Service: Counter-based Redis implementation (recommended)
//   - error: Any initialization errors
func NewRedisWithCounter(config RedisConfig) (Service, error) {
	// Create the Redis counter
	redisCounter, err := counter.NewRedis(counter.RedisConfig{
		RedisURL: config.RedisURL,
		Logger:   config.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis counter: %w", err)
	}

	return NewCounter(CounterConfig{
		DB:      config.DB,
		Logger:  config.Logger,
		Counter: redisCounter,
		TTL:     config.TTL,
	})
}
