// Package buildslot implements the BuildSlotService Restate virtual object,
// which caps how many deployments in a workspace can be actively building at
// the same time.
//
// The virtual object is keyed by workspace_id so that all AcquireOrWait and
// Release calls for a given workspace are serialized, making slot management
// race-free even when multiple (app, environment)-keyed Deploy workflows ask
// for slots concurrently.
//
// State is push-based via Restate awakeables:
//   - `active_slots` holds the set of deployment IDs currently building
//   - `prod_wait_list` is a FIFO of production waiters
//   - `preview_wait_list` is a FIFO of non-production waiters
//
// Both wait lists store {deployment_id, awakeable_id} entries. Production
// deployments respect the workspace's max_concurrent_builds quota — they
// don't bypass the cap — but Release drains prod_wait_list before
// preview_wait_list so a hot-fix priority-queues ahead of preview builds.
//
// This avoids the journal bloat of a poll loop in the Deploy handler: a
// waiting handler is parked on a single Restate wait operation until
// explicitly woken.
package buildslot

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// Service implements the BuildSlotService Restate virtual object.
// Key: workspace_id.
type Service struct {
	hydrav1.UnimplementedBuildSlotServiceServer
	db db.Database
}

var _ hydrav1.BuildSlotServiceServer = (*Service)(nil)

// Config holds configuration for creating a [Service].
type Config struct {
	DB db.Database
}

// New creates a [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedBuildSlotServiceServer: hydrav1.UnimplementedBuildSlotServiceServer{},
		db:                                  cfg.DB,
	}
}
