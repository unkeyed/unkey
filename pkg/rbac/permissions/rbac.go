package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreatePermission authorizes creating permissions in a project.
//
// Valid resource: urn.Project.
type CreatePermission struct{}

func (CreatePermission) ActionFor(urn.Project) {}
func (CreatePermission) String() string        { return "create_permission" }
