package permissions

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
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
			"unkey api permissions list-roles --search=admin",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.Int64("limit", "Maximum number of roles to return per page.", cli.MutuallyExclusive("body")),
			cli.String("cursor", "Pagination cursor from a previous response.", cli.MutuallyExclusive("body")),
			cli.String("search", "Filter roles by ID, name, or description.", cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Permissions.ListRoles, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2PermissionsListRolesResponseBody)
			}

			req := components.V2PermissionsListRolesRequestBody{
				Limit:  nil,
				Cursor: nil,
				Search: nil,
			}

			if v := cmd.Int64("limit"); v != 0 {
				req.Limit = &v
			}

			if v := cmd.String("cursor"); v != "" {
				req.Cursor = &v
			}

			if v := cmd.String("search"); v != "" {
				req.Search = &v
			}

			res, err := client.Permissions.ListRoles(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2PermissionsListRolesResponseBody)
		},
	}
}
