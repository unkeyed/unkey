package versioning

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
)

// Service provides per-region, monotonically increasing versions for state sync.
//
// This is a Restate virtual object that maintains a durable counter per region.
// Each call to NextVersion atomically increments and returns the next version
// number for that region, with exactly-once semantics guaranteed by Restate.
type Service struct {
	hydrav1.UnimplementedVersioningServiceServer
}

var _ hydrav1.VersioningServiceServer = (*Service)(nil)

func New() *Service {
	return &Service{
		UnimplementedVersioningServiceServer: hydrav1.UnimplementedVersioningServiceServer{},
	}
}
