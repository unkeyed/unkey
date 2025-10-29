package migrate

import (
	"github.com/unkeyed/unkey/go/cmd/migrate/actions"
	"github.com/unkeyed/unkey/go/pkg/cli"
)

var Cmd = &cli.Command{
	Name:  "migrate",
	Usage: "Run database migrations",
	Description: `Run various database migrations for Unkey.

This command provides utilities for migrating data between database schemas,
handling data transformations, and managing database updates.

AVAILABLE MIGRATIONS:
- credits: Migrate key credits from keys table to separate credits table

EXAMPLES:
unkey migrate credits                    # Run credits migration
unkey migrate credits --dry-run          # Preview migration without applying changes
unkey migrate credits --batch-size 1000  # Run migration with custom batch size`,
	Commands: []*cli.Command{
		actions.CreditsCmd,
	},
}
