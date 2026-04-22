package buildslot

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Reconcile explicitly runs the self-heal pass: every deployment in
// active_slots and the wait lists is checked against its DB status, and
// any that are terminal are released. Waiters are promoted up to the
// workspace's concurrency limit after terminal entries are removed.
//
// The same primitive is invoked automatically from AcquireOrWait when the
// workspace is at capacity; this RPC is the operator-triggered version.
func (s *Service) Reconcile(
	ctx restate.ObjectContext,
	_ *hydrav1.ReconcileRequest,
) (*hydrav1.ReconcileResponse, error) {
	workspaceID := restate.Key(ctx)

	state, err := loadReconcileState(ctx)
	if err != nil {
		return nil, err
	}
	quota, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Quotas, error) {
		return db.Query.FindQuotaByWorkspaceID(runCtx, s.db.RO(), workspaceID)
	}, restate.WithName("reconcile: fetch quota"))
	if err != nil {
		return nil, fmt.Errorf("fetch quota: %w", err)
	}

	state, released, err := s.sweepAndPromote(ctx, state, quota.MaxConcurrentBuilds)
	if err != nil {
		return nil, err
	}
	saveReconcileState(ctx, state)

	logger.Info("reconcile finished",
		"workspace_id", workspaceID,
		"released_count", len(released),
		"released", released,
	)

	return &hydrav1.ReconcileResponse{
		ReleasedDeploymentIds: released,
	}, nil
}

// ForceRelease releases a specific deployment's slot without consulting
// DB status. Use when Reconcile can't help (e.g. the deployment's DB
// status is still active but the workflow is genuinely dead in Restate).
//
// Internally delegates to the same Release path so a waiter is promoted
// as if the slot had been released normally.
func (s *Service) ForceRelease(
	ctx restate.ObjectContext,
	req *hydrav1.ForceReleaseRequest,
) (*hydrav1.ForceReleaseResponse, error) {
	if req.GetDeploymentId() == "" {
		return nil, restate.TerminalError(fmt.Errorf("deployment_id is required"))
	}

	logger.Info("force releasing build slot",
		"workspace_id", restate.Key(ctx),
		"deployment_id", req.GetDeploymentId(),
	)

	if _, err := s.Release(ctx, &hydrav1.ReleaseSlotRequest{
		DeploymentId: req.GetDeploymentId(),
	}); err != nil {
		return nil, err
	}
	return &hydrav1.ForceReleaseResponse{}, nil
}

// supersededByClearStateMessage is surfaced to losers of the (app,
// environment, branch) dedup that ClearState runs. A deployment that
// loses the dedup didn't fail in the normal sense — a newer commit for
// the same branch was already waiting alongside it, and we're applying
// the "rapid pushes only build the latest" invariant retroactively.
const supersededByClearStateMessage = "Superseded during build slot recovery"

// ClearState is the operator recovery primitive. It rebuilds the VO
// state from scratch:
//  1. Drop every existing active_slots entry (assumed unrecoverable).
//  2. Group waiters by (app_id, environment_id, git_branch) and keep
//     only the newest per group — this extends the existing create-time
//     dedup invariant (rapid pushes only build the latest commit) to
//     the recovery path. Losers are marked superseded in the DB and
//     their awakeables rejected so their parked Deploy handlers
//     unblock and exit cleanly (compensation fires, status stays
//     superseded because UpdateDeploymentStatusIfActive protects it).
//  3. Waiters without a git branch (docker-image deploys) can't be
//     deduped and are all granted slots — they're manual anyway.
//  4. Waiters whose status is already terminal are dropped silently.
//  5. The surviving waiters become the new active_slots with their
//     awakeables resolved.
//
// This may still over-subscribe beyond max_concurrent_builds if a
// workspace has many distinct active branches, but by at most "one
// build per live branch" instead of "every single queued waiter at
// once", which matches the invariant dedup enforces on the happy path.
//
// Does not cancel any in-flight deployments; only resets the
// accounting so future AcquireOrWait calls can proceed cleanly.
func (s *Service) ClearState(
	ctx restate.ObjectContext,
	_ *hydrav1.ClearStateRequest,
) (*hydrav1.ClearStateResponse, error) {
	workspaceID := restate.Key(ctx)

	prodWait, err := loadWaitList(ctx, stateKeyProdWaitList)
	if err != nil {
		return nil, fmt.Errorf("load prod wait list: %w", err)
	}
	previewWait, err := loadWaitList(ctx, stateKeyPreviewWaitList)
	if err != nil {
		return nil, fmt.Errorf("load preview wait list: %w", err)
	}

	waiterIDs := make([]string, 0, len(prodWait)+len(previewWait))
	for _, w := range prodWait {
		waiterIDs = append(waiterIDs, w.DeploymentID)
	}
	for _, w := range previewWait {
		waiterIDs = append(waiterIDs, w.DeploymentID)
	}

	// Batched: one query covers every waiter, across both lists.
	// Missing rows (deployment deleted in DB) are treated as terminal.
	infoByID := make(map[string]db.FindDeploymentDedupInfoByIdsRow, len(waiterIDs))
	if len(waiterIDs) > 0 {
		rows, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.FindDeploymentDedupInfoByIdsRow, error) {
			return db.Query.FindDeploymentDedupInfoByIds(runCtx, s.db.RO(), waiterIDs)
		}, restate.WithName("clear state: fetch waiter dedup info"))
		if err != nil {
			return nil, fmt.Errorf("fetch waiter dedup info: %w", err)
		}
		for _, r := range rows {
			infoByID[r.ID] = r
		}
	}

	// Group by (app_id, env_id, git_branch). Only branched waiters go
	// into groups; branchless and terminal/missing are handled below.
	type groupKey struct{ app, env, branch string }
	groups := make(map[groupKey][]waitEntry)
	var branchless []waitEntry
	var terminalDropped []string

	classify := func(list []waitEntry) {
		for _, w := range list {
			info, ok := infoByID[w.DeploymentID]
			if !ok || isTerminalDeploymentStatus(info.Status) {
				terminalDropped = append(terminalDropped, w.DeploymentID)
				continue
			}
			if !info.GitBranch.Valid || info.GitBranch.String == "" {
				branchless = append(branchless, w)
				continue
			}
			k := groupKey{app: info.AppID, env: info.EnvironmentID, branch: info.GitBranch.String}
			groups[k] = append(groups[k], w)
		}
	}
	classify(prodWait)
	classify(previewWait)

	granted := make(map[string]bool)
	var supersededIDs []string

	for _, members := range groups {
		// Winner = newest created_at.
		winner := members[0]
		winnerCreated := infoByID[winner.DeploymentID].CreatedAt
		for _, m := range members[1:] {
			if infoByID[m.DeploymentID].CreatedAt > winnerCreated {
				winner = m
				winnerCreated = infoByID[m.DeploymentID].CreatedAt
			}
		}
		granted[winner.DeploymentID] = true
		restate.ResolveAwakeable(ctx, winner.AwakeableID, true)

		for _, m := range members {
			if m.DeploymentID == winner.DeploymentID {
				continue
			}
			supersededIDs = append(supersededIDs, m.DeploymentID)
			restate.RejectAwakeable(ctx, m.AwakeableID, supersedeReason(m.DeploymentID))
		}
	}

	for _, w := range branchless {
		granted[w.DeploymentID] = true
		restate.ResolveAwakeable(ctx, w.AwakeableID, true)
	}

	// Mark supersede losers in DB. UpdateDeploymentStatusIfActive skips
	// rows already in terminal status, so if a loser raced to terminal
	// between our read and this write, we don't clobber it.
	if len(supersededIDs) > 0 {
		if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			now := sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()}
			for _, id := range supersededIDs {
				if err := db.Query.UpdateDeploymentStatusIfActive(runCtx, s.db.RW(), db.UpdateDeploymentStatusIfActiveParams{
					ID:        id,
					Status:    db.DeploymentsStatusSuperseded,
					UpdatedAt: now,
				}); err != nil {
					return fmt.Errorf("update %s: %w", id, err)
				}
			}
			return nil
		}, restate.WithName("clear state: mark losers superseded")); err != nil {
			return nil, fmt.Errorf("mark losers superseded: %w", err)
		}
	}

	saveActiveSlots(ctx, granted)
	restate.Clear(ctx, stateKeyProdWaitList)
	restate.Clear(ctx, stateKeyPreviewWaitList)

	logger.Warn("clear state: wait lists drained with per-branch dedup",
		"workspace_id", workspaceID,
		"granted", len(granted),
		"superseded", supersededIDs,
		"dropped_terminal", terminalDropped,
	)

	return &hydrav1.ClearStateResponse{}, nil
}

func supersedeReason(deploymentID string) error {
	return restate.TerminalError(fmt.Errorf("%s: %s", supersededByClearStateMessage, deploymentID))
}
