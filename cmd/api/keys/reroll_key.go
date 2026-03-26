package keys

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

var rerollKeyCmd = &cli.Command{
	Name:  "reroll-key",
	Usage: "Generate a new API key while preserving the configuration from an existing key",
	Description: `Generate a new API key while preserving the configuration from an existing key.

This operation creates a fresh key with a new token while maintaining all settings from the original key:
- Permissions and roles
- Custom metadata
- Rate limit configurations
- Identity associations
- Remaining credits
- Recovery settings

Key Generation:
- The system attempts to extract the prefix from the original key
- If prefix extraction fails, the default API prefix is used
- Key length follows the API's default byte configuration (or 16 bytes if not specified)

Original Key Handling:
- The original key will be revoked after the duration specified in expiration
- Set expiration to 0 to revoke immediately
- This allows for graceful key rotation with an overlap period

Common use cases include:
- Rotating keys for security compliance
- Issuing replacement keys for compromised credentials
- Creating backup keys with identical permissions

Important: Analytics and usage metrics are tracked at both the key level AND identity level. If the original key has an identity, the new key will inherit it, allowing you to track usage across both individual keys and the overall identity.

Required permissions:

Your root key must have:
- api.*.create_key or api.<api_id>.create_key
- api.*.encrypt_key or api.<api_id>.encrypt_key (only when the original key is recoverable)

For full documentation, see https://www.unkey.com/docs/api-reference/v2/keys/reroll-key` + util.Disclaimer,
	Examples: []string{
		"unkey api keys reroll-key --key-id=key_1234abcd --expiration=0",
		"unkey api keys reroll-key --key-id=key_1234abcd --expiration=86400000",
	},
	Flags: []cli.Flag{
		util.RootKeyFlag(),
		util.APIURLFlag(),
		util.ConfigFlag(),
		util.OutputFlag(),
		cli.String("key-id", "The key ID to reroll.", cli.Required()),
		cli.Int64("expiration", "Milliseconds until the original key is revoked. 0 for immediate.", cli.Required()),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		start := time.Now()
		req := components.V2KeysRerollKeyRequestBody{
			KeyID:      cmd.String("key-id"),
			Expiration: cmd.Int64("expiration"),
		}

		res, err := client.Keys.RerollKey(ctx, req)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2KeysRerollKeyResponseBody, time.Since(start))
	},
}
