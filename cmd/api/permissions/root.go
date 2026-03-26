package permissions

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd groups all permissions.* subcommands.
var Cmd = &cli.Command{
	Name:        "permissions",
	Usage:       "Manage permissions and roles",
	Description: "Create, read, and delete permissions and roles for RBAC." + util.Disclaimer,
	Commands: []*cli.Command{
		createPermissionCmd,
		deletePermissionCmd,
		getPermissionCmd,
		listPermissionsCmd,
		createRoleCmd,
		deleteRoleCmd,
		getRoleCmd,
		listRolesCmd,
	},
}
