package keys

import (
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/counter"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
)

// Config holds the configuration for creating a new keys service instance.
type Config struct {
	Logger      logging.Logger        // Logger for service operations
	DB          db.Database           // Database connection
	RateLimiter ratelimit.Service     // Rate limiting service
	RBAC        *rbac.RBAC            // Role-based access control
	Clickhouse  clickhouse.ClickHouse // Clickhouse for telemetry
	Region      string                // Geographic region identifier
	Counter     counter.Counter       // Redis Counter for usage limiting

	KeyCache cache.Cache[string, db.FindKeyForVerificationRow] // Cache for key lookups
}

type service struct {
	logger       logging.Logger
	db           db.Database
	raterLimiter ratelimit.Service
	usageLimiter usagelimiter.Service
	rbac         *rbac.RBAC
	clickhouse   clickhouse.ClickHouse
	region       string

	// hash -> key
	keyCache cache.Cache[string, db.FindKeyForVerificationRow]
}

// New creates a new keys service instance with the provided configuration.
func New(config Config) (*service, error) {
	ulSvc, err := usagelimiter.NewRedisWithCounter(usagelimiter.RedisConfig{
		Logger:  config.Logger,
		DB:      config.DB,
		Counter: config.Counter,
		TTL:     time.Minute * 10,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create usage limiter service: %w", err)
	}

	return &service{
		logger:       config.Logger,
		db:           config.DB,
		rbac:         config.RBAC,
		raterLimiter: config.RateLimiter,
		usageLimiter: ulSvc,
		clickhouse:   config.Clickhouse,
		region:       config.Region,
		keyCache:     config.KeyCache,
	}, nil
}

// Close gracefully shuts down the keys service and its dependencies.
func (s *service) Close() error {
	return s.usageLimiter.Close()
}
