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

var migrateKeysCmd = &cli.Command{
	Name:  "migrate-keys",
	Usage: "Returns HTTP 200 even on partial success; hashes that could not be migrated are listed under data.failed",
	Description: `Returns HTTP 200 even on partial success; hashes that could not be migrated are listed under data.failed.

Required permissions:

Your root key must have one of the following permissions for basic key information:
- api.*.create_key (to migrate keys to any API)
- api.<api_id>.create_key (to migrate keys to a specific API)

For full documentation, see https://www.unkey.com/docs/api-reference/v2/keys/migrate-api-keys` + util.Disclaimer,
	Examples: []string{
		`unkey api keys migrate-keys --migration-id=your_company --api-id=api_123456789 --keys-json='[{"hash":"abc123","enabled":true}]'`,
	},
	Flags: []cli.Flag{
		util.RootKeyFlag(),
		util.APIURLFlag(),
		util.ConfigFlag(),
		util.OutputFlag(),
		cli.String("migration-id", "Migration provider ID from Unkey support.", cli.Required()),
		cli.String("api-id", "The API ID to migrate keys into.", cli.Required()),
		cli.String("keys-json", "JSON array of key migration objects.", cli.Required()),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		start := time.Now()
		var keys []components.V2KeysMigrateKeyData
		if v := cmd.String("keys-json"); v != "" {
			if err := json.Unmarshal([]byte(v), &keys); err != nil {
				return fmt.Errorf("invalid JSON for --keys-json: %w", err)
			}
		}

		req := components.V2KeysMigrateKeysRequestBody{
			MigrationID: cmd.String("migration-id"),
			APIID:       cmd.String("api-id"),
			Keys:        keys,
		}

		res, err := client.Keys.MigrateKeys(ctx, req)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2KeysMigrateKeysResponseBody, time.Since(start))
	},
}
