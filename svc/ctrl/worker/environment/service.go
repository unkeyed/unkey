package environment

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
)

// Service implements the EnvironmentService Restate virtual object for durable
// environment deletion. The virtual object key is the environment ID.
type Service struct {
	hydrav1.UnimplementedEnvironmentServiceServer
	db        db.Database
	admin     *restateadmin.Client
	auditlogs auditlogs.AuditLogService
}

var _ hydrav1.EnvironmentServiceServer = (*Service)(nil)

// Config holds configuration for creating a [Service].
type Config struct {
	DB db.Database

	// Admin cancels in-flight deployment invocations before the env delete
	// cascade drops deployment rows. Required.
	Admin *restateadmin.Client

	// Auditlogs writes the environment.delete event as a durable step inside
	// the deletion workflow. Environment deletes are cascade-only, so the event
	// always carries the actor and correlation ID of the parent teardown.
	Auditlogs auditlogs.AuditLogService
}

// New creates a [Service] with the given configuration.
func New(cfg Config) (*Service, error) {
	if err := assert.All(
		assert.NotNil(cfg.Admin, "Admin must not be nil"),
		assert.NotNil(cfg.Auditlogs, "Auditlogs must not be nil"),
	); err != nil {
		return nil, err
	}
	return &Service{
		UnimplementedEnvironmentServiceServer: hydrav1.UnimplementedEnvironmentServiceServer{},
		db:                                    cfg.DB,
		admin:                                 cfg.Admin,
		auditlogs:                             cfg.Auditlogs,
	}, nil
}
