package permissions

import (
	"context"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/sdks/api/go/v3/optionalnullable"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func updateRoleCmd() *cli.Command {
	return &cli.Command{
		Name:  "update-role",
		Usage: "Update the name or description of a role, identified by its ID or name",
		Description: `Update the name or description of a role, identified by its ID or name.

Omitted fields keep their current values. A request with no fields to update returns the stored role without a write.

This endpoint does not change the role's permissions or keys. Use permissions.setRolePermissions, keys.addRoles, and keys.removeRoles for those.

Important: Keys keep the role after a rename. Requests that name the role, such as keys.addRoles, must use the new name. Key verification can return the old name for a short time while caches refresh. A name that another role in your workspace already uses returns 409.

Required permissions:
- unkey:v1:<workspace_id>:projects/<project_id>/rbac/roles/<role_id>#write

For full documentation, see https://www.unkey.com/docs/api-management/cli/permissions/update-role` + util.Disclaimer,
		Examples: []string{
			"unkey api permissions update-role --role=role_1234abcd --name=support.readonly",
			"unkey api permissions update-role --role=support.readonly --description=",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("role", "The role ID or current name to update.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("name", "New role name, unique within your workspace.", cli.MutuallyExclusive("body")),
			cli.String("description", "New role description; pass an empty string to remove it.", cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.Permissions.UpdateRole, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2PermissionsUpdateRoleResponseBody)
			}

			req := components.V2PermissionsUpdateRoleRequestBody{
				Role:        cmd.String("role"),
				Name:        nil,
				Description: nil,
			}
			if cmd.FlagIsSet("name") {
				req.Name = new(cmd.String("name"))
			}
			if cmd.FlagIsSet("description") {
				req.Description = optionalnullable.From(new(cmd.String("description")))
			}

			res, err := client.Permissions.UpdateRole(ctx, req)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2PermissionsUpdateRoleResponseBody)
		},
	}
}
