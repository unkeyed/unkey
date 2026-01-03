package auditlogs

import (
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// service implements the AuditLogService interface, providing audit logging
// functionality with database persistence and structured logging capabilities.
// The service handles batch insertion of audit logs within transactional contexts.
type service struct {
	db     db.Database
	logger logging.Logger
}

var _ AuditLogService = (*service)(nil)

// Config contains the dependencies required to initialize an audit log service.
// Both fields are required for proper service operation.
type Config struct {
	DB     db.Database    // Database interface for audit log persistence
	Logger logging.Logger // Structured logger for service operations and error reporting
}

// New creates a new audit log service instance with the provided configuration.
// The returned service implements AuditLogService and is ready for immediate use.
// Both database and logger dependencies must be provided in the config.
func New(cfg Config) *service {
	return &service{
		db:     cfg.DB,
		logger: cfg.Logger,
	}
}
