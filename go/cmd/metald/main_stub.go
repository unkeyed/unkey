//go:build !linux
// +build !linux

package metald

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/pkg/cli"
)

var Cmd = &cli.Command{
	Name:  "metald",
	Usage: "VM management service (Linux only)",
	Description: `The metald service manages virtual machines and is only available on Linux systems.

This command is not available on this platform. Please run metald on a Linux system.`,
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return fmt.Errorf("metald is only supported on Linux systems")
	},
}