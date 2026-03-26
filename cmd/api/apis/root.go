package apis

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd groups all apis.* subcommands.
var Cmd = &cli.Command{
	Name:        "apis",
	Usage:       "Manage API namespaces",
	Description: "Create, read, and delete API namespaces." + util.Disclaimer,
	Commands: []*cli.Command{
		createAPICmd,
		deleteAPICmd,
		getAPICmd,
		listKeysCmd,
	},
}
