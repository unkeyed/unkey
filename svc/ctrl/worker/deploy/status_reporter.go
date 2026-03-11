package deploy

import restate "github.com/restatedev/sdk-go"

// deploymentStatusReporter reports deployment progress to an external system
// (e.g. GitHub Deployments API). Implementations must be safe to call even
// when the external system is unavailable — errors are logged, never propagated.
type deploymentStatusReporter interface {
	// Create registers the deployment with the external system and sets
	// an initial "pending" status.
	Create(ctx restate.ObjectSharedContext)

	// Report updates the deployment status (e.g. "success", "failure").
	Report(ctx restate.ObjectSharedContext, state string, description string)
}
