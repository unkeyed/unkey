package identities

import (
	"context"
	"encoding/json"
	"fmt"

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
			`unkey api identities update-identity --identity=user_123 --meta='{"plan":"premium","name":"Alice Smith"}'`,
			`unkey api identities update-identity --identity=user_123 --ratelimits='[{"name":"requests","limit":1000,"duration":3600000,"autoApply":true}]'`,
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("identity", "The identity ID or externalId to update.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("meta", "JSON object of metadata to replace existing metadata.", cli.MutuallyExclusive("body")),
			cli.String("ratelimits", "JSON array of rate limit configurations.", cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Identities.UpdateIdentity, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2IdentitiesUpdateIdentityResponseBody)
			}
			send := func(req components.V2IdentitiesUpdateIdentityRequestBody) error {
				res, err := client.Identities.UpdateIdentity(ctx, req)
				if err != nil {
					return fmt.Errorf("%s", util.FormatError(err))
				}
				return util.Output(cmd, res.V2IdentitiesUpdateIdentityResponseBody)
			}
			req := components.V2IdentitiesUpdateIdentityRequestBody{
				Identity:   cmd.String("identity"),
				Meta:       nil,
				Ratelimits: nil,
			}

			if v := cmd.String("meta"); v != "" {
				var meta map[string]any
				if err := json.Unmarshal([]byte(v), &meta); err != nil {
					return fmt.Errorf("invalid JSON for --meta: %w", err)
				}
				req.Meta = meta
			}

			if v := cmd.String("ratelimits"); v != "" {
				var ratelimits []components.RatelimitRequest
				if err := json.Unmarshal([]byte(v), &ratelimits); err != nil {
					return fmt.Errorf("invalid JSON for --ratelimits: %w", err)
				}
				req.Ratelimits = ratelimits
			}
			return send(req)
		},
	}
}
