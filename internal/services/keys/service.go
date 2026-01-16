package keys

import (
	"fmt"

	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/rbac"
)

// Config holds the configuration for creating a new keys service instance.
type Config struct {
	Logger       logging.Logger        // Logger for service operations
	DB           db.Database           // Database connection
	RateLimiter  ratelimit.Service     // Rate limiting service
	RBAC         *rbac.RBAC            // Role-based access control
	Clickhouse   clickhouse.ClickHouse // Clickhouse for telemetry
	Region       string                // Geographic region identifier
	UsageLimiter usagelimiter.Service  // Redis Counter for usage limiting

	KeyCache cache.Cache[string, db.CachedKeyData] // Cache for key lookups with pre-parsed data
}

type service struct {
	logger       logging.Logger
	db           db.Database
	raterLimiter ratelimit.Service
	usageLimiter usagelimiter.Service
	rbac         *rbac.RBAC
	clickhouse   clickhouse.ClickHouse
	region       string

	// hash -> cached key data (includes pre-parsed IP whitelist)
	keyCache cache.Cache[string, db.CachedKeyData]
}

// New creates a new keys service instance with the provided configuration.
func New(config Config) (*service, error) {
	if err := assert.All(
		assert.NotNil(config.Logger, "logger is required"),
		assert.NotNil(config.DB, "db is required"),
		assert.NotNil(config.RateLimiter, "rate limiter is required"),
		assert.NotNil(config.RBAC, "rbac is required"),
		assert.NotNil(config.Clickhouse, "clickhouse is required"),
		assert.NotNil(config.UsageLimiter, "usage limiter is required"),
		assert.NotNil(config.KeyCache, "key cache is required"),
	); err != nil {
		return nil, fmt.Errorf("invalid keys service config: %w", err)
	}

	return &service{
		logger:       config.Logger,
		db:           config.DB,
		rbac:         config.RBAC,
		raterLimiter: config.RateLimiter,
		usageLimiter: config.UsageLimiter,
		clickhouse:   config.Clickhouse,
		region:       config.Region,
		keyCache:     config.KeyCache,
	}, nil
}

// Close gracefully shuts down the keys service and its dependencies.
func (s *service) Close() error {
	return s.usageLimiter.Close()
}
