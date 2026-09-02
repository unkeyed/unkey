package deployteardown

import (
	"database/sql"
	"fmt"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const (
	// defaultDrainPollInterval is how long Teardown sleeps between drain checks.
	// krane drains a deployment in ~30s (SIGTERM grace), so a 10s cadence reports
	// completion within a poll of the real drain without hammering the database.
	defaultDrainPollInterval = 10 * time.Second

	// defaultDrainGraceTimeout bounds the wait for compute to drain. Past it
	// Teardown returns with drained=false and logs an alert rather than blocking
	// forever on a stuck pod: billing must never hang on a drain that won't
	// finish.
	defaultDrainGraceTimeout = 5 * time.Minute
)

// Teardown stops every running deployment in the workspace and polls until they
// drain. The workspace id is the virtual object key.
//
// For each deployment that is its app's current deployment it first clears
// apps.current_deployment_id: the DeployService state-change guard refuses to
// change the current deployment, and a torn-down app genuinely has no current
// deployment, so clearing it makes the guard's precondition honestly true
// instead of punching a hole in it. Frontline routes off frontline_routes +
// desired_state and ignores current_deployment_id, so clearing it does not
// disturb routing.
//
// The stop itself is fire-and-forget: ScheduleDesiredStateChange records the
// transition on each deployment's own virtual object and self-sends the apply,
// so a slow or stuck deployment cannot stall this handler. Drain is observed by
// polling the database, not by awaiting the children.
func (v *VirtualObject) Teardown(
	ctx restate.ObjectContext,
	req *hydrav1.TeardownRequest,
) (*hydrav1.TeardownResponse, error) {
	workspaceID := restate.Key(ctx)

	running, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListRunningDeploymentsByWorkspaceIdRow, error) {
		return v.db.ListRunningDeploymentsByWorkspaceId(rc, db.ListRunningDeploymentsByWorkspaceIdParams{
			WorkspaceID:    workspaceID,
			ActiveStatuses: mysqltype.ActiveComputeDeploymentStatuses,
		})
	}, restate.WithName("list running deployments"))
	if err != nil {
		return nil, fmt.Errorf("list running deployments: %w", err)
	}

	if len(running) == 0 {
		logger.Info("teardown: nothing running", "workspace_id", workspaceID)
		return &hydrav1.TeardownResponse{DeploymentsStopped: 0, Drained: true}, nil
	}

	ids := make([]string, 0, len(running))

	// appCurrent records each app's current deployment that SUSPEND is stopping
	// so Resume can re-promote exactly that deployment. It is gathered from the
	// same snapshot rows that drive the clear below, so recording it is free.
	appCurrent := make(map[string]string, len(running))

	for _, d := range running {
		ids = append(ids, d.ID)

		// Clear current_deployment_id only for the app's current deployment;
		// clearing it for a non-current one would wrongly drop a different live
		// deployment's pointer.
		if d.CurrentDeploymentID.Valid && d.CurrentDeploymentID.String == d.ID {
			if req.GetMode() == hydrav1.TeardownMode_TEARDOWN_MODE_SUSPEND {
				appCurrent[d.AppID] = d.ID
			}

			if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
				return v.db.ClearAppCurrentDeployment(rc, db.ClearAppCurrentDeploymentParams{
					UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
					AppID:        d.AppID,
					DeploymentID: sql.NullString{Valid: true, String: d.ID},
				})
			}, restate.WithName("clear current deployment "+d.AppID)); err != nil {
				return nil, fmt.Errorf("clear current deployment for app %s: %w", d.AppID, err)
			}
		}

		// A progressing deployment holds its own key with a live Deploy
		// invocation, so the stop Send below queues behind the whole build while
		// the workspace keeps buying compute. Admin cancellation is the one
		// signal that reaches a running invocation instead of its inbox: Deploy
		// aborts at its next journaled step and its compensations unwind the
		// build. CancelInvocation treats 404 as success, so a build that just
		// finished is harmless.
		if v.admin != nil && !d.Status.IsTerminal() && d.InvocationID.Valid && d.InvocationID.String != "" {
			invocationID := d.InvocationID.String
			if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
				return v.admin.CancelInvocation(rc, invocationID)
			}, restate.WithName("cancel invocation "+d.ID)); err != nil {
				return nil, fmt.Errorf("cancel invocation %s for deployment %s: %w", invocationID, d.ID, err)
			}
		}

		// Send (not Request): the per-deployment object owns the state change,
		// its retries, and the krane handoff. A replay does not re-dispatch.
		//
		// Overwrite: without it, ScheduleDesiredStateChange no-ops when the
		// deployment already has a pending transition, and this Send never
		// learns that. A deployment caught mid-transition would then survive
		// the teardown entirely: still running, but with current_deployment_id
		// already cleared above and, for cancel, no entitlement left. Teardown
		// is authoritative, so it supersedes whatever was in flight.
		hydrav1.NewDeployServiceClient(ctx, d.ID).
			ScheduleDesiredStateChange().
			Send(&hydrav1.ScheduleDesiredStateChangeRequest{
				DelayMillis: 0,
				State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
				Overwrite:   true,
			})
	}

	// Persist what this teardown stopped so Resume can reverse it. SUSPEND
	// records the restore map (when there is anything to restore); ARCHIVE is
	// permanent, so it drops any stale record instead.
	switch req.GetMode() {
	case hydrav1.TeardownMode_TEARDOWN_MODE_SUSPEND:
		if len(appCurrent) > 0 {
			// Merge into any existing suspension record: a re-enforcing teardown
			// only sees deployments running now, and replacing would drop the apps
			// the first teardown stopped from the restore map, so Resume would
			// never bring them back. New entries win on collision.
			existing, err := restate.Get[*suspension](ctx, suspensionKey)
			if err != nil {
				return nil, fmt.Errorf("read suspension record: %w", err)
			}
			merged := make(map[string]string, len(appCurrent))
			if existing != nil {
				for appID, deploymentID := range existing.AppCurrent {
					merged[appID] = deploymentID
				}
			}
			for appID, deploymentID := range appCurrent {
				merged[appID] = deploymentID
			}
			restate.Set(ctx, suspensionKey, &suspension{AppCurrent: merged})
		}
	case hydrav1.TeardownMode_TEARDOWN_MODE_ARCHIVE:
		restate.Clear(ctx, suspensionKey)
	case hydrav1.TeardownMode_TEARDOWN_MODE_UNSPECIFIED:
		// Callers always set SUSPEND or ARCHIVE. An unset mode records no
		// suspension and clears none, leaving any prior record untouched.
	}

	logger.Info("teardown stopping deployments",
		"workspace_id", workspaceID,
		"deployments_stopped", len(ids),
	)

	// Poll the database until every stopped deployment drains, bounded by an
	// absolute deadline. The whole poll runs inside a single restate.Run, so it
	// adds one journal entry instead of one per tick (a per-tick Run+Sleep loop
	// journals dozens of entries over the grace window). The deadline is derived
	// from a journaled Now(), so a replay on another node measures against the
	// same absolute cutoff rather than restarting the clock from zero.
	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current time: %w", err)
	}
	deadline := now.Add(v.drainGraceTimeout)

	drained, err := restate.Run(ctx, func(rc restate.RunContext) (bool, error) {
		for {
			active, err := v.db.CountActiveDeploymentsByIds(rc, ids)
			if err != nil {
				return false, err
			}
			if active == 0 {
				return true, nil
			}
			if !time.Now().Before(deadline) {
				return false, nil
			}
			time.Sleep(v.drainPollInterval)
		}
	}, restate.WithName("await drain"))
	if err != nil {
		return nil, fmt.Errorf("await drain: %w", err)
	}

	if !drained {
		// Force completion so billing is never blocked on a stuck pod. The
		// compute is still draining; surface it for an operator rather than
		// hanging the invocation.
		logger.Error("teardown grace timeout: compute still draining",
			"workspace_id", workspaceID,
			"deployments_stopped", len(ids),
			"grace_timeout", v.drainGraceTimeout.String(),
		)
		return &hydrav1.TeardownResponse{DeploymentsStopped: int32(len(ids)), Drained: false}, nil
	}

	logger.Info("teardown drained",
		"workspace_id", workspaceID,
		"deployments_stopped", len(ids),
	)
	return &hydrav1.TeardownResponse{DeploymentsStopped: int32(len(ids)), Drained: true}, nil
}
