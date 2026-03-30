package keys

import (
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/rbac"
)

// Config holds the configuration for creating a new keys service instance.
type Config struct {
	DB           db.Database          // Database connection
	RateLimiter  ratelimit.Service    // Rate limiting service
	RBAC         *rbac.RBAC           // Role-based access control
	Region       string               // Geographic region identifier
	UsageLimiter usagelimiter.Service // Redis Counter for usage limiting

	// KeyVerifications buffers key verification events for ClickHouse.
	KeyVerifications *batch.BatchProcessor[schema.KeyVerification]

	KeyCache   cache.Cache[string, db.CachedKeyData] // Cache for key lookups with pre-parsed data
	QuotaCache cache.Cache[string, db.Quotas]        // Cache for workspace quota lookups
}

type service struct {
	db               db.Database
	rateLimiter      ratelimit.Service
	usageLimiter     usagelimiter.Service
	rbac             *rbac.RBAC
	keyVerifications *batch.BatchProcessor[schema.KeyVerification]
	region           string

	// hash -> cached key data (includes pre-parsed IP whitelist)
	keyCache cache.Cache[string, db.CachedKeyData]

	// workspace_id -> quota (for workspace rate limiting)
	quotaCache cache.Cache[string, db.Quotas]
}

// New creates a new keys service instance with the provided configuration.
func New(config Config) (*service, error) {
	kv := config.KeyVerifications
	if kv == nil {
		kv = batch.NewNoop[schema.KeyVerification]()
	}

	return &service{
		db:               config.DB,
		rbac:             config.RBAC,
		rateLimiter:      config.RateLimiter,
		usageLimiter:     config.UsageLimiter,
		keyVerifications: kv,
		region:           config.Region,
		keyCache:         config.KeyCache,
		quotaCache:       config.QuotaCache,
	}, nil
}

// Close gracefully shuts down the keys service and its dependencies.
func (s *service) Close() error {
	return s.usageLimiter.Close()
}
