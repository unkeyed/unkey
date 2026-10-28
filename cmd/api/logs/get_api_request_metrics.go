package logs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func getAPIRequestMetricsCmd() *cli.Command {
	return &cli.Command{
		Name:  "get-api-request-metrics",
		Usage: "Get API request metrics",
		Description: `Count the requests that the Unkey API served for your workspace per time bucket, split by status class.

The filters have the same shape as in logs.listApiRequests, without requestId. In one field, values under is, contains, startsWith, and endsWith are combined with OR. Values under isNot and notContains, and the range operators gt, gte, lt, and lte, are combined with AND. Different fields are combined with AND. Every text match is exact and case-sensitive.

filters.time sets the window. A missing lower bound is the start of your log retention, and a missing upper bound is now. Unkey picks the bucket width from the window. It is the distance between the time of two points. The first bucket starts at or before the window start and counts its whole width. Windows reach back at most 100 days, and requests outside the log retention of your workspace are not counted.

Required Permissions

Your root key must have the following permission:
- unkey:v1:<workspace_id>:logs/apiRequests#read

For full documentation, see https://www.unkey.com/docs/platform/cli/logs/get-api-request-metrics` + util.Disclaimer,
		Examples: []string{
			"unkey api logs get-api-request-metrics --filters='{\"status\":{\"gte\":400,\"lt\":600}}'",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("filters", "Metric filters as a JSON object.", cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.Logs.GetAPIRequestMetrics, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2LogsGetAPIRequestMetricsResponseBody)
			}

			req := components.V2LogsGetAPIRequestMetricsRequestBody{
				Filters: nil,
			}
			if cmd.FlagIsSet("filters") {
				var value components.V2LogsGetAPIRequestMetricsFilters
				if err := json.Unmarshal([]byte(cmd.String("filters")), &value); err != nil {
					return fmt.Errorf("invalid JSON for --filters: %w", err)
				}
				req.Filters = &value
			}

			res, err := client.Logs.GetAPIRequestMetrics(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2LogsGetAPIRequestMetricsResponseBody)
		},
	}
}
