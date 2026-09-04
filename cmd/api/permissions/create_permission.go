package permissions

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func createPermissionCmd() *cli.Command {
	return &cli.Command{
		Name:  "create-permission",
		Usage: "Create a new permission to define specific actions or capabilities in your RBAC system",
		Description: `Create a new permission to define specific actions or capabilities in your RBAC system. Permissions can be assigned directly to API keys or included in roles.

Use hierarchical naming patterns like documents.read, admin.users.delete, or billing.invoices.create for clear organization.

Important: Permission slugs must be unique within the default project. Once created, permissions are immediately available for assignment.

Required permissions:
- rbac.*.create_permission

For full documentation, see https://www.unkey.com/docs/api-reference/v2/permissions/create-permission` + util.Disclaimer,
		Examples: []string{
			"unkey api permissions create-permission --name=users.read --slug=users-read",
			`unkey api permissions create-permission --name=billing.write --slug=billing-write --description="Grants write access to billing resources"`,
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("name", "Human-readable name describing the permission's purpose.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("slug", "URL-safe identifier that is unique within the default project.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("description", "Detailed documentation of what this permission grants access to.", cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Permissions.CreatePermission, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2PermissionsCreatePermissionResponseBody)
			}

			req := components.V2PermissionsCreatePermissionRequestBody{
				Name:        cmd.String("name"),
				Slug:        cmd.String("slug"),
				Description: nil,
			}

			if v := cmd.String("description"); v != "" {
				req.Description = &v
			}

			res, err := client.Permissions.CreatePermission(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2PermissionsCreatePermissionResponseBody)
		},
	}
}
