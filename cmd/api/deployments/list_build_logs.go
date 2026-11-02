package deployments

import (
	"context"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func listBuildLogsCmd() *cli.Command {
	return &cli.Command{
		Name:  "list-build-logs",
		Usage: "Retrieve the build output of a deployment as a list of log entries, in the order the build printed them",
		Description: `Retrieve the build output of a deployment as a list of log entries, in the
order the build printed them. A log entry is one chunk of build output,
and can contain several lines or part of a line.

Poll this endpoint to follow a running build:

- Send the pagination.cursor of the last response as cursor. The
  response contains only the entries after it.
- When hasMore is true, request again now.
- When hasMore is false, you have all entries so far. Wait,
  then request again with the same cursor.
- The cursor is absent only when the request had no cursor and the build
  has no entries yet.
- A response can contain fewer than limit entries and still have
  hasMore: true when the entries are large.

New entries become visible up to a few seconds after the build prints
them. To know that a build is done, check the deployment status with
getDeployment. After the status is no longer building, request until
you get an empty response at least 5 seconds later.

When a build attempt is retried, an entry from the earlier attempt can
arrive after the cursor has moved past it, and a poll does not return it.
A request without a cursor returns it. A deployment from a prebuilt image
has no build and returns no entries. Build logs are kept for 3 months.

Required permissions:
- unkey:v1:<workspace_id>:projects/<project_id>/apps/<app_id>/environments/<environment_id>/deployments/<deployment_id>/buildLogs#read

For full documentation, see https://www.unkey.com/docs/compute/cli/deployments/list-build-logs` + util.Disclaimer,
		Examples: []string{
			"unkey api deployments list-build-logs --deployment-id=d_1234abcd",
			"unkey api deployments list-build-logs --deployment-id=d_1234abcd --cursor=1704067201000001 --limit=50",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("deployment-id", "The deployment ID or slug whose build logs to retrieve.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("step-id", "Return only entries for this stepId from a previous response.", cli.MutuallyExclusive("body")),
			cli.String("cursor", "Pagination cursor from a previous response to fetch entries after it.", cli.MutuallyExclusive("body")),
			cli.Int64("limit", "Maximum log entries per page (1-500, default 100).", cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.Deployments.ListBuildLogs, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2DeploymentsListBuildLogsResponseBody)
			}

			req := components.V2DeploymentsListBuildLogsRequestBody{
				DeploymentID: cmd.String("deployment-id"),
				StepID:       nil,
				Cursor:       nil,
				Limit:        nil,
			}
			if cmd.FlagIsSet("step-id") {
				req.StepID = new(cmd.String("step-id"))
			}
			if cmd.FlagIsSet("cursor") {
				req.Cursor = new(cmd.String("cursor"))
			}
			if cmd.FlagIsSet("limit") {
				req.Limit = new(cmd.Int64("limit"))
			}

			res, err := client.Deployments.ListBuildLogs(ctx, req)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2DeploymentsListBuildLogsResponseBody)
		},
	}
}
