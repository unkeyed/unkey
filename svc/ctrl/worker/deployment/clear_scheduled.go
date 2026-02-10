package deployment

import (
	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
)

// ClearScheduledStateChanges removes the pending transition record,
// cancelling any scheduled desired state change. A previously enqueued
// ChangeDesiredState call may still fire after the delay, but it will
// encounter a nil transition and return a terminal error rather than applying
// the stale state change.
func (v *VirtualObject) ClearScheduledStateChanges(ctx restate.ObjectContext, req *hydrav1.ClearScheduledStateChangesRequest) (*hydrav1.ClearScheduledStateChangesResponse, error) {

	restate.Clear(ctx, transitionKey)
	return &hydrav1.ClearScheduledStateChangesResponse{}, nil

}
