package keys

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/sdks/api/go/v3/optionalnullable"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func updateKeyCmd() *cli.Command {
	return &cli.Command{
		Name:  "update-key",
		Usage: "Update key properties in response to plan changes, subscription updates, or account status changes",
		Description: `Update key properties in response to plan changes, subscription updates, or account status changes.

Use this for user upgrades/downgrades, role modifications, or administrative changes. Supports partial updates - only specify fields you want to change. Set fields to null to clear them.

Important: Permissions and roles are replaced entirely. Use dedicated add/remove endpoints for incremental changes.

Required permissions:

Your root key must have one of the following permissions:
- api.*.update_key (to update keys in any API)
- api.<api_id>.update_key (to update keys in a specific API)

Side Effects:

If you specify an externalId that doesn't exist, a new identity will be automatically created and linked to the key. Permission updates will auto-create any permissions that don't exist in your workspace. Changes take effect immediately but may take up to 30 seconds to propagate to all edge regions due to cache invalidation.

For full documentation, see https://www.unkey.com/docs/api-reference/v2/keys/update-key-settings` + util.Disclaimer,
		Examples: []string{
			"unkey api keys update-key --key-id=key_1234abcd --name='Updated Key Name'",
			"unkey api keys update-key --key-id=key_1234abcd --enabled=false",
			"unkey api keys update-key --key-id=key_1234abcd --external-id=user_5678 --roles=api_admin,billing_reader",
			`unkey api keys update-key --key-id=key_1234abcd --meta='{"plan":"enterprise","team":"acme"}'`,
			`unkey api keys update-key --key-id=key_1234abcd --credits='{"remaining":5000,"refill":{"interval":"monthly","amount":5000}}'`,
			`unkey api keys update-key --key-id=key_1234abcd --ratelimits='[{"name":"requests","limit":500,"duration":60000,"autoApply":true}]'`,
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("key-id", "The key ID to update.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("name", "Human-readable name for the key.", cli.MutuallyExclusive("body")),
			cli.String("external-id", "Your system's user or entity identifier.", cli.MutuallyExclusive("body")),
			cli.String("meta", "JSON object of arbitrary metadata.", cli.MutuallyExclusive("body")),
			cli.Int64("expires", "Unix timestamp in milliseconds when the key expires.", cli.MutuallyExclusive("body")),
			cli.String("credits", "JSON object for credit and refill configuration.", cli.MutuallyExclusive("body")),
			cli.String("ratelimits", "JSON array of rate limit configurations.", cli.MutuallyExclusive("body")),
			cli.Bool("enabled", "Whether the key is active for verification.", cli.MutuallyExclusive("body")),
			cli.StringSlice("roles", "Comma-separated list of role names to assign.", cli.MutuallyExclusive("body")),
			cli.StringSlice("permissions", "Comma-separated list of permission names to grant.", cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Keys.UpdateKey, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2KeysUpdateKeyResponseBody)
			}

			send := func(req components.V2KeysUpdateKeyRequestBody) error {
				res, err := client.Keys.UpdateKey(ctx, req)
				if err != nil {
					return fmt.Errorf("%s", util.FormatError(err))
				}
				return util.Output(cmd, res.V2KeysUpdateKeyResponseBody)
			}
			req := components.V2KeysUpdateKeyRequestBody{
				KeyID:       cmd.String("key-id"),
				Name:        nil,
				ExternalID:  nil,
				Meta:        nil,
				Expires:     nil,
				Credits:     nil,
				Ratelimits:  nil,
				Enabled:     nil,
				Roles:       nil,
				Permissions: nil,
			}

			if v := cmd.String("name"); v != "" {
				req.Name = optionalnullable.From(&v)
			}

			if v := cmd.String("external-id"); v != "" {
				req.ExternalID = optionalnullable.From(&v)
			}

			if v := cmd.String("meta"); v != "" {
				var meta map[string]any
				if err := json.Unmarshal([]byte(v), &meta); err != nil {
					return fmt.Errorf("invalid JSON for --meta: %w", err)
				}
				req.Meta = optionalnullable.From(&meta)
			}

			if v := cmd.Int64("expires"); v != 0 {
				req.Expires = optionalnullable.From(&v)
			}

			if v := cmd.String("credits"); v != "" {
				var credits components.UpdateKeyCreditsData
				if err := json.Unmarshal([]byte(v), &credits); err != nil {
					return fmt.Errorf("invalid JSON for --credits: %w", err)
				}
				req.Credits = optionalnullable.From(&credits)
			}

			if v := cmd.String("ratelimits"); v != "" {
				var ratelimits []components.RatelimitRequest
				if err := json.Unmarshal([]byte(v), &ratelimits); err != nil {
					return fmt.Errorf("invalid JSON for --ratelimits: %w", err)
				}
				req.Ratelimits = optionalnullable.From(&ratelimits)
			}

			if cmd.FlagIsSet("enabled") {
				req.Enabled = ptr.P(cmd.Bool("enabled"))
			}

			if v := cmd.StringSlice("roles"); len(v) > 0 {
				req.Roles = v
			}

			if v := cmd.StringSlice("permissions"); len(v) > 0 {
				req.Permissions = v
			}
			return send(req)
		},
	}
}
