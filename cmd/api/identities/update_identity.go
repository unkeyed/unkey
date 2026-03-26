package identities

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func updateIdentityCmd() *cli.Command {
	return &cli.Command{
		Name:  "update-identity",
		Usage: "Update an identity's metadata and rate limits.",
		Description: `Update an identity's metadata and rate limits. Only specified fields are modified - others remain unchanged.

Perfect for subscription changes, plan upgrades, or updating user information. Changes take effect immediately.

Important
Requires identity.*.update_identity permission
Rate limit changes propagate within 30 seconds

For full documentation, see https://www.unkey.com/docs/api-reference/v2/identities/update-identity` + util.Disclaimer,
		Examples: []string{
			"unkey api identities update-identity --identity=user_123",
			`unkey api identities update-identity --identity=user_123 --meta-json='{"plan":"premium","name":"Alice Smith"}'`,
			`unkey api identities update-identity --identity=user_123 --ratelimits-json='[{"name":"requests","limit":1000,"duration":3600000,"autoApply":true}]'`,
		},
		Flags: []cli.Flag{
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("identity", "The identity ID or externalId to update.", cli.Required()),
			cli.String("meta-json", "JSON object of metadata to replace existing metadata."),
			cli.String("ratelimits-json", "JSON array of rate limit configurations."),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			req := components.V2IdentitiesUpdateIdentityRequestBody{
				Identity:   cmd.String("identity"),
				Meta:       nil,
				Ratelimits: nil,
			}

			if v := cmd.String("meta-json"); v != "" {
				var meta map[string]any
				if err := json.Unmarshal([]byte(v), &meta); err != nil {
					return fmt.Errorf("invalid JSON for --meta-json: %w", err)
				}
				req.Meta = meta
			}

			if v := cmd.String("ratelimits-json"); v != "" {
				var ratelimits []components.RatelimitRequest
				if err := json.Unmarshal([]byte(v), &ratelimits); err != nil {
					return fmt.Errorf("invalid JSON for --ratelimits-json: %w", err)
				}
				req.Ratelimits = ratelimits
			}

			start := time.Now()
			res, err := client.Identities.UpdateIdentity(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2IdentitiesUpdateIdentityResponseBody, time.Since(start))
		},
	}
}
