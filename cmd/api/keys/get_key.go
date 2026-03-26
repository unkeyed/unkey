package keys

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/ptr"
)

var getKeyCmd = &cli.Command{
	Name:  "get-key",
	Usage: "Retrieve detailed key information for dashboard interfaces and administrative purposes",
	Description: `Retrieve detailed key information for dashboard interfaces and administrative purposes.

Use this to build key management dashboards showing users their key details, status, permissions, and usage data. You can identify keys by keyId or the actual key string.

Important: Set decrypt: true only in secure contexts to retrieve plaintext key values from recoverable keys.

Required permissions:

Your root key must have one of the following permissions for basic key information:
- api.*.read_key (to read keys from any API)
- api.<api_id>.read_key (to read keys from a specific API)

Additional permission required for decrypt functionality:
- api.*.decrypt_key or api.<api_id>.decrypt_key

For full documentation, see https://www.unkey.com/docs/api-reference/v2/keys/get-api-key` + util.Disclaimer,
	Examples: []string{
		"unkey api keys get-key --key-id=key_1234abcd",
		"unkey api keys get-key --key-id=key_1234abcd --decrypt",
	},
	Flags: []cli.Flag{
		util.RootKeyFlag(),
		util.APIURLFlag(),
		util.ConfigFlag(),
		util.OutputFlag(),
		cli.String("key-id", "The key ID to retrieve.", cli.Required()),
		cli.Bool("decrypt", "Whether to include the plaintext key value in the response.", cli.Default(false)),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		start := time.Now()
		req := components.V2KeysGetKeyRequestBody{
			KeyID:   cmd.String("key-id"),
			Decrypt: ptr.P(cmd.Bool("decrypt")),
		}

		res, err := client.Keys.GetKey(ctx, req)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2KeysGetKeyResponseBody, time.Since(start))
	},
}
