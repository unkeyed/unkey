package init

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/cmd/cli/cli"
)

var Command = &cli.Command{
	Name:  "init",
	Usage: "Initialize configuration file for Unkey CLI",
	Description: `Initialize a configuration file to store default values for workspace ID, project ID, and context path.
This will create a configuration file that can be used to avoid specifying common flags repeatedly.

EXAMPLES:
    # Create default config file (./unkey.json)
    unkey init
    
    # Create config file at custom location
    unkey init --config=./my-project.json
    
    # Initialize with specific values
    unkey init --workspace-id=ws_123 --project-id=proj_456`,
	Flags: []cli.Flag{
		cli.String("config", "Configuration file path", "./unkey.json", "", false),
		cli.String("workspace-id", "Default workspace ID to save in config", "", "", false),
		cli.String("project-id", "Default project ID to save in config", "", "", false),
		cli.String("context", "Default Docker context path to save in config", "", "", false),
	},
	Action: run,
}

func run(ctx context.Context, cmd *cli.Command) error {
	configPath := cmd.String("config")
	workspaceID := cmd.String("workspace-id")
	projectID := cmd.String("project-id")
	contextPath := cmd.String("context")

	fmt.Println("🚀 Unkey CLI Configuration Setup")
	fmt.Println("")

	// For now, just show what would be saved
	fmt.Println("Configuration file support coming soon!")
	fmt.Println("")
	fmt.Printf("Config file location: %s\n", configPath)

	if workspaceID != "" {
		fmt.Printf("Workspace ID: %s\n", workspaceID)
	}
	if projectID != "" {
		fmt.Printf("Project ID: %s\n", projectID)
	}
	if contextPath != "" {
		fmt.Printf("Context path: %s\n", contextPath)
	}

	fmt.Println("")
	fmt.Println("For now, use flags directly:")
	fmt.Println("")
	fmt.Println("Example:")
	fmt.Println("  unkey deploy \\")
	fmt.Println("    --workspace-id=ws_4QgQsKsKfdm3nGeC \\")
	fmt.Println("    --project-id=proj_9aiaks2dzl6mcywnxjf \\")
	fmt.Println("    --context=./demo_api")
	fmt.Println("")

	return nil
}
