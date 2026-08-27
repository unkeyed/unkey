package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadEnvironment authorizes reading an environment.
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
