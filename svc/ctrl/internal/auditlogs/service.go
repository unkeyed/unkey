package auditlogs

import (
	"fmt"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// service persists audit events through the control plane database.
type service struct {
	db db.Database
}

var _ AuditLogService = (*service)(nil)

// Config contains the dependencies required to initialize an audit log service.
type Config struct {
	// DB is the control plane database used for audit log persistence.
	DB db.Database
}

// New creates an audit log service instance for the control plane.
func New(cfg Config) (*service, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "db is required"),
	); err != nil {
		return nil, fmt.Errorf("invalid auditlogs service config: %w", err)
	}

	return &service{
		db: cfg.DB,
	}, nil
}
