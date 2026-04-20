package buildslot

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/observability"
)

const workflowBuildSlotAcquire = "buildslot_acquire"

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
) (resp *hydrav1.AcquireOrWaitResponse, retErr error) {
	defer observability.RunTimer(workflowBuildSlotAcquire, &retErr)()

	workspaceID := restate.Key(ctx)
	deploymentID := req.GetDeploymentId()
	awakeableID := req.GetAwakeableId()

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

	if active[deploymentID] {
		restate.ResolveAwakeable(ctx, awakeableID, true)
		return &hydrav1.AcquireOrWaitResponse{}, nil
	}

	if waitListContains(prodWait, deploymentID) || waitListContains(previewWait, deploymentID) {
		return &hydrav1.AcquireOrWaitResponse{}, nil
	}

	quota, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Quotas, error) {
		return db.Query.FindQuotaByWorkspaceID(runCtx, s.db.RO(), workspaceID)
	}, restate.WithName("fetch quota"))
	if err != nil {
		return nil, fmt.Errorf("fetch quota: %w", err)
	}

	if uint32(len(active)) < quota.MaxConcurrentBuilds {
		return s.grantSlot(ctx, active, workspaceID, deploymentID, awakeableID, quota.MaxConcurrentBuilds, req.GetIsProduction())
	}

	// At capacity: park the caller. Production goes to its own list so
	// Release can drain it ahead of preview waiters.
	entry := waitEntry{
		DeploymentID: deploymentID,
		AwakeableID:  awakeableID,
	}
	if req.GetIsProduction() {
		prodWait = append(prodWait, entry)
		saveWaitList(ctx, stateKeyProdWaitList, prodWait)
	} else {
		previewWait = append(previewWait, entry)
		saveWaitList(ctx, stateKeyPreviewWaitList, previewWait)
	}

	logger.Info("build slot full, deployment queued",
		"workspace_id", workspaceID,
		"deployment_id", deploymentID,
		"is_production", req.GetIsProduction(),
		"active", len(active),
		"prod_wait", len(prodWait),
		"preview_wait", len(previewWait),
		"limit", quota.MaxConcurrentBuilds,
	)

	return &hydrav1.AcquireOrWaitResponse{}, nil
}

func (s *Service) grantSlot(
	ctx restate.ObjectContext,
	active map[string]bool,
	workspaceID, deploymentID, awakeableID string,
	limit uint32,
	isProduction bool,
) (*hydrav1.AcquireOrWaitResponse, error) {
	active[deploymentID] = true
	saveActiveSlots(ctx, active)

	restate.ResolveAwakeable(ctx, awakeableID, true)

	logger.Info("build slot granted",
		"workspace_id", workspaceID,
		"deployment_id", deploymentID,
		"is_production", isProduction,
		"active", len(active),
		"limit", limit,
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
