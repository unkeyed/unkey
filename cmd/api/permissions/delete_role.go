package permissions

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func deleteRoleCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete-role",
		Usage: "Remove a role from your workspace",
		Description: `Remove a role from your workspace. This also removes the role from all assigned API keys.

Important: This operation cannot be undone and immediately affects all API keys that had this role assigned.

Required permissions:
- rbac.*.delete_role

For full documentation, see https://www.unkey.com/docs/api-reference/v2/permissions/delete-role` + util.Disclaimer,
		Examples: []string{
			"unkey api permissions delete-role --role=role_dns_manager",
			"unkey api permissions delete-role --role=admin",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			util.YesFlag(),
			cli.String("role", "The role ID or name to permanently delete.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			req, err := util.RequestBody(cmd, components.V2PermissionsDeleteRoleRequestBody{Role: cmd.String("role")})
			if err != nil {
				return err
			}
			if err := util.Confirm(cmd, fmt.Sprintf("Delete role %q from the workspace and all assigned keys?", req.Role)); err != nil {
				return err
			}

			res, err := client.Permissions.DeleteRole(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2PermissionsDeleteRoleResponseBody)
		},
	}
}
