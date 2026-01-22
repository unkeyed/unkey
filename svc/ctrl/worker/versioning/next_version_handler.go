package versioning

import (
	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
)

// versionStateKey is the Restate state key used to store the version counter.
// Each virtual object instance (keyed by region) has its own isolated state,
// so this key is scoped to the region.
const versionStateKey = "version"

// NextVersion atomically increments and returns the next version number.
//
// The version is durably stored in Restate's virtual object state, guaranteeing
// monotonically increasing values within each region. Version numbers start at 1
// (the first call to a new region returns 1, not 0). Restate guarantees exactly-once
// execution, so retried invocations return the same version that was originally
// assigned.
//
// Returns an error only if Restate state operations fail, which indicates an
// infrastructure problem. On success, the returned version is always positive.
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
// This is useful for stale cursor detection: if a client's cursor version is
// older than the minimum retained version in the database (due to row deletion
// or compaction), the client must perform a full bootstrap instead of
// incremental sync.
//
// Returns 0 if no versions have been generated yet for this region.
func (s *Service) GetVersion(ctx restate.ObjectContext, _ *hydrav1.GetVersionRequest) (*hydrav1.GetVersionResponse, error) {
	current, err := restate.Get[uint64](ctx, versionStateKey)
	if err != nil {
		return nil, err
	}

	return &hydrav1.GetVersionResponse{
		Version: current,
	}, nil
}
