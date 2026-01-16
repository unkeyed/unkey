package usagelimiter

import (
	"fmt"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
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
// For higher performance, use NewRedisWithCounter instead.
func New(config Config) (*service, error) {
	if err := assert.All(
		assert.NotNil(config.DB, "db is required"),
		assert.NotNil(config.Logger, "logger is required"),
	); err != nil {
		return nil, fmt.Errorf("invalid usagelimiter service config: %w", err)
	}

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
	//nolint:exhaustruct // ReplayWorkers defaults to 8 in NewCounter when unset
	return NewCounter(CounterConfig{
		DB:      config.DB,
		Logger:  config.Logger,
		Counter: config.Counter,
		TTL:     config.TTL,
	})
}
