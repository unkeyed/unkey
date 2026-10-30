package ratelimit

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func listNamespacesCmd() *cli.Command {
	return &cli.Command{
		Name:  "list-namespaces",
		Usage: "Retrieve a paginated list of the rate limit namespaces in your workspace",
		Description: `Retrieve a paginated list of the rate limit namespaces in your workspace.

Use this to find the namespaces your services rate limit against, for example to build an admin view or to look up a namespace ID. Results are ordered newest first and returned in pages. When hasMore is true, pass the returned cursor to fetch the next page.

Deleted namespaces are not returned. The list contains only the namespaces your root key can read.

Required permissions:
- unkey:v1:<workspace_id>:projects/*/ratelimits/namespaces/*#read (to read any namespace)
- unkey:v1:<workspace_id>:projects/<project_id>/ratelimits/namespaces/<namespace_id>#read (to read a specific namespace)

For full documentation, see https://www.unkey.com/docs/api-management/cli/ratelimit/list-namespaces` + util.Disclaimer,
		Examples: []string{
			"unkey api ratelimit list-namespaces",
			"unkey api ratelimit list-namespaces --search=email --limit=20",
			"unkey api ratelimit list-namespaces --cursor=rlns_1234abcd",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.Int64("limit", "Maximum namespaces per page (1-100, default 100).", cli.MutuallyExclusive("body")),
			cli.String("cursor", "Pagination cursor from a previous response with hasMore=true.", cli.MutuallyExclusive("body")),
			cli.String("search", "Filter namespace IDs and names by a case-insensitive substring.", cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.Ratelimit.ListNamespaces, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2RatelimitListNamespacesResponseBody)
			}

			req := components.V2RatelimitListNamespacesRequestBody{
				Limit:  nil,
				Cursor: nil,
				Search: nil,
			}
			if cmd.FlagIsSet("limit") {
				req.Limit = new(cmd.Int64("limit"))
			}
			if cmd.FlagIsSet("cursor") {
				req.Cursor = new(cmd.String("cursor"))
			}
			if cmd.FlagIsSet("search") {
				req.Search = new(cmd.String("search"))
			}

			res, err := client.Ratelimit.ListNamespaces(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2RatelimitListNamespacesResponseBody)
		},
	}
}
