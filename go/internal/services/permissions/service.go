package permissions

import (
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
)

type service struct {
	db     db.Database
	logger logging.Logger
	rbac   *rbac.RBAC

	// keyId -> permissions
	cache cache.Cache[string, []string]
}

var _ PermissionService = (*service)(nil)

// Config contains the dependencies needed to create a new permission service.
type Config struct {
	// Database connection
	DB db.Database
	// Logger for service operations
	Logger logging.Logger
	// Clock for time-related operations
	Clock clock.Clock
	// Cache for permissions by key ID
	Cache cache.Cache[string, []string]
}

// New creates a new permission service with the provided configuration.
// It returns a service that implements the PermissionService interface.
//
// Example:
//
//	permSvc, err := permissions.New(permissions.Config{
//		DB:     database,
//		Logger: logger,
//		Clock:  clock.New(),
//		Cache:  caches.PermissionsByKeyId,
//	})
//	if err != nil {
//		log.Fatalf("Failed to create permission service: %v", err)
//	}
func New(config Config) (*service, error) {

	return &service{
		db:     config.DB,
		logger: config.Logger,
		rbac:   rbac.New(),
		cache:  config.Cache,
	}, nil
}
