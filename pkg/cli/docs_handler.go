package cli

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	ErrCommandNotFound = errors.New("command not found")
	ErrMDXGeneration   = errors.New("failed to generate MDX")
)

// handleMDXGeneration checks if this is an MDX generation request and handles it
// Update handleMDXGeneration to pass command path to GenerateMDX
func (c *Command) handleMDXGeneration(args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if c == nil {
		return false, ErrCommandNil
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
		return true, fmt.Errorf("%w: '%s'", ErrCommandNotFound, strings.Join(commandPath, " "))
	}

	// Generate appropriate frontmatter based on command path
	frontMatter := c.generateFrontMatterFromPath(commandPath, targetCmd)

	// Set the command path on the target command before generating MDX
	targetCmd.commandPath = strings.Join(commandPath, " ")

	// Generate and output MDX
	mdxContent, err := targetCmd.GenerateMDX(frontMatter)
	if err != nil {
		return true, fmt.Errorf("%w: %v", ErrMDXGeneration, err)
	}

	fmt.Print(mdxContent)
	return true, nil
}

// findCommandByPath traverses the command tree to find the target command
func (c *Command) findCommandByPath(path []string) *Command {
	if c == nil {
		return nil
	}
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
		if cmd == nil {
			continue
		}
		if cmd.Name == searchPath[0] {
			return cmd.findCommandByPath(searchPath[1:])
		}
	}

	return nil
}

// generateFrontMatterFromPath creates appropriate frontmatter based on command path
func (c *Command) generateFrontMatterFromPath(path []string, targetCmd *Command) *FrontMatter {
	if c == nil || targetCmd == nil {
		return &FrontMatter{
			Title:       "Unknown Command",
			Description: "No description available",
		}
	}

	caser := cases.Title(language.English)

	if len(path) == 0 {
		return &FrontMatter{
			Title:       caser.String(c.Name),
			Description: c.Usage,
		}
	}

	// Build title from command path
	title := ""
	if len(path) > 0 && path[len(path)-1] != "" {
		title = caser.String(path[len(path)-1])
	}

	description := targetCmd.Usage
	if description == "" {
		description = "No description available"
	}

	return &FrontMatter{
		Title:       title,
		Description: description,
	}
}
