package projects

import (
	"context"
	"fmt"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func deleteProjectCmd() *cli.Command {
	return &cli.Command{Name: "delete-project", Usage: "Delete an existing project in your workspace, identified by its id.", Description: `Delete an existing project in your workspace, identified by its id.

Deletion is asynchronous and eventually consistent. The project and all of its associated resources (apps, environments, deployments, custom domains) are torn down by a background workflow. A successful response indicates the deletion was enqueued, not that every resource has already been removed.

Projects with delete protection enabled cannot be deleted until protection is disabled.

Required Permissions
- project.*.delete_project (to delete any project)
- project.<project_id>.delete_project (to delete a specific project)

For full documentation, see https://www.unkey.com/docs/api-reference/v2/projects/delete-project` + util.Disclaimer, Examples: []string{"unkey api projects delete-project --project=proj_1234abcd"}, Flags: []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body"))}, Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		if cmd.FlagIsSet("body") {
			body := cmd.String("body")
			res, err := util.SendBody(ctx, client.Projects.DeleteProject, body)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2ProjectsDeleteProjectResponseBody)
		}
		req := components.V2ProjectsDeleteProjectRequestBody{Project: cmd.String("project")}
		res, err := client.Projects.DeleteProject(ctx, req)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2ProjectsDeleteProjectResponseBody)
	}}
}
