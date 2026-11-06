package deployteardown

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deployment"
)

// restoreCheck is the journaled verdict on whether a suspension-record entry
// is still safe to restore. Fields are exported so Restate can serialize it
// into the journal.
type restoreCheck struct {
	Eligible      bool
	Reason        string
	EnvironmentID string
	AppID         string
}

func staleRestore(reason string) restoreCheck {
	return restoreCheck{Eligible: false, Reason: reason, EnvironmentID: "", AppID: ""}
}

// restoreEntry is one suspended deployment and the environment or, for records
// written before the live pointer moved to environments, the app it was
// current for.
type restoreEntry struct {
	DeploymentID  string
	EnvironmentID string
	AppID         string
}

func (e restoreEntry) owns(environmentID, appID string) bool {
	if e.EnvironmentID != "" {
		return e.EnvironmentID == environmentID
	}
	return e.AppID == appID
}

// restoreEntries lists the record's entries in a stable order so replays are
// deterministic.
func restoreEntries(susp *suspension) []restoreEntry {
	entries := make([]restoreEntry, 0, len(susp.EnvironmentCurrent)+len(susp.AppCurrent))
	for environmentID, deploymentID := range susp.EnvironmentCurrent {
		entries = append(entries, restoreEntry{DeploymentID: deploymentID, EnvironmentID: environmentID, AppID: ""})
	}
	for appID, deploymentID := range susp.AppCurrent {
		entries = append(entries, restoreEntry{DeploymentID: deploymentID, EnvironmentID: "", AppID: appID})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].EnvironmentID != entries[j].EnvironmentID {
			return entries[i].EnvironmentID < entries[j].EnvironmentID
		}
		return entries[i].AppID < entries[j].AppID
	})
	return entries
}

// Resume reverses a SUSPEND: it returns each suspended deployment to running and
// restores its environment's current deployment from the record Teardown(SUSPEND)
// saved. The workspace id is the virtual object key.
//
// Per environment: restore current_deployment_id first, guarded on the pointer still
// being unset, and only wake the deployment's compute if that guarded restore
// actually claimed it. Ordering the guarded pointer restore ahead of the
// desired-state write closes a race: if a promote set a different current while
// the workspace was suspended, the restore no-ops, Resume sees it lost, and it
// leaves the deployment stopped rather than waking compute that is no longer
// current, which nothing would tear down again.
//
// The wake writes the running desired state directly via
// deployment.ApplyDesiredState rather than routing through
// DeploymentService.ScheduleDesiredStateChange: the latter applies
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
	if susp == nil || len(susp.EnvironmentCurrent)+len(susp.AppCurrent) == 0 {
		logger.Info("resume: nothing suspended", "workspace_id", workspaceID)
		return &hydrav1.ResumeResponse{DeploymentsResumed: 0}, nil
	}

	resumed := 0
	for _, entry := range restoreEntries(susp) {
		deploymentID := entry.DeploymentID

		// The suspension record is a snapshot; the world may have moved on
		// while suspended. Verify the recorded deployment still exists and
		// belongs to this workspace and the recorded environment or app, and
		// that nothing promoted a new current deployment in the meantime:
		// restoring over either would start stale compute or roll the
		// environment back to an old version. Not-found and mismatch outcomes
		// are returned as values (not errors) so Restate journals the verdict
		// instead of retrying it forever.
		check, err := restate.Run(ctx, func(rc restate.RunContext) (restoreCheck, error) {
			d, dErr := v.db.FindDeploymentById(rc, deploymentID)
			if dErr != nil {
				if db.IsNotFound(dErr) {
					return staleRestore("deployment no longer exists"), nil
				}
				return restoreCheck{}, dErr
			}
			if d.WorkspaceID != workspaceID || !entry.owns(d.EnvironmentID, d.AppID) {
				return staleRestore("deployment no longer belongs to this workspace/environment"), nil
			}
			environment, eErr := v.db.FindEnvironmentById(rc, d.EnvironmentID)
			if eErr != nil {
				if db.IsNotFound(eErr) {
					return staleRestore("environment no longer exists"), nil
				}
				return restoreCheck{}, eErr
			}
			if environment.CurrentDeploymentID.Valid && environment.CurrentDeploymentID.String != "" {
				return staleRestore("a newer current deployment was promoted while suspended"), nil
			}
			return restoreCheck{Eligible: true, Reason: "", EnvironmentID: d.EnvironmentID, AppID: d.AppID}, nil
		}, restate.WithName("verify restore "+deploymentID))
		if err != nil {
			return nil, fmt.Errorf("verify restore for deployment %s: %w", deploymentID, err)
		}
		if !check.Eligible {
			logger.Warn("resume: skipping deployment, suspension record is stale",
				"workspace_id", workspaceID,
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
			return db.TxWithResult(rc, v.db.RW(), func(txCtx context.Context, tx db.DBTX) (bool, error) {
				queries := db.NewQueries(tx)
				now := sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()}
				if err := queries.SetEnvironmentCurrentDeployment(txCtx, db.SetEnvironmentCurrentDeploymentParams{
					DeploymentID:  sql.NullString{Valid: true, String: deploymentID},
					UpdatedAt:     now,
					EnvironmentID: check.EnvironmentID,
				}); err != nil {
					return false, err
				}
				environment, eErr := queries.FindEnvironmentById(txCtx, check.EnvironmentID)
				if eErr != nil {
					return false, eErr
				}
				if !environment.CurrentDeploymentID.Valid || environment.CurrentDeploymentID.String != deploymentID {
					return false, nil
				}
				if err := queries.SetAppCurrentDeployment(txCtx, db.SetAppCurrentDeploymentParams{
					DeploymentID: sql.NullString{Valid: true, String: deploymentID},
					UpdatedAt:    now,
					AppID:        check.AppID,
				}); err != nil {
					return false, err
				}
				return true, nil
			})
		}, restate.WithName("restore current "+deploymentID))
		if err != nil {
			return nil, fmt.Errorf("restore current deployment %s: %w", deploymentID, err)
		}
		if !claimed {
			logger.Warn("resume: skipping deployment, current deployment changed before restore",
				"workspace_id", workspaceID,
				"environment_id", check.EnvironmentID,
				"deployment_id", deploymentID,
			)
			continue
		}

		// Now current again, so wake its compute. ApplyDesiredState has no
		// current-deployment guard (unlike ScheduleDesiredStateChange), so the
		// write goes through even though the deployment is current.
		if err := deployment.ApplyDesiredState(
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
