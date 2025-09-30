package cli

import (
	"fmt"
	"strings"
)

// showHelp displays comprehensive help information for the command
// This includes name, description, usage, subcommands, and flags
func (c *Command) showHelp() {
	c.showHeader()
	c.showUsageLine()
	c.showVersion()
	c.showCommands()
	c.showFlags()
	c.showGlobalOptions()
}

// showHeader displays the command name and description
func (c *Command) showHeader() {
	fmt.Printf("NAME:\n   %s", c.Name)
	if c.Usage != "" {
		fmt.Printf(" - %s", c.Usage)
	}
	fmt.Printf("\n\n")

	if c.Description != "" {
		fmt.Printf("DESCRIPTION:\n   %s\n\n", c.Description)
	}
}

// showUsageLine displays the command usage syntax
func (c *Command) showUsageLine() {
	fmt.Printf("USAGE:\n   ")

	path := c.buildCommandPath()
	fmt.Printf("%s", strings.Join(path, " "))

	// Add syntax indicators based on what's available
	if len(c.Flags) > 0 {
		fmt.Printf(" [options]")
	}
	if len(c.Commands) > 0 {
		fmt.Printf(" [command]")
	}

	// Add arguments indicator if this command has an action but no subcommands
	if c.Action != nil && len(c.Commands) == 0 {
		fmt.Printf(" [arguments...]")
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

// showVersion displays version information for root commands
func (c *Command) showVersion() {
	if c.Version != "" {
		fmt.Printf("VERSION:\n   %s\n\n", c.Version)
	}
}

// showCommands displays all available subcommands in a formatted table
func (c *Command) showCommands() {
	if len(c.Commands) == 0 {
		return
	}

	fmt.Printf("COMMANDS:\n")

	// Calculate alignment for clean formatting
	maxLen := c.calculateMaxCommandLength()

	// Display each command with aliases
	for _, cmd := range c.Commands {
		c.showSingleCommand(cmd, maxLen)
	}

	// Add built-in help command
	fmt.Printf("   %-*s %s\n", maxLen+10, "help, h", "Shows help for commands")
	fmt.Printf("\n")
}

// calculateMaxCommandLength finds the longest command name for alignment
func (c *Command) calculateMaxCommandLength() int {
	maxLen := 0
	for _, cmd := range c.Commands {
		nameWithAliases := cmd.Name
		if len(cmd.Aliases) > 0 {
			nameWithAliases += fmt.Sprintf(", %s", strings.Join(cmd.Aliases, ", "))
		}
		if len(nameWithAliases) > maxLen {
			maxLen = len(nameWithAliases)
		}
	}
	// Also consider the help command length
	helpLen := len("help, h")
	if helpLen > maxLen {
		maxLen = helpLen
	}
	return maxLen
}

// showSingleCommand displays one command with proper formatting
func (c *Command) showSingleCommand(cmd *Command, maxLen int) {
	name := cmd.Name
	if len(cmd.Aliases) > 0 {
		name += fmt.Sprintf(", %s", strings.Join(cmd.Aliases, ", "))
	}
	fmt.Printf("   %-*s %s\n", maxLen+10, name, cmd.Usage)
}

// showFlags displays command-specific flags if any exist
func (c *Command) showFlags() {
	if len(c.Flags) == 0 {
		return
	}

	fmt.Printf("OPTIONS:\n")
	for _, flag := range c.Flags {
		c.showFlag(flag)
	}
	fmt.Printf("\n")
}

// showGlobalOptions displays help and version flags
func (c *Command) showGlobalOptions() {
	fmt.Printf("GLOBAL OPTIONS:\n")
	fmt.Printf("   %-25s %s\n", "--help, -h", "show help")

	// Add version flag only for root command (commands with Version set)
	if c.Version != "" {
		fmt.Printf("   %-25s %s\n", "--version, -v", "print the version")
	}
	fmt.Printf("\n")
}

// showFlag displays a single flag with proper formatting
func (c *Command) showFlag(flag Flag) {
	flagName := c.buildFlagName(flag)
	usage := c.buildFlagUsage(flag)

	fmt.Printf("   %-25s %s\n", flagName, usage)
}

// buildFlagName constructs the flag name string with proper formatting
func (c *Command) buildFlagName(flag Flag) string {
	name := flag.Name()

	// For single character flags, show both short and long form
	if len(name) == 1 {
		return fmt.Sprintf("-%s", name)
	}

	// For multi-character flags, just show long form
	return fmt.Sprintf("--%s", name)
}

// buildFlagUsage constructs the complete usage string for a flag
func (c *Command) buildFlagUsage(flag Flag) string {
	usage := flag.Usage()

	// Add required indicator
	if flag.Required() {
		usage += " (required)"
	}

	// Add environment variable info if available
	if envVar := c.getEnvVar(flag); envVar != "" {
		usage += fmt.Sprintf(" [$%s]", envVar)
	}

	// Add default value if present and not required
	if !flag.Required() {
		if defaultVal := c.getDefaultValue(flag); defaultVal != "" {
			usage += fmt.Sprintf(" (default: %s)", defaultVal)
		}
	}

	return usage
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
	case *FloatFlag:
		return f.EnvVar()
	case *StringSliceFlag:
		return f.EnvVar()
	case *DurationFlag:
		return f.EnvVar()
	default:
		return ""
	}
}

// getDefaultValue extracts and formats the default value from a flag
func (c *Command) getDefaultValue(flag Flag) string {
	switch f := flag.(type) {
	case *StringFlag:
		if val := f.Value(); val != "" {
			return fmt.Sprintf(`"%s"`, val)
		}
	case *BoolFlag:
		if f.HasValue() {
			return fmt.Sprintf("%t", f.Value())
		}
	case *IntFlag:
		if f.HasValue() {
			return fmt.Sprintf("%d", f.Value())
		}
	case *FloatFlag:
		if f.HasValue() {
			return fmt.Sprintf("%.2f", f.Value())
		}
	case *StringSliceFlag:
		if val := f.Value(); len(val) > 0 {
			return fmt.Sprintf(`["%s"]`, strings.Join(val, `", "`))
		}
	case *DurationFlag:
		if f.HasValue() {
			return fmt.Sprintf("%s", f.Value())
		}
	}
	return ""
}
