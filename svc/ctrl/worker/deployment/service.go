package deployment

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// VirtualObject manages desired state transitions for a single deployment.
//
// It is a Restate virtual object keyed by deployment ID, which guarantees that
// only one state transition executes per deployment at a time. This serialization
// is critical because desired state changes can be scheduled with arbitrary delays
// and a newer schedule must be able to supersede an older one without races.
//
// The object exposes two operations. ScheduleDesiredStateChange records a
// pending transition and enqueues a delayed self-call. ChangeDesiredState
// executes the transition only if its nonce still matches the latest schedule,
// implementing last-writer-wins semantics so that superseded transitions
// become harmless no-ops.
type VirtualObject struct {
	hydrav1.UnimplementedDeploymentServiceServer
	db db.Database
}

var _ hydrav1.DeploymentServiceServer = (*VirtualObject)(nil)

// Config holds the dependencies required to create a VirtualObject.
type Config struct {
	// DB is the main database connection for workspace, project, and deployment data.
	DB db.Database
}

// New creates a new VirtualObject from the given configuration.
func New(cfg Config) *VirtualObject {
	return &VirtualObject{
		UnimplementedDeploymentServiceServer: hydrav1.UnimplementedDeploymentServiceServer{},
		db:                                   cfg.DB,
	}
}
