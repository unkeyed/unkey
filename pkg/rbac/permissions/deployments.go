package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateDeployment authorizes creating deployments.
//
// Valid resource: urn.Deployment. Grants use a wildcard deployment id because
// the deployment does not exist when the request is authorized.
type CreateDeployment struct{}

func (CreateDeployment) ActionFor(urn.Deployment) {}
func (CreateDeployment) String() string           { return "create_deployment" }

// StopDeployment authorizes stopping a running deployment.
//
// Valid resource: urn.Deployment.
type StopDeployment struct{}

func (StopDeployment) ActionFor(urn.Deployment) {}
func (StopDeployment) String() string           { return "stop_deployment" }
