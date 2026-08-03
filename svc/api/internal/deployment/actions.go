package deployment

import (
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// availableActions returns the lifecycle operations that would succeed on a
// deployment in this state. It encodes the same preconditions the
// promote/rollback/stop/start handlers enforce, so a caller can pick a legal
// action instead of discovering the rules through 412s.
//
// promote and rollback are gated separately because the handlers differ: both
// need the app to already have a live deployment, but rollback rejects the
// current pointer outright while promote still allows it when the app is in a
// rolled-back state (promoting forward off a rollback).
func availableActions(in Input) []openapi.DeploymentAction {
	d := in.Deployment
	actions := []openapi.DeploymentAction{}

	// An empty slug means the environment could not be resolved; without it the
	// production gate is unknowable, so offer nothing rather than guess.
	if in.State.EnvironmentSlug == "" {
		return actions
	}

	ready := d.Status == mysqltype.DeploymentsStatusReady
	running := d.DesiredState == mysqltype.DeploymentsDesiredStateRunning

	if in.State.EnvironmentIsProduction {
		currentDeploymentID := in.State.AppCurrentDeploymentID.String
		hasLiveDeployment := currentDeploymentID != ""
		isCurrentDeployment := currentDeploymentID == d.ID
		if ready && running && hasLiveDeployment {
			// promote is illegal only when this is already the promoted-live
			// deployment (current pointer and not in a rolled-back state).
			if !isCurrentDeployment || in.State.AppIsRolledBack {
				actions = append(actions, openapi.DeploymentActionPromote)
			}
			// rollback is illegal when this is the current pointer, regardless of
			// rolled-back state.
			if !isCurrentDeployment {
				actions = append(actions, openapi.DeploymentActionRollback)
			}
		}
		return actions
	}

	if ready && running {
		actions = append(actions, openapi.DeploymentActionStop)
	}
	if d.Status == mysqltype.DeploymentsStatusStopped {
		actions = append(actions, openapi.DeploymentActionStart)
	}
	return actions
}
