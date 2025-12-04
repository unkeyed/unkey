// Package cli provides a command-line interface framework for building CLI applications.
// It supports nested commands, various flag types, and structured error handling.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

var (
	ErrFlagNotFound  = errors.New("flag not found")
	ErrWrongFlagType = errors.New("wrong flag type")
	ErrNoArguments   = errors.New("no arguments provided")
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
	AcceptsArgs bool       // Whether this command accepts positional arguments
	commandPath string     // Full command path for MDX generation (e.g., "run api")

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

// RequireString returns the value of a string flag by name
// Panics if flag doesn't exist or isn't a StringFlag
func (c *Command) RequireString(name string) string {
	flag, ok := c.flagMap[name]
	if !ok {
		panic(c.newFlagNotFoundError(name))
	}

	sf, ok := flag.(*StringFlag)
	if !ok {
		panic(c.newWrongFlagTypeError(name, flag, "StringFlag"))
	}

	return sf.Value()
}

// Duration returns the value of a duration flag by name
// Returns 0 if flag doesn't exist or isn't a DurationFlag
func (c *Command) Duration(name string) time.Duration {
	if flag, ok := c.flagMap[name]; ok {
		if sf, ok := flag.(*DurationFlag); ok {
			return sf.Value()
		}
	}
	return time.Duration(0)
}

// RequireDuration returns the value of a duration flag by name
// Panics if flag doesn't exist or isn't a DurationFlag
func (c *Command) RequireDuration(name string) time.Duration {
	flag, ok := c.flagMap[name]
	if !ok {
		panic(c.newFlagNotFoundError(name))
	}

	sf, ok := flag.(*DurationFlag)
	if !ok {
		panic(c.newWrongFlagTypeError(name, flag, "DurationFlag"))
	}

	return sf.Value()
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

// RequireBool returns the value of a boolean flag by name
// Panics if flag doesn't exist or isn't a BoolFlag
func (c *Command) RequireBool(name string) bool {
	flag, ok := c.flagMap[name]
	if !ok {
		panic(c.newFlagNotFoundError(name))
	}

	bf, ok := flag.(*BoolFlag)
	if !ok {
		panic(c.newWrongFlagTypeError(name, flag, "BoolFlag"))
	}

	return bf.Value()
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

// RequireInt returns the value of an integer flag by name
// Panics if flag doesn't exist or isn't an IntFlag
func (c *Command) RequireInt(name string) int {
	flag, ok := c.flagMap[name]
	if !ok {
		panic(c.newFlagNotFoundError(name))
	}

	inf, ok := flag.(*IntFlag)
	if !ok {
		panic(c.newWrongFlagTypeError(name, flag, "IntFlag"))
	}

	return inf.Value()
}

// Int64 returns the value of an int64 flag by name
// Returns 0 if flag doesn't exist or isn't an Int64Flag
func (c *Command) Int64(name string) int64 {
	if flag, ok := c.flagMap[name]; ok {
		if i64f, ok := flag.(*Int64Flag); ok {
			return i64f.Value()
		}
	}
	return 0
}

// RequireInt64 returns the value of an int64 flag by name
// Panics if flag doesn't exist or isn't an Int64Flag
func (c *Command) RequireInt64(name string) int64 {
	flag, ok := c.flagMap[name]
	if !ok {
		panic(c.newFlagNotFoundError(name))
	}

	i64f, ok := flag.(*Int64Flag)
	if !ok {
		panic(c.newWrongFlagTypeError(name, flag, "Int64Flag"))
	}

	return i64f.Value()
}

// Float returns the value of a float flag by name
// Returns 0.0 if flag doesn't exist or isn't a FloatFlag
func (c *Command) Float(name string) float64 {
	if flag, ok := c.flagMap[name]; ok {
		if ff, ok := flag.(*FloatFlag); ok {
			return ff.Value()
		}
	}
	return 0.0
}

// RequireFloat returns the value of a float flag by name
// Panics if flag doesn't exist or isn't a FloatFlag
func (c *Command) RequireFloat(name string) float64 {
	flag, ok := c.flagMap[name]
	if !ok {
		panic(c.newFlagNotFoundError(name))
	}

	ff, ok := flag.(*FloatFlag)
	if !ok {
		panic(c.newWrongFlagTypeError(name, flag, "FloatFlag"))
	}

	return ff.Value()
}

// StringSlice returns the value of a string slice flag by name
// Returns empty slice if flag doesn't exist or isn't a StringSliceFlag
func (c *Command) StringSlice(name string) []string {
	if flag, ok := c.flagMap[name]; ok {
		if ssf, ok := flag.(*StringSliceFlag); ok {
			return ssf.Value()
		}
	}
	return []string{}
}

// RequireStringSlice returns the value of a string slice flag by name
// Panics if flag doesn't exist or isn't a StringSliceFlag
func (c *Command) RequireStringSlice(name string) []string {
	flag, ok := c.flagMap[name]
	if !ok {
		panic(c.newFlagNotFoundError(name))
	}

	ssf, ok := flag.(*StringSliceFlag)
	if !ok {
		panic(c.newWrongFlagTypeError(name, flag, "StringSliceFlag"))
	}

	return ssf.Value()
}

// Run executes the command with the given arguments (typically os.Args)
// This is the main entry point for CLI execution
func (c *Command) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return ErrNoArguments
	}
	// Handle MDX generation first
	if handled, err := c.handleMDXGeneration(args); handled {
		return err
	}
	// Parse arguments starting from index 1 (skip program name)
	return c.parse(ctx, args[1:])
}

// newFlagNotFoundError creates a structured error for missing flags
func (c *Command) newFlagNotFoundError(flagName string) error {
	return fmt.Errorf("%w: %q in command %q - available flags: %s",
		ErrFlagNotFound, flagName, c.Name, c.getAvailableFlags())
}

// newWrongFlagTypeError creates a structured error for type mismatches
func (c *Command) newWrongFlagTypeError(flagName string, flag Flag, expectedType string) error {
	actualType := c.getFlagType(flag)
	availableOfType := c.getFlagsByType(expectedType)

	return fmt.Errorf("%w: flag %q is %s, expected %s in command %q - available %s flags: %s",
		ErrWrongFlagType, flagName, actualType, expectedType, c.Name,
		strings.ToLower(expectedType), availableOfType)
}

// Helper functions for error reporting

// getFlagType returns a human-readable type name for a flag
func (c *Command) getFlagType(flag Flag) string {
	switch flag.(type) {
	case *StringFlag:
		return "StringFlag"
	case *BoolFlag:
		return "BoolFlag"
	case *IntFlag:
		return "IntFlag"
	case *Int64Flag:
		return "Int64Flag"
	case *FloatFlag:
		return "FloatFlag"
	case *StringSliceFlag:
		return "StringSliceFlag"
	default:
		return "unknown flag type"
	}
}

// getAvailableFlags returns a comma-separated list of all available flag names
func (c *Command) getAvailableFlags() string {
	if len(c.Flags) == 0 {
		return "none"
	}

	names := make([]string, len(c.Flags))
	for i, flag := range c.Flags {
		names[i] = flag.Name()
	}

	return strings.Join(names, ", ")
}

// getFlagsByType returns a comma-separated list of flags of the specified type
func (c *Command) getFlagsByType(flagType string) string {
	var matching []string

	for _, flag := range c.Flags {
		if c.getFlagType(flag) == flagType {
			matching = append(matching, flag.Name())
		}
	}

	if len(matching) == 0 {
		return "none"
	}

	return strings.Join(matching, ", ")
}

// ExitFunc allows dependency injection for testing
var ExitFunc = os.Exit

// Exit provides a clean way to exit with an error message and code
// This is a convenience function that prints the message and calls os.Exit
func Exit(message string, code int) error {
	fmt.Println(message)
	ExitFunc(code)
	return nil // unreachable but satisfies error interface
}
