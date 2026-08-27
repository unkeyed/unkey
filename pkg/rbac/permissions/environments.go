package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadEnvironment authorizes reading a specific environment.
//
// Valid resource: urn.Environment.
type ReadEnvironment struct{}

func (ReadEnvironment) ActionFor(urn.Environment) {}
func (ReadEnvironment) String() string            { return "read_environment" }

// WriteEnvironment authorizes creating or updating an environment, including
// promotion and rollback.
type WriteEnvironment struct{}

func (WriteEnvironment) ActionFor(urn.Environment) {}
func (WriteEnvironment) String() string            { return "write_environment" }

// DeleteEnvironment authorizes deleting an environment.
type DeleteEnvironment struct{}

func (DeleteEnvironment) ActionFor(urn.Environment) {}
func (DeleteEnvironment) String() string            { return "delete_environment" }

// UpdateEnvironment authorizes updating a specific environment's settings.
//
// Valid resource: urn.Environment.
type UpdateEnvironment struct{}

func (UpdateEnvironment) ActionFor(urn.Environment) {}
func (UpdateEnvironment) String() string            { return "update_environment" }

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
