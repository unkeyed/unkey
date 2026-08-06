package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadEnvironment authorizes reading a specific environment.
//
// Valid resource: urn.Environment.
type ReadEnvironment struct{}

func (ReadEnvironment) ActionFor(urn.Environment) {}
func (ReadEnvironment) String() string            { return "read_environment" }

// UpdateEnvironment authorizes updating a specific environment's settings.
//
// Valid resource: urn.Environment.
type UpdateEnvironment struct{}

func (UpdateEnvironment) ActionFor(urn.Environment) {}
func (UpdateEnvironment) String() string            { return "update_environment" }

// CreateDeployment authorizes creating deployments in an environment.
//
// Valid resource: urn.Environment.
type CreateDeployment struct{}

func (CreateDeployment) ActionFor(urn.Environment) {}
func (CreateDeployment) String() string            { return "create_deployment" }

// StopDeployment authorizes stopping a running deployment in an environment.
//
// Valid resource: urn.Environment.
type StopDeployment struct{}

func (StopDeployment) ActionFor(urn.Environment) {}
func (StopDeployment) String() string            { return "stop_deployment" }

// StartDeployment authorizes starting a stopped deployment in an environment.
//
// Valid resource: urn.Environment.
type StartDeployment struct{}

func (StartDeployment) ActionFor(urn.Environment) {}
func (StartDeployment) String() string            { return "start_deployment" }

// RollbackDeployment authorizes rolling live traffic in an environment back to a previous deployment.
//
// Valid resource: urn.Environment.
type RollbackDeployment struct{}

func (RollbackDeployment) ActionFor(urn.Environment) {}
func (RollbackDeployment) String() string            { return "rollback_deployment" }

// PromoteDeployment authorizes promoting a deployment in an environment to live.
//
// Valid resource: urn.Environment.
type PromoteDeployment struct{}

func (PromoteDeployment) ActionFor(urn.Environment) {}
func (PromoteDeployment) String() string            { return "promote_deployment" }

// CreateDomain authorizes creating domains in an environment.
//
// Valid resource: urn.Environment.
type CreateDomain struct{}

func (CreateDomain) ActionFor(urn.Environment) {}
func (CreateDomain) String() string            { return "create_domain" }

// CreateVariable authorizes creating variables in an environment.
//
// Valid resource: urn.Environment.
type CreateVariable struct{}

func (CreateVariable) ActionFor(urn.Environment) {}
func (CreateVariable) String() string            { return "create_variable" }
