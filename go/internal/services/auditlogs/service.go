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

// Config contains the dependencies needed to create a new audit log service.
type Config struct {
	// DB is the database connection used to store audit logs
	DB     db.Database
	// Logger is used for internal logging of the service
	Logger logging.Logger
}

// New creates a new audit log service with the provided configuration.
// It returns a service that implements the AuditLogService interface.
//
// Example usage:
//
//	svc := auditlogs.New(auditlogs.Config{
//		DB:     database,
//		Logger: logger,
//	})
//
//	// Record an audit log
//	err := svc.Insert(ctx, nil, []auditlog.AuditLog{
//		{
//			Event:       "user.login",
//			WorkspaceID: "ws_123",
//			Display:     "User logged in",
//			ActorType:   "user",
//			ActorID:     "user_123",
//			// ...additional fields
//		},
//	})
func New(cfg Config) *service {
	return &service{
		db:     cfg.DB,
		logger: cfg.Logger,
	}
}
