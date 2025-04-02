package auditlogs

import (
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type service struct {
	db          db.Database
	logger      logging.Logger
	bucketCache cache.Cache[string, string]
}

var _ AuditLogService = (*service)(nil)

type Config struct {
	DB          db.Database
	Logger      logging.Logger
	BucketCache cache.Cache[string, string]
}

func New(cfg Config) *service {
	return &service{
		db:          cfg.DB,
		logger:      cfg.Logger,
		bucketCache: cfg.BucketCache,
	}
}
