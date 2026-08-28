package apis

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func listKeysCmd() *cli.Command {
	return &cli.Command{
		Name:  "list-keys",
		Usage: "Retrieve a paginated list of API keys for dashboard and administrative interfaces",
		Description: `Retrieve a paginated list of API keys for dashboard and administrative interfaces.

Use this to build key management dashboards, filter keys by user with externalId, or retrieve key details for administrative purposes. Each key includes status, metadata, permissions, and usage limits.

Important: Set decrypt: true only in secure contexts to retrieve plaintext key values from recoverable keys.

Required permissions:
- api.*.read_key
- api.<api_id>.read_key
- api.*.read_api
- api.<api_id>.read_api

Additional permission required for decrypt functionality:
- api.*.decrypt_key
- api.<api_id>.decrypt_key

For full documentation, see https://www.unkey.com/docs/api-reference/v2/apis/list-api-keys` + util.Disclaimer,
		Examples: []string{
			"unkey api apis list-keys --api-id=api_1234abcd",
			"unkey api apis list-keys --api-id=api_1234abcd --limit=50",
			"unkey api apis list-keys --api-id=api_1234abcd --external-id=user_1234abcd",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("api-id", "The API ID whose keys to list.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.Int64("limit", "Maximum number of keys to return per page.", cli.MutuallyExclusive("body")),
			cli.String("cursor", "Pagination cursor from a previous response.", cli.MutuallyExclusive("body")),
			cli.String("external-id", "Filter keys by external ID.", cli.MutuallyExclusive("body")),
			cli.Bool("decrypt", "Include the plaintext key value in the response.", cli.Default(false), cli.MutuallyExclusive("body")),
			cli.Bool("revalidate-keys-cache", "Bypass the cache and read keys directly from the database.", cli.Default(false), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Apis.ListKeys, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2ApisListKeysResponseBody)
			}

			req := components.V2ApisListKeysRequestBody{
				APIID:               cmd.String("api-id"),
				Limit:               nil,
				Cursor:              nil,
				ExternalID:          nil,
				Decrypt:             ptr.P(cmd.Bool("decrypt")),
				RevalidateKeysCache: ptr.P(cmd.Bool("revalidate-keys-cache")),
			}

			if v := cmd.Int64("limit"); v != 0 {
				req.Limit = &v
			}

			if v := cmd.String("cursor"); v != "" {
				req.Cursor = &v
			}

			if v := cmd.String("external-id"); v != "" {
				req.ExternalID = &v
			}

			res, err := client.Apis.ListKeys(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}

			return util.Output(cmd, res.V2ApisListKeysResponseBody)
		},
	}
}
