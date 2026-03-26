package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func limitCmd() *cli.Command {
	return &cli.Command{
		Name:  "limit",
		Usage: "Check and enforce rate limits for any identifier",
		Description: `Check and enforce rate limits for any identifier (user ID, IP address, API client, etc.).

Use this for rate limiting beyond API keys - limit users by ID, IPs by address, or any custom identifier. Supports namespace organization, variable costs, and custom overrides.

Response Codes: Rate limit checks return HTTP 200 regardless of whether the limit is exceeded - check the success field in the response to determine if the request should be allowed. A 429 may be returned if the workspace exceeds its API rate limit. Other 4xx responses indicate auth, namespace existence/deletion, or validation errors (e.g., 410 Gone for deleted namespaces). 5xx responses indicate server errors.

Required permissions:

Your root key must have one of the following permissions:
- ratelimit.*.limit (to check limits in any namespace)
- ratelimit.<namespace_id>.limit (to check limits in a specific namespace)

For full documentation, see https://www.unkey.com/docs/api-reference/v2/ratelimit/apply-rate-limiting` + util.Disclaimer,
		Examples: []string{
			"unkey api ratelimit limit --namespace=api.requests --identifier=user_abc123 --limit=100 --duration=60000",
			"unkey api ratelimit limit --namespace=auth.login --identifier=203.0.113.42 --limit=5 --duration=60000",
			"unkey api ratelimit limit --namespace=api.heavy_operations --identifier=user_def456 --limit=50 --duration=3600000 --cost=5",
		},
		Flags: []cli.Flag{
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("namespace", "The id or name of the namespace.", cli.Required()),
			cli.String("identifier", "The entity being rate limited (user ID, IP, etc.).", cli.Required()),
			cli.Int64("limit", "Maximum operations allowed within the duration window.", cli.Required()),
			cli.Int64("duration", "Rate limit window duration in milliseconds.", cli.Required()),
			cli.Int64("cost", "How much quota this request consumes (default 1)."),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			req := components.V2RatelimitLimitRequestBody{
				Namespace:  cmd.String("namespace"),
				Identifier: cmd.String("identifier"),
				Limit:      cmd.Int64("limit"),
				Duration:   cmd.Int64("duration"),
				Cost:       nil,
			}

			if v := cmd.Int64("cost"); v != 0 {
				req.Cost = &v
			}

			start := time.Now()
			res, err := client.Ratelimit.Limit(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2RatelimitLimitResponseBody, time.Since(start))
		},
	}
}
