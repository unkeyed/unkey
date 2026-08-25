package analytics

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func getRatelimitsCmd() *cli.Command {
	return &cli.Command{
		Name:  "get-ratelimits",
		Usage: "Query rate limit data",
		Description: `Queries may reference only the five public rate limit analytics aliases: ratelimits_v1, ratelimits_per_minute_v1, ratelimits_per_hour_v1, ratelimits_per_day_v1, or ratelimits_per_month_v1. CTEs, subqueries, UNION, and EXCEPT are supported.
Queries are always restricted to the authenticated workspace. Wildcard analytics permission can read every namespace in that workspace; namespace-scoped permissions automatically restrict results to the permitted namespace IDs.
Workspace retention and query limits apply.

For full documentation, see https://www.unkey.com/docs/platform/analytics/get-ratelimits` + util.Disclaimer,
		Examples: []string{
			`unkey api analytics get-ratelimits --query="SELECT namespace_id, COUNT(*) AS total FROM ratelimits_v1 WHERE namespace_id = 'rlns_123' GROUP BY namespace_id"`,
			`unkey api analytics get-ratelimits --query="SELECT time, sum(total) AS requests FROM ratelimits_per_hour_v1 WHERE namespace_id = 'rlns_123' GROUP BY time ORDER BY time"`,
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("query", "SQL query to run against rate limit analytics data.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.Analytics.GetRatelimits, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2AnalyticsGetRatelimitsResponseBody)
			}

			res, err := client.Analytics.GetRatelimits(ctx, components.V2AnalyticsGetRatelimitsRequestBody{
				Query: cmd.String("query"),
			})
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2AnalyticsGetRatelimitsResponseBody)
		},
	}
}
