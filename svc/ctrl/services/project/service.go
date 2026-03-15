package project

import (
	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/ctrl/services/app"
)

// Service implements the ProjectService ConnectRPC API. Creates delegate
// app creation to AppService; deletes are delegated to Restate for
// durable cleanup of associated resources.
type Service struct {
	ctrlv1connect.UnimplementedProjectServiceHandler
	db         db.Database
	restate    *restateingress.Client
	appService *app.Service
}

// Config holds the configuration for creating a new [Service].
type Config struct {
	Database   db.Database
	Restate    *restateingress.Client
	AppService *app.Service
}

// New creates a new [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedProjectServiceHandler: ctrlv1connect.UnimplementedProjectServiceHandler{},
		db:                                 cfg.Database,
		restate:                            cfg.Restate,
		appService:                         cfg.AppService,
	}
}
