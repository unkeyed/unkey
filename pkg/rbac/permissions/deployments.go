package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateDeployment authorizes creating deployments.
//
// Valid resource: urn.Deployment. Grants use a wildcard deployment id because
// the deployment does not exist when the request is authorized.
type CreateDeployment struct{}

func (CreateDeployment) ActionFor(urn.Deployment) {}
func (CreateDeployment) String() string           { return "create_deployment" }

// ReadDeployment authorizes reading a deployment.
//
// Valid resource: urn.Deployment.
type ReadDeployment struct{}

func (ReadDeployment) ActionFor(urn.Deployment) {}
func (ReadDeployment) String() string           { return "read_deployment" }

// StopDeployment authorizes stopping a running deployment.
//
// Valid resource: urn.Deployment.
type StopDeployment struct{}

func (StopDeployment) ActionFor(urn.Deployment) {}
func (StopDeployment) String() string           { return "stop_deployment" }

// StartDeployment authorizes starting a stopped deployment.
//
// Valid resource: urn.Deployment.
type StartDeployment struct{}

func (StartDeployment) ActionFor(urn.Deployment) {}
func (StartDeployment) String() string           { return "start_deployment" }

// RollbackDeployment authorizes rolling live traffic back to a previous deployment.
//
// Valid resource: urn.Deployment, the rollback target.
type RollbackDeployment struct{}

func (RollbackDeployment) ActionFor(urn.Deployment) {}
func (RollbackDeployment) String() string           { return "rollback_deployment" }

// PromoteDeployment authorizes promoting a deployment to live.
//
// Valid resource: urn.Deployment.
type PromoteDeployment struct{}

func (PromoteDeployment) ActionFor(urn.Deployment) {}
func (PromoteDeployment) String() string           { return "promote_deployment" }
