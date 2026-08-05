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
// Valid resource: urn.Deployment.
type CreateDeployment struct{}

func (CreateDeployment) ActionFor(urn.Deployment) {}
func (CreateDeployment) String() string           { return "create_deployment" }

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

// ReadVariables authorizes reading an environment's variable collection.
type ReadVariables struct{}

func (ReadVariables) ActionFor(urn.Environment) {}
func (ReadVariables) String() string            { return "read_variables" }

// SetVariables authorizes replacing or upserting environment variables.
type SetVariables struct{}

func (SetVariables) ActionFor(urn.Environment) {}
func (SetVariables) String() string            { return "set_variables" }

// RemoveVariables authorizes removing environment variables.
type RemoveVariables struct{}

func (RemoveVariables) ActionFor(urn.Environment) {}
func (RemoveVariables) String() string            { return "remove_variables" }
