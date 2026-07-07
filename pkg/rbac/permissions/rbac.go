package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// CreatePermission authorizes creating RBAC permission resources.
type CreatePermission struct{}

func (CreatePermission) ActionFor(urn.V1) {}
func (CreatePermission) String() string   { return "create_permission" }
