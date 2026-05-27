package deploy

import (
	"context"
	"database/sql"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
)

// buildSlotWakeupFallback bounds how long the Deploy handler parks on its
// awakeable before re-attempting the acquire query. The DB is the source of
// truth, so a dropped wake-up (handler dead, race with release tx) is
// self-healing: on timeout we just loop and re-check capacity. Acquire is
// instant on the happy path because Release resolves the awakeable in ms.
const buildSlotWakeupFallback = 60 * time.Second

// skipIfSuperseded marks the current deployment as superseded and returns
// (true, nil) when a newer deployment for the same (app, environment, branch)
// has already been queued. Rapid pushes to the same branch only build the
// latest commit. `skipped` is reserved for "watch paths didn't match", so
// supersession uses its own status here.
//
// Returns (false, nil) when the deployment should proceed normally, or
// (false, err) if the dedup query or status update fails.
//
// This catches the race where the proactive dedup in
// services/deployment.create_deployment didn't manage to cancel the older
// sibling before it started running (e.g. invocation_id hadn't been
// persisted yet). The workflow self-checks at the top so it can bow out
// before acquiring a build slot.
func (w *Workflow) skipIfSuperseded(
	ctx restate.ObjectContext,
	deployment db.Deployment,
) (bool, error) {
	hasNewer, err := restate.Run(ctx, func(runCtx restate.RunContext) (bool, error) {
		return db.Query.HasNewerActiveDeployment(runCtx, w.db.RO(), db.HasNewerActiveDeploymentParams{
			AppID:         deployment.AppID,
			EnvironmentID: deployment.EnvironmentID,
			GitBranch:     deployment.GitBranch,
			CreatedAt:     deployment.CreatedAt,
			DeploymentID:  deployment.ID,
		})
	}, restate.WithName("check for newer active deployment"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return false, fault.Wrap(err, fault.Public("Failed to check for newer deployments."))
	}
	if !hasNewer {
		return false, nil
	}

	logger.Info("self-superseding deployment",
		"deployment_id", deployment.ID,
		"app_id", deployment.AppID,
		"branch", deployment.GitBranch.String,
	)

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		now := sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()}
		if updErr := db.Query.UpdateDeploymentStatus(runCtx, w.db.RW(), db.UpdateDeploymentStatusParams{
			ID:        deployment.ID,
			Status:    db.DeploymentsStatusSuperseded,
			UpdatedAt: now,
		}); updErr != nil {
			return updErr
		}
		return db.Query.EndDeploymentStep(runCtx, w.db.RW(), db.EndDeploymentStepParams{
			DeploymentID: deployment.ID,
			Step:         db.DeploymentStepsStepQueued,
			EndedAt:      now,
			Error:        sql.NullString{Valid: true, String: "superseded by newer commit"},
		})
	}, restate.WithName("mark deployment superseded"), restate.WithMaxRetryAttempts(runMaxAttempts)); err != nil {
		return false, fault.Wrap(err, fault.Public("Failed to mark deployment as superseded."))
	}

	return true, nil
}

// waitForBuildSlot blocks until the workspace has capacity for this
// deployment. State lives in the build_slots / build_slot_waiters tables:
//
//   - acquire is an atomic INSERT...WHERE that joins build_slots against
//     deployments.status, so leaked rows from purged invocations don't
//     count against capacity (no deadlock class).
//   - when at capacity, the handler parks on a Restate awakeable that the
//     next [releaseBuildSlot] resolves. A 60s timeout drops back into the
//     loop so a missed wake-up (race, dead handler) self-heals.
//
// Production waiters jump ahead of preview waiters via the ORDER BY in
// [PickNextBuildSlotWaiter]. Strict FIFO is not guaranteed across
// concurrent acquires (Restate doesn't give us that today), but the
// production-first invariant is.
//
// The caller is responsible for releasing the slot on both the success and
// failure paths via [releaseBuildSlot]. On cancellation mid-wait, the
// Deploy handler's compensation calls releaseBuildSlot which drops the
// waiter row (or active row) and promotes the next waiter.
func (w *Workflow) waitForBuildSlot(
	ctx restate.ObjectContext,
	deployment db.Deployment,
	isProduction bool,
) error {
	for {
		// Two questions in one Run: "do we already hold a row?" then,
		// only if not, "can we insert one?". `build_slots` is the source
		// of truth, so any path that leaves our row in the table means
		// we own the slot — including:
		//
		//   - Restate replay of an earlier successful TryAcquire.
		//   - Awakeable fired after ReleaseAndPromoteBuildSlot pre-granted us.
		//   - Wake-up timeout after the holder crashed between its
		//     release tx commit and ResolveAwakeable. The pre-grant row
		//     is still there; the awakeable resolution was lost.
		//
		// Without the EXISTS check, the second and third cases re-enter
		// TryAcquireBuildSlot which either deadlocks
		// (max_concurrent_builds=1: our own pre-granted row makes
		// count == limit) or PK-violates (max>1: INSERT proceeds, hits
		// duplicate deployment_id).
		granted, err := restate.Run(ctx, func(runCtx restate.RunContext) (bool, error) {
			holds, runErr := db.Query.HoldsBuildSlot(runCtx, w.db.RO(), deployment.ID)
			if runErr != nil {
				return false, runErr
			}
			if holds {
				return true, nil
			}
			rows, runErr := db.Query.TryAcquireBuildSlot(runCtx, w.db.RW(), db.TryAcquireBuildSlotParams{
				DeploymentID:     deployment.ID,
				WorkspaceID:      deployment.WorkspaceID,
				AcquiredAt:       time.Now().UnixMilli(),
				TerminalStatuses: db.TerminalDeploymentStatuses,
			})
			return rows == 1, runErr
		}, restate.WithName("try acquire build slot"), restate.WithMaxRetryAttempts(runMaxAttempts))
		if err != nil {
			return fault.Wrap(err, fault.Public("Failed to request build slot."))
		}
		if granted {
			logger.Info("build slot acquired",
				"workspace_id", deployment.WorkspaceID,
				"deployment_id", deployment.ID,
			)
			return nil
		}

		awk := restate.Awakeable[bool](ctx)
		if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return db.Query.RegisterBuildSlotWaiter(runCtx, w.db.RW(), db.RegisterBuildSlotWaiterParams{
				DeploymentID: deployment.ID,
				WorkspaceID:  deployment.WorkspaceID,
				AwakeableID:  awk.Id(),
				IsProduction: isProduction,
				EnqueuedAt:   time.Now().UnixMilli(),
			})
		}, restate.WithName("register build slot waiter"), restate.WithMaxRetryAttempts(runMaxAttempts)); err != nil {
			return fault.Wrap(err, fault.Public("Failed to enqueue for build slot."))
		}

		logger.Info("waiting for build slot",
			"workspace_id", deployment.WorkspaceID,
			"deployment_id", deployment.ID,
			"is_production", isProduction,
		)

		// Race the awakeable against a fallback timeout. We discard the
		// winner: the next iteration's HoldsBuildSlot check is the
		// source of truth for whether the slot is ours, whether the
		// awakeable fired, the timeout fired with a successful
		// pre-grant (holder crashed between tx commit and
		// ResolveAwakeable), or neither happened (still at capacity,
		// just re-park).
		timeout := restate.After(ctx, buildSlotWakeupFallback)
		if _, err := restate.WaitFirst(ctx, awk, timeout); err != nil {
			return fault.Wrap(err, fault.Public("Failed while waiting for build slot."))
		}

		// Drop the waiter row so a follow-up Release doesn't pick a
		// stale awakeable that this handler will never read from again.
		// On the awakeable-fired path Release already unregistered us
		// in its tx; this DELETE is a no-op there. On the timeout path
		// it's required so the next loop iteration's re-register isn't
		// blocked by our stale row's awakeable_id.
		if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return db.Query.UnregisterBuildSlotWaiter(runCtx, w.db.RW(), deployment.ID)
		}, restate.WithName("unregister build slot waiter"), restate.WithMaxRetryAttempts(runMaxAttempts)); err != nil {
			return fault.Wrap(err, fault.Public("Failed to deregister build slot waiter."))
		}
	}
}

// BuildSlotPromotion identifies the waiter that just received the freed
// slot. ReleaseAndPromoteBuildSlot returns nil when no eligible waiter
// exists, so callers branch on nil-ness rather than inspecting fields.
type BuildSlotPromotion struct {
	AwakeableID  string
	DeploymentID string
}

// ReleaseAndPromoteBuildSlot frees the slot row and any waiter row for
// deploymentID, then picks the next eligible waiter in workspaceID and
// pre-grants the freed slot to them. Waiters whose deployments have gone
// terminal are unregistered and the next is tried, so a dead waiter
// never blocks the queue. Returns nil when no eligible waiter remains.
//
// Callers must invoke this inside a transaction so the entire
// release-and-promote sequence is atomic with respect to concurrent
// releases (two parallel releases must not both pre-grant the same
// waiter). The awakeable is *not* resolved here — the caller does that
// outside the tx so the row lock isn't held across a Restate round-trip.
func ReleaseAndPromoteBuildSlot(
	ctx context.Context,
	tx db.DBTX,
	workspaceID, deploymentID string,
	now int64,
) (*BuildSlotPromotion, error) {
	if err := db.Query.ReleaseBuildSlot(ctx, tx, deploymentID); err != nil {
		return nil, err
	}
	// Also drop a waiter row in case this deployment was parked
	// (cancelled before its slot came up).
	if err := db.Query.UnregisterBuildSlotWaiter(ctx, tx, deploymentID); err != nil {
		return nil, err
	}

	for {
		next, err := db.Query.PickNextBuildSlotWaiter(ctx, tx, workspaceID)
		if err != nil {
			if db.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}

		rows, err := db.Query.PreGrantBuildSlot(ctx, tx, db.PreGrantBuildSlotParams{
			DeploymentID:     next.DeploymentID,
			WorkspaceID:      workspaceID,
			AcquiredAt:       now,
			TerminalStatuses: db.TerminalDeploymentStatuses,
		})
		if err != nil {
			return nil, err
		}

		// Unregister whether the deployment was terminal (drop dead
		// waiter) or live (it's now in build_slots, no longer parked).
		if err := db.Query.UnregisterBuildSlotWaiter(ctx, tx, next.DeploymentID); err != nil {
			return nil, err
		}

		if rows == 0 {
			// Waiter's deployment already terminal — try the next one.
			continue
		}

		return &BuildSlotPromotion{
			AwakeableID:  next.AwakeableID,
			DeploymentID: next.DeploymentID,
		}, nil
	}
}

// releaseBuildSlot frees the slot held (or waiter row registered) by a
// deployment and hands the slot to the next eligible waiter — production
// first, then by enqueue order.
//
// Idempotent: releasing a deployment that holds no slot and is not parked
// in the waiters table is a no-op. Safe to call from both the happy path
// and the compensation stack.
func (w *Workflow) releaseBuildSlot(ctx restate.ObjectContext, workspaceID, deploymentID string) {
	prom, err := restate.Run(ctx, func(runCtx restate.RunContext) (*BuildSlotPromotion, error) {
		var promoted *BuildSlotPromotion
		err := db.TxRetry(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
			p, runErr := ReleaseAndPromoteBuildSlot(txCtx, tx, workspaceID, deploymentID, time.Now().UnixMilli())
			promoted = p
			return runErr
		})
		return promoted, err
	}, restate.WithName("release build slot"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		// Release is fire-and-forget — log and move on. The fallback
		// timeout in waitForBuildSlot ensures waiters re-check the DB
		// even if a release tx fails persistently.
		logger.Warn("failed to release build slot",
			"workspace_id", workspaceID,
			"deployment_id", deploymentID,
			"error", err,
		)
		return
	}

	if prom == nil {
		logger.Info("build slot released",
			"workspace_id", workspaceID,
			"deployment_id", deploymentID,
		)
		return
	}

	restate.ResolveAwakeable[bool](ctx, prom.AwakeableID, true)
	logger.Info("build slot handed off",
		"workspace_id", workspaceID,
		"released", deploymentID,
		"promoted", prom.DeploymentID,
	)
}
