package gateway

import (
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func Cmd() *cli.Command {
	return &cli.Command{Name: "gateway", Usage: "Manage gateway policies", Description: "List, replace, and update gateway policies." + util.Disclaimer, Commands: []*cli.Command{listPoliciesCmd(), setPoliciesCmd(), updatePolicyCmd()}}
}
