package app

import (
	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/db"
)

// Service implements the AppService ConnectRPC API. Creates are handled
// directly via DB transactions; deletes are delegated to Restate for
// durable cleanup of associated resources.
type Service struct {
	ctrlv1connect.UnimplementedAppServiceHandler
	db      db.Database
	restate *restateingress.Client
}

// Config holds the configuration for creating a new [Service].
type Config struct {
	Database db.Database
	Restate  *restateingress.Client
}

// New creates a new [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedAppServiceHandler: ctrlv1connect.UnimplementedAppServiceHandler{},
		db:                             cfg.Database,
		restate:                        cfg.Restate,
	}
}
