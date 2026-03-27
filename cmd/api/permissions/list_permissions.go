package permissions

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func listPermissionsCmd() *cli.Command {
	return &cli.Command{
		Name:  "list-permissions",
		Usage: "Retrieve all permissions in your workspace",
		Description: `Retrieve all permissions in your workspace.
Results are paginated and sorted by their id.

Required permissions:
- rbac.*.read_permission

For full documentation, see https://www.unkey.com/docs/api-reference/v2/permissions/list-permissions` + util.Disclaimer,
		Examples: []string{
			"unkey api permissions list-permissions",
			"unkey api permissions list-permissions --limit=50",
			"unkey api permissions list-permissions --limit=50 --cursor=eyJrZXkiOiJwZXJtXzEyMzQifQ==",
		},
		Flags: []cli.Flag{
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.Int64("limit", "Maximum number of permissions to return per page."),
			cli.String("cursor", "Pagination cursor from a previous response."),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			req := components.V2PermissionsListPermissionsRequestBody{
				Limit:  nil,
				Cursor: nil,
			}

			if v := cmd.Int64("limit"); v != 0 {
				req.Limit = &v
			}

			if v := cmd.String("cursor"); v != "" {
				req.Cursor = &v
			}

			start := time.Now()
			res, err := client.Permissions.ListPermissions(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2PermissionsListPermissionsResponseBody, time.Since(start))
		},
	}
}
