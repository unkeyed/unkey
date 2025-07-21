package init

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/unkeyed/unkey/go/cmd/config"
	"github.com/unkeyed/unkey/go/cmd/deploy"
	"github.com/unkeyed/unkey/go/pkg/cli"
)

var Cmd = &cli.Command{
	Name:  "init",
	Usage: "Initialize configuration file",
	Description: `Initialize a configuration file to store default values for workspace ID, 
project ID, and build context path.

This will create a unkey.json file in the specified directory with template
values that you can customize for your project.

EXAMPLES:
    # Create unkey.json in current directory
    unkey init

    # Create unkey.json in a specific directory
    unkey init --config=./my-project

    # Force overwrite existing config
    unkey init --force`,
	Flags: []cli.Flag{
		cli.String("config", "Directory where unkey.json will be created", cli.Default(".")),
		cli.Bool("force", "Overwrite existing configuration file without prompting"),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	configDir := cmd.String("config")
	configPath := config.GetConfigFilePath(configDir)
	force := cmd.Bool("force")
	// TODO: Move this to a `/pkg` and make more generic
	ui := deploy.NewUI()

	fmt.Printf("Unkey Configuration Setup\n")
	fmt.Printf("──────────────────────────────────────────────────\n")

	// Check if config file already exists
	if config.ConfigExists(configDir) {
		fmt.Printf("Configuration file already exists at: %s\n", configPath)
		if !force && !promptConfirm("Do you want to overwrite it?") {
			fmt.Printf("Configuration setup cancelled.\n")
			return nil
		}
		fmt.Printf("\n")
	}

	// Create template config file
	ui.Print("Creating configuration template")
	if err := config.CreateTemplate(configDir); err != nil {
		ui.PrintError("Failed to create config template")
		return fmt.Errorf("failed to create config template: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Configuration template created at: %s", configPath))
	fmt.Printf("\n")

	printNextSteps()
	return nil
}

func printNextSteps() {
	fmt.Printf("Next Steps:\n")
	fmt.Printf("1. Edit the configuration file and replace placeholder values\n")
	fmt.Printf("2. Set your workspace ID and project ID\n")
	fmt.Printf("3. Customize the build context path if needed\n")
	fmt.Printf("\n")
	fmt.Printf("After configuration, you can deploy without flags:\n")
	fmt.Printf("  unkey deploy\n")
	fmt.Printf("\n")
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
