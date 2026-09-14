package buildslot

import (
	"context"
	"database/sql"
	"sort"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/logger"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// auditDeployments verifies the given deployments against the database and
// Restate, and returns the IDs that are dead. It runs as one bounded,
// journaled step. The callers pass slot holders, wait-list entries, or
// both.
//
// Virtual Object state outlives invocations: a deployment ID stays in
// active_slots or a wait list until something removes it. A `restate kill`,
// a purge, or a forced service-deployment removal ends the invocation
// without its Release compensation. The dead entry then blocks the whole
// workspace. This audit is the pull-based safety net. It does not depend
// on an earlier scheduled event.
//
// A deployment is dead when one of these is true:
//   - the deployment row is gone from the database
//   - the deployment status is terminal
//   - the recorded Restate invocation no longer executes (killed,
//     cancelled, completed, or purged)
//
// A non-terminal deployment without a recorded invocation ID stays. The
// slot lease still bounds it.
func (s *Service) auditDeployments(
	ctx restate.ObjectContext,
	deploymentIDs []string,
) ([]string, error) {
	sort.Strings(deploymentIDs)

	return restate.Run(ctx, func(runCtx restate.RunContext) ([]string, error) {
		return reapDeadDeployments(runCtx, s.db, s.restateAdmin, deploymentIDs)
	}, restate.WithName("audit build slot deployments"), restate.WithMaxRetryAttempts(runMaxAttempts))
}

// reapDeadDeployments is the core of the audit. It returns the dead
// deployment IDs. A row that is still active while its invocation is gone
// is also force-failed in the database, so the dashboard does not show a
// phantom build after the reclaim. It runs under a plain context inside
// restate.Run and is unit-testable.
func reapDeadDeployments(
	ctx context.Context,
	database db.Database,
	liveness InvocationLiveness,
	deploymentIDs []string,
) ([]string, error) {
	staleIDs := []string{}
	type pendingCheck struct {
		deploymentID string
		invocationID string
	}
	toCheck := []pendingCheck{}

	for _, deploymentID := range deploymentIDs {
		deployment, err := database.FindDeploymentById(ctx, deploymentID)
		if db.IsNotFound(err) {
			staleIDs = append(staleIDs, deploymentID)
			continue
		}
		if err != nil {
			return nil, err
		}
		if deployment.Status.IsTerminal() {
			staleIDs = append(staleIDs, deploymentID)
			continue
		}
		if deployment.InvocationID.Valid && deployment.InvocationID.String != "" {
			toCheck = append(toCheck, pendingCheck{
				deploymentID: deploymentID,
				invocationID: deployment.InvocationID.String,
			})
		}
	}

	if len(toCheck) > 0 {
		invocationIDs := make([]string, len(toCheck))
		for i, c := range toCheck {
			invocationIDs[i] = c.invocationID
		}
		live, err := liveness.FindLiveInvocations(ctx, invocationIDs)
		if err != nil {
			return nil, err
		}
		for _, c := range toCheck {
			if !live[c.invocationID] {
				staleIDs = append(staleIDs, c.deploymentID)
				// The row still shows an active status while its Deploy
				// invocation is gone. Force-fail it so the database agrees
				// with the reclaim and the dashboard does not show a
				// phantom build. The update is guarded: a concurrent
				// legitimate terminal transition wins.
				if failErr := forceFailDeployment(ctx, database, c.deploymentID,
					"build slot audit: deploy invocation no longer exists in Restate"); failErr != nil {
					return nil, failErr
				}
			}
		}
	}

	sort.Strings(staleIDs)
	return staleIDs, nil
}

// forceFailDeployment marks a deployment failed and ends its active build
// steps with the given reason. The status update only applies while the
// row is non-terminal, so a legitimate concurrent transition is never
// overwritten. Idempotent, safe under restate.Run retries.
func forceFailDeployment(
	ctx context.Context,
	database db.Database,
	deploymentID, reason string,
) error {
	now := sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()}
	if err := database.UpdateDeploymentStatusIfActive(ctx, db.UpdateDeploymentStatusIfActiveParams{
		ID:                  deploymentID,
		Status:              mysqltype.DeploymentsStatusFailed,
		UpdatedAt:           now,
		ProgressingStatuses: mysqltype.ProgressingDeploymentStatuses,
	}); err != nil {
		return err
	}
	return database.EndActiveDeploymentStepsWithError(ctx, db.EndActiveDeploymentStepsWithErrorParams{
		DeploymentID: deploymentID,
		EndedAt:      now,
		Error:        sql.NullString{Valid: true, String: reason},
	})
}

// pruneDeadWaiters audits both wait lists and removes every entry whose
// deployment is dead, so a dead waiter is never promoted into a freed
// slot. Best-effort: when the audit fails, both lists come back unchanged
// and the wait-entry lease removes dead entries later. The caller must
// persist the returned lists.
func (s *Service) pruneDeadWaiters(
	ctx restate.ObjectContext,
	workspaceID string,
	prodWait, previewWait []waitEntry,
) ([]waitEntry, []waitEntry) {
	waiterIDs := waitListIDs(prodWait, previewWait)
	if len(waiterIDs) == 0 {
		return prodWait, previewWait
	}

	deadIDs, err := s.auditDeployments(ctx, waiterIDs)
	if err != nil {
		logger.Warn("wait list audit failed, promoting without pruning",
			"workspace_id", workspaceID,
			"error", err,
		)
		return prodWait, previewWait
	}

	return dropWaitEntries(ctx, workspaceID, deadIDs, prodWait, previewWait)
}

// dropWaitEntries removes the given deployments from both wait lists and
// rejects their awakeables. A reject on a dead invocation is harmless.
func dropWaitEntries(
	ctx restate.ObjectContext,
	workspaceID string,
	deploymentIDs []string,
	prodWait, previewWait []waitEntry,
) ([]waitEntry, []waitEntry) {
	for _, id := range deploymentIDs {
		awakeableID := findAwakeableID(prodWait, previewWait, id)
		if awakeableID == "" {
			continue
		}
		restate.RejectAwakeable(ctx, awakeableID,
			restate.TerminalErrorf("build slot wait entry removed: deploy invocation no longer exists"))
		prodWait = removeFromWaitList(prodWait, id)
		previewWait = removeFromWaitList(previewWait, id)

		logger.Warn("removed dead build slot waiter",
			"workspace_id", workspaceID,
			"deployment_id", id,
		)
	}
	return prodWait, previewWait
}

// waitListIDs collects the deployment IDs of every entry in the given
// wait lists.
func waitListIDs(lists ...[]waitEntry) []string {
	ids := []string{}
	for _, list := range lists {
		for _, w := range list {
			ids = append(ids, w.DeploymentID)
		}
	}
	return ids
}

// reclaimStaleSlots removes the stale deployments from active_slots and
// promotes as many waiters as the freed capacity allows, production first.
// The caller must persist active and both wait lists afterwards.
func reclaimStaleSlots(
	ctx restate.ObjectContext,
	workspaceID string,
	staleIDs []string,
	active map[string]bool,
	prodWait, previewWait []waitEntry,
	buildLimit uint32,
) (map[string]bool, []waitEntry, []waitEntry) {
	for _, id := range staleIDs {
		delete(active, id)
	}

	logger.Warn("reclaimed stale build slots",
		"workspace_id", workspaceID,
		"stale_deployment_ids", staleIDs,
		"active", len(active),
		"limit", buildLimit,
	)

	for uint32(len(active)) < buildLimit {
		var promoted *waitEntry
		promoted, prodWait, previewWait = pickNextWaiter(prodWait, previewWait)
		if promoted == nil {
			break
		}
		active[promoted.DeploymentID] = true
		restate.ResolveAwakeable(ctx, promoted.AwakeableID, true)
		scheduleExpiry(ctx, workspaceID, promoted.DeploymentID, slotLeaseDuration, 0)

		logger.Info("build slot handed off after stale reclaim",
			"workspace_id", workspaceID,
			"promoted", promoted.DeploymentID,
			"active", len(active),
		)
	}

	return active, prodWait, previewWait
}
