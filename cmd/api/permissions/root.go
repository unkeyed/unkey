package permissions

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd groups all permissions.* subcommands.
func Cmd() *cli.Command {
	return &cli.Command{
		Name:        "permissions",
		Usage:       "Manage permissions and roles",
		Description: "Create, read, update, and delete permissions and roles for RBAC." + util.Disclaimer,
		Commands: []*cli.Command{
			createPermissionCmd(),
			deletePermissionCmd(),
			getPermissionCmd(),
			listPermissionsCmd(),
			updatePermissionCmd(),
			createRoleCmd(),
			deleteRoleCmd(),
			getRoleCmd(),
			listRolesCmd(),
			setRolePermissionsCmd(),
			updateRoleCmd(),
		},
	}
}
