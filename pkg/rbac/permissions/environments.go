package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreateEnvironment authorizes creating an environment resource.
//
// Valid resource: urn.Environment.
type CreateEnvironment struct{}

func (CreateEnvironment) ActionFor(urn.Environment) {}
func (CreateEnvironment) String() string            { return "create_environment" }

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

// CreateVariable authorizes creating a variable resource.
//
// Valid resource: urn.Variable.
type CreateVariable struct{}

func (CreateVariable) ActionFor(urn.Variable) {}
func (CreateVariable) String() string         { return "create_variable" }

// CreateVariables authorizes creating or replacing environment variables.
type CreateVariables struct{}

func (CreateVariables) ActionFor(urn.Environment) {}
func (CreateVariables) String() string            { return "create_variables" }

// DeleteVariables authorizes deleting environment variables.
type DeleteVariables struct{}

func (DeleteVariables) ActionFor(urn.Environment) {}
func (DeleteVariables) String() string            { return "delete_variables" }

// ReadVariables authorizes reading an environment's variable collection.
type ReadVariables struct{}

func (ReadVariables) ActionFor(urn.Environment) {}
func (ReadVariables) String() string            { return "read_variables" }

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
