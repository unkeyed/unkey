package rootkeys

import (
	"context"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func rerollKeyCmd() *cli.Command {
	return &cli.Command{
		Name:  "reroll-key",
		Usage: "Reroll root key",
		Description: `Creates a new root key secret with the same name, enabled state, expiration, and
effective permissions as an existing root key in the authenticated workspace.
Requires unkey:v1:<workspace_id>:rootKeys/<key_id>#write, and the caller must
already hold every permission of the original key. When expiration is not null,
it also requires unkey:v1:<workspace_id>:rootKeys/<key_id>#delete.

expiration controls the original key: 0 revokes it immediately, a positive value
keeps it valid for that many milliseconds, and null keeps its current expiration.

For full documentation, see https://www.unkey.com/docs/platform/cli/root-keys/reroll-key` + util.Disclaimer,
		Examples: []string{
			"unkey api root-keys reroll-key --key-id=key_example --expiration=86400000",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("key-id", "Root key ID.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.Int64("expiration", "Milliseconds until the original key expires; 0 revokes it immediately. Use --body with null to keep its expiration.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.RootKeys.RerollKey, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2RootKeysRerollKeyResponseBody)
			}

			req := components.V2RootKeysRerollKeyRequestBody{
				KeyID:      cmd.String("key-id"),
				Expiration: new(cmd.Int64("expiration")),
			}

			res, err := client.RootKeys.RerollKey(ctx, req)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2RootKeysRerollKeyResponseBody)
		},
	}
}
