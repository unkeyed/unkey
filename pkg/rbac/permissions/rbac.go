package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreatePermission authorizes creating RBAC permission resources.
type CreatePermission struct{}

func (CreatePermission) ActionFor(urn.PermissionDefinition) {}
func (CreatePermission) String() string                     { return "create_permission" }

// ReadPermission authorizes reading an RBAC permission definition.
type ReadPermission struct{}

func (ReadPermission) ActionFor(urn.PermissionDefinition) {}
func (ReadPermission) String() string                     { return "read_permission" }

// DeletePermission authorizes deleting an RBAC permission definition.
type DeletePermission struct{}

func (DeletePermission) ActionFor(urn.PermissionDefinition) {}
func (DeletePermission) String() string                     { return "delete_permission" }

// CreateRole authorizes creating an RBAC role.
type CreateRole struct{}

func (CreateRole) ActionFor(urn.Role) {}
func (CreateRole) String() string     { return "create_role" }

// ReadRole authorizes reading an RBAC role.
type ReadRole struct{}

func (ReadRole) ActionFor(urn.Role) {}
func (ReadRole) String() string     { return "read_role" }

// DeleteRole authorizes deleting an RBAC role.
type DeleteRole struct{}

func (DeleteRole) ActionFor(urn.Role) {}
func (DeleteRole) String() string     { return "delete_role" }
