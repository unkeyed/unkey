package keys

import (
	"github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/rbac"
)

// Config holds the configuration for creating a new keys service instance.
type Config struct {
	DB          mysql.MySQL       // Database with read/write replicas
	RateLimiter ratelimit.Service // Rate limiting service
	RBAC        *rbac.RBAC        // Role-based access control
	Region      string            // Geographic region identifier
	// Source tags every verification this service emits with where it
	// originated (schema.SourceAPI for the public API, schema.SourceGateway
	// for the Deploy gateway), so billing can exclude gateway traffic.
	Source       string
	UsageLimiter usagelimiter.Service // Redis Counter for usage limiting

	KeyCache cache.Cache[string, db.CachedKeyData] // Cache for key lookups with pre-parsed data
}

type service struct {
	db           *db.Database
	rateLimiter  ratelimit.Service
	usageLimiter usagelimiter.Service
	rbac         *rbac.RBAC
	region       string
	source       string

	// hash -> cached key data (includes pre-parsed IP whitelist)
	keyCache cache.Cache[string, db.CachedKeyData]
}

// New creates a new keys service instance with the provided configuration.
func New(config Config) (*service, error) {
	return &service{
		db:           db.New(config.DB),
		rbac:         config.RBAC,
		rateLimiter:  config.RateLimiter,
		usageLimiter: config.UsageLimiter,
		region:       config.Region,
		source:       config.Source,
		keyCache:     config.KeyCache,
	}, nil
}

// Close gracefully shuts down the keys service and its dependencies.
func (s *service) Close() error {
	return s.usageLimiter.Close()
}
