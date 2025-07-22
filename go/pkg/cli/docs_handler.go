package cli

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// handleMDXGeneration checks if this is an MDX generation request and handles it
func (c *Command) handleMDXGeneration(ctx context.Context, args []string) (bool, error) {
	// Check if third argument is "mdx" (args[0] = program, args[1] = command, args[2] = "mdx")
	if len(args) < 3 || args[2] != "mdx" {
		return false, nil // Not an MDX request
	}

	// Find the target command using args[1] (the command name)
	commandName := args[1]
	targetCmd := c.findCommand(commandName)
	if targetCmd == nil {
		return true, fmt.Errorf("command '%s' not found", commandName)
	}

	// Create MDX command with output flag
	mdxCmd := &Command{
		Name:  "mdx",
		Usage: "Generate MDX documentation",
		Flags: []Flag{
			String("output", "Output file path (default: stdout)"),
		},
		Action: func(ctx context.Context, cmd *Command) error {
			return generateMDXForCommand(targetCmd, cmd.String("output"))
		},
	}

	// Run the MDX command with remaining args (skip program name, command name, and "mdx")
	return true, mdxCmd.Run(ctx, args[2:])
}

// findCommand recursively finds a command by name
func (c *Command) findCommand(name string) *Command {
	if c.Name == name {
		return c
	}

	for _, cmd := range c.Commands {
		if found := cmd.findCommand(name); found != nil {
			return found
		}
	}

	return nil
}

// generateMDXForCommand generates MDX documentation for the specified command
func generateMDXForCommand(cmd *Command, outputFile string) error {
	// Use command's Name and Usage as default frontmatter
	frontMatter := &FrontMatter{
		Title:       cases.Title(language.English).String(cmd.Name) + " Command",
		Description: cmd.Usage,
	}

	mdxContent, err := cmd.GenerateMDX(frontMatter)
	if err != nil {
		return fmt.Errorf("failed to generate MDX: %w", err)
	}

	if outputFile != "" {
		// Write to file
		if err := os.WriteFile(outputFile, []byte(mdxContent), 0644); err != nil {
			return fmt.Errorf("failed to write to file %s: %w", outputFile, err)
		}
		fmt.Printf("MDX documentation written to: %s\n", outputFile)
	} else {
		// Print to stdout
		fmt.Print(mdxContent)
	}

	return nil
}
