package workspace

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/cli/session"
)

// Cmd lists stored CLI workspaces and switches the active one.
var Cmd = &cli.Command{
	Name:        "workspace",
	Usage:       "Choose the active CLI workspace",
	Description: "List the workspaces this machine has signed into, or switch the active one.",
	Commands: []*cli.Command{
		{
			Name:  "list",
			Usage: "List stored CLI workspaces",
			Action: func(context.Context, *cli.Command) error {
				active, sessions, err := session.List()
				if err != nil {
					return err
				}
				if len(sessions) == 0 {
					fmt.Println("No CLI sessions stored. Run unkey login.")
					return nil
				}
				for _, item := range sessions {
					marker := " "
					if item.WorkspaceID == active {
						marker = "*"
					}
					fmt.Printf("%s %s\n", marker, item.WorkspaceID)
				}
				return nil
			},
		},
		{
			Name:        "switch",
			Usage:       "Switch the active CLI workspace",
			Description: "unkey workspace switch <workspace-id>",
			Action: func(_ context.Context, cmd *cli.Command) error {
				if len(cmd.Args()) != 1 {
					return fmt.Errorf("usage: unkey workspace switch <workspace-id>")
				}
				if err := session.Switch(cmd.Args()[0]); err != nil {
					return fmt.Errorf("switch workspace: %w", err)
				}
				fmt.Printf("Active workspace is %s.\n", cmd.Args()[0])
				return nil
			},
		},
	},
}
