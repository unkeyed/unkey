package versions

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/cmd/cli/commands/deploy"
	"github.com/unkeyed/unkey/go/pkg/cli"
)

// VersionListOptions holds options for version list command
type VersionListOptions struct {
	Branch string
	Status string
	Limit  int
}

// Command defines the version CLI command with subcommands
var Command = &cli.Command{
	Name:  "version",
	Usage: "Manage API versions",
	Description: `Create, list, and manage versions of your API.

Versions are immutable snapshots of your code, configuration, and infrastructure settings.

EXAMPLES:
    # Create new version
    unkey version create --workspace-id=ws_123 --project-id=proj_456

    # List versions
    unkey version list
    unkey version list --branch=main --limit=20

    # Get specific version
    unkey version get v_abc123def456`,
	Commands: []*cli.Command{
		createCmd,
		listCmd,
		getCmd,
	},
}

// createCmd handles version create (alias for deploy)
var createCmd = &cli.Command{
	Name:        "create",
	Aliases:     []string{"deploy"},
	Usage:       "Create a new version (same as deploy)",
	Description: "Same as 'unkey deploy'. See 'unkey help deploy' for details.",
	Flags:       deploy.DeployFlags,
	Action:      deploy.DeployAction,
}

// listCmd handles version listing
var listCmd = &cli.Command{
	Name:  "list",
	Usage: "List versions",
	Description: `List all versions with optional filtering.

EXAMPLES:
    # List all versions
    unkey version list

    # Filter by branch
    unkey version list --branch=main

    # Filter by status and limit results
    unkey version list --status=active --limit=5`,
	Flags: []cli.Flag{
		cli.String("branch", "Filter by branch"),
		cli.String("status", "Filter by status (pending, building, active, failed)"),
		cli.Int("limit", "Number of versions to show", cli.Default(10)),
	},
	Action: listAction,
}

// getCmd handles getting version details
var getCmd = &cli.Command{
	Name:  "get",
	Usage: "Get version details",
	Description: `Get detailed information about a specific version.

USAGE:
    unkey version get <version-id>

EXAMPLES:
    unkey version get v_abc123def456`,
	Action: getAction,
}

// listAction handles the version list command execution
func listAction(ctx context.Context, cmd *cli.Command) error {
	opts := &VersionListOptions{
		Branch: cmd.String("branch"),
		Status: cmd.String("status"),
		Limit:  cmd.Int("limit"),
	}

	// Display filter info if provided
	filters := []string{}
	if opts.Branch != "" {
		filters = append(filters, fmt.Sprintf("branch=%s", opts.Branch))
	}
	if opts.Status != "" {
		filters = append(filters, fmt.Sprintf("status=%s", opts.Status))
	}
	filters = append(filters, fmt.Sprintf("limit=%d", opts.Limit))

	if len(filters) > 1 {
		fmt.Printf("Listing versions (%s)\n", fmt.Sprintf("%v", filters))
	} else {
		fmt.Printf("Listing versions (limit=%d)\n", opts.Limit)
	}
	fmt.Println()

	// TODO: Add actual version listing logic here
	// This would typically:
	// 1. Call control plane API with filters
	// 2. Parse response
	// 3. Format and display results

	// Mock data for demonstration
	fmt.Println("ID               STATUS    BRANCH    CREATED")
	fmt.Println("v_abc123def456   ACTIVE    main      2024-01-01 12:00:00")
	if opts.Branch == "" || opts.Branch == "feature" {
		fmt.Println("v_def456ghi789   ACTIVE    feature   2024-01-01 11:00:00")
	}
	if opts.Status == "" || opts.Status == "failed" {
		fmt.Println("v_ghi789jkl012   FAILED    main      2024-01-01 10:00:00")
	}

	return nil
}

// getAction handles the version get command execution
func getAction(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args()
	if len(args) == 0 {
		return cli.Exit("version get requires a version ID", 1)
	}

	versionID := args[0]
	fmt.Printf("Getting version: %s\n", versionID)
	fmt.Println()

	// TODO: Add actual version get logic here
	// This would typically:
	// 1. Call control plane API with version ID
	// 2. Parse response
	// 3. Display detailed information

	// Mock data for demonstration
	fmt.Printf("Version: %s\n", versionID)
	fmt.Printf("Status: ACTIVE\n")
	fmt.Printf("Branch: main\n")
	fmt.Printf("Created: 2024-01-01 12:00:00\n")
	fmt.Printf("Docker Image: ghcr.io/unkeyed/deploy:main-abc123\n")
	fmt.Printf("Commit: abc123def456789\n")
	fmt.Println()
	fmt.Printf("Hostnames:\n")
	fmt.Printf("  - https://main-abc123-workspace.unkey.app\n")
	fmt.Printf("  - https://api.acme.com\n")

	return nil
}
