package buildslot

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// deploymentCheck is the journal-safe result of the lease audit's ground
// truth reads. The not-found case is folded into the value because error
// types do not survive Restate journaling.
type deploymentCheck struct {
	Exists   bool `json:"exists"`
	Terminal bool `json:"terminal"`
	// InvocationGone is true when the deployment row records a Restate
	// invocation ID that no longer exists in sys_invocation: the Deploy
	// invocation was killed or purged without updating the deployment row.
	InvocationGone bool `json:"invocation_gone"`
}

// ExpireSlot audits one deployment's slot lease or wait-list entry. It is
// scheduled as a delayed self-call on every grant (slotLeaseDuration) and
// every enqueue (waiterExpiryDelay), and is the mechanism that makes slot
// accounting self-healing: no matter how a Deploy invocation dies — killed,
// purged, crashed before compensation — its slot is eventually reclaimed.
//
// Cases:
//
//  1. Deployment no longer tracked: it released normally. No-op; this is
//     the overwhelmingly common path.
//  2. Tracked, but the deployment row is terminal or missing: the owning
//     invocation died without releasing. Reclaim and promote the next
//     waiter.
//  3. Holds a slot and is still non-terminal after the full lease: the
//     invocation overran any plausible build duration. Force-fail the
//     deployment in the database, then reclaim, so one stuck build cannot
//     wedge the workspace queue.
//  4. Still waiting and non-terminal past waiterExpiryDelay: its own wait
//     timeout (MaxWaitDuration, strictly shorter) should have removed it,
//     so the invocation is dead. Reject its awakeable defensively and drop
//     the entry. No promotion needed — a waiter holds no capacity.
//
// If the database check itself keeps failing, the audit re-arms itself with
// expireRetryDelay rather than giving up: losing a lease check would
// reintroduce the permanent-leak bug this handler exists to fix.
func (s *Service) ExpireSlot(
	ctx restate.ObjectContext,
	req *hydrav1.ExpireSlotRequest,
) (*hydrav1.ExpireSlotResponse, error) {
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

	holdsSlot := active[deploymentID]
	isWaiting := waitListContains(prodWait, deploymentID) || waitListContains(previewWait, deploymentID)
	if !holdsSlot && !isWaiting {
		// Released normally before the lease fired. The common case.
		return &hydrav1.ExpireSlotResponse{}, nil
	}

	check, err := restate.Run(ctx, func(runCtx restate.RunContext) (deploymentCheck, error) {
		deployment, dbErr := s.db.FindDeploymentById(runCtx, deploymentID)
		if db.IsNotFound(dbErr) {
			return deploymentCheck{Exists: false, Terminal: false, InvocationGone: false}, nil
		}
		if dbErr != nil {
			return deploymentCheck{}, dbErr
		}
		result := deploymentCheck{
			Exists:         true,
			Terminal:       deployment.Status.IsTerminal(),
			InvocationGone: false,
		}
		// The database can lie: a killed Deploy invocation leaves the row
		// stuck in a non-terminal status forever. Restate itself is the
		// ground truth for whether the invocation still exists.
		if !result.Terminal && deployment.InvocationID.Valid && deployment.InvocationID.String != "" {
			live, admErr := s.restateAdmin.FindLiveInvocations(runCtx, []string{deployment.InvocationID.String})
			if admErr != nil {
				return deploymentCheck{}, admErr
			}
			result.InvocationGone = !live[deployment.InvocationID.String]
		}
		return result, nil
	}, restate.WithName("check deployment status for lease audit"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		// Database persistently unavailable. Re-arm instead of dropping the
		// audit — the deployment stays tracked and will be checked again.
		logger.Warn("build slot lease audit could not read deployment, re-arming",
			"workspace_id", workspaceID,
			"deployment_id", deploymentID,
			"error", err,
		)
		scheduleExpiry(ctx, workspaceID, deploymentID, expireRetryDelay)
		return &hydrav1.ExpireSlotResponse{}, nil
	}

	if check.Exists && !check.Terminal {
		if holdsSlot || check.InvocationGone {
			// Either the deployment overran its lease while supposedly still
			// building, or its Deploy invocation is gone from Restate while
			// the row still claims an active status. Force-fail it so the
			// database agrees with the reclaim below;
			// UpdateDeploymentStatusIfActive won't clobber a concurrent
			// legitimate terminal transition.
			failReason := "build slot lease expired: deployment exceeded maximum build duration"
			if check.InvocationGone {
				failReason = "build slot lease expired: deploy invocation no longer exists in Restate"
			}
			if runErr := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
				now := sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()}
				if updErr := s.db.UpdateDeploymentStatusIfActive(runCtx, db.UpdateDeploymentStatusIfActiveParams{
					ID:               deploymentID,
					Status:           mysqltype.DeploymentsStatusFailed,
					UpdatedAt:        now,
					TerminalStatuses: mysqltype.TerminalDeploymentStatuses,
				}); updErr != nil {
					return updErr
				}
				return s.db.EndActiveDeploymentStepsWithError(runCtx, db.EndActiveDeploymentStepsWithErrorParams{
					DeploymentID: deploymentID,
					EndedAt:      now,
					Error:        sql.NullString{Valid: true, String: failReason},
				})
			}, restate.WithName("force-fail deployment on lease expiry"), restate.WithMaxRetryAttempts(runMaxAttempts)); runErr != nil {
				logger.Warn("build slot lease audit could not force-fail deployment, re-arming",
					"workspace_id", workspaceID,
					"deployment_id", deploymentID,
					"error", runErr,
				)
				scheduleExpiry(ctx, workspaceID, deploymentID, expireRetryDelay)
				return &hydrav1.ExpireSlotResponse{}, nil
			}

			logger.Error("build slot lease expired, force-failed and reclaiming",
				"workspace_id", workspaceID,
				"deployment_id", deploymentID,
				"invocation_gone", check.InvocationGone,
				"lease", slotLeaseDuration.String(),
			)
		}
		// A live waiter past waiterExpiryDelay falls through to the sweep
		// below without touching the database: its wait timeout
		// (MaxWaitDuration < waiterExpiryDelay) already failed the
		// deployment, so only the stale entry needs removing.
	} else {
		logger.Warn("build slot lease audit reclaiming slot from dead deployment",
			"workspace_id", workspaceID,
			"deployment_id", deploymentID,
			"deployment_exists", check.Exists,
			"held_slot", holdsSlot,
			"was_waiting", isWaiting,
		)
	}

	// Defensive: if the expired waiter's invocation is somehow still parked
	// on its awakeable, reject it so it fails fast instead of hanging.
	// Rejecting an already-completed awakeable is harmless.
	if staleAwakeableID := findAwakeableID(prodWait, previewWait, deploymentID); staleAwakeableID != "" {
		restate.RejectAwakeable(ctx, staleAwakeableID, restate.TerminalErrorf("build slot wait entry expired"))
	}

	// Reclaim: drop the deployment from all tracking.
	delete(active, deploymentID)
	prodWait = removeFromWaitList(prodWait, deploymentID)
	previewWait = removeFromWaitList(previewWait, deploymentID)

	// Only a freed slot creates capacity for a promotion.
	if holdsSlot {
		var promoted *waitEntry
		promoted, prodWait, previewWait = pickNextWaiter(prodWait, previewWait)
		if promoted != nil {
			active[promoted.DeploymentID] = true
			restate.ResolveAwakeable(ctx, promoted.AwakeableID, true)
			scheduleExpiry(ctx, workspaceID, promoted.DeploymentID, slotLeaseDuration)

			logger.Info("build slot handed off after lease reclaim",
				"workspace_id", workspaceID,
				"reclaimed", deploymentID,
				"promoted", promoted.DeploymentID,
				"active", len(active),
			)
		}
	}

	saveActiveSlots(ctx, active)
	saveWaitList(ctx, stateKeyProdWaitList, prodWait)
	saveWaitList(ctx, stateKeyPreviewWaitList, previewWait)

	return &hydrav1.ExpireSlotResponse{}, nil
}
