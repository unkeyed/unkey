package project

import (
	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/db"
)

// Service implements the ProjectService ConnectRPC API. Projects are created
// empty; deletes are delegated to Restate for durable cleanup of associated
// resources.
type Service struct {
	ctrlv1connect.UnimplementedProjectServiceHandler
	db      db.Database
	restate *restateingress.Client
	bearer  string
}

// Config holds the configuration for creating a new [Service].
type Config struct {
	// Database provides read and write access for managing projects.
	Database db.Database

	// Restate is the ingress client used to trigger durable project deletion workflows.
	Restate *restateingress.Client

	// Bearer is the preshared token that callers must provide in the Authorization header.
	Bearer string
}

// New creates a new [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedProjectServiceHandler: ctrlv1connect.UnimplementedProjectServiceHandler{},
		db:                                 cfg.Database,
		restate:                            cfg.Restate,
		bearer:                             cfg.Bearer,
	}
}
