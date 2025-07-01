package run

import (
	"github.com/unkeyed/unkey/go/cmd/api"
	"github.com/unkeyed/unkey/go/cmd/controlplane"
	"github.com/urfave/cli/v3"
)

var Cmd = &cli.Command{
	Name:  "run",
	Usage: "Run Unkey services",
	Description: `Run various Unkey services including:
  - api: The main API server for validating and managing API keys
  - controlplane: The control plane service for managing infrastructure`,

	Commands: []*cli.Command{
		api.Cmd,
		controlplane.Cmd,
	},
}
