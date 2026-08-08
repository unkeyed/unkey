package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateDeployment authorizes creating a deployment resource.
type CreateDeployment struct{}

func (CreateDeployment) ActionFor(urn.Deployment) {}
func (CreateDeployment) String() string           { return "create_deployment" }

// ReadDeployment authorizes reading a deployment.
type ReadDeployment struct{}

func (ReadDeployment) ActionFor(urn.Deployment) {}
func (ReadDeployment) String() string           { return "read_deployment" }

// StartDeployment authorizes starting a deployment.
type StartDeployment struct{}

func (StartDeployment) ActionFor(urn.Deployment) {}
func (StartDeployment) String() string           { return "start_deployment" }

// StopDeployment authorizes stopping a deployment.
type StopDeployment struct{}

func (StopDeployment) ActionFor(urn.Deployment) {}
func (StopDeployment) String() string           { return "stop_deployment" }

// PromoteDeployment authorizes changing an environment's current deployment.
type PromoteDeployment struct{}

func (PromoteDeployment) ActionFor(urn.Environment) {}
func (PromoteDeployment) String() string            { return "promote_deployment" }

// RollbackDeployment authorizes restoring an environment's previous deployment.
type RollbackDeployment struct{}

func (RollbackDeployment) ActionFor(urn.Environment) {}
func (RollbackDeployment) String() string            { return "rollback_deployment" }
