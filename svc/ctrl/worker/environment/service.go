package environment

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// Service implements the EnvironmentService Restate virtual object for durable
// environment deletion. The virtual object key is the environment ID.
type Service struct {
	hydrav1.UnimplementedEnvironmentServiceServer
	db db.Database
}

var _ hydrav1.EnvironmentServiceServer = (*Service)(nil)

// Config holds configuration for creating a [Service].
type Config struct {
	DB db.Database
}

// New creates a [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedEnvironmentServiceServer: hydrav1.UnimplementedEnvironmentServiceServer{},
		db:                                    cfg.DB,
	}
}
