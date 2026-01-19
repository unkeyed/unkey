package versioning

import (
	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
)

const versionStateKey = "version"

// NextVersion atomically increments and returns the next version number.
//
// The version is durably stored in Restate's virtual object state, guaranteeing monotonically increasing values
func (s *Service) NextVersion(ctx restate.ObjectContext, _ *hydrav1.NextVersionRequest) (*hydrav1.NextVersionResponse, error) {
	current, err := restate.Get[uint64](ctx, versionStateKey)
	if err != nil {
		return nil, err
	}

	next := current + 1
	restate.Set(ctx, versionStateKey, next)

	return &hydrav1.NextVersionResponse{
		Version: next,
	}, nil
}

// GetVersion returns the current version without incrementing.
//
// Useful for stale cursor detection: if a client's version is older than the
// minimum retained version in the database, they must perform a full bootstrap.
func (s *Service) GetVersion(ctx restate.ObjectContext, _ *hydrav1.GetVersionRequest) (*hydrav1.GetVersionResponse, error) {
	current, err := restate.Get[uint64](ctx, versionStateKey)
	if err != nil {
		return nil, err
	}

	return &hydrav1.GetVersionResponse{
		Version: current,
	}, nil
}
