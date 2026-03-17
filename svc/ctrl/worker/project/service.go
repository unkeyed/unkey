package project

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// Service implements the ProjectService Restate virtual object for durable
// project deletion. The virtual object key is the project ID.
type Service struct {
	hydrav1.UnimplementedProjectServiceServer
	db db.Database
}

var _ hydrav1.ProjectServiceServer = (*Service)(nil)

// Config holds configuration for creating a [Service].
type Config struct {
	DB db.Database
}

// New creates a [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedProjectServiceServer: hydrav1.UnimplementedProjectServiceServer{},
		db:                                cfg.DB,
	}
}
