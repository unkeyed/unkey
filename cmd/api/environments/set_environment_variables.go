package environments

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func setEnvironmentVariablesCmd() *cli.Command {
	return &cli.Command{
		Name:        "set-environment-variables",
		Usage:       "Set environment variables",
		Description: "Upsert environment variables atomically. With --prune=true, variables omitted from the supplied array are removed.\n\nFor full documentation, see https://www.unkey.com/docs/api-reference/v2/environments/set-environment-variables" + util.Disclaimer,
		Examples: []string{
			`unkey api environments set-environment-variables --project=payments --app=payments-api --environment=production --variables='[{"key":"TOKEN","value":"secret"}]'`,
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			util.YesFlag(),
			cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("app", "App ID or slug.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("environment", "Environment ID or slug.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("variables", "Variables as a JSON array.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.Bool("prune", "Remove variables not included.", cli.Default(false), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			var variables []components.EnvironmentVariableInput
			if !cmd.FlagIsSet("body") {
				if err := json.Unmarshal([]byte(cmd.String("variables")), &variables); err != nil {
					return fmt.Errorf("invalid JSON for --variables: %w", err)
				}
			}
			req, err := util.RequestBody(cmd, components.V2EnvironmentsSetEnvironmentVariablesRequestBody{
				Project: cmd.String("project"), App: cmd.String("app"), Environment: cmd.String("environment"), Variables: variables, Prune: new(cmd.Bool("prune")),
			})
			if err != nil {
				return err
			}
			if req.Prune != nil && *req.Prune {
				if err := util.Confirm(ctx, cmd, fmt.Sprintf("Remove variables not included in this request from environment %q, app %q, project %q?", req.Environment, req.App, req.Project)); err != nil {
					return err
				}
			}
			res, err := client.Environments.SetEnvironmentVariables(ctx, req)
			if err != nil {
				return err
			}
			return util.Output(ctx, cmd, res.V2EnvironmentsSetEnvironmentVariablesResponseBody)
		},
	}
}
