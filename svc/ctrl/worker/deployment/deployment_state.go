package deployment

import (
	"database/sql"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

const transitionKey = "transition"

// transition is the Restate-persisted state for a pending desired state change.
// Only the most recently written transition is considered active; older ones are
// identified and discarded by nonce mismatch in ChangeDesiredState.
type transition struct {
	Nonce string
	After int64
	To    hydrav1.DeploymentDesiredState
}

// ScheduleDesiredStateChange records a future desired state transition for this
// deployment. It generates a unique nonce, persists the transition in Restate
// state, and sends a delayed ChangeDesiredState call to itself. If called again
// before the delay elapses, the new nonce overwrites the old one, causing the
// previous delayed call to no-op on nonce mismatch.
func (v *VirtualObject) ScheduleDesiredStateChange(ctx restate.ObjectContext, req *hydrav1.ScheduleDesiredStateChangeRequest) (*hydrav1.ScheduleDesiredStateChangeResponse, error) {

	nonce := restate.UUID(ctx).String()

	t := transition{
		Nonce: nonce,
		After: req.GetAfter(),
		To:    req.GetState(),
	}

	restate.Set(ctx, transitionKey, &t)

	hydrav1.NewDeploymentServiceClient(ctx, restate.Key(ctx)).ChangeDesiredState().Send(&hydrav1.ChangeDesiredStateRequest{
		Nonce: nonce,
		State: req.GetState(),
	}, restate.WithDelay(time.Until(time.UnixMilli(req.GetAfter()))))

	return &hydrav1.ScheduleDesiredStateChangeResponse{}, nil
}

// ChangeDesiredState applies a previously scheduled desired state transition to
// the database. It first validates the request nonce against the stored
// transition: a mismatch means a newer schedule has superseded this one, so
// the call returns successfully without making any changes. On match, it maps
// the protobuf state enum to the database representation and updates the
// deployment's desired state.
func (v *VirtualObject) ChangeDesiredState(ctx restate.ObjectContext, req *hydrav1.ChangeDesiredStateRequest) (*hydrav1.ChangeDesiredStateResponse, error) {

	t, err := restate.Get[*transition](ctx, transitionKey)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, restate.TerminalErrorf("no state found")
	}
	if t.Nonce != req.GetNonce() {
		// This is a noop, since the request is outdated
		return &hydrav1.ChangeDesiredStateResponse{}, nil
	}

	var desiredState db.DeploymentsDesiredState
	switch req.GetState() {
	case hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_RUNNING:
		desiredState = db.DeploymentsDesiredStateRunning
	case hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STANDBY:
		desiredState = db.DeploymentsDesiredStateStandby
	case hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_ARCHIVED:
		desiredState = db.DeploymentsDesiredStateArchived
	case hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_UNSPECIFIED:
		return nil, restate.TerminalErrorf("invalid state: %s", req.GetState())
	}

	// actually do state change here
	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.UpdateDeploymentDesiredState(runCtx, v.db.RW(), db.UpdateDeploymentDesiredStateParams{
			ID:           restate.Key(ctx),
			DesiredState: desiredState,
			UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	})
	if err != nil {
		return nil, err
	}

	return &hydrav1.ChangeDesiredStateResponse{}, nil
}
