package deployteardown

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deployment"
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
// Per app the order is the inverse of suspend: apply the running desired state
// first, while the deployment is still not current, then restore
// current_deployment_id. This matters because of the DeploymentService guard,
// which refuses to change the desired state of an app's current deployment.
//
// For the same reason Resume does NOT route the running transition through
// DeploymentService.ScheduleDesiredStateChange: that applies asynchronously via
// a self-sent ChangeDesiredState which would race the current-restore below and
// then be refused by the guard once current points back at the deployment.
// Resume instead writes the running desired state directly via
// deployment.ApplyDesiredState (the same writes ChangeDesiredState performs,
// minus the guard and nonce), then restores current.
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
			d, dErr := db.Query.FindDeploymentById(rc, v.db.RO(), deploymentID)
			if dErr != nil {
				if db.IsNotFound(dErr) {
					return restoreCheck{Eligible: false, Reason: "deployment no longer exists"}, nil
				}
				return restoreCheck{}, dErr
			}
			if d.WorkspaceID != workspaceID || d.AppID != appID {
				return restoreCheck{Eligible: false, Reason: "deployment no longer belongs to this workspace/app"}, nil
			}
			app, aErr := db.Query.FindAppById(rc, v.db.RO(), appID)
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

		// Apply running while the deployment is still not current, so the guard
		// stays satisfied. See the doc comment for why ScheduleDesiredStateChange
		// is not used here.
		if err := deployment.ApplyDesiredState(
			ctx,
			v.db,
			deploymentID,
			db.DeploymentsDesiredStateRunning,
			db.DeploymentTopologyDesiredStatusRunning,
		); err != nil {
			return nil, fmt.Errorf("resume deployment %s to running: %w", deploymentID, err)
		}

		// The query itself re-checks current_deployment_id IS NULL, so a
		// promotion racing past the verification above loses nothing: the
		// restore becomes a no-op rather than a rollback.
		if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
			return db.Query.SetAppCurrentDeployment(rc, v.db.RW(), db.SetAppCurrentDeploymentParams{
				DeploymentID: sql.NullString{Valid: true, String: deploymentID},
				UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				AppID:        appID,
			})
		}, restate.WithName("restore current "+appID)); err != nil {
			return nil, fmt.Errorf("restore current deployment for app %s: %w", appID, err)
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
