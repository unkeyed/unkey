package apis

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
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
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("api-id", "The API ID to retrieve.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Apis.GetAPI, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2ApisGetAPIResponseBody)
			}

			res, err := client.Apis.GetAPI(ctx, components.V2ApisGetAPIRequestBody{
				APIID: cmd.String("api-id"),
			})
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}

			return util.Output(cmd, res.V2ApisGetAPIResponseBody)
		},
	}
}
