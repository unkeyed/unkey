package app

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/cmd/cli/commands"
)

// CLI represents our command line interface
type CLI struct {
	args []string
}

// New creates a new CLI instance
func New(args []string) *CLI {
	return &CLI{args: args}
}

// Run executes the CLI
func (c *CLI) Run(ctx context.Context) error {
	if len(c.args) < 2 {
		PrintUsage()
		return nil // Don't return error for help case
	}

	command := c.args[1]

	switch command {
	case "init":
		return commands.Init(c.args[2:])
	case "deploy":
		return commands.Deploy(ctx, c.args[2:])
	case "version":
		return commands.Version(ctx, c.args[2:])
	case "help", "-h", "--help":
		return c.runHelp()
	default:
		PrintUsage()
		return fmt.Errorf("unknown command: %s", command)
	}
}

// runHelp handles the help command
func (c *CLI) runHelp() error {
	if len(c.args) < 3 {
		// General help
		PrintUsage()
		return nil
	}

	// Help for specific command
	helpTopic := c.args[2]
	switch helpTopic {
	case "init":
		commands.PrintInitHelp()
	case "deploy":
		commands.PrintDeployHelp()
	case "version":
		commands.PrintVersionHelp()
	default:
		fmt.Printf("No help available for '%s'\n", helpTopic)
		PrintUsage()
	}

	return nil
}

// PrintUsage prints general usage information
func PrintUsage() {
	fmt.Println("unkey - Deploy and manage your API versions")
	fmt.Println("")
	fmt.Println("USAGE:")
	fmt.Println("    unkey <command> [flags]")
	fmt.Println("")
	fmt.Println("COMMANDS:")
	fmt.Println("    init       Initialize configuration file")
	fmt.Println("    deploy     Deploy a new version")
	fmt.Println("    version    Manage API versions")
	fmt.Println("    help       Show help information")
	fmt.Println("")
	fmt.Println("FLAGS:")
	fmt.Println("    -h, --help    Show help")
	fmt.Println("")
	fmt.Println("EXAMPLES:")
	fmt.Println("    unkey help")
	fmt.Println("    unkey help deploy")
	fmt.Println("    unkey init")
	fmt.Println("    unkey deploy --workspace-id=ws_123 --project-id=proj_456")
	fmt.Println("")
	fmt.Println("For detailed help on a command, use 'unkey help <command>'")
}
