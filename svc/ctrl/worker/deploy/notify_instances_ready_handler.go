package deploy

import (
	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
)

// NotifyInstancesReady tolerates a repeat: a second krane report after the
// promise resolved must not fail
func (w *Workflow) NotifyInstancesReady(ctx restate.WorkflowSharedContext, _ *hydrav1.NotifyInstancesReadyRequest) (*hydrav1.NotifyInstancesReadyResponse, error) {
	if err := restate.Promise[restate.Void](ctx, instancesReadyPromise).Resolve(restate.Void{}); err != nil && !restate.IsTerminalError(err) {
		return nil, err
	}
	return &hydrav1.NotifyInstancesReadyResponse{}, nil
}
