package cli

import (
	"context"
	"fmt"
	"os"
)

// Action represents a command handler function that receives context and the parsed command
type Action func(context.Context, *Command) error

// Command represents a CLI command with its configuration and runtime state
type Command struct {
	// Configuration
	Name        string     // Command name (e.g., "deploy", "version")
	Usage       string     // Short description shown in help
	Description string     // Longer description for detailed help
	Version     string     // Version string (only used for root command)
	Commands    []*Command // Subcommands
	Flags       []Flag     // Available flags for this command
	Action      Action     // Function to execute when command is run
	Aliases     []string   // Alternative names for this command

	// Runtime state (populated during parsing)
	args    []string        // Non-flag arguments passed to command
	flagMap map[string]Flag // Map for O(1) flag lookup
	parent  *Command        // Parent command (for building usage paths)
}

// Args returns the non-flag arguments passed to the command
// Example: "mycli deploy myapp" -> Args() returns ["myapp"]
func (c *Command) Args() []string {
	return c.args
}

// String returns the value of a string flag by name
// Returns empty string if flag doesn't exist or isn't a StringFlag
func (c *Command) String(name string) string {
	if flag, ok := c.flagMap[name]; ok {
		if sf, ok := flag.(*StringFlag); ok {
			return sf.Value()
		}
	}
	return ""
}

// Bool returns the value of a boolean flag by name
// Returns false if flag doesn't exist or isn't a BoolFlag
func (c *Command) Bool(name string) bool {
	if flag, ok := c.flagMap[name]; ok {
		if bf, ok := flag.(*BoolFlag); ok {
			return bf.Value()
		}
	}
	return false
}

// Int returns the value of an integer flag by name
// Returns 0 if flag doesn't exist or isn't an IntFlag
func (c *Command) Int(name string) int {
	if flag, ok := c.flagMap[name]; ok {
		if inf, ok := flag.(*IntFlag); ok {
			return inf.Value()
		}
	}
	return 0
}

// Run executes the command with the given arguments (typically os.Args)
// This is the main entry point for CLI execution
func (c *Command) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no arguments provided")
	}
	// Initialize flag lookup map for O(1) access
	c.flagMap = make(map[string]Flag)
	for _, flag := range c.Flags {
		c.flagMap[flag.Name()] = flag
	}
	// Parse arguments starting from index 1 (skip program name)
	return c.parse(ctx, args[1:])
}

// Exit provides a clean way to exit with an error message and code
// This is a convenience function that prints the message and calls os.Exit
func Exit(message string, code int) error {
	fmt.Println(message)
	os.Exit(code)
	return nil // unreachable but satisfies error interface
}
