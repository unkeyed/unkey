package environment

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Service implements the EnvironmentService Restate virtual object for durable
// environment deletion. The virtual object key is the environment ID.
type Service struct {
	hydrav1.UnimplementedEnvironmentServiceServer
	db    db.Database
	admin *restateadmin.Client
}

var _ hydrav1.EnvironmentServiceServer = (*Service)(nil)

// Config holds configuration for creating a [Service].
type Config struct {
	DB db.Database

	// Admin cancels in-flight deployment invocations before the env delete
	// cascade drops deployment rows. Required.
	Admin *restateadmin.Client
}

// New creates a [Service] with the given configuration.
func New(cfg Config) (*Service, error) {
	if err := assert.NotNil(cfg.Admin, "Admin must not be nil"); err != nil {
		return nil, err
	}
	return &Service{
		UnimplementedEnvironmentServiceServer: hydrav1.UnimplementedEnvironmentServiceServer{},
		db:                                    cfg.DB,
		admin:                                 cfg.Admin,
	}, nil
}
