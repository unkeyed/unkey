package namespace

import (
	"fmt"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
	"golang.org/x/sync/singleflight"
)

// Config holds the dependencies for creating a new namespace service.
type Config struct {
	DB        db.Database
	Cache     cache.Cache[cache.ScopedKey, db.FindRatelimitNamespace]
	Auditlogs auditlogs.AuditLogService
}

type service struct {
	db           db.Database
	cache        cache.Cache[cache.ScopedKey, db.FindRatelimitNamespace]
	auditlogs    auditlogs.AuditLogService
	createFlight singleflight.Group
}

var _ Service = (*service)(nil)

// New creates a new namespace service with the provided configuration.
func New(cfg Config) (Service, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "db is required"),
		assert.NotNil(cfg.Cache, "cache is required"),
		assert.NotNil(cfg.Auditlogs, "auditlogs is required"),
	); err != nil {
		return nil, fmt.Errorf("invalid namespace service config: %w", err)
	}

	return &service{
		db:           cfg.DB,
		cache:        cfg.Cache,
		auditlogs:    cfg.Auditlogs,
		createFlight: singleflight.Group{},
	}, nil
}
