package version

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/unkeyed/unkey/go/pkg/cli"
)

var Cmd = &cli.Command{
	Name:  "version",
	Usage: "Manage API versions",
	Description: `Create, list, and manage versions of your API.

Versions are immutable snapshots of your code, configuration, and infrastructure settings. 
Each version represents a specific deployment state that can be rolled back to at any time.

## Available Commands

- ` + "`get`" + ` - Get details about a specific version
- ` + "`list`" + ` - List all versions with optional filtering
- ` + "`rollback`" + ` - Rollback to a previous version`,
	Commands: []*cli.Command{
		getCmd,
		listCmd,
		rollbackCmd,
	},
}

var getCmd = &cli.Command{
	Name:  "get",
	Usage: "Get details about a version",
	Description: `Get comprehensive details about a specific version including status, branch, creation time, and associated hostnames.

## Usage

` + "`unkey version get <version-id>`" + `

## Examples

Get details for a specific version:
` + "`unkey version get v_abc123def456`" + `

Get details for the latest version:
` + "`unkey version get v_def456ghi789`",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		logger := slog.Default()

		args := cmd.Args()
		if len(args) < 1 {
			return fmt.Errorf("version ID required")
		}

		versionID := args[0]
		logger.Info("Getting version details", "version_id", versionID)

		// Call control plane API to get version
		fmt.Printf("Version: %s\n", versionID)
		fmt.Println("Status: ACTIVE")
		fmt.Println("Branch: main")
		fmt.Println("Created: 2024-01-01 12:00:00")
		fmt.Println("Hostnames:")
		fmt.Println("  - https://abc123-workspace.unkey.app")

		return nil
	},
}

var listCmd = &cli.Command{
	Name:  "list",
	Usage: "List versions with optional filtering",
	Description: `List all versions for the current project with support for filtering by branch, status, and limiting results.

## Usage

` + "`unkey version list [flags]`" + `

## Examples

List all versions:
` + "`unkey version list`" + `

List versions from main branch:
` + "`unkey version list --branch main`" + `

List only active versions:
` + "`unkey version list --status active`" + `

List last 5 versions:
` + "`unkey version list --limit 5`" + `

Combine filters:
` + "`unkey version list --branch main --status active --limit 3`",
	Flags: []cli.Flag{
		cli.String("branch", "Filter by branch name"),
		cli.String("status", "Filter by status (pending, building, active, failed)"),
		cli.Int("limit", "Number of versions to show", cli.Default(10)),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		// Hardcoded for demo
		workspace := "Acme"
		project := "my-api"

		fmt.Printf("Versions for %s/%s:\n", workspace, project)
		fmt.Println()
		fmt.Println("ID               STATUS    BRANCH    CREATED")
		fmt.Println("v_abc123def456   ACTIVE    main      2024-01-01 12:00:00")
		fmt.Println("v_def456ghi789   ACTIVE    feature   2024-01-01 11:00:00")
		fmt.Println("v_ghi789jkl012   FAILED    main      2024-01-01 10:00:00")

		return nil
	},
}

var rollbackCmd = &cli.Command{
	Name:  "rollback",
	Usage: "Rollback to a previous version",
	Description: `Rollback a hostname to a previous version. This operation will switch traffic from the current 
version to the specified target version.

**Warning:** This operation affects live traffic. Use the ` + "`--force`" + ` flag to skip the confirmation prompt 
in automated environments.

## Usage

` + "`unkey version rollback <hostname> <version-id> [flags]`" + `

## Examples

Rollback with confirmation prompt:
` + "`unkey version rollback my-api.unkey.app v_abc123def456`" + `

Rollback without confirmation (automation):
` + "`unkey version rollback my-api.unkey.app v_abc123def456 --force`" + `

Rollback staging environment:
` + "`unkey version rollback staging-api.unkey.app v_def456ghi789`",
	Flags: []cli.Flag{
		cli.Bool("force", "Skip confirmation prompt for automated deployments"),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		logger := slog.Default()

		args := cmd.Args()
		if len(args) < 2 {
			return fmt.Errorf("hostname and version ID required")
		}

		hostname := args[0]
		versionID := args[1]
		force := cmd.Bool("force")

		logger.Info("Rolling back version",
			"hostname", hostname,
			"version_id", versionID,
			"force", force,
		)

		if !force {
			fmt.Printf("⚠ Are you sure you want to rollback %s to version %s? [y/N] ", hostname, versionID)
			// Read user confirmation
		}

		// Call control plane API to rollback
		fmt.Printf("Rolling back %s to version %s...\n", hostname, versionID)
		fmt.Println("✓ Rollback completed successfully!")

		return nil
	},
}
