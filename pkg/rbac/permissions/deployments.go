package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadDeployment authorizes reading a deployment.
type ReadDeployment struct{}

func (ReadDeployment) ActionFor(urn.Deployment) {}
func (ReadDeployment) String() string           { return "read_deployment" }

// WriteDeployment authorizes creating, updating, starting, or stopping a deployment.
type WriteDeployment struct{}

func (WriteDeployment) ActionFor(urn.Deployment) {}
func (WriteDeployment) String() string           { return "write_deployment" }

// DeleteDeployment authorizes deleting a deployment.
type DeleteDeployment struct{}

func (DeleteDeployment) ActionFor(urn.Deployment) {}
func (DeleteDeployment) String() string           { return "delete_deployment" }
