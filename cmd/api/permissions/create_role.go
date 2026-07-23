package permissions

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func createRoleCmd() *cli.Command {
	return &cli.Command{
		Name:  "create-role",
		Usage: "Create a new role to group related permissions for easier management",
		Description: `Create a new role to group related permissions for easier management. Roles enable consistent permission assignment across multiple API keys.

Important: Role names must be unique within the workspace. Once created, roles are immediately available for assignment.

Required permissions:
- rbac.*.create_role

For full documentation, see https://www.unkey.com/docs/api-reference/v2/permissions/create-role` + util.Disclaimer,
		Examples: []string{
			`unkey api permissions create-role --name=content.editor --description="Can read and write content"`,
			"unkey api permissions create-role --name=api.reader",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("name", "Unique name for the role within your workspace.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("description", "Documentation of what this role encompasses and what access it grants.", cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Permissions.CreateRole, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2PermissionsCreateRoleResponseBody)
			}

			req := components.V2PermissionsCreateRoleRequestBody{
				Name:        cmd.String("name"),
				Description: nil,
			}

			if v := cmd.String("description"); v != "" {
				req.Description = &v
			}

			res, err := client.Permissions.CreateRole(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2PermissionsCreateRoleResponseBody)
		},
	}
}
