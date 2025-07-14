package app

import (
	"context"
	"fmt"
	"os"

	"github.com/unkeyed/unkey/go/cmd/cli/commands"
)

// CLI represents our command line interface
type CLI struct {
	args    []string
	name    string
	usage   string
	version string
	env     map[string]string
}

// New creates a new CLI instance
func New(args []string, name, usage, version string) *CLI {
	env := map[string]string{
		"UNKEY_WORKSPACE_ID": os.Getenv("UNKEY_WORKSPACE_ID"),
		"UNKEY_PROJECT_ID":   os.Getenv("UNKEY_PROJECT_ID"),
	}

	return &CLI{
		args:    args,
		name:    name,
		usage:   usage,
		version: version,
		env:     env,
	}
}

// Run executes the CLI
func (c *CLI) Run(ctx context.Context) error {
	if len(c.args) < 2 {
		c.PrintUsage()
		return nil
	}

	command := c.args[1]
	switch command {
	case "init":
		return commands.Init(c.args[2:], c.env)
	case "deploy":
		return commands.Deploy(ctx, c.args[2:], c.env)
	case "version":
		return commands.Version(ctx, c.args[2:], c.env)
	case "help", "-h", "--help":
		return c.runHelp()
	case "-v", "--version":
		fmt.Println(c.version)
		return nil
	default:
		c.PrintUsage()
		return fmt.Errorf("unknown command: %s", command)
	}
}

// runHelp handles the help command
func (c *CLI) runHelp() error {
	if len(c.args) < 3 {
		c.PrintUsage()
		return nil
	}

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
		c.PrintUsage()
	}
	return nil
}

// PrintUsage prints general usage information
func (c *CLI) PrintUsage() {
	fmt.Printf("%s - %s\n", c.name, c.usage)
	fmt.Println("")
	fmt.Println("USAGE:")
	fmt.Printf("    %s <command> [flags]\n", c.name)
	fmt.Println("")
	fmt.Println("VERSION:")
	fmt.Printf("    %s\n", c.version)
	fmt.Println("")
	fmt.Println("COMMANDS:")
	fmt.Println("    init       Initialize configuration file")
	fmt.Println("    deploy     Deploy a new version")
	fmt.Println("    version    Manage API versions")
	fmt.Println("    help       Show help information")
	fmt.Println("")
	fmt.Println("FLAGS:")
	fmt.Println("    -h, --help       Show help")
	fmt.Println("    -v, --version    Show version")
	fmt.Println("")
	fmt.Println("ENVIRONMENT VARIABLES:")
	fmt.Println("    UNKEY_WORKSPACE_ID    Workspace ID (can be overridden by --workspace-id)")
	fmt.Println("    UNKEY_PROJECT_ID      Project ID (can be overridden by --project-id)")
	fmt.Println("    UNKEY_API_KEY         API key for authentication")
	fmt.Println("    UNKEY_BASE_URL        Base URL for API calls")
	fmt.Println("")
	fmt.Println("EXAMPLES:")
	fmt.Printf("    %s help\n", c.name)
	fmt.Printf("    %s help deploy\n", c.name)
	fmt.Printf("    %s init\n", c.name)
	fmt.Printf("    %s deploy --workspace-id=ws_123 --project-id=proj_456\n", c.name)
	fmt.Printf("    UNKEY_WORKSPACE_ID=ws_123 %s deploy\n", c.name)
	fmt.Printf("    UNKEY_WORKSPACE_ID=ws_123 UNKEY_PROJECT_ID=proj_456 %s deploy\n", c.name)
	fmt.Println("")
	fmt.Printf("For detailed help on a command, use '%s help <command>'\n", c.name)
}
