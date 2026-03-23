package deployqueue

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// Service implements the DeployQueueService virtual object. Each instance is
// keyed by app_id:branch and manages a priority queue of deploy requests for
// that specific app+branch combination.
//
// It handles:
//   - Priority ordering (production > preview, FIFO within each tier)
//   - Superseding: new deploys cancel older queued/active ones for the same branch
//   - Build slot coordination with the workspace-level DeploySchedulerService
type Service struct {
	hydrav1.UnimplementedDeployQueueServiceServer
	db db.Database
}

var _ hydrav1.DeployQueueServiceServer = (*Service)(nil)

// Config holds the configuration for creating a new [Service].
type Config struct {
	DB db.Database
}

// New creates a new deployment queue service instance.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedDeployQueueServiceServer: hydrav1.UnimplementedDeployQueueServiceServer{},
		db:                                    cfg.DB,
	}
}
