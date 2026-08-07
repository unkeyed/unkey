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

// ReadEnvironmentVariables authorizes reading a specific environment's variables.
//
// Valid resource: urn.Environment.
type ReadEnvironmentVariables struct{}

func (ReadEnvironmentVariables) ActionFor(urn.Environment) {}
func (ReadEnvironmentVariables) String() string            { return "read_environment_variables" }

// SetEnvironmentVariables authorizes creating and overwriting a specific
// environment's variables.
//
// Valid resource: urn.Environment.
type SetEnvironmentVariables struct{}

func (SetEnvironmentVariables) ActionFor(urn.Environment) {}
func (SetEnvironmentVariables) String() string            { return "set_environment_variables" }

// RemoveEnvironmentVariables authorizes removing variables from a specific
// environment.
//
// Valid resource: urn.Environment.
type RemoveEnvironmentVariables struct{}

func (RemoveEnvironmentVariables) ActionFor(urn.Environment) {}
func (RemoveEnvironmentVariables) String() string            { return "remove_environment_variables" }
