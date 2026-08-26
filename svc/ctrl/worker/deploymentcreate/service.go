package deploymentcreate

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/svc/ctrl/dedup"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Service implements the DeploymentCreateService Restate service.
type Service struct {
	hydrav1.UnimplementedDeploymentCreateServiceServer
	db        db.Database
	auditlogs auditlogs.AuditLogService
	dedup     *dedup.Service
}

var _ hydrav1.DeploymentCreateServiceServer = (*Service)(nil)

// Config holds the configuration for creating a [Service].
type Config struct {
	DB db.Database
	// Auditlogs records the deployment.create audit event in the same
	// transaction as the row insert. Required.
	Auditlogs auditlogs.AuditLogService
	// RestateAdmin is used to cancel in-flight Deploy invocations when this
	// create supersedes an older queued sibling. Optional: when nil, sibling
	// rows are still marked superseded but their invocations keep running.
	RestateAdmin *restateadmin.Client
}

// New creates a new [Service] with the provided configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedDeploymentCreateServiceServer: hydrav1.UnimplementedDeploymentCreateServiceServer{},
		db:        cfg.DB,
		auditlogs: cfg.Auditlogs,
		dedup:     dedup.New(cfg.DB, cfg.RestateAdmin),
	}
}
