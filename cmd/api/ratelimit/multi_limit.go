package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

var multiLimitCmd = &cli.Command{
	Name:  "multi-limit",
	Usage: "Check and enforce multiple rate limits in a single request",
	Description: `Check and enforce multiple rate limits in a single request for any identifiers (user IDs, IP addresses, API clients, etc.).

Use this to efficiently check multiple rate limits at once. Each rate limit check is independent and returns its own result with a top-level passed indicator showing if all checks succeeded.

Response Codes: Rate limit checks return HTTP 200 regardless of whether limits are exceeded - check the passed field to see if all limits passed, or the success field in each individual result. A 429 may be returned if the workspace exceeds its API rate limit. Other 4xx responses indicate auth, namespace existence/deletion, or validation errors (e.g., 410 Gone for deleted namespaces). 5xx responses indicate server errors.

Required permissions:

Your root key must have one of the following permissions:
- ratelimit.*.limit (to check limits in any namespace)
- ratelimit.<namespace_id>.limit (to check limits in all specific namespaces being checked)

For full documentation, see https://www.unkey.com/docs/api-reference/v2/ratelimit/apply-multiple-rate-limit-checks` + util.Disclaimer,
	Examples: []string{
		`unkey api ratelimit multi-limit --limits-json='[{"namespace":"api.requests","identifier":"user_abc123","limit":100,"duration":60000},{"namespace":"auth.login","identifier":"user_abc123","limit":5,"duration":60000}]'`,
		`unkey api ratelimit multi-limit --limits-json='[{"namespace":"api.light_operations","identifier":"user_xyz789","limit":100,"duration":60000,"cost":1},{"namespace":"api.heavy_operations","identifier":"user_xyz789","limit":50,"duration":3600000,"cost":5}]'`,
	},
	Flags: []cli.Flag{
		util.RootKeyFlag(),
		util.APIURLFlag(),
		util.ConfigFlag(),
		util.OutputFlag(),
		cli.String("limits-json", "JSON array of rate limit check objects.", cli.Required()),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		var limits []components.V2RatelimitLimitRequestBody
		if err := json.Unmarshal([]byte(cmd.String("limits-json")), &limits); err != nil {
			return fmt.Errorf("invalid JSON for --limits-json: %w", err)
		}

		start := time.Now()
		res, err := client.Ratelimit.MultiLimit(ctx, limits)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2RatelimitMultiLimitResponseBody, time.Since(start))
	},
}
