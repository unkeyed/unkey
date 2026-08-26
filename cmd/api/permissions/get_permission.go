package permissions

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func getPermissionCmd() *cli.Command {
	return &cli.Command{
		Name:  "get-permission",
		Usage: "Retrieve details about a specific permission",
		Description: `Retrieve details about a specific permission including its name, description, and metadata.

Required permissions:
- rbac.*.read_permission

For full documentation, see https://www.unkey.com/docs/api-reference/v2/permissions/get-permission` + util.Disclaimer,
		Examples: []string{
			"unkey api permissions get-permission --permission=perm_1234567890abcdef",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("permission", "The unique identifier of the permission to retrieve.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Permissions.GetPermission, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2PermissionsGetPermissionResponseBody)
			}

			res, err := client.Permissions.GetPermission(ctx, components.V2PermissionsGetPermissionRequestBody{
				Permission: cmd.String("permission"),
			})
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2PermissionsGetPermissionResponseBody)
		},
	}
}
