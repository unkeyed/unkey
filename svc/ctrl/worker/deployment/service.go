package deployment

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// VirtualObject serialises all mutations targeting a single deployment. See
// the package documentation for an explanation of the virtual object keying
// and the nonce-based last-writer-wins mechanism used for scheduled state
// changes.
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
