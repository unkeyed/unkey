package apis

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func deleteAPICmd() *cli.Command {
	return &cli.Command{
		Name:  "delete-api",
		Usage: "Permanently delete an API namespace and immediately invalidate all associated keys",
		Description: `Permanently delete an API namespace and immediately invalidate all associated keys.

Use this for cleaning up development environments, retiring deprecated services, or removing unused resources.
All keys in the namespace are immediately marked as deleted and will fail verification with code=NOT_FOUND.

Important: This operation is immediate and permanent. Verify you have the correct API ID before deletion.
If delete protection is enabled, disable it first through the dashboard or API configuration.

Required permissions:
- api.*.delete_api
- api.<api_id>.delete_api

For full documentation, see https://www.unkey.com/docs/api-reference/v2/apis/delete-api-namespace` + util.Disclaimer,
		Examples: []string{
			"unkey api apis delete-api --api-id=api_1234abcd",
		},
		Flags: []cli.Flag{
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("api-id", "The API ID to delete.", cli.Required()),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			start := time.Now()
			res, err := client.Apis.DeleteAPI(ctx, components.V2ApisDeleteAPIRequestBody{
				APIID: cmd.String("api-id"),
			})
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}

			return util.Output(cmd, res.V2ApisDeleteAPIResponseBody, time.Since(start))
		},
	}
}
