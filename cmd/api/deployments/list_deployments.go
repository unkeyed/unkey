package deployments

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

var deploymentStatuses = []string{
	string(components.DeploymentStatusPending), string(components.DeploymentStatusStarting),
	string(components.DeploymentStatusBuilding), string(components.DeploymentStatusDeploying),
	string(components.DeploymentStatusNetwork), string(components.DeploymentStatusFinalizing),
	string(components.DeploymentStatusReady), string(components.DeploymentStatusFailed),
	string(components.DeploymentStatusSkipped), string(components.DeploymentStatusAwaitingApproval),
	string(components.DeploymentStatusStopped), string(components.DeploymentStatusSuperseded),
	string(components.DeploymentStatusCancelled),
}

func validateDeploymentStatuses(value string) error {
	for _, status := range strings.Split(value, ",") {
		status = strings.TrimSpace(status)
		if status != "" && !slices.Contains(deploymentStatuses, status) {
			return fmt.Errorf("invalid status %q; valid choices: %s", status, strings.Join(deploymentStatuses, ", "))
		}
	}
	return nil
}

func listDeploymentsCmd() *cli.Command {
	return &cli.Command{
		Name: "list-deployments", Usage: "List deployments with optional resource and status filters", Description: "List deployments in reverse chronological order. Filter by project, app, environment, or one or more lifecycle statuses. Use the cursor returned by a response to fetch the next page.\n\nFor full documentation, see https://www.unkey.com/docs/api-reference/v2/deployments/list-deployments" + util.Disclaimer,
		Examples: []string{"unkey api deployments list-deployments --project=payments --app=payments-api --environment=production", "unkey api deployments list-deployments --status=ready,failed --limit=25"},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("project", "Project ID or slug to filter by.", cli.MutuallyExclusive("body")), cli.String("app", "App ID or slug to filter by.", cli.MutuallyExclusive("body")), cli.String("environment", "Environment ID or slug to filter by.", cli.MutuallyExclusive("body")),
			cli.StringSlice("status", "Lifecycle statuses to include. Valid choices: "+strings.Join(deploymentStatuses, ", ")+".", cli.Validate(validateDeploymentStatuses), cli.MutuallyExclusive("body")), cli.Int64("limit", "Maximum deployments to return per page (default 100).", cli.MutuallyExclusive("body")), cli.String("cursor", "Pagination cursor from a previous response.", cli.MutuallyExclusive("body"))},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Deployments.ListDeployments, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2DeploymentsListDeploymentsResponseBody)
			}
			req := components.V2DeploymentsListDeploymentsRequestBody{Project: nil, App: nil, Environment: nil, Status: nil, Limit: nil, Cursor: nil}
			if v := cmd.String("project"); v != "" {
				req.Project = &v
			}
			if v := cmd.String("app"); v != "" {
				req.App = &v
			}
			if v := cmd.String("environment"); v != "" {
				req.Environment = &v
			}
			for _, v := range cmd.StringSlice("status") {
				req.Status = append(req.Status, components.DeploymentStatus(v))
			}
			if v := cmd.Int64("limit"); v != 0 {
				req.Limit = &v
			}
			if v := cmd.String("cursor"); v != "" {
				req.Cursor = &v
			}
			res, err := client.Deployments.ListDeployments(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2DeploymentsListDeploymentsResponseBody)
		},
	}
}
