package buildslot

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

const (
	stateKeyActiveSlots     = "active_slots"
	stateKeyProdWaitList    = "prod_wait_list"
	stateKeyPreviewWaitList = "preview_wait_list"
)

type waitEntry struct {
	DeploymentID string `json:"deployment_id"`
	AwakeableID  string `json:"awakeable_id"`
}

// AcquireOrWait either grants a build slot immediately or parks the caller
// on a FIFO wait list. The caller's awakeable is resolved with true when the
// slot is granted (now or later).
//
// Production deployments still respect the workspace's max_concurrent_builds
// quota — they don't get a free pass — but they enqueue into a separate
// prod_wait_list that Release drains before the preview wait list. So
// production hot-fixes priority-queue ahead of preview builds without
// blowing past the workspace cap.
//
// Idempotent: if the deployment already holds a slot, the awakeable is
// re-resolved; if it is already waiting, we return immediately (the
// existing entry will be resolved when a slot frees up).
func (s *Service) AcquireOrWait(
	ctx restate.ObjectContext,
	req *hydrav1.AcquireOrWaitRequest,
) (*hydrav1.AcquireOrWaitResponse, error) {
	workspaceID := restate.Key(ctx)
	deploymentID := req.GetDeploymentId()
	awakeableID := req.GetAwakeableId()

	state, err := loadReconcileState(ctx)
	if err != nil {
		return nil, err
	}

	if state.active[deploymentID] {
		restate.ResolveAwakeable(ctx, awakeableID, true)
		return &hydrav1.AcquireOrWaitResponse{}, nil
	}
	if waitListContains(state.prodWait, deploymentID) || waitListContains(state.previewWait, deploymentID) {
		return &hydrav1.AcquireOrWaitResponse{}, nil
	}

	quota, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Quotas, error) {
		return db.Query.FindQuotaByWorkspaceID(runCtx, s.db.RO(), workspaceID)
	}, restate.WithName("fetch quota"))
	if err != nil {
		return nil, fmt.Errorf("fetch quota: %w", err)
	}

	// At capacity — self-heal before parking. A previous deployment may have
	// leaked a slot (Restate internal termination, dropped Send, etc.)
	// leaving active_slots full of dead deployments with nothing to sweep
	// them. One batched DB read finds and releases them.
	if uint32(len(state.active)) >= quota.MaxConcurrentBuilds {
		var released []string
		state, released, err = s.sweepAndPromote(ctx, state, quota.MaxConcurrentBuilds)
		if err != nil {
			return nil, fmt.Errorf("reconcile before park: %w", err)
		}
		if len(released) > 0 {
			logger.Info("reconcile freed leaked slots at capacity",
				"workspace_id", workspaceID,
				"released", released,
			)
		}
	}

	if uint32(len(state.active)) < quota.MaxConcurrentBuilds {
		state.active[deploymentID] = true
		restate.ResolveAwakeable(ctx, awakeableID, true)
		saveReconcileState(ctx, state)
		logger.Info("build slot granted",
			"workspace_id", workspaceID,
			"deployment_id", deploymentID,
			"is_production", req.GetIsProduction(),
			"active", len(state.active),
			"limit", quota.MaxConcurrentBuilds,
		)
		return &hydrav1.AcquireOrWaitResponse{}, nil
	}

	// Still at capacity: park the caller. Production goes to its own list
	// so Release can drain it ahead of preview waiters.
	entry := waitEntry{DeploymentID: deploymentID, AwakeableID: awakeableID}
	if req.GetIsProduction() {
		state.prodWait = append(state.prodWait, entry)
	} else {
		state.previewWait = append(state.previewWait, entry)
	}
	saveReconcileState(ctx, state)

	logger.Info("build slot full, deployment queued",
		"workspace_id", workspaceID,
		"deployment_id", deploymentID,
		"is_production", req.GetIsProduction(),
		"active", len(state.active),
		"prod_wait", len(state.prodWait),
		"preview_wait", len(state.previewWait),
		"limit", quota.MaxConcurrentBuilds,
	)

	return &hydrav1.AcquireOrWaitResponse{}, nil
}

func loadActiveSlots(ctx restate.ObjectContext) (map[string]bool, error) {
	slots, err := restate.Get[map[string]bool](ctx, stateKeyActiveSlots)
	if err != nil {
		return nil, err
	}
	if slots == nil {
		slots = make(map[string]bool)
	}
	return slots, nil
}

func saveActiveSlots(ctx restate.ObjectContext, slots map[string]bool) {
	restate.Set(ctx, stateKeyActiveSlots, slots)
}

func loadWaitList(ctx restate.ObjectContext, key string) ([]waitEntry, error) {
	list, err := restate.Get[[]waitEntry](ctx, key)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []waitEntry{}
	}
	return list, nil
}

func saveWaitList(ctx restate.ObjectContext, key string, list []waitEntry) {
	restate.Set(ctx, key, list)
}

func waitListContains(list []waitEntry, deploymentID string) bool {
	for _, w := range list {
		if w.DeploymentID == deploymentID {
			return true
		}
	}
	return false
}
