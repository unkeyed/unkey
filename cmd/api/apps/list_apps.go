package apps

import (
	"context"
	"fmt"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func listAppsCmd() *cli.Command {
	return &cli.Command{Name: "list-apps", Usage: "Retrieve a paginated list of apps within a project.", Description: `Retrieve a paginated list of apps within a project.

Use this to enumerate every app in a project. Results are ordered by app id and paginated; when hasMore is true, pass the returned cursor to fetch the next page.

Required Permissions
- app.*.read_app (to read apps in any project)

For full documentation, see https://www.unkey.com/docs/api-reference/v2/apps/list-apps` + util.Disclaimer, Examples: []string{"unkey api apps list-apps --project=payments", "unkey api apps list-apps --project=payments --limit=25 --search=checkout"}, Flags: []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.Int64("limit", "Maximum number of apps to return per request.", cli.Default(int64(100)), cli.MutuallyExclusive("body")), cli.String("cursor", "Pagination cursor from a previous response.", cli.MutuallyExclusive("body")), cli.String("search", "Free-form text to filter apps.", cli.MutuallyExclusive("body"))}, Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		if cmd.FlagIsSet("body") {
			body := cmd.String("body")
			res, err := util.SendBody(ctx, client.Apps.ListApps, body)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2AppsListAppsResponseBody)
		}
		req := components.V2AppsListAppsRequestBody{Project: cmd.String("project"), Limit: ptr.P(cmd.Int64("limit")), Cursor: nil, Search: nil}
		if v := cmd.String("cursor"); v != "" {
			req.Cursor = &v
		}
		if v := cmd.String("search"); v != "" {
			req.Search = &v
		}
		res, err := client.Apps.ListApps(ctx, req)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2AppsListAppsResponseBody)
	}}
}
