package keys

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func deleteKeyCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete-key",
		Usage: "Delete API keys permanently from user accounts or for cleanup purposes",
		Description: `Delete API keys permanently from user accounts or for cleanup purposes.

Use this for user-requested key deletion, account deletion workflows, or cleaning up unused keys. Keys are immediately invalidated. Two modes: soft delete (default, preserves audit records) and permanent delete.

Important: For temporary access control, use updateKey with enabled: false instead of deletion.

Required permissions:

Your root key must have one of the following permissions:
- api.*.delete_key (to delete keys in any API)
- api.<api_id>.delete_key (to delete keys in a specific API)

For full documentation, see https://www.unkey.com/docs/api-reference/v2/keys/delete-api-keys` + util.Disclaimer,
		Examples: []string{
			"unkey api keys delete-key --key-id=key_1234abcd",
			"unkey api keys delete-key --key-id=key_1234abcd --permanent",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("key-id", "The key ID to delete.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.Bool("permanent", "Whether to permanently erase the key instead of soft-deleting.", cli.Default(false), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Keys.DeleteKey, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2KeysDeleteKeyResponseBody)
			}

			req := components.V2KeysDeleteKeyRequestBody{
				KeyID:     cmd.String("key-id"),
				Permanent: ptr.P(cmd.Bool("permanent")),
			}

			res, err := client.Keys.DeleteKey(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2KeysDeleteKeyResponseBody)
		},
	}
}
