package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// handleMDXGeneration checks if this is an MDX generation request and handles it
func (c *Command) handleMDXGeneration(ctx context.Context, args []string) (bool, error) {
	// Check if the last argument is "mdx"
	if len(args) == 0 {
		return false, nil
	}

	lastArg := args[len(args)-1]
	if lastArg != "mdx" {
		return false, nil // Not an MDX request
	}

	// Build the command path from args (excluding program name and "mdx")
	commandPath := args[1 : len(args)-1] // Skip program name and "mdx"

	// Find the target command by traversing the command tree
	targetCmd := c.findCommandByPath(commandPath)
	if targetCmd == nil {
		return true, fmt.Errorf("command '%s' not found", strings.Join(commandPath, " "))
	}

	// Generate appropriate frontmatter based on command path
	frontMatter := c.generateFrontMatterFromPath(commandPath, targetCmd)

	// Generate and output MDX
	mdxContent, err := targetCmd.GenerateMDX(frontMatter)
	if err != nil {
		return true, fmt.Errorf("failed to generate MDX: %w", err)
	}

	fmt.Print(mdxContent)
	return true, nil
}

// findCommandByPath traverses the command tree to find the target command
func (c *Command) findCommandByPath(path []string) *Command {
	if len(path) == 0 {
		return c
	}

	// Skip the first element if it matches current command name
	searchPath := path
	if len(path) > 0 && path[0] == c.Name {
		searchPath = path[1:]
	}

	if len(searchPath) == 0 {
		return c
	}

	// Find the next command in the path
	for _, cmd := range c.Commands {
		if cmd.Name == searchPath[0] {
			return cmd.findCommandByPath(searchPath[1:])
		}
	}

	return nil
}

// generateFrontMatterFromPath creates appropriate frontmatter based on command path
func (c *Command) generateFrontMatterFromPath(path []string, targetCmd *Command) *FrontMatter {
	if len(path) == 0 {
		return &FrontMatter{
			Title:       cases.Title(language.English).String(c.Name) + " Command",
			Description: c.Usage,
		}
	}

	// Build title from command path
	var titleParts []string
	for _, part := range path {
		titleParts = append(titleParts, cases.Title(language.English).String(part))
	}

	title := strings.Join(titleParts, " ") + " Command"
	description := targetCmd.Usage

	return &FrontMatter{
		Title:       title,
		Description: description,
	}
}

// findCommand recursively finds a command by name (kept for backward compatibility)
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

// generateMDXForCommand generates MDX documentation for the specified command (kept for backward compatibility)
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
