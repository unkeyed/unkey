package seed

import (
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd is the seed command that provides subcommands for seeding test data
// into various backends including local development, frontline, and verifications.
var Cmd = &cli.Command{
	Name:  "seed",
	Usage: "Seed data for testing",
	Commands: []*cli.Command{
		localCmd,
		verificationsCmd,
	},
}
