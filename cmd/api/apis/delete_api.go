package apis

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
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
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			util.YesFlag(),
			cli.String("api-id", "The API ID to delete.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			req, err := util.RequestBody(cmd, components.V2ApisDeleteAPIRequestBody{APIID: cmd.String("api-id")})
			if err != nil {
				return err
			}
			if err := util.Confirm(cmd, fmt.Sprintf("Delete API %q and invalidate all its keys?", req.APIID)); err != nil {
				return err
			}

			res, err := client.Apis.DeleteAPI(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}

			return util.Output(cmd, res.V2ApisDeleteAPIResponseBody)
		},
	}
}
