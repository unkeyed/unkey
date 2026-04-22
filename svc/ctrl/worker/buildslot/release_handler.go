package buildslot

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Release frees the build slot held by a deployment and hands the slot to
// the next live waiter — production waiters first, then preview.
//
// Three cases:
//  1. Deployment in active_slots: remove it, promote next live waiter (prod
//     first, fallback to preview). Waiters whose DB status is terminal are
//     skipped: their Deploy workflow is already dead, so resolving their
//     awakeable would park the slot on a corpse.
//  2. Deployment in either wait list: remove it (cancelled before slot came up).
//  3. Deployment in neither: no-op. Idempotent — safe to call from both
//     the happy path and the compensation stack.
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

	// Promote the next live waiter — production first. Dead waiters (status
	// already terminal in DB) are dropped silently and their slot given to
	// the next candidate. Without this check a terminal waiter at the head
	// of the queue would swallow the slot forever: its awakeable resolves
	// into nothing and no one will call Release on it again.
	if held {
		promoted, droppedProd, droppedPreview, prodRem, previewRem, promoteErr :=
			s.promoteNextLive(ctx, prodWait, previewWait)
		if promoteErr != nil {
			return nil, promoteErr
		}
		prodWait = prodRem
		previewWait = previewRem

		if len(droppedProd)+len(droppedPreview) > 0 {
			logger.Warn("dropped terminal waiters during promotion",
				"workspace_id", workspaceID,
				"dropped_prod", droppedProd,
				"dropped_preview", droppedPreview,
			)
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

// promoteNextLive pops waiters off prod_wait_list first, then
// preview_wait_list, skipping any whose DB status is terminal. Returns
// the first live waiter (or nil if none), the dropped-terminal IDs, and
// the remaining wait lists. All wait-list statuses are fetched in a
// single batched DB query to stay O(1) round-trips.
func (s *Service) promoteNextLive(
	ctx restate.ObjectContext,
	prodWait, previewWait []waitEntry,
) (*waitEntry, []string, []string, []waitEntry, []waitEntry, error) {
	if len(prodWait) == 0 && len(previewWait) == 0 {
		return nil, nil, nil, prodWait, previewWait, nil
	}

	ids := make([]string, 0, len(prodWait)+len(previewWait))
	for _, w := range prodWait {
		ids = append(ids, w.DeploymentID)
	}
	for _, w := range previewWait {
		ids = append(ids, w.DeploymentID)
	}

	rows, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.FindDeploymentStatusesByIdsRow, error) {
		return db.Query.FindDeploymentStatusesByIds(runCtx, s.db.RO(), ids)
	}, restate.WithName("promote: fetch waiter statuses"))
	if err != nil {
		return nil, nil, nil, prodWait, previewWait, fmt.Errorf("fetch waiter statuses: %w", err)
	}
	statuses := make(map[string]db.DeploymentsStatus, len(rows))
	for _, r := range rows {
		statuses[r.ID] = r.Status
	}
	// Missing from DB = deleted = treat as terminal.
	terminal := func(id string) bool {
		status, ok := statuses[id]
		if !ok {
			return true
		}
		return isTerminalDeploymentStatus(status)
	}

	droppedProd := []string{}
	droppedPreview := []string{}

	// rejectDead unblocks any Deploy workflow that's still parked on the
	// awakeable despite its DB status being terminal — e.g. a row
	// hard-deleted by env cascade, or a status flipped externally without
	// the invocation being cancelled. If the workflow is already dead,
	// the reject is a no-op.
	rejectDead := func(w waitEntry) {
		restate.RejectAwakeable(ctx, w.AwakeableID, waiterDeadReason)
	}

	for len(prodWait) > 0 {
		head := prodWait[0]
		prodWait = prodWait[1:]
		if terminal(head.DeploymentID) {
			rejectDead(head)
			droppedProd = append(droppedProd, head.DeploymentID)
			continue
		}
		return &head, droppedProd, droppedPreview, prodWait, previewWait, nil
	}
	for len(previewWait) > 0 {
		head := previewWait[0]
		previewWait = previewWait[1:]
		if terminal(head.DeploymentID) {
			rejectDead(head)
			droppedPreview = append(droppedPreview, head.DeploymentID)
			continue
		}
		return &head, droppedProd, droppedPreview, prodWait, previewWait, nil
	}
	return nil, droppedProd, droppedPreview, prodWait, previewWait, nil
}

func removeFromWaitList(list []waitEntry, deploymentID string) []waitEntry {
	for i, w := range list {
		if w.DeploymentID == deploymentID {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}
