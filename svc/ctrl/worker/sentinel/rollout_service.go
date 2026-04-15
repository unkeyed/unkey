package sentinel

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// RolloutService implements the SentinelRolloutService Restate virtual object
// for orchestrating progressive sentinel image rollouts across the fleet.
type RolloutService struct {
	hydrav1.UnimplementedSentinelRolloutServiceServer
	db db.Database
}

var _ hydrav1.SentinelRolloutServiceServer = (*RolloutService)(nil)

// RolloutConfig holds the configuration for the sentinel rollout service.
type RolloutConfig struct {
	DB db.Database
}

// NewRolloutService creates a new sentinel rollout service.
func NewRolloutService(cfg RolloutConfig) *RolloutService {
	return &RolloutService{
		UnimplementedSentinelRolloutServiceServer: hydrav1.UnimplementedSentinelRolloutServiceServer{},
		db: cfg.DB,
	}
}
