package buildslot

import (
	"context"
	"sort"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// auditActiveSlots verifies every deployment in active_slots against the
// database and Restate, and returns the IDs whose slots are stale. It runs
// as one bounded, journaled step.
//
// Virtual Object state outlives invocations: a deployment ID stays in
// active_slots until something removes it. A `restate kill`, a purge, or a
// forced service-deployment removal ends the invocation without its Release
// compensation. The dead entry then blocks the whole workspace. This audit
// is the pull-based safety net. It does not depend on an earlier scheduled
// event.
//
// A slot is stale when one of these is true:
//   - the deployment row is gone from the database
//   - the deployment status is terminal
//   - the recorded Restate invocation is gone from sys_invocation
//     (Restate drops killed, purged, and completed invocations)
//
// A non-terminal deployment without a recorded invocation ID stays. The
// slot lease still bounds it.
func (s *Service) auditActiveSlots(
	ctx restate.ObjectContext,
	workspaceID string,
	active map[string]bool,
) ([]string, error) {
	deploymentIDs := make([]string, 0, len(active))
	for id := range active {
		deploymentIDs = append(deploymentIDs, id)
	}
	sort.Strings(deploymentIDs)

	return restate.Run(ctx, func(runCtx restate.RunContext) ([]string, error) {
		return computeStaleSlots(runCtx, s.db, s.restateAdmin, deploymentIDs)
	}, restate.WithName("audit active build slots"), restate.WithMaxRetryAttempts(runMaxAttempts))
}

// computeStaleSlots is the core of the audit. It runs under a plain context
// inside restate.Run and is unit-testable.
func computeStaleSlots(
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
			}
		}
	}

	sort.Strings(staleIDs)
	return staleIDs, nil
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
		scheduleExpiry(ctx, workspaceID, promoted.DeploymentID, slotLeaseDuration)

		logger.Info("build slot handed off after stale reclaim",
			"workspace_id", workspaceID,
			"promoted", promoted.DeploymentID,
			"active", len(active),
		)
	}

	return active, prodWait, previewWait
}
