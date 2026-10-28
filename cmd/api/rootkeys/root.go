package rootkeys

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func Cmd() *cli.Command {
	return &cli.Command{
		Name:        "root-keys",
		Usage:       "Manage root keys",
		Description: "Manage root keys through the Unkey API." + util.Disclaimer,
		Commands: []*cli.Command{
			createKeyCmd(),
			deleteKeyCmd(),
			listKeysCmd(),
			rerollKeyCmd(),
			updateKeyCmd(),
		},
	}
}
