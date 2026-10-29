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
					fmt.Println("No saved workspaces. Run 'unkey login' to sign in.")
					return nil
				}
				fmt.Println("  WORKSPACE")
				for _, item := range sessions {
					marker := " "
					if item.WorkspaceID == active {
						marker = "*"
					}
					fmt.Printf("%s %s\n", marker, item.WorkspaceID)
				}
				fmt.Println("\n* Active workspace")
				return nil
			},
		},
		{
			Name:        "switch",
			Usage:       "Switch the active CLI workspace",
			Description: "unkey workspace switch <workspace-id>",
			AcceptsArgs: true,
			Action: func(_ context.Context, cmd *cli.Command) error {
				if len(cmd.Args()) != 1 {
					return fmt.Errorf("usage: unkey workspace switch <workspace-id>")
				}
				if err := session.Switch(cmd.Args()[0]); err != nil {
					return fmt.Errorf("switch workspace: %w", err)
				}
				fmt.Printf("Using workspace %s.\n", cmd.Args()[0])
				return nil
			},
		},
	},
}
