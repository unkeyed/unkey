package project

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Service implements the ProjectService Restate virtual object for durable
// project deletion. The virtual object key is the project ID.
type Service struct {
	hydrav1.UnimplementedProjectServiceServer
	db        db.Database
	auditlogs auditlogs.AuditLogService
}

var _ hydrav1.ProjectServiceServer = (*Service)(nil)

// Config holds configuration for creating a [Service].
type Config struct {
	DB db.Database

	// Auditlogs writes the project.delete event as a durable step inside the
	// deletion workflow, tying the audit record to the retried unit.
	Auditlogs auditlogs.AuditLogService
}

// New creates a [Service] with the given configuration.
func New(cfg Config) (*Service, error) {
	if err := assert.NotNil(cfg.Auditlogs, "Auditlogs must not be nil"); err != nil {
		return nil, err
	}
	return &Service{
		UnimplementedProjectServiceServer: hydrav1.UnimplementedProjectServiceServer{},
		db:                                cfg.DB,
		auditlogs:                         cfg.Auditlogs,
	}, nil
}
