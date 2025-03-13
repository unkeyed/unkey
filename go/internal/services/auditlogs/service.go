package auditlogs

import (
	"time"

	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/logging"
)

type service struct {
	db          db.Database
	logger      logging.Logger
	bucketCache cache.Cache[string, string]
}

var _ AuditLogService = (*service)(nil)

type Config struct {
	DB     db.Database
	Logger logging.Logger
}

func New(cfg Config) *service {
	c := cache.New(cache.Config[string, string]{
		MaxSize:  100_000,
		Fresh:    time.Minute,
		Stale:    time.Hour * 24,
		Resource: "audit_log_bucket_id_by_workspace_id_and_name",
		Logger:   cfg.Logger,
		Clock:    clock.New(),
	})

	return &service{
		db:          cfg.DB,
		logger:      cfg.Logger,
		bucketCache: c,
	}
}
