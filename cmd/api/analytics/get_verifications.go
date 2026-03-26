package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func getVerificationsCmd() *cli.Command {
	return &cli.Command{
		Name:  "get-verifications",
		Usage: "Execute custom SQL queries against your key verification analytics",
		Description: `Execute custom SQL queries against your key verification analytics.
For complete documentation including available tables, columns, data types, query examples, see the schema reference in the API documentation.

For full documentation, see https://www.unkey.com/docs/api-reference/v2/analytics/query-key-verification-data` + util.Disclaimer,
		Examples: []string{
			`unkey api analytics get-verifications --query="SELECT COUNT(*) as total FROM key_verifications_v1 WHERE outcome = 'VALID' AND time >= now() - INTERVAL 7 DAY"`,
			`unkey api analytics get-verifications --query="SELECT key_id, outcome, COUNT(*) as cnt FROM key_verifications_v1 GROUP BY key_id, outcome"`,
		},
		Flags: []cli.Flag{
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("query", "SQL SELECT query to run against analytics data.", cli.Required()),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			start := time.Now()
			res, err := client.Analytics.GetVerifications(ctx, components.V2AnalyticsGetVerificationsRequestBody{
				Query: cmd.String("query"),
			})
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2AnalyticsGetVerificationsResponseBody, time.Since(start))
		},
	}
}
