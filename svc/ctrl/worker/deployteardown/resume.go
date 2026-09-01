package deployteardown

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deploy"
)

// restoreCheck is the journaled verdict on whether a suspension-record entry
// is still safe to restore. Fields are exported so Restate can serialize it
// into the journal.
type restoreCheck struct {
	Eligible bool
	Reason   string
}

// Resume reverses a SUSPEND: it returns each suspended deployment to running and
// restores its app's current deployment from the record Teardown(SUSPEND)
// saved. The workspace id is the virtual object key.
//
// Per app: restore current_deployment_id first, guarded on the pointer still
// being unset, and only wake the deployment's compute if that guarded restore
// actually claimed it. Ordering the guarded pointer restore ahead of the
// desired-state write closes a race: if a promote set a different current while
// the workspace was suspended, the restore no-ops, Resume sees it lost, and it
// leaves the deployment stopped rather than waking compute that is no longer
// current, which nothing would tear down again.
//
// The wake writes the running desired state directly via
// deploy.ApplyDesiredState rather than routing through
// DeployService.ScheduleDesiredStateChange: the latter applies
// asynchronously and its guard would refuse a desired-state change on a
// now-current deployment. ApplyDesiredState performs the same writes without
// that guard, which is safe here because current already points back at the
// deployment.
//
// Idempotent: an absent or empty record is a no-op.
func (v *VirtualObject) Resume(ctx restate.ObjectContext, _ *hydrav1.ResumeRequest) (*hydrav1.ResumeResponse, error) {
	workspaceID := restate.Key(ctx)

	susp, err := restate.Get[*suspension](ctx, suspensionKey)
	if err != nil {
		return nil, fmt.Errorf("read suspension record: %w", err)
	}
	if susp == nil || len(susp.AppCurrent) == 0 {
		logger.Info("resume: nothing suspended", "workspace_id", workspaceID)
		return &hydrav1.ResumeResponse{DeploymentsResumed: 0}, nil
	}

	// Restore in a stable order so replays are deterministic.
	appIDs := make([]string, 0, len(susp.AppCurrent))
	for appID := range susp.AppCurrent {
		appIDs = append(appIDs, appID)
	}
	sort.Strings(appIDs)

	resumed := 0
	for _, appID := range appIDs {
		deploymentID := susp.AppCurrent[appID]

		// The suspension record is a snapshot; the world may have moved on
		// while suspended. Verify the recorded deployment still exists and
		// belongs to this workspace and app, and that nothing promoted a new
		// current deployment in the meantime: restoring over either would
		// start stale compute or roll the app back to an old version.
		// Not-found and mismatch outcomes are returned as values (not errors)
		// so Restate journals the verdict instead of retrying it forever.
		check, err := restate.Run(ctx, func(rc restate.RunContext) (restoreCheck, error) {
			d, dErr := v.db.FindDeploymentById(rc, deploymentID)
			if dErr != nil {
				if db.IsNotFound(dErr) {
					return restoreCheck{Eligible: false, Reason: "deployment no longer exists"}, nil
				}
				return restoreCheck{}, dErr
			}
			if d.WorkspaceID != workspaceID || d.AppID != appID {
				return restoreCheck{Eligible: false, Reason: "deployment no longer belongs to this workspace/app"}, nil
			}
			app, aErr := v.db.FindAppById(rc, appID)
			if aErr != nil {
				if db.IsNotFound(aErr) {
					return restoreCheck{Eligible: false, Reason: "app no longer exists"}, nil
				}
				return restoreCheck{}, aErr
			}
			if app.CurrentDeploymentID.Valid && app.CurrentDeploymentID.String != "" {
				return restoreCheck{Eligible: false, Reason: "a newer current deployment was promoted while suspended"}, nil
			}
			return restoreCheck{Eligible: true, Reason: ""}, nil
		}, restate.WithName("verify restore "+appID))
		if err != nil {
			return nil, fmt.Errorf("verify restore for app %s: %w", appID, err)
		}
		if !check.Eligible {
			logger.Warn("resume: skipping app, suspension record is stale",
				"workspace_id", workspaceID,
				"app_id", appID,
				"deployment_id", deploymentID,
				"reason", check.Reason,
			)
			continue
		}

		// Restore current_deployment_id first, guarded on the pointer still being
		// unset (the UPDATE has AND current_deployment_id IS NULL), then re-read to
		// learn whether this resume actually claimed it. A promotion racing past
		// the eligibility check makes the guarded update a no-op; we detect that
		// here and skip waking compute, so a deployment that is no longer current
		// is never turned back on.
		claimed, err := restate.Run(ctx, func(rc restate.RunContext) (bool, error) {
			if err := v.db.SetAppCurrentDeployment(rc, db.SetAppCurrentDeploymentParams{
				DeploymentID: sql.NullString{Valid: true, String: deploymentID},
				UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				AppID:        appID,
			}); err != nil {
				return false, err
			}
			app, aErr := v.db.FindAppById(rc, appID)
			if aErr != nil {
				return false, aErr
			}
			return app.CurrentDeploymentID.Valid && app.CurrentDeploymentID.String == deploymentID, nil
		}, restate.WithName("restore current "+appID))
		if err != nil {
			return nil, fmt.Errorf("restore current deployment for app %s: %w", appID, err)
		}
		if !claimed {
			logger.Warn("resume: skipping app, current deployment changed before restore",
				"workspace_id", workspaceID,
				"app_id", appID,
				"deployment_id", deploymentID,
			)
			continue
		}

		// Now current again, so wake its compute. ApplyDesiredState has no
		// current-deployment guard (unlike ScheduleDesiredStateChange), so the
		// write goes through even though the deployment is current.
		if err := deploy.ApplyDesiredState(
			ctx,
			v.db,
			deploymentID,
			mysqltype.DeploymentsDesiredStateRunning,
			db.DeploymentTopologyDesiredStatusRunning,
		); err != nil {
			return nil, fmt.Errorf("resume deployment %s to running: %w", deploymentID, err)
		}

		resumed++
	}

	restate.Clear(ctx, suspensionKey)

	logger.Info("resume complete",
		"workspace_id", workspaceID,
		"deployments_resumed", resumed,
	)

	return &hydrav1.ResumeResponse{DeploymentsResumed: int32(resumed)}, nil
}
