package deployschedule

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// Service implements the DeploySchedulerService virtual object. Each instance is
// keyed by workspace_id and manages build concurrency slots for that workspace.
//
// It handles:
//   - Routing deploy requests to per-branch DeployQueueService instances
//   - Allocating and releasing build slots (up to max_concurrent_builds)
//   - Maintaining a waitlist of queue keys that are waiting for build slots
type Service struct {
	hydrav1.UnimplementedDeploySchedulerServiceServer
	db db.Database
}

var _ hydrav1.DeploySchedulerServiceServer = (*Service)(nil)

// Config holds the configuration for creating a new [Service].
type Config struct {
	DB db.Database
}

// New creates a new deploy scheduler service instance.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedDeploySchedulerServiceServer: hydrav1.UnimplementedDeploySchedulerServiceServer{},
		db: cfg.DB,
	}
}
