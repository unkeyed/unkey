package buildslot

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// deploymentCheck is the journal-safe result of the lease audit's database
// and Restate reads. The not-found case is part of the value because error
// types do not survive Restate journaling.
type deploymentCheck struct {
	Exists   bool `json:"exists"`
	Terminal bool `json:"terminal"`
	// InvocationGone is true when the deployment row records a Restate
	// invocation that no longer executes: the Deploy invocation was
	// killed, cancelled, or purged without an update to the deployment
	// row.
	InvocationGone bool `json:"invocation_gone"`
}

// ExpireSlot audits one deployment's slot lease or wait-list entry. Every
// slot grant schedules it after slotLeaseDuration. Every enqueue schedules
// it after waiterExpiryDelay. This makes sure a dead Deploy invocation
// (killed, purged, or crashed before compensation) cannot hold a slot
// forever.
//
// Cases:
//
//  1. Deployment not tracked: it released normally. No-op. This is the
//     common path.
//  2. Tracked, but the deployment row is terminal or missing: the owning
//     invocation died without a release. Reclaim and promote the next
//     waiter.
//  3. Holds a slot, non-terminal, and the Deploy invocation is live in
//     Restate: the build still runs. Renew the lease and check again
//     after slotLeaseDuration, up to maxSlotLeaseRenewals. Past the cap,
//     force-fail the deployment in the database, then reclaim, so one
//     hung build cannot block the workspace queue.
//  4. Still waiting and non-terminal after waiterExpiryDelay: its own wait
//     timeout (MaxWaitDuration, strictly shorter) should have removed it,
//     so the invocation is dead. Reject its awakeable and drop the entry.
//     No promotion is needed because a waiter holds no capacity.
//
// If the database check keeps failing, the audit schedules itself again
// after expireRetryDelay. A lost lease check would bring back the permanent
// leak this handler exists to fix.
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
		// The database row can be stale: a killed Deploy invocation leaves
		// the row in a non-terminal status forever. Ask Restate whether the
		// invocation still exists.
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
		// Database not available. Schedule the audit again instead of
		// dropping it. The deployment stays tracked until the next check.
		logger.Warn("build slot lease audit could not read deployment, re-arming",
			"workspace_id", workspaceID,
			"deployment_id", deploymentID,
			"error", err,
		)
		scheduleExpiry(ctx, workspaceID, deploymentID, expireRetryDelay, req.GetRenewals())
		return &hydrav1.ExpireSlotResponse{}, nil
	}

	if check.Exists && !check.Terminal {
		// A live invocation that holds a slot still builds. Renew the
		// lease instead of failing it, up to the renewal cap.
		if holdsSlot && !check.InvocationGone && req.GetRenewals() < maxSlotLeaseRenewals {
			logger.Info("build slot lease renewed for live deployment",
				"workspace_id", workspaceID,
				"deployment_id", deploymentID,
				"renewals", req.GetRenewals()+1,
				"max_renewals", maxSlotLeaseRenewals,
			)
			scheduleExpiry(ctx, workspaceID, deploymentID, slotLeaseDuration, req.GetRenewals()+1)
			return &hydrav1.ExpireSlotResponse{}, nil
		}

		if holdsSlot || check.InvocationGone {
			// Either the deployment used all lease renewals, or its Deploy
			// invocation is gone from Restate while the row still shows an
			// active status. Force-fail it so the database agrees with the
			// reclaim below. UpdateDeploymentStatusIfActive does not
			// overwrite a concurrent legitimate terminal transition.
			failReason := "build slot lease expired: deployment exceeded maximum build duration"
			if check.InvocationGone {
				failReason = "build slot lease expired: deploy invocation no longer exists in Restate"
			}
			if runErr := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
				return forceFailDeployment(runCtx, s.db, deploymentID, failReason)
			}, restate.WithName("force-fail deployment on lease expiry"), restate.WithMaxRetryAttempts(runMaxAttempts)); runErr != nil {
				logger.Warn("build slot lease audit could not force-fail deployment, re-arming",
					"workspace_id", workspaceID,
					"deployment_id", deploymentID,
					"error", runErr,
				)
				scheduleExpiry(ctx, workspaceID, deploymentID, expireRetryDelay, req.GetRenewals())
				return &hydrav1.ExpireSlotResponse{}, nil
			}

			logger.Error("build slot lease expired, force-failed and reclaiming",
				"workspace_id", workspaceID,
				"deployment_id", deploymentID,
				"invocation_gone", check.InvocationGone,
				"renewals", req.GetRenewals(),
				"lease_interval", slotLeaseDuration.String(),
			)
		}
		// A live waiter past waiterExpiryDelay falls through to the sweep
		// below without a database write. Its wait timeout
		// (MaxWaitDuration < waiterExpiryDelay) already failed the
		// deployment, so only the stale entry needs removal.
	} else {
		logger.Warn("build slot lease audit reclaiming slot from dead deployment",
			"workspace_id", workspaceID,
			"deployment_id", deploymentID,
			"deployment_exists", check.Exists,
			"held_slot", holdsSlot,
			"was_waiting", isWaiting,
		)
	}

	// Defensive: if the expired waiter's invocation still waits on its
	// awakeable, reject it so it fails fast instead of hanging. A reject on
	// a completed awakeable is harmless.
	if staleAwakeableID := findAwakeableID(prodWait, previewWait, deploymentID); staleAwakeableID != "" {
		restate.RejectAwakeable(ctx, staleAwakeableID, restate.TerminalErrorf("build slot wait entry expired"))
	}

	// Reclaim: drop the deployment from all tracking.
	delete(active, deploymentID)
	prodWait = removeFromWaitList(prodWait, deploymentID)
	previewWait = removeFromWaitList(previewWait, deploymentID)

	// Only a freed slot creates capacity for a promotion. Prune first so
	// the slot does not go to another dead waiter.
	if holdsSlot {
		prodWait, previewWait = s.pruneDeadWaiters(ctx, workspaceID, prodWait, previewWait)

		var promoted *waitEntry
		promoted, prodWait, previewWait = pickNextWaiter(prodWait, previewWait)
		if promoted != nil {
			active[promoted.DeploymentID] = true
			restate.ResolveAwakeable(ctx, promoted.AwakeableID, true)
			scheduleExpiry(ctx, workspaceID, promoted.DeploymentID, slotLeaseDuration, 0)

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
