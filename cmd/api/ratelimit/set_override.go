package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func setOverrideCmd() *cli.Command {
	return &cli.Command{
		Name:  "set-override",
		Usage: "Create or update a custom rate limit for specific identifiers",
		Description: `Create or update a custom rate limit for specific identifiers, bypassing the namespace default.

Use this to create premium tiers with higher limits, apply stricter limits to specific users, or implement emergency throttling.

Important: Overrides take effect immediately and completely replace the default limit for matching identifiers. Use wildcard patterns (e.g., premium_*) to match multiple identifiers.

Required permissions:
- ratelimit.*.set_override
- ratelimit.<namespace_id>.set_override

For full documentation, see https://www.unkey.com/docs/api-reference/v2/ratelimit/set-ratelimit-override` + util.Disclaimer,
		Examples: []string{
			"unkey api ratelimit set-override --namespace=api.requests --identifier=premium_user_123 --limit=1000 --duration=60000",
			"unkey api ratelimit set-override --namespace=api.requests --identifier='premium_*' --limit=500 --duration=60000",
		},
		Flags: []cli.Flag{
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("namespace", "The ID or name of the rate limit namespace.", cli.Required()),
			cli.String("identifier", "Identifier of the entity receiving this custom rate limit.", cli.Required()),
			cli.Int64("limit", "Maximum number of requests allowed for this override.", cli.Required()),
			cli.Int64("duration", "Duration in milliseconds for the rate limit window.", cli.Required()),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			start := time.Now()
			res, err := client.Ratelimit.SetOverride(ctx, components.V2RatelimitSetOverrideRequestBody{
				Namespace:  cmd.String("namespace"),
				Identifier: cmd.String("identifier"),
				Limit:      cmd.Int64("limit"),
				Duration:   cmd.Int64("duration"),
			})
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2RatelimitSetOverrideResponseBody, time.Since(start))
		},
	}
}
