package permissions

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/sdks/api/go/v3/optionalnullable"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func updatePermissionCmd() *cli.Command {
	return &cli.Command{
		Name:  "update-permission",
		Usage: "Update the name, slug, or description of a permission, identified by its ID or slug",
		Description: `Update the name, slug, or description of a permission, identified by its ID or slug.

Omitted fields keep their current values. A request with no fields to update returns the stored permission without a write.

Important: Keys and roles reference the permission by ID, so they keep the permission after a slug change. Key verification can return the old slug for a short time while caches refresh. A slug that another permission in your workspace already uses returns 409.

Required permissions:
- unkey:v1:<workspace_id>:projects/<project_id>/rbac/permissions/<permission_id>#write

For full documentation, see https://www.unkey.com/docs/api-management/cli/permissions/update-permission` + util.Disclaimer,
		Examples: []string{
			`unkey api permissions update-permission --permission=documents-read --name="Read documents" --slug=documents.read`,
			"unkey api permissions update-permission --permission=documents.read --description=",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("permission", "The permission ID or current slug to update.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("name", "New human-readable name for the permission.", cli.MutuallyExclusive("body")),
			cli.String("slug", "New permission slug, unique within your workspace.", cli.MutuallyExclusive("body")),
			cli.String("description", "New permission description; pass an empty string to remove it.", cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.Permissions.UpdatePermission, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2PermissionsUpdatePermissionResponseBody)
			}

			req := components.V2PermissionsUpdatePermissionRequestBody{
				Permission:  cmd.String("permission"),
				Name:        nil,
				Slug:        nil,
				Description: nil,
			}
			if cmd.FlagIsSet("name") {
				req.Name = new(cmd.String("name"))
			}
			if cmd.FlagIsSet("slug") {
				req.Slug = new(cmd.String("slug"))
			}
			if cmd.FlagIsSet("description") {
				req.Description = optionalnullable.From(new(cmd.String("description")))
			}

			res, err := client.Permissions.UpdatePermission(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2PermissionsUpdatePermissionResponseBody)
		},
	}
}
