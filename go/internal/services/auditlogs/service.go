package auditlogs

import (
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type service struct {
	db     db.Database
	logger logging.Logger
}

var _ AuditLogService = (*service)(nil)

type Config struct {
	DB     db.Database
	Logger logging.Logger
}

func New(cfg Config) *service {
	return &service{
		db:     cfg.DB,
		logger: cfg.Logger,
	}
}
