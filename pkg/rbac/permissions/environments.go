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

// CreateDomain authorizes creating domains in an environment.
//
// Valid resource: urn.Environment.
type CreateDomain struct{}

func (CreateDomain) ActionFor(urn.Environment) {}
func (CreateDomain) String() string            { return "create_domain" }

// ReadDomain authorizes reading an environment's domains.
//
// Valid resource: urn.Environment.
type ReadDomain struct{}

func (ReadDomain) ActionFor(urn.Environment) {}
func (ReadDomain) String() string            { return "read_domain" }

// DeleteDomain authorizes removing a domain from an environment.
//
// Valid resource: urn.Environment.
type DeleteDomain struct{}

func (DeleteDomain) ActionFor(urn.Environment) {}
func (DeleteDomain) String() string            { return "delete_domain" }

// VerifyDomain authorizes restarting verification for an environment's domains.
//
// Valid resource: urn.Environment.
type VerifyDomain struct{}

func (VerifyDomain) ActionFor(urn.Environment) {}
func (VerifyDomain) String() string            { return "verify_domain" }

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
