package domains

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd groups all domains.* subcommands.
func Cmd() *cli.Command {
	return &cli.Command{Name: "domains", Usage: "Manage custom domains", Description: "Create, read, verify, and delete custom domains." + util.Disclaimer, Commands: []*cli.Command{
		createDomainCmd(), deleteDomainCmd(), getDomainCmd(), listDomainsCmd(), verifyDomainCmd(),
	}}
}
