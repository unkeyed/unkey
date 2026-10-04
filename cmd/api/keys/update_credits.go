package keys

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/sdks/api/go/v3/optionalnullable"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func updateCreditsCmd() *cli.Command {
	return &cli.Command{
		Name:  "update-credits",
		Usage: "Update credit quotas in response to plan changes, billing cycles, or usage purchases",
		Description: `Update credit quotas in response to plan changes, billing cycles, or usage purchases.

Use this for user upgrades/downgrades, monthly quota resets, credit purchases, or promotional bonuses. Supports three operations: set, increment, or decrement credits. Set to null for unlimited usage.

Important: Setting unlimited credits automatically clears existing refill configurations.

Required permissions:

Your root key must have one of the following permissions:
- api.*.update_key (to update keys in any API)
- api.<api_id>.update_key (to update keys in a specific API)

Side Effects:

Credit updates remove the key from cache immediately. Setting credits to unlimited automatically clears any existing refill settings. Changes take effect instantly but may take up to 30 seconds to propagate to all edge regions.

For full documentation, see https://www.unkey.com/docs/api-reference/v2/keys/update-key-credits` + util.Disclaimer,
		Examples: []string{
			"unkey api keys update-credits --key-id=key_1234abcd --operation=set --value=1000",
			"unkey api keys update-credits --key-id=key_1234abcd --operation=increment --value=500",
			"unkey api keys update-credits --key-id=key_1234abcd --operation=decrement --value=100",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("key-id", "The key ID to update credits for.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("operation", "How to modify credits: set, increment, or decrement.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.Int64("value", "The credit amount for the operation.", cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Keys.UpdateCredits, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2KeysUpdateCreditsResponseBody)
			}

			req := components.V2KeysUpdateCreditsRequestBody{
				KeyID:     cmd.String("key-id"),
				Operation: components.Operation(cmd.String("operation")),
				Value:     nil,
			}

			if v := cmd.Int64("value"); v != 0 {
				req.Value = optionalnullable.From(&v)
			}

			res, err := client.Keys.UpdateCredits(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2KeysUpdateCreditsResponseBody)
		},
	}
}
