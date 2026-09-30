package projects

import (
	"context"
	"fmt"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func getProjectCmd() *cli.Command {
	return &cli.Command{Name: "get-project", Usage: "Retrieve a single project in your workspace by its id.", Description: `Retrieve a single project in your workspace by its id.

Use this to fetch project details after creation, verify a project exists before performing operations, or resolve a project's metadata from its id.

Required Permissions
- project.*.read_project (to read any project)
- project.<project_id>.read_project (to read a specific project)

For full documentation, see https://www.unkey.com/docs/api-reference/v2/projects/get-project` + util.Disclaimer, Examples: []string{"unkey api projects get-project --project=proj_1234abcd"}, Flags: []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body"))}, Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		if cmd.FlagIsSet("body") {
			body := cmd.String("body")
			res, err := util.SendBody(ctx, client.Projects.GetProject, body)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2ProjectsGetProjectResponseBody)
		}
		req := components.V2ProjectsGetProjectRequestBody{Project: cmd.String("project")}
		res, err := client.Projects.GetProject(ctx, req)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2ProjectsGetProjectResponseBody)
	}}
}
