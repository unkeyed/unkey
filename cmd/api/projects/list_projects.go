package projects

import (
	"context"
	"fmt"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func listProjectsCmd() *cli.Command {
	return &cli.Command{Name: "list-projects", Usage: "Retrieve a paginated list of projects in your workspace.", Description: `Retrieve a paginated list of projects in your workspace.

Use this to build project management dashboards or to enumerate projects for administrative purposes. Results are ordered by project id and returned in pages. When hasMore is true, pass the returned cursor to fetch the next page.

Required Permissions
- project.*.read_project (to read projects in your workspace)

For full documentation, see https://www.unkey.com/docs/api-reference/v2/projects/list-projects` + util.Disclaimer, Examples: []string{"unkey api projects list-projects", "unkey api projects list-projects --limit=25 --search=billing"}, Flags: []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.Int64("limit", "Maximum number of projects to return per request.", cli.Default(int64(100)), cli.MutuallyExclusive("body")), cli.String("cursor", "Pagination cursor from a previous response.", cli.MutuallyExclusive("body")), cli.String("search", "Free-form text to filter projects.", cli.MutuallyExclusive("body"))}, Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		if cmd.FlagIsSet("body") {
			body := cmd.String("body")
			res, err := util.SendBody(ctx, client.Projects.ListProjects, body)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2ProjectsListProjectsResponseBody)
		}
		req := components.V2ProjectsListProjectsRequestBody{Limit: ptr.P(cmd.Int64("limit")), Cursor: nil, Search: nil}
		if v := cmd.String("cursor"); v != "" {
			req.Cursor = &v
		}
		if v := cmd.String("search"); v != "" {
			req.Search = &v
		}
		res, err := client.Projects.ListProjects(ctx, req)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2ProjectsListProjectsResponseBody)
	}}
}
