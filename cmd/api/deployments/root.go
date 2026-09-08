package deployments

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd returns the deployments group command.
func Cmd() *cli.Command {
	return &cli.Command{Name: "deployments", Usage: "Manage deployments", Description: "Manage Unkey deployments." + util.Disclaimer, Commands: []*cli.Command{
		createDeploymentCmd(),
		getDeploymentCmd(),
		listDeploymentsCmd(),
		promoteDeploymentCmd(),
		rollbackDeploymentCmd(),
		startDeploymentCmd(),
		stopDeploymentCmd(),
	}}
}
