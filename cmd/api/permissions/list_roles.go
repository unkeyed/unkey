package permissions

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func listRolesCmd() *cli.Command {
	return &cli.Command{
		Name:  "list-roles",
		Usage: "Retrieve all roles in your workspace including their assigned permissions",
		Description: `Retrieve all roles in your workspace including their assigned permissions.
Results are paginated and sorted by their id.

Required permissions:
- rbac.*.read_role

For full documentation, see https://www.unkey.com/docs/api-reference/v2/permissions/list-roles` + util.Disclaimer,
		Examples: []string{
			"unkey api permissions list-roles",
			"unkey api permissions list-roles --limit=50",
			"unkey api permissions list-roles --limit=50 --cursor=eyJrZXkiOiJyb2xlXzEyMzQifQ==",
		},
		Flags: []cli.Flag{
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.Int64("limit", "Maximum number of roles to return per page."),
			cli.String("cursor", "Pagination cursor from a previous response."),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			req := components.V2PermissionsListRolesRequestBody{
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
			res, err := client.Permissions.ListRoles(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2PermissionsListRolesResponseBody, time.Since(start))
		},
	}
}
