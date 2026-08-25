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
			cli.String("role", "The role ID or name to permanently delete.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Permissions.DeleteRole, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2PermissionsDeleteRoleResponseBody)
			}

			res, err := client.Permissions.DeleteRole(ctx, components.V2PermissionsDeleteRoleRequestBody{
				Role: cmd.String("role"),
			})
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2PermissionsDeleteRoleResponseBody)
		},
	}
}
