package buildslot

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/db"
)

// isTerminalDeploymentStatus reports whether a deployment is in a state from
// which no forward progress is possible. A deployment in a terminal state
// that still holds an active slot or a wait-list entry is a leak — the
// handler that was supposed to release it either already died or never got
// the chance to run its compensation.
//
// Must stay aligned with services/deployment.isTerminalDeploymentStatus —
// they answer the same question from opposite ends of the workflow.
func isTerminalDeploymentStatus(status db.DeploymentsStatus) bool {
	switch status {
	case db.DeploymentsStatusReady,
		db.DeploymentsStatusFailed,
		db.DeploymentsStatusSkipped,
		db.DeploymentsStatusStopped,
		db.DeploymentsStatusSuperseded,
		db.DeploymentsStatusCancelled:
		return true
	case db.DeploymentsStatusPending,
		db.DeploymentsStatusStarting,
		db.DeploymentsStatusBuilding,
		db.DeploymentsStatusDeploying,
		db.DeploymentsStatusNetwork,
		db.DeploymentsStatusFinalizing,
		db.DeploymentsStatusAwaitingApproval:
		return false
	default:
		return false
	}
}

// reconcileState is the in-memory snapshot of a workspace's BuildSlotService
// state. Used both as input and output of [Service.sweepAndPromote] so the
// caller can continue operating on the post-reconcile state without
// reloading from Restate.
type reconcileState struct {
	active      map[string]bool
	prodWait    []waitEntry
	previewWait []waitEntry
}

// sweepAndPromote removes terminal entries from state and promotes waiters
// up to the concurrency cap. Pure transformation on the in-memory state:
// callers own loading and saving. Returns the updated state, the list of
// deployment IDs released, and whether promotion actually happened (so
// callers can decide whether to re-check capacity without a reload).
//
// Shared primitive used by the admin [Service.Reconcile] RPC and the lazy
// self-heal in [Service.AcquireOrWait]. Batched DB read keeps this O(1)
// round-trips regardless of how much state exists.
func (s *Service) sweepAndPromote(
	ctx restate.ObjectContext,
	state reconcileState,
	quotaLimit uint32,
) (reconcileState, []string, error) {
	ids := make([]string, 0, len(state.active)+len(state.prodWait)+len(state.previewWait))
	for id := range state.active {
		ids = append(ids, id)
	}
	for _, w := range state.prodWait {
		ids = append(ids, w.DeploymentID)
	}
	for _, w := range state.previewWait {
		ids = append(ids, w.DeploymentID)
	}
	if len(ids) == 0 {
		return state, nil, nil
	}

	rows, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.FindDeploymentStatusesByIdsRow, error) {
		return db.Query.FindDeploymentStatusesByIds(runCtx, s.db.RO(), ids)
	}, restate.WithName("reconcile: fetch deployment statuses"))
	if err != nil {
		return state, nil, fmt.Errorf("fetch deployment statuses: %w", err)
	}

	// A deployment in `ids` that doesn't come back has been deleted from
	// the DB — treat as terminal and release.
	statuses := make(map[string]db.DeploymentsStatus, len(rows))
	for _, r := range rows {
		statuses[r.ID] = r.Status
	}
	terminal := func(id string) bool {
		status, ok := statuses[id]
		if !ok {
			return true
		}
		return isTerminalDeploymentStatus(status)
	}

	released := make([]string, 0)
	for id := range state.active {
		if terminal(id) {
			delete(state.active, id)
			released = append(released, id)
		}
	}
	var prodDropped, previewDropped []waitEntry
	state.prodWait, prodDropped = filterWaitList(state.prodWait, terminal)
	state.previewWait, previewDropped = filterWaitList(state.previewWait, terminal)
	// Reject dropped waiters' awakeables: if their workflow is still alive
	// and parked (e.g. status was flipped to terminal externally, or the
	// row was hard-deleted without cancelling the invocation), the reject
	// unblocks them so their defer fires instead of leaving a zombie
	// suspended in Restate. If the workflow is already dead, the reject is
	// a no-op. See also [Service.promoteNextLive].
	for _, w := range prodDropped {
		restate.RejectAwakeable(ctx, w.AwakeableID, waiterDeadReason)
		released = append(released, w.DeploymentID)
	}
	for _, w := range previewDropped {
		restate.RejectAwakeable(ctx, w.AwakeableID, waiterDeadReason)
		released = append(released, w.DeploymentID)
	}

	// Promote live waiters in priority order (prod first) up to the cap.
	for uint32(len(state.active)) < quotaLimit {
		var promoted *waitEntry
		switch {
		case len(state.prodWait) > 0:
			promoted = &state.prodWait[0]
			state.prodWait = state.prodWait[1:]
		case len(state.previewWait) > 0:
			promoted = &state.previewWait[0]
			state.previewWait = state.previewWait[1:]
		}
		if promoted == nil {
			break
		}
		state.active[promoted.DeploymentID] = true
		restate.ResolveAwakeable(ctx, promoted.AwakeableID, true)
	}

	return state, released, nil
}

// loadReconcileState reads the whole VO state snapshot in one place.
func loadReconcileState(ctx restate.ObjectContext) (reconcileState, error) {
	active, err := loadActiveSlots(ctx)
	if err != nil {
		return reconcileState{}, fmt.Errorf("load active slots: %w", err)
	}
	prodWait, err := loadWaitList(ctx, stateKeyProdWaitList)
	if err != nil {
		return reconcileState{}, fmt.Errorf("load prod wait list: %w", err)
	}
	previewWait, err := loadWaitList(ctx, stateKeyPreviewWaitList)
	if err != nil {
		return reconcileState{}, fmt.Errorf("load preview wait list: %w", err)
	}
	return reconcileState{active: active, prodWait: prodWait, previewWait: previewWait}, nil
}

// saveReconcileState writes the whole snapshot back.
func saveReconcileState(ctx restate.ObjectContext, state reconcileState) {
	saveActiveSlots(ctx, state.active)
	saveWaitList(ctx, stateKeyProdWaitList, state.prodWait)
	saveWaitList(ctx, stateKeyPreviewWaitList, state.previewWait)
}

// filterWaitList partitions a wait list into entries the caller wants to
// keep (terminal returns false) and entries to drop. Returns both halves
// so the caller can reject dropped awakeables without losing the
// awakeable_id.
func filterWaitList(list []waitEntry, terminal func(string) bool) ([]waitEntry, []waitEntry) {
	kept := list[:0]
	dropped := []waitEntry{}
	for _, w := range list {
		if terminal(w.DeploymentID) {
			dropped = append(dropped, w)
			continue
		}
		kept = append(kept, w)
	}
	return kept, dropped
}

// waiterDeadReason is the error used to reject a waiter's awakeable when
// its deployment's DB status is terminal (or the row is gone). Surfaces
// as the error returned from awakeable.Result() in the Deploy workflow's
// waitForBuildSlot, which then flows into the compensation stack.
var waiterDeadReason = restate.TerminalError(fmt.Errorf("deployment is in terminal status; wait-list entry swept by BuildSlotService"))
