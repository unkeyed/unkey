package deploy

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// ScheduleDesiredStateChange records a delayed desired-state transition:
// store a nonce with the target state, self-send a delayed ChangeDesiredState
// carrying that nonce. Restate timers cannot be cancelled, so scheduling again
// overwrites the nonce and the older delayed call no-ops on mismatch.
//
// Cross-key callers only. A handler already running on the deployment's key
// uses applyDesiredStateNow: a Request here would deadlock on its own busy
// key, and a delay-0 self-Send would queue behind the running handler.
func (w *Workflow) ScheduleDesiredStateChange(ctx restate.ObjectContext, req *hydrav1.ScheduleDesiredStateChangeRequest) (*hydrav1.ScheduleDesiredStateChangeResponse, error) {
	if !req.Overwrite {
		t, err := restate.Get[*transition](ctx, transitionKey)
		if err != nil {
			return nil, err
		}
		if t != nil {
			// This is a noop, since we don't overwrite
			return &hydrav1.ScheduleDesiredStateChangeResponse{}, nil
		}
	}

	nonce := restate.UUID(ctx).String()

	t := transition{
		Nonce: nonce,
		To:    req.GetState(),
	}

	restate.Set(ctx, transitionKey, &t)

	delay := time.Duration(req.GetDelayMillis()) * time.Millisecond

	options := []restate.SendOption{}
	if delay > 0 {
		options = append(options, restate.WithDelay(delay))
	}

	hydrav1.NewDeployServiceClient(ctx, restate.Key(ctx)).ChangeDesiredState().Send(&hydrav1.ChangeDesiredStateRequest{
		Nonce: nonce,
		State: req.GetState(),
	}, options...)

	return &hydrav1.ScheduleDesiredStateChangeResponse{}, nil
}

// ChangeDesiredState applies a scheduled transition if its nonce still
// matches the stored one. No record or a mismatch means the transition was
// cleared or superseded: return success without touching the database.
func (w *Workflow) ChangeDesiredState(ctx restate.ObjectContext, req *hydrav1.ChangeDesiredStateRequest) (*hydrav1.ChangeDesiredStateResponse, error) {
	t, err := restate.Get[*transition](ctx, transitionKey)
	if err != nil {
		return nil, err
	}
	if t == nil {
		// This is a noop, since the request was removed
		return &hydrav1.ChangeDesiredStateResponse{}, nil
	}
	if t.Nonce != req.GetNonce() {
		// This is a noop, since the request is outdated
		return &hydrav1.ChangeDesiredStateResponse{}, nil
	}

	if err := w.applyDesiredStateGuarded(ctx, restate.Key(ctx), req.GetState()); err != nil {
		return nil, err
	}

	restate.Clear(ctx, transitionKey)

	return &hydrav1.ChangeDesiredStateResponse{}, nil
}

// ClearScheduledStateChanges removes the pending transition record. An
// already enqueued delayed ChangeDesiredState still fires, finds no record,
// and no-ops.
func (w *Workflow) ClearScheduledStateChanges(ctx restate.ObjectContext, req *hydrav1.ClearScheduledStateChangesRequest) (*hydrav1.ClearScheduledStateChangesResponse, error) {
	restate.Clear(ctx, transitionKey)
	return &hydrav1.ClearScheduledStateChangesResponse{}, nil
}

// ApplyDesiredState writes the desired state and propagates it to every
// region's topology, without the current-deployment guard. Only for callers
// that legitimately mutate the current deployment: Resume uses it to wake a
// deployment it just made current again. Everything else goes through the
// guarded paths.
func ApplyDesiredState(ctx restate.ObjectContext, database db.Database, deploymentID string, desiredState mysqltype.DeploymentsDesiredState, topologyStatus db.DeploymentTopologyDesiredStatus) error {
	err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return database.UpdateDeploymentDesiredState(runCtx, db.UpdateDeploymentDesiredStateParams{
			ID:           deploymentID,
			DesiredState: desiredState,
			UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("updating desired state"))
	if err != nil {
		return err
	}

	return applyTopologyDesiredStatus(ctx, database, deploymentID, topologyStatus)
}

// transitionKey holds the pending desired-state transition for this
// deployment.
const transitionKey = "transition"

// transition is the pending desired-state change. Only the most recently
// written nonce is live; ChangeDesiredState discards the rest.
type transition struct {
	Nonce string
	To    hydrav1.DeploymentDesiredState
}

// applyDesiredStateNow is what same-key handlers (stop, wake) use instead of
// ScheduleDesiredStateChange: supersede any pending transition, then apply
// the new state before returning.
//
// The order is load-bearing, and the shared object is what makes it airtight:
// this handler holds the deployment's one inbox, so nothing runs between the
// clear and the apply, and a pending delayed stop later finds no transition
// record and no-ops instead of stopping a deployment a user just woke.
func (w *Workflow) applyDesiredStateNow(ctx restate.ObjectContext, deploymentID string, state hydrav1.DeploymentDesiredState) error {
	restate.Clear(ctx, transitionKey)
	return w.applyDesiredStateGuarded(ctx, deploymentID, state)
}

// applyDesiredStateGuarded writes the desired state under the invariant that
// the deployment currently serving its app never changes state. Guard and
// write share one transaction so a concurrent promote cannot slip between
// them. A row deleted by an environment cascade is a silent no-op.
func (w *Workflow) applyDesiredStateGuarded(ctx restate.ObjectContext, deploymentID string, state hydrav1.DeploymentDesiredState) error {
	var desiredState mysqltype.DeploymentsDesiredState
	var topologyDesiredStatus db.DeploymentTopologyDesiredStatus

	switch state {
	case hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_RUNNING:
		desiredState = mysqltype.DeploymentsDesiredStateRunning
		topologyDesiredStatus = db.DeploymentTopologyDesiredStatusRunning
	case hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED:
		desiredState = mysqltype.DeploymentsDesiredStateStopped
		topologyDesiredStatus = db.DeploymentTopologyDesiredStatusStopped
	case hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_UNSPECIFIED:
		return restate.TerminalErrorf("invalid state: %s", state)
	default:
		return restate.TerminalErrorf("unhandled state: %s", state)
	}

	applied, err := restate.Run(ctx, func(runCtx restate.RunContext) (bool, error) {
		return db.TxWithResult(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) (bool, error) {
			deployment, err := db.NewQueries(tx).FindDeploymentById(txCtx, deploymentID)
			if err != nil {
				if db.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}
			app, err := db.NewQueries(tx).FindAppById(txCtx, deployment.AppID)
			if err != nil {
				if db.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}

			if app.CurrentDeploymentID.Valid && app.CurrentDeploymentID.String == deploymentID {
				return false, restate.TerminalErrorf("not allowed to modify the current deployment")
			}

			err = db.NewQueries(tx).UpdateDeploymentDesiredState(txCtx, db.UpdateDeploymentDesiredStateParams{
				ID:           deploymentID,
				DesiredState: desiredState,
				UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
			if err != nil {
				return false, err
			}

			return true, nil
		})
	}, restate.WithName("updating desired state"))
	if err != nil {
		return err
	}
	if !applied {
		return nil
	}

	return applyTopologyDesiredStatus(ctx, w.db, deploymentID, topologyDesiredStatus)
}

// applyTopologyDesiredStatus writes the status to every region's topology row
// and inserts a deployment_changes row per region so WatchDeploymentChanges
// picks the change up.
func applyTopologyDesiredStatus(ctx restate.ObjectContext, database db.Database, deploymentID string, topologyStatus db.DeploymentTopologyDesiredStatus) error {
	regions, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.Region, error) {
		return database.FindDeploymentRegions(runCtx, deploymentID)
	}, restate.WithName("find deployment regions"))
	if err != nil {
		return fmt.Errorf("failed to find deployment regions: %w", err)
	}

	for _, region := range regions {
		err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return db.Tx(runCtx, database.RW(), func(txCtx context.Context, tx db.DBTX) error {
				err := db.NewQueries(tx).UpdateDeploymentTopologyDesiredStatus(txCtx, db.UpdateDeploymentTopologyDesiredStatusParams{
					DesiredStatus: topologyStatus,
					UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
					DeploymentID:  deploymentID,
					RegionID:      region.ID,
				})
				if err != nil {
					return err
				}
				return db.NewQueries(tx).InsertDeploymentChange(txCtx, db.InsertDeploymentChangeParams{
					ResourceType: db.DeploymentChangesResourceTypeDeploymentTopology,
					ResourceID:   deploymentID,
					RegionID:     region.ID,
					CreatedAt:    time.Now().UnixMilli(),
				})
			})
		}, restate.WithName(fmt.Sprintf("updating topology desired status in %s", region.ID)))
		if err != nil {
			return fmt.Errorf("failed to update topology desired status in %s: %w", region.ID, err)
		}
	}

	return nil
}
