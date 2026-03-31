package cli

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
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
	c.showExamples()
}

// showHeader displays the command name and description
func (c *Command) showHeader() {
	fmt.Printf("NAME:\n   %s", c.Name)
	if c.Usage != "" {
		fmt.Printf(" - %s", c.Usage)
	}
	fmt.Printf("\n\n")

	if c.Description != "" {
		const indent = "   "
		termWidth := getTerminalWidth()
		descWidth := termWidth - len(indent)
		if descWidth < 30 {
			descWidth = 77
		}

		fmt.Printf("DESCRIPTION:\n")
		for _, line := range strings.Split(c.Description, "\n") {
			if line == "" {
				fmt.Println()
			} else {
				for _, wrapped := range wrapText(line, descWidth) {
					fmt.Printf("%s%s\n", indent, wrapped)
				}
			}
		}
		fmt.Println()
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

// showExamples displays example invocations if any are defined
func (c *Command) showExamples() {
	if len(c.Examples) == 0 {
		return
	}

	fmt.Printf("EXAMPLES:\n")
	for _, example := range c.Examples {
		fmt.Printf("   %s\n", example)
	}
	fmt.Println()
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

	const flagColWidth = 25
	prefix := fmt.Sprintf("   %-*s ", flagColWidth, flagName)

	// If the flag name is longer than the column, put description on next line
	if len(flagName) > flagColWidth {
		fmt.Printf("   %s\n", flagName)
		prefix = strings.Repeat(" ", flagColWidth+4)
	}

	termWidth := getTerminalWidth()
	descWidth := termWidth - len(prefix)
	if descWidth < 30 {
		// Terminal too narrow for wrapping, just print it
		fmt.Printf("%s%s\n", prefix, usage)
		return
	}

	lines := wrapText(usage, descWidth)
	for i, line := range lines {
		if i == 0 {
			fmt.Printf("%s%s\n", prefix, line)
		} else {
			fmt.Printf("%s%s\n", strings.Repeat(" ", len(prefix)), line)
		}
	}
}

// getTerminalWidth returns the current terminal width, defaulting to 80.
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 80
	}
	return width
}

// wrapText breaks text into lines of at most maxWidth characters,
// splitting on word boundaries.
func wrapText(text string, maxWidth int) []string {
	if len(text) <= maxWidth {
		return []string{text}
	}

	var lines []string
	for len(text) > 0 {
		if len(text) <= maxWidth {
			lines = append(lines, text)
			break
		}

		// Find the last space within maxWidth
		cut := strings.LastIndex(text[:maxWidth], " ")
		if cut <= 0 {
			// No space found, force cut at maxWidth
			cut = maxWidth
		}

		lines = append(lines, text[:cut])
		text = strings.TrimLeft(text[cut:], " ")
	}

	return lines
}

// buildFlagName constructs the flag name string with proper formatting
func (c *Command) buildFlagName(flag Flag) string {
	name := flag.Name()

	var prefix string
	if len(name) == 1 {
		prefix = fmt.Sprintf("-%s", name)
	} else {
		prefix = fmt.Sprintf("--%s", name)
	}

	typeLabel := flagTypeLabel(flag)
	if typeLabel != "" {
		return fmt.Sprintf("%s %s", prefix, typeLabel)
	}
	return prefix
}

// flagTypeLabel returns a human-readable type hint for a flag.
func flagTypeLabel(flag Flag) string {
	switch flag.(type) {
	case *StringFlag:
		return "string"
	case *BoolFlag:
		return "bool"
	case *IntFlag:
		return "int"
	case *Int64Flag:
		return "int"
	case *FloatFlag:
		return "float"
	case *StringSliceFlag:
		return "strings"
	case *DurationFlag:
		return "duration"
	default:
		return ""
	}
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
			return f.Value().String()
		}
	}
	return ""
}
