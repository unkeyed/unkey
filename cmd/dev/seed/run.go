package seed

import (
	"github.com/unkeyed/unkey/cmd/dev/seed/checkpoints"
	"github.com/unkeyed/unkey/cmd/dev/seed/deployusage"
	"github.com/unkeyed/unkey/pkg/cli"
)

var Cmd = &cli.Command{
	Name:  "seed",
	Usage: "Seed data for testing",
	Commands: []*cli.Command{
		localCmd,
		verificationsCmd,
		frontlineCmd,
		checkpoints.Cmd,
		deployusage.Cmd,
	},
}
