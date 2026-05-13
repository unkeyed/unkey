package buildslot

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Release frees the build slot held by a deployment and hands the slot to
// the next waiter — production waiters first, then preview.
//
// Three cases:
//  1. Deployment in active_slots: remove it, promote head waiter (prod first,
//     fallback to preview).
//  2. Deployment in either wait list: remove it (cancelled before slot came up).
//  3. Deployment in neither: no-op. Idempotent — safe to call from both
//     the happy path and the compensation stack.
//
// If a promoted waiter's workflow was already cancelled, the resolve lands
// on a dead handler. That waiter's own compensation eventually calls
// Release, which promotes the *next* waiter, so no slot is permanently lost.
func (s *Service) Release(
	ctx restate.ObjectContext,
	req *hydrav1.ReleaseSlotRequest,
) (*hydrav1.ReleaseSlotResponse, error) {
	workspaceID := restate.Key(ctx)
	deploymentID := req.GetDeploymentId()

	active, err := loadActiveSlots(ctx)
	if err != nil {
		return nil, fmt.Errorf("load active slots: %w", err)
	}

	prodWait, err := loadWaitList(ctx, stateKeyProdWaitList)
	if err != nil {
		return nil, fmt.Errorf("load prod wait list: %w", err)
	}

	previewWait, err := loadWaitList(ctx, stateKeyPreviewWaitList)
	if err != nil {
		return nil, fmt.Errorf("load preview wait list: %w", err)
	}

	_, held := active[deploymentID]
	if held {
		delete(active, deploymentID)
	}

	// Sweep both wait lists in case this is a cancelled waiter that never
	// got a slot.
	prodWait = removeFromWaitList(prodWait, deploymentID)
	previewWait = removeFromWaitList(previewWait, deploymentID)

	// Promote the next waiter — production first.
	if held {
		var promoted *waitEntry
		switch {
		case len(prodWait) > 0:
			promoted = &prodWait[0]
			prodWait = prodWait[1:]
		case len(previewWait) > 0:
			promoted = &previewWait[0]
			previewWait = previewWait[1:]
		}

		if promoted != nil {
			active[promoted.DeploymentID] = true
			restate.ResolveAwakeable(ctx, promoted.AwakeableID, true)

			logger.Info("build slot handed off",
				"workspace_id", workspaceID,
				"released", deploymentID,
				"promoted", promoted.DeploymentID,
				"active", len(active),
				"prod_wait", len(prodWait),
				"preview_wait", len(previewWait),
			)
		} else {
			logger.Info("build slot released",
				"workspace_id", workspaceID,
				"deployment_id", deploymentID,
				"active", len(active),
			)
		}
	}

	saveActiveSlots(ctx, active)
	saveWaitList(ctx, stateKeyProdWaitList, prodWait)
	saveWaitList(ctx, stateKeyPreviewWaitList, previewWait)

	return &hydrav1.ReleaseSlotResponse{}, nil
}

func removeFromWaitList(list []waitEntry, deploymentID string) []waitEntry {
	for i, w := range list {
		if w.DeploymentID == deploymentID {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}
