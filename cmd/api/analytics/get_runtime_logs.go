package analytics

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func getRuntimeLogsCmd() *cli.Command {
	return &cli.Command{
		Name:  "get-runtime-logs",
		Usage: "Query runtime log data",
		Description: `A query can use only the public alias runtime_logs_v1. CTEs, subqueries, UNION, and EXCEPT are permitted.
The root key must have the project.*.read_runtime_logs permission.
Unkey limits each query to the workspace of the root key. To get the logs of one project, app, environment, or deployment, add a filter on project_id, app_id, environment_id, or deployment_id.
The workspace retention period and the workspace query limits apply.

For full documentation, see https://www.unkey.com/docs/platform/analytics/get-runtime-logs` + util.Disclaimer,
		Examples: []string{
			`unkey api analytics get-runtime-logs --query="SELECT time, message FROM runtime_logs_v1 WHERE deployment_id = 'dpl_123' ORDER BY time DESC LIMIT 100"`,
			`unkey api analytics get-runtime-logs --query="SELECT severity, COUNT(*) AS total FROM runtime_logs_v1 WHERE environment_id = 'env_123' GROUP BY severity"`,
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("query", "SQL query to run against runtime log data.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.Analytics.GetRuntimeLogs, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2AnalyticsGetRuntimeLogsResponseBody)
			}

			res, err := client.Analytics.GetRuntimeLogs(ctx, components.V2AnalyticsGetRuntimeLogsRequestBody{
				Query: cmd.String("query"),
			})
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2AnalyticsGetRuntimeLogsResponseBody)
		},
	}
}
