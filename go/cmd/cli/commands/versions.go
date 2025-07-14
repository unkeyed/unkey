package commands

import (
	"context"
	"flag"
	"fmt"

	"github.com/unkeyed/unkey/go/cmd/cli/commands/deploy"
)

// VersionListOptions holds options for version list command
type VersionListOptions struct {
	Branch string
	Status string
	Limit  int
}

// Version handles the version command and its subcommands
func Version(ctx context.Context, args []string, env map[string]string) error {
	if len(args) < 1 {
		PrintVersionCommandHelp()
		return fmt.Errorf("version command requires a subcommand")
	}

	subcommand := args[0]
	switch subcommand {
	case "create":
		return VersionCreate(ctx, args[1:], env)
	case "list":
		return VersionList(args[1:])
	case "get":
		return VersionGet(args[1:])
	case "help", "-h", "--help":
		PrintVersionHelp()
		return nil
	default:
		PrintVersionCommandHelp()
		return fmt.Errorf("unknown version subcommand: %s", subcommand)
	}
}

// VersionCreate handles version create (same as deploy)
func VersionCreate(ctx context.Context, args []string, env map[string]string) error {
	return deploy.Deploy(ctx, args, env)
}

// VersionList handles version list command
func VersionList(args []string) error {
	opts, err := parseVersionListFlags(args)
	if err != nil {
		return err
	}

	fmt.Printf("Listing versions (branch=%s, status=%s, limit=%d)\n",
		opts.Branch, opts.Status, opts.Limit)

	// TODO: Add actual version listing logic
	fmt.Println("ID               STATUS    BRANCH    CREATED")
	fmt.Println("v_abc123def456   ACTIVE    main      2024-01-01 12:00:00")
	fmt.Println("v_def456ghi789   ACTIVE    feature   2024-01-01 11:00:00")

	return nil
}

// parseVersionListFlags parses flags for version list command
func parseVersionListFlags(args []string) (*VersionListOptions, error) {
	fs := flag.NewFlagSet("version list", flag.ExitOnError)

	opts := &VersionListOptions{}

	fs.StringVar(&opts.Branch, "branch", "", "Filter by branch")
	fs.StringVar(&opts.Status, "status", "", "Filter by status")
	fs.IntVar(&opts.Limit, "limit", 10, "Number of versions to show")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("failed to parse version list flags: %w", err)
	}

	return opts, nil
}

// VersionGet handles version get command
func VersionGet(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("version get requires a version ID")
	}

	versionID := args[0]
	fmt.Printf("Getting version: %s\n", versionID)

	// TODO: Add actual version get logic
	fmt.Printf("Version: %s\n", versionID)
	fmt.Println("Status: ACTIVE")
	fmt.Println("Branch: main")
	fmt.Println("Created: 2024-01-01 12:00:00")

	return nil
}

// PrintVersionCommandHelp shows help for version subcommands
func PrintVersionCommandHelp() {
	fmt.Println("'version' requires a subcommand.")
	fmt.Println("")
	fmt.Println("Valid subcommands for 'version':")
	fmt.Println("    create    Create a new version")
	fmt.Println("    list      List versions")
	fmt.Println("    get       Get version details")
	fmt.Println("")
	fmt.Println("For detailed help: unkey help version")
}

// PrintVersionHelp prints detailed help for version command
func PrintVersionHelp() {
	fmt.Println("unkey version - Manage API versions")
	fmt.Println("")
	fmt.Println("USAGE:")
	fmt.Println("    unkey version <subcommand> [FLAGS]")
	fmt.Println("")
	fmt.Println("SUBCOMMANDS:")
	fmt.Println("    create    Create a new version (same as deploy)")
	fmt.Println("    list      List versions")
	fmt.Println("    get       Get version details")
	fmt.Println("")
	fmt.Println("VERSION CREATE:")
	fmt.Println("    Same as 'unkey deploy'. See 'unkey help deploy' for details.")
	fmt.Println("")
	fmt.Println("VERSION LIST FLAGS:")
	fmt.Println("    --branch <n>      Filter by branch")
	fmt.Println("    --status <status>    Filter by status (pending, building, active, failed)")
	fmt.Println("    --limit <number>     Number of versions to show (default: 10)")
	fmt.Println("")
	fmt.Println("VERSION GET:")
	fmt.Println("    unkey version get <version-id>")
	fmt.Println("")
	fmt.Println("EXAMPLES:")
	fmt.Println("    # Create new version")
	fmt.Println("    unkey version create --workspace-id=ws_123 --project-id=proj_456")
	fmt.Println("")
	fmt.Println("    # List versions")
	fmt.Println("    unkey version list")
	fmt.Println("    unkey version list --branch=main --limit=20")
	fmt.Println("")
	fmt.Println("    # Get specific version")
	fmt.Println("    unkey version get v_abc123def456")
}
