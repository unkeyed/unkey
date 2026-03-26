package keys

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func createKeyCmd() *cli.Command {
	return &cli.Command{
		Name:  "create-key",
		Usage: "Create a new API key for user authentication and authorization",
		Description: `Create a new API key for user authentication and authorization.

Use this endpoint when users sign up, upgrade subscription tiers, or need additional keys. Keys are cryptographically secure and unique to the specified API namespace.

Important: The key is returned only once. Store it immediately and provide it to your user, as it cannot be retrieved later.

Common use cases:
- Generate keys for new user registrations
- Create additional keys for different applications
- Issue keys with specific permissions or limits

Required permissions:

Your root key needs one of:
- api.*.create_key (create keys in any API)
- api.<api_id>.create_key (create keys in specific API)

For full documentation, see https://www.unkey.com/docs/api-reference/v2/keys/create-api-key` + util.Disclaimer,
		Examples: []string{
			"unkey api keys create-key --api-id=api_1234abcd",
			"unkey api keys create-key --api-id=api_1234abcd --prefix=prod --name='Payment Service Key'",
			"unkey api keys create-key --api-id=api_1234abcd --external-id=user_1234abcd --roles=api_admin,billing_reader",
			`unkey api keys create-key --api-id=api_1234abcd --meta-json='{"plan":"pro","team":"acme"}'`,
			`unkey api keys create-key --api-id=api_1234abcd --credits-json='{"remaining":1000,"refill":{"interval":"monthly","amount":100}}'`,
			`unkey api keys create-key --api-id=api_1234abcd --ratelimits-json='[{"name":"requests","limit":100,"duration":60000,"autoApply":true}]'`,
		},
		Flags: []cli.Flag{
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("api-id", "The API namespace this key belongs to.", cli.Required()),
			cli.String("prefix", "Prefix prepended to the generated key string."),
			cli.String("name", "Human-readable name for the key."),
			cli.Int64("byte-length", "Cryptographic key length in bytes."),
			cli.String("external-id", "Your system's user or entity identifier to link to this key."),
			cli.String("meta-json", "JSON object of arbitrary metadata returned during verification."),
			cli.StringSlice("roles", "Comma-separated list of role names to assign."),
			cli.StringSlice("permissions", "Comma-separated list of permission names to grant."),
			cli.Int64("expires", "Unix timestamp in milliseconds when the key expires."),
			cli.String("credits-json", "JSON object of credit and refill configuration."),
			cli.String("ratelimits-json", "JSON array of rate limit configurations."),
			cli.Bool("enabled", "Whether the key is active for verification.", cli.Default(true)),
			cli.Bool("recoverable", "Whether the plaintext key is stored for later retrieval.", cli.Default(false)),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			start := time.Now()
			req := components.V2KeysCreateKeyRequestBody{
				APIID:       cmd.String("api-id"),
				Prefix:      nil,
				Name:        nil,
				ByteLength:  nil,
				ExternalID:  nil,
				Meta:        nil,
				Roles:       nil,
				Permissions: nil,
				Expires:     nil,
				Credits:     nil,
				Ratelimits:  nil,
				Enabled:     ptr.P(cmd.Bool("enabled")),
				Recoverable: ptr.P(cmd.Bool("recoverable")),
			}

			if v := cmd.String("prefix"); v != "" {
				req.Prefix = &v
			}

			if v := cmd.String("name"); v != "" {
				req.Name = &v
			}

			if v := cmd.Int64("byte-length"); v != 0 {
				req.ByteLength = &v
			}

			if v := cmd.String("external-id"); v != "" {
				req.ExternalID = &v
			}

			if v := cmd.String("meta-json"); v != "" {
				var meta map[string]any
				if err := json.Unmarshal([]byte(v), &meta); err != nil {
					return fmt.Errorf("invalid JSON for --meta-json: %w", err)
				}
				req.Meta = meta
			}

			if v := cmd.StringSlice("roles"); len(v) > 0 {
				req.Roles = v
			}

			if v := cmd.StringSlice("permissions"); len(v) > 0 {
				req.Permissions = v
			}

			if v := cmd.Int64("expires"); v != 0 {
				req.Expires = &v
			}

			if v := cmd.String("credits-json"); v != "" {
				var credits components.KeyCreditsData
				if err := json.Unmarshal([]byte(v), &credits); err != nil {
					return fmt.Errorf("invalid JSON for --credits-json: %w", err)
				}
				req.Credits = &credits
			}

			if v := cmd.String("ratelimits-json"); v != "" {
				var ratelimits []components.RatelimitRequest
				if err := json.Unmarshal([]byte(v), &ratelimits); err != nil {
					return fmt.Errorf("invalid JSON for --ratelimits-json: %w", err)
				}
				req.Ratelimits = ratelimits
			}

			res, err := client.Keys.CreateKey(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2KeysCreateKeyResponseBody, time.Since(start))
		},
	}
}
