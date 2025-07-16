package init

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/unkeyed/unkey/go/cmd/cli/cli"
	"github.com/unkeyed/unkey/go/cmd/cli/config"
)

var Command = &cli.Command{
	Name:  "init",
	Usage: "Initialize configuration file for Unkey CLI",
	Description: `Initialize a configuration file to store default values for workspace ID, project ID, and context path.
This will create a unkey.json file in the specified directory.

EXAMPLES:
    # Create unkey.json in current directory
    unkey init
    
    # Create unkey.json in a specific directory
    unkey init --config=./test-docker`,
	Flags: []cli.Flag{
		cli.String("config", "Directory where unkey.json will be created", ".", "", false),
	},
	Action: run,
}

func run(ctx context.Context, cmd *cli.Command) error {
	configDir := cmd.String("config")
	configPath := config.GetConfigFilePath(configDir)

	fmt.Println("🚀 Unkey CLI Configuration Setup")
	fmt.Println("")

	// Check if config file already exists
	if config.ConfigExists(configDir) {
		fmt.Printf("⚠️  Configuration file already exists at: %s\n", configPath)
		if !promptConfirm("Do you want to overwrite it?") {
			fmt.Println("Configuration setup cancelled.")
			return nil
		}
		fmt.Println("")
	}

	// Create template config file
	if err := config.CreateTemplate(configDir); err != nil {
		return fmt.Errorf("failed to create config template: %w", err)
	}

	fmt.Printf("✅ Configuration template created at: %s\n", configPath)
	fmt.Println("")
	fmt.Println("Please replace the placeholder values with your actual values:")
	fmt.Println("")
	fmt.Println("After editing, you can run commands without flags:")
	fmt.Println("  unkey deploy")
	fmt.Println("")

	return nil
}

func readLine() string {
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func promptConfirm(message string) bool {
	fmt.Printf("%s (y/N): ", message)
	response := strings.ToLower(strings.TrimSpace(readLine()))
	return response == "y" || response == "yes"
}
