package permissions

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func getRoleCmd() *cli.Command {
	return &cli.Command{
		Name:  "get-role",
		Usage: "Retrieve details about a specific role including its assigned permissions",
		Description: `Retrieve details about a specific role including its assigned permissions.

Required permissions:
- rbac.*.read_role

For full documentation, see https://www.unkey.com/docs/api-reference/v2/permissions/get-role` + util.Disclaimer,
		Examples: []string{
			"unkey api permissions get-role --role=role_1234567890abcdef",
			`unkey api permissions get-role --role=my-role-name`,
		},
		Flags: []cli.Flag{
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("role", "Role ID (starting with role_) or role name to retrieve.", cli.Required()),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			start := time.Now()
			res, err := client.Permissions.GetRole(ctx, components.V2PermissionsGetRoleRequestBody{
				Role: cmd.String("role"),
			})
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2PermissionsGetRoleResponseBody, time.Since(start))
		},
	}
}
