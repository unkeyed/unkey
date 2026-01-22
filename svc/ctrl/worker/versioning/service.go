package versioning

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
)

// Service provides per-region, monotonically increasing versions for state sync.
//
// This is a Restate virtual object that maintains a durable counter per region.
// Each call to [Service.NextVersion] atomically increments and returns the next
// version number for that region, with exactly-once semantics guaranteed by Restate.
// The service is stateless; all state is stored in Restate's virtual object storage.
type Service struct {
	hydrav1.UnimplementedVersioningServiceServer
}

var _ hydrav1.VersioningServiceServer = (*Service)(nil)

// New creates a new versioning service instance. The returned service should be
// registered with a Restate router using [hydrav1.NewVersioningServiceServer].
func New() *Service {
	return &Service{
		UnimplementedVersioningServiceServer: hydrav1.UnimplementedVersioningServiceServer{},
	}
}
