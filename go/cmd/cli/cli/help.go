package cli

import (
	"fmt"
	"strings"
)

// showHelp displays comprehensive help information for the command
// This includes name, description, usage, subcommands, and flags
func (c *Command) showHelp() {
	// Command name and usage description
	fmt.Printf("NAME:\n   %s", c.Name)
	if c.Usage != "" {
		fmt.Printf(" - %s", c.Usage)
	}
	fmt.Printf("\n\n")

	// Extended description if available
	if c.Description != "" {
		fmt.Printf("DESCRIPTION:\n   %s\n\n", c.Description)
	}

	// Build and show usage line
	c.showUsageLine()

	// Show version for root command
	if c.Version != "" {
		fmt.Printf("VERSION:\n   %s\n\n", c.Version)
	}

	// Show available subcommands
	if len(c.Commands) > 0 {
		c.showCommands()
	}

	// Show command-specific flags if any exist
	if len(c.Flags) > 0 {
		fmt.Printf("OPTIONS:\n")
		for _, flag := range c.Flags {
			c.showFlag(flag)
		}
		fmt.Printf("\n")
	}

	// Always show global options
	fmt.Printf("GLOBAL OPTIONS:\n")
	fmt.Printf("   %-25s %s\n", "--help, -h", "show help")

	// Add version flag only for root command (commands with Version set)
	if c.Version != "" {
		fmt.Printf("   %-25s %s\n", "--version, -v", "print the version")
	}
	fmt.Printf("\n")
}

// showUsageLine displays the command usage syntax
func (c *Command) showUsageLine() {
	fmt.Printf("USAGE:\n   ")

	// Build full command path (parent commands + this command)
	path := c.buildCommandPath()
	fmt.Printf("%s", strings.Join(path, " "))

	// Add syntax indicators
	if len(c.Flags) > 0 {
		fmt.Printf(" [options]")
	}
	if len(c.Commands) > 0 {
		fmt.Printf(" [command]")
	}
	fmt.Printf("\n\n")
}

// buildCommandPath constructs the full command path from root to current command
func (c *Command) buildCommandPath() []string {
	var path []string

	// Walk up the parent chain to build full path
	cmd := c
	for cmd != nil {
		path = append([]string{cmd.Name}, path...)
		cmd = cmd.parent
	}
	return path
}

// showCommands displays all available subcommands in a formatted table
func (c *Command) showCommands() {
	fmt.Printf("COMMANDS:\n")

	// Find the longest command name for alignment
	maxLen := 0
	for _, cmd := range c.Commands {
		if len(cmd.Name) > maxLen {
			maxLen = len(cmd.Name)
		}
	}

	// Display each command with aliases
	for _, cmd := range c.Commands {
		name := cmd.Name
		if len(cmd.Aliases) > 0 {
			name += fmt.Sprintf(", %s", strings.Join(cmd.Aliases, ", "))
		}
		fmt.Printf("   %-*s %s\n", maxLen+10, name, cmd.Usage)
	}

	// Add built-in help command
	fmt.Printf("   %-*s %s\n", maxLen+10, "help, h", "Shows help for commands")
	fmt.Printf("\n")
}

// showFlag displays a single flag with proper formatting
func (c *Command) showFlag(flag Flag) {
	// Build flag name(s) - support both short and long forms
	name := fmt.Sprintf("--%s", flag.Name())
	if len(flag.Name()) == 1 {
		name = fmt.Sprintf("-%s, --%s", flag.Name(), flag.Name())
	}

	// Build usage description
	usage := flag.Usage()

	// Add required indicator
	if flag.Required() {
		usage += " (required)"
	}

	// Add environment variable info if available
	envVar := c.getEnvVar(flag)
	if envVar != "" {
		usage += fmt.Sprintf(" [$%s]", envVar)
	}

	// Display with consistent formatting
	fmt.Printf("   %-25s %s\n", name, usage)
}

// getEnvVar extracts environment variable name from flag if it supports it
func (c *Command) getEnvVar(flag Flag) string {
	switch f := flag.(type) {
	case *StringFlag:
		return f.EnvVar()
	case *BoolFlag:
		return f.EnvVar()
	case *IntFlag:
		return f.EnvVar()
	default:
		return ""
	}
}
