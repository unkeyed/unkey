package app

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// Service implements the AppService Restate virtual object for durable
// app deletion. The virtual object key is the app ID.
type Service struct {
	hydrav1.UnimplementedAppServiceServer
	db db.Database
}

var _ hydrav1.AppServiceServer = (*Service)(nil)

// Config holds configuration for creating a [Service].
type Config struct {
	DB db.Database
}

// New creates a [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedAppServiceServer: hydrav1.UnimplementedAppServiceServer{},
		db:                            cfg.DB,
	}
}
