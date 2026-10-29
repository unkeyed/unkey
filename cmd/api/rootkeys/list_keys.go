package rootkeys

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func listKeysCmd() *cli.Command {
	return &cli.Command{
		Name:  "list-keys",
		Usage: "List root keys",
		Description: `Lists readable root keys in the authenticated workspace, ordered by key ID.
Requires unkey:v1:<workspace_id>:rootKeys/<key_id>#read for each returned key.
Use rootKeys/*#read to read all root keys. Write permission does not imply read.
A caller with no readable keys receives an empty page.

Includes disabled and expired keys from the new root-key store.
Deleted keys are excluded. Secrets and hashes are never returned.
Pass the returned cursor when hasMore is true to retrieve the next page.

For full documentation, see https://www.unkey.com/docs/platform/cli/root-keys/list-keys` + util.Disclaimer,
		Examples: []string{
			"unkey api root-keys list-keys --limit=20",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.Int64("limit", "Maximum keys per page, between 1 and 100.", cli.Default(int64(100)), cli.MutuallyExclusive("body")),
			cli.String("cursor", "Opaque cursor from a previous response.", cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.RootKeys.ListKeys, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2RootKeysListKeysResponseBody)
			}

			req := components.V2RootKeysListKeysRequestBody{
				Limit:  new(cmd.Int64("limit")),
				Cursor: nil,
			}
			if cmd.FlagIsSet("cursor") {
				req.Cursor = new(cmd.String("cursor"))
			}

			res, err := client.RootKeys.ListKeys(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2RootKeysListKeysResponseBody)
		},
	}
}
