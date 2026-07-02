package app

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Service implements the AppService Restate virtual object for durable
// app deletion. The virtual object key is the app ID.
type Service struct {
	hydrav1.UnimplementedAppServiceServer
	db        db.Database
	auditlogs auditlogs.AuditLogService
}

var _ hydrav1.AppServiceServer = (*Service)(nil)

// Config holds configuration for creating a [Service].
type Config struct {
	DB db.Database

	// Auditlogs writes the app.delete event as a durable step inside the
	// deletion workflow, so the audit record is tied to the retried unit
	// rather than the synchronous RPC that enqueued it.
	Auditlogs auditlogs.AuditLogService
}

// New creates a [Service] with the given configuration.
func New(cfg Config) (*Service, error) {
	if err := assert.NotNil(cfg.Auditlogs, "Auditlogs must not be nil"); err != nil {
		return nil, err
	}
	return &Service{
		UnimplementedAppServiceServer: hydrav1.UnimplementedAppServiceServer{},
		db:                            cfg.DB,
		auditlogs:                     cfg.Auditlogs,
	}, nil
}
