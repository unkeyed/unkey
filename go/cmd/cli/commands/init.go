package commands

import (
	"fmt"
)

// Init handles the init command
func Init(args []string) error {
	fmt.Println("Init command - config file support coming soon!")
	fmt.Println("For now, use flags directly:")
	fmt.Println("")
	fmt.Println("Example:")
	fmt.Println("  unkey deploy \\")
	fmt.Println("    --workspace-id=ws_4QgQsKsKfdm3nGeC \\")
	fmt.Println("    --project-id=proj_9aiaks2dzl6mcywnxjf \\")
	fmt.Println("    --context=./demo_api")

	return nil
}

// PrintInitHelp prints detailed help for init command
func PrintInitHelp() {
	fmt.Println("unkey init - Initialize configuration")
	fmt.Println("")
	fmt.Println("USAGE:")
	fmt.Println("    unkey init [FLAGS]")
	fmt.Println("")
	fmt.Println("DESCRIPTION:")
	fmt.Println("    Initialize a configuration file to store default values for")
	fmt.Println("    workspace ID, project ID, and docker context path.")
	fmt.Println("")
	fmt.Println("FLAGS:")
	fmt.Println("    --config <path>    Configuration file path (default: ./unkey.json)")
	fmt.Println("")
	fmt.Println("EXAMPLES:")
	fmt.Println("    unkey init")
	fmt.Println("    unkey init --config=./my-project.json")
}
