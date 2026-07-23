package project

import (
	"context"

	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// DepotProjects deletes the Depot build project backing an Unkey project.
// Satisfied by depotclient.Client (and Noop). Implementations must treat
// an already-deleted project as success.
type DepotProjects interface {
	DeleteProject(ctx context.Context, depotProjectID string) error
}

// Service implements the ProjectService Restate virtual object for durable
// project deletion. The virtual object key is the project ID.
type Service struct {
	hydrav1.UnimplementedProjectServiceServer
	db        db.Database
	auditlogs auditlogs.AuditLogService
	depot     DepotProjects
}

var _ hydrav1.ProjectServiceServer = (*Service)(nil)

// Config holds configuration for creating a [Service].
type Config struct {
	// DB is the primary application database. Must not be nil.
	DB db.Database

	// Auditlogs writes the project.delete event as a durable step inside the
	// deletion workflow, tying the audit record to the retried unit.
	Auditlogs auditlogs.AuditLogService

	// Depot deletes the project's Depot build project during teardown.
	// Must not be nil — pass depotclient.NewNoop() where Depot is not
	// configured.
	Depot DepotProjects
}

// New creates a [Service] with the given configuration.
func New(cfg Config) (*Service, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Auditlogs, "Auditlogs must not be nil"),
		assert.NotNil(cfg.Depot, "Depot must not be nil; use depotclient.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &Service{
		UnimplementedProjectServiceServer: hydrav1.UnimplementedProjectServiceServer{},
		db:                                cfg.DB,
		auditlogs:                         cfg.Auditlogs,
		depot:                             cfg.Depot,
	}, nil
}
