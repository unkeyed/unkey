package keys

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

var whoamiCmd = &cli.Command{
	Name:  "whoami",
	Usage: "Find out what key this is",
	Description: `Find out what key this is.

Required permissions:

Your root key must have one of the following permissions for basic key information:
- api.*.read_key (to read keys from any API)
- api.<api_id>.read_key (to read keys from a specific API)

If your rootkey lacks permissions but the key exists, we may return a 404 status here to prevent leaking the existance of a key to unauthorized clients. If you believe that a key should exist, but receive a 404, please double check your root key has the correct permissions.

For full documentation, see https://www.unkey.com/docs/api-reference/v2/keys/get-api-key-by-hash` + util.Disclaimer,
	Examples: []string{
		"unkey api keys whoami --key=sk_1234abcdef5678",
	},
	Flags: []cli.Flag{
		util.RootKeyFlag(),
		util.APIURLFlag(),
		util.ConfigFlag(),
		util.OutputFlag(),
		cli.String("key", "The full API key string, including any prefix.", cli.Required()),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		start := time.Now()
		req := components.V2KeysWhoamiRequestBody{
			Key: cmd.String("key"),
		}

		res, err := client.Keys.Whoami(ctx, req)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2KeysWhoamiResponseBody, time.Since(start))
	},
}
