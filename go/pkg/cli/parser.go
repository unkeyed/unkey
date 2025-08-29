package cli

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// parse processes command line arguments and executes the appropriate action
// This handles flag parsing, subcommand routing, and help display
func (c *Command) parse(ctx context.Context, args []string) error {
	// Initialize flagMap if not already done
	c.initFlagMap()

	var commandArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Handle help flags first - these short-circuit normal processing
		if arg == "-h" || arg == "--help" || arg == "help" {
			c.showHelp()
			return nil
		}

		// Handle version flags - print version and exit
		if (arg == "-v" || arg == "--version") && c.Version != "" {
			fmt.Println(c.Version)
			return nil
		}

		// Handle "help <command>" pattern - show help for specific subcommand
		if arg == "help" && i+1 < len(args) {
			cmdName := args[i+1]
			for _, subcmd := range c.Commands {
				if subcmd.Name == cmdName {
					subcmd.parent = c
					subcmd.showHelp()
					return nil
				}
				// Check aliases
				if slices.Contains(subcmd.Aliases, cmdName) {
					subcmd.parent = c
					subcmd.showHelp()
					return nil
				}
			}
			return fmt.Errorf("unknown command: %s", cmdName)
		}

		// Check for subcommands (non-flag arguments)
		if !strings.HasPrefix(arg, "-") {
			// Look for matching subcommand
			for _, subcmd := range c.Commands {
				if subcmd.Name == arg {
					subcmd.parent = c
					return subcmd.parse(ctx, args[i+1:])
				}
				// Check aliases
				if slices.Contains(subcmd.Aliases, arg) {
					subcmd.parent = c
					return subcmd.parse(ctx, args[i+1:])
				}
			}

			if len(c.Commands) > 0 {
				availableCommands := make([]string, len(c.Commands))
				for j, cmd := range c.Commands {
					availableCommands[j] = cmd.Name
				}
				return fmt.Errorf("unknown command: %s\nAvailable commands: %s",
					arg, strings.Join(availableCommands, ", "))
			}

			// If no subcommands defined, treat as regular argument
			commandArgs = append(commandArgs, arg)
			continue
		}

		// Parse flags (arguments starting with -)
		if err := c.parseFlag(args, &i); err != nil {
			return err
		}
	}

	// Store parsed arguments
	c.args = commandArgs

	if len(commandArgs) > 0 {
		availableCommands := make([]string, len(c.Commands))
		for j, cmd := range c.Commands {
			availableCommands[j] = cmd.Name
		}
		availableFlags := c.getAvailableFlags()
		return fmt.Errorf("unexpected argument: %s\nAvailable flags: %s",
			commandArgs[0], availableFlags)
	}

	// Validate all required flags are present
	if err := c.validateRequiredFlags(); err != nil {
		fmt.Printf("Error: %v\n\n", err)
		c.showHelp()
		return err
	}

	// Execute action if present
	if c.Action != nil {
		return c.Action(ctx, c)
	}

	// No action defined - show help if we have subcommands
	if len(c.Commands) > 0 {
		c.showHelp()
	}

	return nil
}

// parseFlag handles parsing of a single flag and its value
// It modifies the index i to skip consumed arguments
func (c *Command) parseFlag(args []string, i *int) error {
	arg := args[*i]

	// Remove leading dashes properly
	var flagName string
	if strings.HasPrefix(arg, "--") {
		flagName = arg[2:] // Remove exactly "--"
	} else if strings.HasPrefix(arg, "-") {
		flagName = arg[1:] // Remove exactly "-"
	} else {
		return fmt.Errorf("invalid flag format: %s", arg)
	}

	// Handle --flag=value format
	var flagValue string
	var hasValue bool
	if eqIndex := strings.Index(flagName, "="); eqIndex != -1 {
		flagValue = flagName[eqIndex+1:]
		flagName = flagName[:eqIndex]
		hasValue = true
	}

	// Look up the flag
	flag, exists := c.flagMap[flagName]
	if !exists {
		availableFlags := c.getAvailableFlags()
		return fmt.Errorf("unknown flag: %s\nAvailable flags: %s", flagName, availableFlags)
	}

	// Handle boolean flags specially - they don't require values
	if bf, ok := flag.(*BoolFlag); ok {
		if hasValue {
			// --bool-flag=true/false format
			return bf.Parse(flagValue)
		} else {
			// --bool-flag format (implies true)
			return bf.Parse("")
		}
	}

	// For non-boolean flags, we need a value
	if !hasValue {
		// Value should be in next argument
		if *i+1 >= len(args) {
			return fmt.Errorf("flag %s requires a value", flagName)
		}
		*i++ // Move to next argument
		flagValue = args[*i]
	}

	// Parse the flag value
	return flag.Parse(flagValue)
}

func (c *Command) initFlagMap() {
	if c.flagMap != nil {
		return
	}
	c.flagMap = make(map[string]Flag)
	for _, flag := range c.Flags {
		c.flagMap[flag.Name()] = flag
	}
}

// validateRequiredFlags checks that all required flags have been set
func (c *Command) validateRequiredFlags() error {
	for _, flag := range c.Flags {
		if flag.Required() && !flag.HasValue() {
			return fmt.Errorf("required flag missing: %s", flag.Name())
		}
	}
	return nil
}
