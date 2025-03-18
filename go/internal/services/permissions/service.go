package permissions

import (
	"time"

	"github.com/unkeyed/unkey/go/pkg/cache"
	cacheMiddleware "github.com/unkeyed/unkey/go/pkg/cache/middleware"
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

type Config struct {
	DB     db.Database
	Logger logging.Logger
	Clock  clock.Clock
}

func New(config Config) (*service, error) {

	c, err := cache.New[string, []string](cache.Config[string, []string]{
		// How long the data is considered fresh
		// Subsequent requests in this time will try to use the cache
		Fresh:   10 * time.Second,
		Stale:   60 * time.Second,
		Logger:  config.Logger,
		MaxSize: 1_000_000,

		Resource: "permissions",
		Clock:    config.Clock,
	})
	if err != nil {
		return nil, err
	}

	return &service{
		db:     config.DB,
		logger: config.Logger,
		rbac:   rbac.New(),
		cache:  cacheMiddleware.WithTracing(c),
	}, nil
}
