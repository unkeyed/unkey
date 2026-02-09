package cluster

import (
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/db"
)

// Service implements [ctrlv1connect.ClusterServiceHandler] to synchronize desired state
// between the control plane and krane agents. It provides streaming RPCs for watching
// deployment and sentinel changes, point queries for fetching individual resource states,
// and status reporting endpoints for agents to report observed state back to the control plane.
type Service struct {
	ctrlv1connect.UnimplementedClusterServiceHandler
	db     db.Database
	bearer string
}

// Config holds the configuration for creating a new cluster [Service].
type Config struct {
	// Database provides read and write access for querying and updating resource state.
	Database db.Database

	// Bearer is the authentication token that agents must provide in the Authorization header.
	Bearer string
}

// New creates a new cluster [Service] with the given configuration. The returned service
// is ready to be registered with a Connect server.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedClusterServiceHandler: ctrlv1connect.UnimplementedClusterServiceHandler{},
		db:                                 cfg.Database,
		bearer:                             cfg.Bearer,
	}
}

var _ ctrlv1connect.ClusterServiceHandler = (*Service)(nil)
