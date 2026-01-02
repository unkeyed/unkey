package seed

import (
	"github.com/unkeyed/unkey/pkg/cli"
)

var Cmd = &cli.Command{
	Name:  "seed",
	Usage: "Seed data for testing",
	Commands: []*cli.Command{
		localCmd,
		frontlineCmd,
		verificationsCmd,
	},
}
