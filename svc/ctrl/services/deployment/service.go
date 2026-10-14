package deployment

import (
	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Service implements the DeployService ConnectRPC API. It coordinates
// deployment operations by persisting state to the database and delegating
// workflow execution to Restate.
type Service struct {
	ctrlv1connect.UnimplementedDeployServiceHandler
	db      db.Database
	restate *restateingress.Client
	bearer  string
}

// Config holds the configuration for creating a new [Service].
type Config struct {
	// Database provides read/write access to deployment metadata.
	Database db.Database
	// Restate is the ingress client for triggering durable workflows.
	Restate *restateingress.Client
	// Bearer is the preshared token that callers must provide in the Authorization header.
	Bearer string
}

// New creates a new [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedDeployServiceHandler: ctrlv1connect.UnimplementedDeployServiceHandler{},
		db:                                cfg.Database,
		restate:                           cfg.Restate,
		bearer:                            cfg.Bearer,
	}
}
