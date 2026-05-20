package project

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
)

// Service implements the ProjectService Restate virtual object for durable
// project deletion. The virtual object key is the project ID.
type Service struct {
	hydrav1.UnimplementedProjectServiceServer
	db    db.Database
	admin *restateadmin.Client
}

var _ hydrav1.ProjectServiceServer = (*Service)(nil)

// Config holds configuration for creating a [Service].
type Config struct {
	DB db.Database

	// Admin is used to cancel in-flight deployment invocations during the
	// project delete cascade. Optional: when nil, the Delete handler still
	// runs but skips the Restate-side cancel.
	Admin *restateadmin.Client
}

// New creates a [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedProjectServiceServer: hydrav1.UnimplementedProjectServiceServer{},
		db:                                cfg.DB,
		admin:                             cfg.Admin,
	}
}
