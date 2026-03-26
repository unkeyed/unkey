package keys

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd groups all keys.* subcommands.
func Cmd() *cli.Command {
	return &cli.Command{
		Name:        "keys",
		Usage:       "Manage API keys",
		Description: "Create, verify, update, delete, and manage permissions and roles for API keys." + util.Disclaimer,
		Commands: []*cli.Command{
			createKeyCmd(),
			deleteKeyCmd(),
			getKeyCmd(),
			verifyKeyCmd(),
			updateKeyCmd(),
			rerollKeyCmd(),
			whoamiCmd(),
			migrateKeysCmd(),
			addPermissionsCmd(),
			removePermissionsCmd(),
			setPermissionsCmd(),
			addRolesCmd(),
			removeRolesCmd(),
			setRolesCmd(),
			updateCreditsCmd(),
		},
	}
}
