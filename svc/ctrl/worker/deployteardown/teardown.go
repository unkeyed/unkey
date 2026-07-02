package deployteardown

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
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
// apps.current_deployment_id: the DeploymentService guard refuses to change the
// current deployment, and a torn-down app genuinely has no current deployment,
// so clearing it makes the guard's precondition honestly true instead of
// punching a hole in it. Frontline routes off frontline_routes + desired_state
// and ignores current_deployment_id, so clearing it does not disturb routing.
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

	desiredState, err := desiredStateFor(req.GetMode())
	if err != nil {
		return nil, err
	}

	running, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListRunningDeploymentsByWorkspaceIdRow, error) {
		return db.Query.ListRunningDeploymentsByWorkspaceId(rc, v.db.RO(), workspaceID)
	}, restate.WithName("list running deployments"))
	if err != nil {
		return nil, fmt.Errorf("list running deployments: %w", err)
	}

	if len(running) == 0 {
		logger.Info("teardown: nothing running",
			"workspace_id", workspaceID,
			"mode", req.GetMode().String(),
		)
		return &hydrav1.TeardownResponse{DeploymentsStopped: 0, Drained: true}, nil
	}

	ids := make([]string, 0, len(running))
	for _, d := range running {
		ids = append(ids, d.ID)

		// Clear current_deployment_id only for the app's current deployment;
		// clearing it for a non-current one would wrongly drop a different live
		// deployment's pointer.
		if d.CurrentDeploymentID.Valid && d.CurrentDeploymentID.String == d.ID {
			if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
				return db.Query.ClearAppCurrentDeployment(rc, v.db.RW(), db.ClearAppCurrentDeploymentParams{
					UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
					AppID:        d.AppID,
					DeploymentID: sql.NullString{Valid: true, String: d.ID},
				})
			}, restate.WithName("clear current deployment "+d.AppID)); err != nil {
				return nil, fmt.Errorf("clear current deployment for app %s: %w", d.AppID, err)
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
		hydrav1.NewDeploymentServiceClient(ctx, d.ID).
			ScheduleDesiredStateChange().
			Send(&hydrav1.ScheduleDesiredStateChangeRequest{
				DelayMillis: 0,
				State:       desiredState,
				Overwrite:   true,
			})
	}

	logger.Info("teardown stopping deployments",
		"workspace_id", workspaceID,
		"mode", req.GetMode().String(),
		"deployments_stopped", len(ids),
	)

	// Poll until every deployment drains, sleeping between checks. The loop is
	// bounded: maxPolls * pollInterval == graceTimeout. The counter is pure Go
	// arithmetic and the sleeps are journaled in order, so this is deterministic
	// across replays.
	maxPolls := int(v.drainGraceTimeout / v.drainPollInterval)
	for poll := 0; ; poll++ {
		active, err := restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
			return db.Query.CountActiveDeploymentsByIds(rc, v.db.RO(), ids)
		}, restate.WithName("count active deployments"))
		if err != nil {
			return nil, fmt.Errorf("count active deployments: %w", err)
		}

		if active == 0 {
			logger.Info("teardown drained",
				"workspace_id", workspaceID,
				"mode", req.GetMode().String(),
				"deployments_stopped", len(ids),
			)
			return &hydrav1.TeardownResponse{DeploymentsStopped: int32(len(ids)), Drained: true}, nil
		}

		if poll >= maxPolls {
			// Force completion so billing is never blocked on a stuck pod. The
			// compute is still draining; surface it for an operator rather than
			// hanging the invocation.
			logger.Error("teardown grace timeout: compute still draining",
				"workspace_id", workspaceID,
				"mode", req.GetMode().String(),
				"active_deployments", active,
				"deployments_stopped", len(ids),
				"grace_timeout", v.drainGraceTimeout.String(),
			)
			return &hydrav1.TeardownResponse{DeploymentsStopped: int32(len(ids)), Drained: false}, nil
		}

		if err := restate.Sleep(ctx, v.drainPollInterval); err != nil {
			return nil, err
		}
	}
}

// desiredStateFor maps a teardown mode to the deployment desired state it
// drives: ARCHIVE is permanent, SUSPEND is resumable. Both map (in
// DeploymentService) to a stopped topology, so krane drains either way.
func desiredStateFor(mode hydrav1.TeardownMode) (hydrav1.DeploymentDesiredState, error) {
	switch mode {
	case hydrav1.TeardownMode_TEARDOWN_MODE_ARCHIVE, hydrav1.TeardownMode_TEARDOWN_MODE_SUSPEND:
		return hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED, nil
	case hydrav1.TeardownMode_TEARDOWN_MODE_UNSPECIFIED:
		return 0, restate.TerminalErrorf("teardown mode unspecified")
	default:
		return 0, restate.TerminalErrorf("unhandled teardown mode: %s", mode)
	}
}
