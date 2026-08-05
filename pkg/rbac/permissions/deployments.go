package permissions

import "github.com/unkeyed/unkey/pkg/urn"

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

// PromoteDeployment authorizes promoting a deployment.
type PromoteDeployment struct{}

func (PromoteDeployment) ActionFor(urn.Deployment) {}
func (PromoteDeployment) String() string           { return "promote_deployment" }

// RollbackDeployment authorizes rolling back a deployment.
type RollbackDeployment struct{}

func (RollbackDeployment) ActionFor(urn.Deployment) {}
func (RollbackDeployment) String() string           { return "rollback_deployment" }
