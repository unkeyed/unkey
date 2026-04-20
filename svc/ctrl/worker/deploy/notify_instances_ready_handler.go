package deploy

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
)

// NotifyInstancesReady resolves the awakeable stored by [Workflow.waitForDeployments]
// so the suspended Deploy handler can proceed. Called from
// services/cluster's ReportDeploymentStatus once krane reports enough
// instances as running to satisfy the per-region minimum-replica
// requirement.
//
// This is a SHARED handler so it can run concurrently with a suspended
// Deploy on the same VO key. Since the VO is keyed by deployment_id, each
// instance owns the awakeable for exactly one deployment.
func (w *Workflow) NotifyInstancesReady(
	ctx restate.ObjectSharedContext,
	req *hydrav1.NotifyInstancesReadyRequest,
) (*hydrav1.NotifyInstancesReadyResponse, error) {
	awakeableID, err := restate.Get[string](ctx, instancesReadyAwakeableKey)
	if err != nil {
		return nil, fmt.Errorf("get instances-ready awakeable: %w", err)
	}
	if awakeableID == "" {
		// No Deploy handler is currently waiting on this VO.
		return &hydrav1.NotifyInstancesReadyResponse{}, nil
	}

	restate.ResolveAwakeable[restate.Void](ctx, awakeableID, restate.Void{})
	return &hydrav1.NotifyInstancesReadyResponse{}, nil
}
