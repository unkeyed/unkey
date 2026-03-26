package keys

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func verifyKeyCmd() *cli.Command {
	return &cli.Command{
		Name:  "verify-key",
		Usage: "Verify an API key's validity and permissions for request authentication",
		Description: `Verify an API key's validity and permissions for request authentication.

Use this endpoint on every incoming request to your protected resources. It checks key validity, permissions, rate limits, and usage quotas in a single call.

Important: Returns HTTP 200 for all verification outcomes -- check the valid field in response data to determine if the key is authorized. A 429 may be returned if the workspace exceeds its API rate limit.

Common use cases:
- Authenticate API requests before processing
- Enforce permission-based access control
- Track usage and apply rate limits

Required permissions:

Your root key needs one of:
- api.*.verify_key (verify keys in any API)
- api.<api_id>.verify_key (verify keys in specific API)

Note: If your root key has no verify permissions at all, you will receive a 403 Forbidden error. If your root key has verify permissions for a different API than the key you're verifying, you will receive a 200 response with code: NOT_FOUND to avoid leaking key existence.

For full documentation, see https://www.unkey.com/docs/api-reference/v2/keys/verify-api-key` + util.Disclaimer,
		Examples: []string{
			"unkey api keys verify-key --key=sk_1234abcdef",
			"unkey api keys verify-key --key=sk_1234abcdef --permissions='documents.read AND users.view'",
			"unkey api keys verify-key --key=sk_1234abcdef --tags=endpoint=/users/profile,method=GET",
			`unkey api keys verify-key --key=sk_1234abcdef --credits-json='{"cost":5}'`,
			`unkey api keys verify-key --key=sk_1234abcdef --ratelimits-json='[{"name":"requests","limit":100,"duration":60000}]'`,
		},
		Flags: []cli.Flag{
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("key", "The API key to verify, including any prefix.", cli.Required()),
			cli.StringSlice("tags", "Metadata tags for analytics in key=value format."),
			cli.String("permissions", "Permission query to check, supports AND/OR operators."),
			cli.String("credits-json", "JSON object for credit consumption configuration."),
			cli.String("ratelimits-json", "JSON array of rate limit checks to enforce."),
			cli.String("migration-id", "Migration provider ID for on-demand key migration."),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			start := time.Now()
			req := components.V2KeysVerifyKeyRequestBody{
				Key:         cmd.String("key"),
				Tags:        nil,
				Permissions: nil,
				Credits:     nil,
				Ratelimits:  nil,
				MigrationID: nil,
			}

			if v := cmd.StringSlice("tags"); len(v) > 0 {
				req.Tags = v
			}

			if v := cmd.String("permissions"); v != "" {
				req.Permissions = &v
			}

			if v := cmd.String("credits-json"); v != "" {
				var credits components.KeysVerifyKeyCredits
				if err := json.Unmarshal([]byte(v), &credits); err != nil {
					return fmt.Errorf("invalid JSON for --credits-json: %w", err)
				}
				req.Credits = &credits
			}

			if v := cmd.String("ratelimits-json"); v != "" {
				var ratelimits []components.KeysVerifyKeyRatelimit
				if err := json.Unmarshal([]byte(v), &ratelimits); err != nil {
					return fmt.Errorf("invalid JSON for --ratelimits-json: %w", err)
				}
				req.Ratelimits = ratelimits
			}

			if v := cmd.String("migration-id"); v != "" {
				req.MigrationID = &v
			}

			res, err := client.Keys.VerifyKey(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2KeysVerifyKeyResponseBody, time.Since(start))
		},
	}
}
