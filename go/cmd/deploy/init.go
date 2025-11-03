package deploy

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/unkeyed/unkey/go/pkg/cli"
)

const (
	// Init messages
	InitHeaderTitle     = "Unkey Configuration Setup"
	InitHeaderSeparator = "──────────────────────────────────────────────────"
)

func handleInit(cmd *cli.Command, ui *UI) error {
	configDir := cmd.String("config")
	if configDir == "" {
		configDir = "."
	}

	configPath := getConfigFilePath(configDir)
	force := cmd.Bool("force")

	fmt.Printf("%s\n", InitHeaderTitle)
	fmt.Printf("%s\n", InitHeaderSeparator)

	// Check if config file already exists
	if configExists(configDir) {
		fmt.Printf("Configuration file already exists at: %s\n", configPath)
		if !force && !promptConfirm("Do you want to overwrite it?") {
			fmt.Printf("Configuration setup cancelled.\n")
			return nil
		}
		fmt.Printf("\n")
	}

	// Interactive prompts for configuration
	fmt.Printf("Please provide the following configuration details:\n\n")

	fmt.Printf("Project ID: ")
	projectID := readLine()
	if projectID == "" {
		return fmt.Errorf("project ID is required")
	}

	fmt.Printf("Build context path [.]: ")
	context := readLine()
	if context == "" {
		context = "."
	}

	fmt.Printf("\n") // Add spacing before status messages

	// Create configuration with user input
	ui.Print("Creating configuration file")
	if err := createConfigWithValues(configDir, projectID, context); err != nil {
		ui.PrintError("Failed to create config file")
		return fmt.Errorf("failed to create config file: %w", err)
	}
	ui.PrintSuccess(fmt.Sprintf("Configuration file created at: %s", configPath))

	printInitNextSteps()
	return nil
}

func printInitNextSteps() {
	fmt.Printf("\n") // Consistent spacing
	fmt.Printf("Configuration complete!\n")
	fmt.Printf("\n")
	fmt.Printf("You can now deploy without any flags:\n")
	fmt.Printf("  unkey deploy\n")
	fmt.Printf("\n")
	fmt.Printf("Or override specific values:\n")
	fmt.Printf("  unkey deploy --project-id=proj_different\n")
	fmt.Printf("  unkey deploy --context=./other-app\n")
}

func promptConfirm(message string) bool {
	fmt.Printf("%s (y/N): ", message)
	response := strings.ToLower(strings.TrimSpace(readLine()))
	return response == "y" || response == "yes"
}

func readLine() string {
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}
