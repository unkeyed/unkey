package keys

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func migrateKeysCmd() *cli.Command {
	return &cli.Command{
		Name:  "migrate-keys",
		Usage: "Returns HTTP 200 even on partial success; hashes that could not be migrated are listed under data.failed",
		Description: `Returns HTTP 200 even on partial success; hashes that could not be migrated are listed under data.failed.

Required permissions:

Your root key must have one of the following permissions for basic key information:
- api.*.create_key (to migrate keys to any API)
- api.<api_id>.create_key (to migrate keys to a specific API)

For full documentation, see https://www.unkey.com/docs/api-reference/v2/keys/migrate-api-keys` + util.Disclaimer,
		Examples: []string{
			`unkey api keys migrate-keys --migration-id=your_company --api-id=api_123456789 --keys='[{"hash":"abc123","enabled":true}]'`,
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("migration-id", "Migration provider ID from Unkey support.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("api-id", "The API ID to migrate keys into.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("keys", "JSON array of key migration objects.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Keys.MigrateKeys, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2KeysMigrateKeysResponseBody)
			}

			send := func(req components.V2KeysMigrateKeysRequestBody) error {
				res, err := client.Keys.MigrateKeys(ctx, req)
				if err != nil {
					return fmt.Errorf("%s", util.FormatError(err))
				}
				return util.Output(cmd, res.V2KeysMigrateKeysResponseBody)
			}
			var keys []components.V2KeysMigrateKeyData
			if v := cmd.String("keys"); v != "" {
				if err := json.Unmarshal([]byte(v), &keys); err != nil {
					return fmt.Errorf("invalid JSON for --keys: %w", err)
				}
			}

			req := components.V2KeysMigrateKeysRequestBody{
				MigrationID: cmd.String("migration-id"),
				APIID:       cmd.String("api-id"),
				Keys:        keys,
			}

			return send(req)
		},
	}
}
