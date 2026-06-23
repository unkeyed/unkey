package project

import (
	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/db"
)

// Service implements the ProjectService ConnectRPC API. Projects are created
// empty; deletes are delegated to Restate for durable cleanup of associated
// resources.
type Service struct {
	ctrlv1connect.UnimplementedProjectServiceHandler
	db                db.Database
	restate           *restateingress.Client
	auditlogs         auditlogs.AuditLogService
	bearer            string
	enforceDeployGate bool
}

// Config holds the configuration for creating a new [Service].
type Config struct {
	// Database provides read and write access for managing projects.
	Database db.Database

	// Restate is the ingress client used to trigger durable project deletion workflows.
	Restate *restateingress.Client

	// Auditlogs records project mutations within the same transaction as the write.
	Auditlogs auditlogs.AuditLogService

	// Bearer is the preshared token that callers must provide in the Authorization header.
	Bearer string

	// EnforceDeployGate hard-blocks project creation for workspaces without an
	// Unkey Deploy entitlement. When false (default), the gate runs in observe
	// mode: it logs when it would block but allows creation.
	EnforceDeployGate bool
}

// New creates a new [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedProjectServiceHandler: ctrlv1connect.UnimplementedProjectServiceHandler{},
		db:                                 cfg.Database,
		restate:                            cfg.Restate,
		auditlogs:                          cfg.Auditlogs,
		bearer:                             cfg.Bearer,
		enforceDeployGate:                  cfg.EnforceDeployGate,
	}
}
