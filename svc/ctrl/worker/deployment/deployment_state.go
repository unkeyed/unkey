package deployment

import (
	"context"
	"database/sql"
	"fmt"
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
		To:    req.GetState(),
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
// without making any changes. On match, it maps the protobuf state enum to the
// database representation, updates the deployment's desired state and all
// topology entries, and clears the stored transition.
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

	var desiredState db.DeploymentsDesiredState
	var topologyDesiredStatus db.DeploymentTopologyDesiredStatus
	switch req.GetState() {
	case hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_RUNNING:
		desiredState = db.DeploymentsDesiredStateRunning
		topologyDesiredStatus = db.DeploymentTopologyDesiredStatusStarted
	case hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STANDBY:
		desiredState = db.DeploymentsDesiredStateStandby
		topologyDesiredStatus = db.DeploymentTopologyDesiredStatusStopped
	case hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_ARCHIVED:
		desiredState = db.DeploymentsDesiredStateArchived
		topologyDesiredStatus = db.DeploymentTopologyDesiredStatusStopped
	case hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_UNSPECIFIED:
		return nil, restate.TerminalErrorf("invalid state: %s", req.GetState())
	default:
		return nil, restate.TerminalErrorf("unhandled state: %s", req.GetState())
	}

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {

		return db.Tx(runCtx, v.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
			deployment, err := db.Query.FindDeploymentById(txCtx, tx, deploymentID)
			if err != nil {
				return err
			}
			project, err := db.Query.FindProjectById(txCtx, tx, deployment.ProjectID)
			if err != nil {
				return err
			}

			if project.LiveDeploymentID.Valid && project.LiveDeploymentID.String == deploymentID {
				return restate.TerminalErrorf("not allowed to modify the current live deployment")
			}

			err = db.Query.UpdateDeploymentDesiredState(txCtx, tx, db.UpdateDeploymentDesiredStateParams{
				ID:           deploymentID,
				DesiredState: desiredState,
				UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
			if err != nil {
				return err
			}
			return nil

		})

	}, restate.WithName("updating desired state"))

	if err != nil {
		return nil, err
	}

	// Update all topology entries so WatchDeployments picks up the change.
	// Each region needs a new version from VersioningService.
	regions, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]string, error) {
		return db.Query.FindDeploymentRegions(runCtx, v.db.RO(), deploymentID)
	}, restate.WithName("find deployment regions"))
	if err != nil {
		return nil, fmt.Errorf("failed to find deployment regions: %w", err)
	}

	for _, region := range regions {
		versionResp, err := hydrav1.NewVersioningServiceClient(ctx, region).NextVersion().Request(&hydrav1.NextVersionRequest{})
		if err != nil {
			return nil, fmt.Errorf("failed to get next version for region %s: %w", region, err)
		}

		err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return db.Query.UpdateDeploymentTopologyDesiredStatus(runCtx, v.db.RW(), db.UpdateDeploymentTopologyDesiredStatusParams{
				DesiredStatus: topologyDesiredStatus,
				Version:       versionResp.GetVersion(),
				UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				DeploymentID:  deploymentID,
				Region:        region,
			})
		}, restate.WithName(fmt.Sprintf("updating topology desired status in %s", region)))
		if err != nil {
			return nil, fmt.Errorf("failed to update topology desired status in %s: %w", region, err)
		}
	}

	restate.Clear(ctx, transitionKey)

	return &hydrav1.ChangeDesiredStateResponse{}, nil
}
