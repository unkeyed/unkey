package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const transitionKey = "transition"

const (
	pinnedRetryDelayMin = time.Minute
	pinnedRetryDelayMax = 15 * time.Minute
)

// transition is the Restate-persisted state for a pending desired state change.
// Only the most recently written transition is considered active; older ones are
// identified and discarded by nonce mismatch in ChangeDesiredState.
type transition struct {
	Nonce            string
	To               hydrav1.DeploymentDesiredState
	DeferWhilePinned bool
	PinnedRetries    int
}

// ScheduleDesiredStateChange records a future desired state transition for this
// deployment. It generates a unique nonce, persists the transition in Restate
// state, and sends a delayed ChangeDesiredState call to itself. If called again
// before the delay elapses, the new nonce overwrites the old one, causing the
// previous delayed call to no-op on nonce mismatch.
func (v *VirtualObject) ScheduleDesiredStateChange(ctx restate.ObjectContext, req *hydrav1.ScheduleDesiredStateChangeRequest) (*hydrav1.ScheduleDesiredStateChangeResponse, error) {
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
		Nonce:            nonce,
		To:               req.GetState(),
		DeferWhilePinned: req.GetDeferWhilePinned(),
		PinnedRetries:    0,
	}

	restate.Set(ctx, transitionKey, &t)

	delay := time.Duration(req.GetDelayMillis()) * time.Millisecond

	options := []restate.SendOption{}
	if delay > 0 {
		options = append(options, restate.WithDelay(delay))
	}

	hydrav1.NewDeploymentServiceClient(ctx, restate.Key(ctx)).ChangeDesiredState().Send(&hydrav1.ChangeDesiredStateRequest{
		Nonce: nonce,
		State: req.GetState(),
	}, options...)

	return &hydrav1.ScheduleDesiredStateChangeResponse{}, nil
}

// ChangeDesiredState applies a previously scheduled desired state transition to
// the database. It validates the request nonce against the stored transition:
// if no transition exists (already applied and cleared) or the nonce mismatches
// (a newer schedule has superseded this one), the call returns successfully
// without making any changes. If the deployment or app row no longer exists,
// typically because an environment delete removed it while a delayed transition
// was still pending, the call also no-ops. On match, it applies the state and
// clears the stored transition.
func (v *VirtualObject) ChangeDesiredState(ctx restate.ObjectContext, req *hydrav1.ChangeDesiredStateRequest) (*hydrav1.ChangeDesiredStateResponse, error) {
	deploymentID := restate.Key(ctx)

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

	deferred, changeErr := v.setDesiredState(ctx, deploymentID, req.GetState(), t.DeferWhilePinned)
	if changeErr != nil {
		if te := restate.AsTerminalError(changeErr); te == nil || te.Code() != 404 {
			return nil, changeErr
		}
	}
	if deferred {
		delay := pinnedRetryDelay(t.PinnedRetries)
		logger.Info("deployment stop deferred because an app binding pins it",
			"deployment_id", deploymentID, "deferrals", t.PinnedRetries+1, "retry_in", delay.String())
		t.PinnedRetries++
		restate.Set(ctx, transitionKey, t)
		hydrav1.NewDeploymentServiceClient(ctx, deploymentID).ChangeDesiredState().Send(req, restate.WithDelay(delay))
		return &hydrav1.ChangeDesiredStateResponse{}, nil
	}

	restate.Clear(ctx, transitionKey)
	return &hydrav1.ChangeDesiredStateResponse{}, nil
}

func pinnedRetryDelay(retries int) time.Duration {
	delay := pinnedRetryDelayMin
	for range retries {
		if delay >= pinnedRetryDelayMax/2 {
			return pinnedRetryDelayMax
		}
		delay *= 2
	}
	return delay
}

func (v *VirtualObject) setDesiredState(ctx restate.ObjectContext, deploymentID string, state hydrav1.DeploymentDesiredState, deferWhilePinned bool) (bool, error) {
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
		return false, restate.TerminalErrorf("invalid state: %s", state)
	default:
		return false, restate.TerminalErrorf("unhandled state: %s", state)
	}

	deferred, err := restate.Run(ctx, func(runCtx restate.RunContext) (bool, error) {
		return db.TxWithResult(runCtx, v.db.RW(), func(txCtx context.Context, tx db.DBTX) (bool, error) {
			queries := db.NewQueries(tx)
			deployment, err := queries.LockDeploymentWithApp(txCtx, deploymentID)
			if err != nil {
				if db.IsNotFound(err) {
					return false, restate.ToTerminalError(fmt.Errorf("deployment not found"), restate.WithErrorCode(404))
				}
				return false, err
			}

			pinned, err := queries.ExistsAppBindingPinningDeployment(txCtx, sql.NullString{String: deploymentID, Valid: true})
			if err != nil {
				return false, err
			}
			if pinned && desiredState == mysqltype.DeploymentsDesiredStateStopped && deferWhilePinned {
				return true, nil
			}

			if deployment.CurrentDeploymentID.Valid && deployment.CurrentDeploymentID.String == deploymentID {
				return false, restate.TerminalErrorf("not allowed to modify the current deployment")
			}

			err = queries.UpdateDeploymentDesiredState(txCtx, db.UpdateDeploymentDesiredStateParams{
				ID:           deploymentID,
				DesiredState: desiredState,
				UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
			return false, err
		})
	}, restate.WithName("updating desired state"))
	if err != nil {
		return false, err
	}
	if deferred {
		return true, nil
	}

	return false, applyTopologyDesiredStatus(ctx, v.db, deploymentID, topologyDesiredStatus)
}

// ApplyDesiredState updates the deployment and its topology in each region.
// It does not check whether this is the app's current deployment. Resume needs
// this so it can start a suspended deployment before making it current.
// Use ChangeDesiredState when the update needs that check.
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

// applyTopologyDesiredStatus updates the deployment's topology in each region.
func applyTopologyDesiredStatus(ctx restate.ObjectContext, database db.Database, deploymentID string, topologyStatus db.DeploymentTopologyDesiredStatus) error {
	regions, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.Region, error) {
		return database.FindDeploymentRegions(runCtx, deploymentID)
	}, restate.WithName("find deployment regions"))
	if err != nil {
		return fmt.Errorf("failed to find deployment regions: %w", err)
	}

	for _, region := range regions {
		err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return database.UpdateDeploymentTopologyDesiredStatus(runCtx, db.UpdateDeploymentTopologyDesiredStatusParams{
				DesiredStatus: topologyStatus,
				UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				DeploymentID:  deploymentID,
				RegionID:      region.ID,
			})
		}, restate.WithName(fmt.Sprintf("updating topology desired status in %s", region.ID)))
		if err != nil {
			return fmt.Errorf("failed to update topology desired status in %s: %w", region.ID, err)
		}
	}

	return nil
}
