package permissions

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func setRolePermissionsCmd() *cli.Command {
	return &cli.Command{
		Name:  "set-role-permissions",
		Usage: "Atomically replace all permissions directly assigned to a role.",
		Description: `Atomically replaces all permissions directly assigned to a role. An empty permissions array removes every permission from the role. Permissions that do not exist are created when the caller has permission to create them.

Required Permissions

Your root key must have:
- rbac.*.add_permission_to_role
- rbac.*.remove_permission_from_role

When any requested permission slug does not exist, it must also have:
- rbac.*.create_permission

For full documentation, see https://www.unkey.com/docs/platform/apis/features/authorization/roles-and-permissions` + util.Disclaimer,
		Examples: []string{
			"unkey api permissions set-role-permissions --role-id=role_1234abcd --permissions=documents.read,documents.write",
			"unkey api permissions set-role-permissions --role-id=role_1234abcd --permissions=",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("role-id", "Role ID or slug whose directly assigned permissions will be replaced.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.StringSlice("permissions", "Complete set of permission slugs to assign directly to the role.", cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Permissions.SetRolePermissions, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2PermissionsSetRolePermissionsResponseBody)
			}
			if !cmd.FlagIsSet("permissions") {
				return fmt.Errorf("required flag missing: permissions")
			}

			req := components.V2PermissionsSetRolePermissionsRequestBody{
				RoleID:      cmd.String("role-id"),
				Permissions: cmd.StringSlice("permissions"),
			}
			res, err := client.Permissions.SetRolePermissions(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2PermissionsSetRolePermissionsResponseBody)
		},
	}
}
