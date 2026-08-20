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

// ReadPolicies authorizes reading a specific environment's gateway policies.
//
// Valid resource: urn.Environment.
type ReadPolicies struct{}

func (ReadPolicies) ActionFor(urn.Environment) {}
func (ReadPolicies) String() string            { return "read_policies" }

// WritePolicies authorizes replacing a specific environment's entire gateway
// policy list or updating a single policy in place.
//
// Valid resource: urn.Environment.
type WritePolicies struct{}

func (WritePolicies) ActionFor(urn.Environment) {}
func (WritePolicies) String() string            { return "write_policies" }
