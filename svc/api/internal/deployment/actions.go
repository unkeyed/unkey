package deployment

import (
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// productionSlug is the environment whose current deployment serves production
// traffic. promote and rollback only apply there; stop and start only apply
// elsewhere. Mirrors the gate in the lifecycle handlers.
const productionSlug = "production"

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

	ready := d.Status == db.DeploymentsStatusReady
	running := d.DesiredState == db.DeploymentsDesiredStateRunning

	if in.State.EnvironmentSlug == productionSlug {
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
	if d.Status == db.DeploymentsStatusStopped {
		actions = append(actions, openapi.DeploymentActionStart)
	}
	return actions
}
