package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadEnvironmentVariable authorizes reading an environment variable.
type ReadEnvironmentVariable struct{}

func (ReadEnvironmentVariable) ActionFor(urn.EnvironmentVariable) {}
func (ReadEnvironmentVariable) String() string                    { return "read_environment_variable" }

// WriteEnvironmentVariable authorizes creating or updating an environment variable.
type WriteEnvironmentVariable struct{}

func (WriteEnvironmentVariable) ActionFor(urn.EnvironmentVariable) {}
func (WriteEnvironmentVariable) String() string                    { return "write_environment_variable" }

// DeleteEnvironmentVariable authorizes deleting an environment variable.
type DeleteEnvironmentVariable struct{}

func (DeleteEnvironmentVariable) ActionFor(urn.EnvironmentVariable) {}
func (DeleteEnvironmentVariable) String() string                    { return "delete_environment_variable" }
