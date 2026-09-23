package deploy

import (
	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
)

// Deprecated: Use NotifyReadiness. Queued notifications still use this endpoint.
func (w *Workflow) NotifyInstancesReady(ctx restate.WorkflowSharedContext, _ *hydrav1.NotifyInstancesReadyRequest) (*hydrav1.NotifyInstancesReadyResponse, error) {
	_, err := w.NotifyReadiness(ctx, &hydrav1.NotifyReadinessRequest{State: hydrav1.NotifyReadinessRequest_STATE_READY})
	if err != nil {
		return nil, err
	}
	return &hydrav1.NotifyInstancesReadyResponse{}, nil
}
