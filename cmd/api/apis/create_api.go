package apis

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

var createAPICmd = &cli.Command{
	Name:  "create-api",
	Usage: "Create an API namespace for organizing keys by environment, service, or product",
	Description: `Create an API namespace for organizing keys by environment, service, or product.

Use this to separate production from development keys, isolate different services, or manage multiple products. Each API gets a unique identifier and dedicated infrastructure for secure key operations.

Important: API names must be unique within your workspace and cannot be changed after creation.

Required permissions:
- api.*.create_api

For full documentation, see https://www.unkey.com/docs/api-reference/v2/apis/create-api-namespace` + util.Disclaimer,
	Examples: []string{
		"unkey api apis create-api --name=payment-service-prod",
		"unkey api apis create-api --name=user-api-dev",
	},
	Flags: []cli.Flag{
		util.RootKeyFlag(),
		util.APIURLFlag(),
		util.ConfigFlag(),
		util.OutputFlag(),
		cli.String("name", "The name for the new API namespace.", cli.Required()),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		start := time.Now()
		res, err := client.Apis.CreateAPI(ctx, components.V2ApisCreateAPIRequestBody{
			Name: cmd.String("name"),
		})
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}

		return util.Output(cmd, res.V2ApisCreateAPIResponseBody, time.Since(start))
	},
}
