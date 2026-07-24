package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreatePermission authorizes creating RBAC permissions in a project.
type CreatePermission struct{}

func (CreatePermission) ActionFor(urn.Project) {}
func (CreatePermission) String() string        { return "create_permission" }
