package permissions

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func deletePermissionCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete-permission",
		Usage: "Remove a permission from your workspace",
		Description: `Remove a permission from your workspace. This also removes the permission from all API keys and roles.

Important: This operation cannot be undone and immediately affects all API keys and roles that had this permission assigned.

Required permissions:
- rbac.*.delete_permission

For full documentation, see https://www.unkey.com/docs/api-reference/v2/permissions/delete-permission` + util.Disclaimer,
		Examples: []string{
			"unkey api permissions delete-permission --permission=perm_1234567890abcdef",
			"unkey api permissions delete-permission --permission=documents.read",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("permission", "The permission ID or slug to permanently delete.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Permissions.DeletePermission, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2PermissionsDeletePermissionResponseBody)
			}

			res, err := client.Permissions.DeletePermission(ctx, components.V2PermissionsDeletePermissionRequestBody{
				Permission: cmd.String("permission"),
			})
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2PermissionsDeletePermissionResponseBody)
		},
	}
}
