package permissions

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
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
			util.YesFlag(),
			cli.String("permission", "The permission ID or slug to permanently delete.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			req, err := util.RequestBody(cmd, components.V2PermissionsDeletePermissionRequestBody{Permission: cmd.String("permission")})
			if err != nil {
				return err
			}
			if err := util.Confirm(cmd, fmt.Sprintf("Delete permission %q from the workspace and all keys and roles?", req.Permission)); err != nil {
				return err
			}

			res, err := client.Permissions.DeletePermission(ctx, req)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2PermissionsDeletePermissionResponseBody)
		},
	}
}
