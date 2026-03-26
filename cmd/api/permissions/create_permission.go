package permissions

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

var createPermissionCmd = &cli.Command{
	Name:  "create-permission",
	Usage: "Create a new permission to define specific actions or capabilities in your RBAC system",
	Description: `Create a new permission to define specific actions or capabilities in your RBAC system. Permissions can be assigned directly to API keys or included in roles.

Use hierarchical naming patterns like documents.read, admin.users.delete, or billing.invoices.create for clear organization.

Important: Permission names must be unique within the workspace. Once created, permissions are immediately available for assignment.

Required permissions:
- rbac.*.create_permission

For full documentation, see https://www.unkey.com/docs/api-reference/v2/permissions/create-permission` + util.Disclaimer,
	Examples: []string{
		"unkey api permissions create-permission --name=users.read --slug=users-read",
		`unkey api permissions create-permission --name=billing.write --slug=billing-write --description="Grants write access to billing resources"`,
	},
	Flags: []cli.Flag{
		util.RootKeyFlag(),
		util.APIURLFlag(),
		util.ConfigFlag(),
		util.OutputFlag(),
		cli.String("name", "Human-readable name describing the permission's purpose.", cli.Required()),
		cli.String("slug", "URL-safe identifier for use in APIs and integrations.", cli.Required()),
		cli.String("description", "Detailed documentation of what this permission grants access to."),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		req := components.V2PermissionsCreatePermissionRequestBody{
			Name:        cmd.String("name"),
			Slug:        cmd.String("slug"),
			Description: nil,
		}

		if v := cmd.String("description"); v != "" {
			req.Description = &v
		}

		start := time.Now()
		res, err := client.Permissions.CreatePermission(ctx, req)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2PermissionsCreatePermissionResponseBody, time.Since(start))
	},
}
