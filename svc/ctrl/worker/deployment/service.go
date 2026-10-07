package deployment

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const (
	// regionReadyTimeout bounds WakeDeployment's health poll
	regionReadyTimeout = 15 * time.Minute

	// runMaxAttempts turns a persistently failing Run into a terminal error
	runMaxAttempts uint = 5
)

// VirtualObject serialises all mutations targeting a single deployment. See
// the package documentation for an explanation of the virtual object keying
// and the nonce-based last-writer-wins mechanism used for scheduled state
// changes.
type VirtualObject struct {
	hydrav1.UnimplementedDeploymentServiceServer
	db        db.Database
	auditlogs auditlogs.AuditLogService
}

var _ hydrav1.DeploymentServiceServer = (*VirtualObject)(nil)

// Config holds the dependencies required to create a VirtualObject.
type Config struct {
	// DB is the main database connection for workspace, project, and deployment data.
	DB db.Database

	// Auditlogs records API-originated Stop and Wake calls
	Auditlogs auditlogs.AuditLogService
}

// New creates a new VirtualObject from the given configuration.
func New(cfg Config) (*VirtualObject, error) {
	if err := assert.NotNil(cfg.Auditlogs, "Auditlogs must not be nil"); err != nil {
		return nil, err
	}

	return &VirtualObject{
		UnimplementedDeploymentServiceServer: hydrav1.UnimplementedDeploymentServiceServer{},
		db:                                   cfg.DB,
		auditlogs:                            cfg.Auditlogs,
	}, nil
}

func (v *VirtualObject) loadDeployment(ctx restate.ObjectContext, deploymentID, purpose string) (db.FindDeploymentWithEnvironmentAndAppRow, error) {
	return restate.Run(ctx, func(runCtx restate.RunContext) (db.FindDeploymentWithEnvironmentAndAppRow, error) {
		row, err := v.db.FindDeploymentWithEnvironmentAndApp(runCtx, deploymentID)
		if err != nil {
			if db.IsNotFound(err) {
				return db.FindDeploymentWithEnvironmentAndAppRow{}, restate.ToTerminalError(fmt.Errorf("deployment not found"), restate.WithErrorCode(404))
			}
			return db.FindDeploymentWithEnvironmentAndAppRow{}, fmt.Errorf("load deployment: %w", err)
		}
		return row, nil
	}, restate.WithName("load deployment for "+purpose), restate.WithMaxRetryAttempts(runMaxAttempts))
}
