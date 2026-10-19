package usagelimiter

import (
	"fmt"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
)

type service struct {
	db *db.Replica
}

var _ Service = (*service)(nil)

// Config requires a primary database connection for atomic credit admission.
type Config struct {
	DB *db.Replica
}

// New creates a DB limiter that locks a key's credits until admission commits.
func New(config Config) (*service, error) {
	if err := assert.NotNil(config.DB, "DB is required"); err != nil {
		return nil, fmt.Errorf("invalid usagelimiter service config: %w", err)
	}

	return &service{db: config.DB}, nil
}

// NewRedisWithCounter creates a counter-based limiter with transactional DB fallback.
func NewRedisWithCounter(config RedisConfig) (Service, error) {
	return NewCounter(CounterConfig{
		DB:                  config.DB,
		FindKeyCredits:      config.FindKeyCredits,
		DecrementKeyCredits: config.DecrementKeyCredits,
		Counter:             config.Counter,
		TTL:                 config.TTL,
		ReplayWorkers:       defaultReplayWorkers,
	})
}
