package apis

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func getAPICmd() *cli.Command {
	return &cli.Command{
		Name:  "get-api",
		Usage: "Retrieve basic information about an API namespace including its ID and name",
		Description: `Retrieve basic information about an API namespace including its ID and name.

Use this to verify an API exists before performing operations, get the human-readable name when you only have the API ID, or confirm access to a specific namespace. For detailed key information, use the listKeys endpoint instead.

Required permissions:
- api.*.read_api
- api.<api_id>.read_api

For full documentation, see https://www.unkey.com/docs/api-reference/v2/apis/get-api-namespace` + util.Disclaimer,
		Examples: []string{
			"unkey api apis get-api --api-id=api_1234abcd",
		},
		Flags: []cli.Flag{
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("api-id", "The API ID to retrieve.", cli.Required()),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			start := time.Now()
			res, err := client.Apis.GetAPI(ctx, components.V2ApisGetAPIRequestBody{
				APIID: cmd.String("api-id"),
			})
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}

			return util.Output(cmd, res.V2ApisGetAPIResponseBody, time.Since(start))
		},
	}
}
