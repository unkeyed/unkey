package analytics

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func getGatewayRequestsCmd() *cli.Command {
	return &cli.Command{
		Name:  "get-gateway-requests",
		Usage: "Query gateway request data",
		Description: `A query can use only the public alias gateway_requests_v1. CTEs, subqueries, UNION, and EXCEPT are permitted.
The root key must have the project.*.read_gateway_requests permission.
Unkey limits each query to the workspace of the root key. To get the data for one project, app, or environment, add a filter on project_id, app_id, or environment_id.
The workspace retention period and the workspace query limits apply.

For full documentation, see https://www.unkey.com/docs/platform/analytics/get-gateway-requests` + util.Disclaimer,
		Examples: []string{
			`unkey api analytics get-gateway-requests --query="SELECT project_id, COUNT(*) AS total FROM gateway_requests_v1 GROUP BY project_id"`,
			`unkey api analytics get-gateway-requests --query="SELECT time, status, latency FROM gateway_requests_v1 WHERE environment_id = 'env_123' ORDER BY time DESC LIMIT 100"`,
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("query", "SQL query to run against gateway request data.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.Analytics.GetGatewayRequests, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2AnalyticsGetGatewayRequestsResponseBody)
			}

			res, err := client.Analytics.GetGatewayRequests(ctx, components.V2AnalyticsGetGatewayRequestsRequestBody{
				Query: cmd.String("query"),
			})
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2AnalyticsGetGatewayRequestsResponseBody)
		},
	}
}
