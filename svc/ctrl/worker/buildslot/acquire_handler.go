package buildslot

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
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
// Production deployments still respect the workspace's concurrent build limit,
// but they enqueue into a separate
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

	// limitsResult carries the fetch outcome through the Restate journal.
	// The not-found case is folded into the value (instead of returning
	// ErrNoRows) for two reasons: error types don't survive journaling, and
	// more importantly an error here would be retried — and this VO holds
	// the per-workspace key lock while retrying. An unbounded retry on a
	// workspace without a limits row previously wedged the entire workspace
	// queue forever. The retry is bounded AND no-rows is not an error.
	type limitsResult struct {
		Found bool   `json:"found"`
		Max   uint32 `json:"max"`
	}
	limits, err := restate.Run(ctx, func(runCtx restate.RunContext) (limitsResult, error) {
		row, dbErr := s.db.FindLimitsByWorkspaceID(runCtx, workspaceID)
		if db.IsNotFound(dbErr) {
			return limitsResult{Found: false, Max: 0}, nil
		}
		if dbErr != nil {
			return limitsResult{}, dbErr
		}
		return limitsResult{Found: true, Max: uint32(row.BuildsConcurrentMax)}, nil
	}, restate.WithName("fetch limits"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fmt.Errorf("fetch limits: %w", err)
	}

	buildLimit := defaultBuildLimit
	if limits.Found {
		buildLimit = limits.Max
	} else {
		logger.Warn("workspace has no limits row, using default build limit",
			"workspace_id", workspaceID,
			"default_limit", defaultBuildLimit,
		)
	}
	if uint32(len(active)) < buildLimit {
		return s.grantSlot(ctx, active, workspaceID, deploymentID, awakeableID, buildLimit, req.GetIsProduction())
	}

	// At capacity: before parking the caller, verify the current occupants
	// against ground truth (database status + Restate invocation liveness).
	// This is the moment a stale slot actually hurts, and auditing here
	// means the queue self-heals on demand — it never depends on a
	// previously scheduled lease having survived kills or state written
	// before this code was deployed. Best-effort: if the audit itself fails,
	// the caller is parked normally and its wait timeout still bounds the
	// damage.
	staleIDs, auditErr := s.auditActiveSlots(ctx, workspaceID, active)
	if auditErr != nil {
		logger.Warn("build slot audit failed, queueing without reclaim",
			"workspace_id", workspaceID,
			"deployment_id", deploymentID,
			"error", auditErr,
		)
	} else if len(staleIDs) > 0 {
		// Existing waiters are promoted into the freed capacity before the
		// caller so queue order stays fair.
		active, prodWait, previewWait = reclaimStaleSlots(ctx, workspaceID, staleIDs, active, prodWait, previewWait, buildLimit)
		saveActiveSlots(ctx, active)
		saveWaitList(ctx, stateKeyProdWaitList, prodWait)
		saveWaitList(ctx, stateKeyPreviewWaitList, previewWait)

		if uint32(len(active)) < buildLimit {
			return s.grantSlot(ctx, active, workspaceID, deploymentID, awakeableID, buildLimit, req.GetIsProduction())
		}
	}

	// Still at capacity: park the caller. Production goes to its own list
	// so Release can drain it ahead of preview waiters.
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

	// Schedule the wait-entry audit. A live waiter times itself out and
	// releases its entry before this fires; a dead one (killed invocation
	// whose compensation never ran) is swept here.
	scheduleExpiry(ctx, workspaceID, deploymentID, waiterExpiryDelay)

	logger.Info("build slot full, deployment queued",
		"workspace_id", workspaceID,
		"deployment_id", deploymentID,
		"is_production", req.GetIsProduction(),
		"active", len(active),
		"prod_wait", len(prodWait),
		"preview_wait", len(previewWait),
		"limit", buildLimit,
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

	// Start the slot lease: if this deployment never releases (killed
	// invocation, lost compensation), ExpireSlot reclaims the slot instead
	// of leaking workspace capacity forever.
	scheduleExpiry(ctx, workspaceID, deploymentID, slotLeaseDuration)

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

// scheduleExpiry arms a delayed self-call to ExpireSlot for the deployment.
// The send is journaled by Restate, so once this handler commits, the lease
// check is guaranteed to fire even across worker restarts.
func scheduleExpiry(ctx restate.ObjectContext, workspaceID, deploymentID string, delay time.Duration) {
	hydrav1.NewBuildSlotServiceClient(ctx, workspaceID).ExpireSlot().Send(
		&hydrav1.ExpireSlotRequest{DeploymentId: deploymentID},
		restate.WithDelay(delay),
	)
}

// findAwakeableID returns the awakeable id of the wait entry for the given
// deployment, searching both wait lists. Empty string when not waiting.
func findAwakeableID(prodWait, previewWait []waitEntry, deploymentID string) string {
	for _, list := range [][]waitEntry{prodWait, previewWait} {
		for _, w := range list {
			if w.DeploymentID == deploymentID {
				return w.AwakeableID
			}
		}
	}
	return ""
}

// pickNextWaiter pops the head of the production wait list, falling back to
// the preview wait list. Returns nil when both lists are empty. Pure so the
// prod-before-preview ordering is unit-testable without a Restate context.
func pickNextWaiter(prodWait, previewWait []waitEntry) (promoted *waitEntry, newProd, newPreview []waitEntry) {
	switch {
	case len(prodWait) > 0:
		return &prodWait[0], prodWait[1:], previewWait
	case len(previewWait) > 0:
		return &previewWait[0], prodWait, previewWait[1:]
	default:
		return nil, prodWait, previewWait
	}
}
