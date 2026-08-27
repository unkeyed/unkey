package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadRole authorizes reading a role.
type ReadRole struct{}

func (ReadRole) ActionFor(urn.Role) {}
func (ReadRole) String() string     { return "read_role" }

// WriteRole authorizes creating or updating a role and its permission assignments.
type WriteRole struct{}

func (WriteRole) ActionFor(urn.Role) {}
func (WriteRole) String() string     { return "write_role" }

// DeleteRole authorizes deleting a role.
type DeleteRole struct{}

func (DeleteRole) ActionFor(urn.Role) {}
func (DeleteRole) String() string     { return "delete_role" }
