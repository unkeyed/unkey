package cli

import (
	"fmt"
	"os"
	"strconv"
)

// Flag represents a command line flag interface
// All flag types must implement these methods
type Flag interface {
	Name() string             // The flag name (without dashes)
	Usage() string            // Help text describing the flag
	Required() bool           // Whether this flag is mandatory
	Parse(value string) error // Parse string value into the flag's type
	IsSet() bool              // Whether the flag was explicitly set by user
}

// StringFlag represents a string command line flag
type StringFlag struct {
	name     string // Flag name
	usage    string // Help description
	value    string // Current value
	envVar   string // Environment variable to check for default
	required bool   // Whether flag is mandatory
	set      bool   // Whether user explicitly provided this flag
}

// Name returns the flag name
func (f *StringFlag) Name() string { return f.name }

// Usage returns the flag's help text
func (f *StringFlag) Usage() string { return f.usage }

// Required returns whether this flag is mandatory
func (f *StringFlag) Required() bool { return f.required }

// IsSet returns whether the user explicitly provided this flag
func (f *StringFlag) IsSet() bool { return f.set }

// Parse sets the flag value from a string
func (f *StringFlag) Parse(value string) error {
	f.value = value
	f.set = true
	return nil
}

// Value returns the current string value
func (f *StringFlag) Value() string { return f.value }

// EnvVar returns the environment variable name for this flag
func (f *StringFlag) EnvVar() string { return f.envVar }

// BoolFlag represents a boolean command line flag
type BoolFlag struct {
	name     string // Flag name
	usage    string // Help description
	value    bool   // Current value
	envVar   string // Environment variable to check for default
	required bool   // Whether flag is mandatory
	set      bool   // Whether user explicitly provided this flag
}

// Name returns the flag name
func (f *BoolFlag) Name() string { return f.name }

// Usage returns the flag's help text
func (f *BoolFlag) Usage() string { return f.usage }

// Required returns whether this flag is mandatory
func (f *BoolFlag) Required() bool { return f.required }

// IsSet returns whether the user explicitly provided this flag
func (f *BoolFlag) IsSet() bool { return f.set }

// Parse sets the flag value from a string
// Empty string means the flag was provided without a value (--flag), which sets it to true
// Otherwise parses as boolean: "true", "false", "1", "0", etc.
func (f *BoolFlag) Parse(value string) error {
	if value == "" {
		f.value = true
		f.set = true
		return nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("invalid boolean value: %s", value)
	}
	f.value = parsed
	f.set = true
	return nil
}

// Value returns the current boolean value
func (f *BoolFlag) Value() bool { return f.value }

// EnvVar returns the environment variable name for this flag
func (f *BoolFlag) EnvVar() string { return f.envVar }

// IntFlag represents an integer command line flag
type IntFlag struct {
	name     string // Flag name
	usage    string // Help description
	value    int    // Current value
	envVar   string // Environment variable to check for default
	required bool   // Whether flag is mandatory
	set      bool   // Whether user explicitly provided this flag
}

// Name returns the flag name
func (f *IntFlag) Name() string { return f.name }

// Usage returns the flag's help text
func (f *IntFlag) Usage() string { return f.usage }

// Required returns whether this flag is mandatory
func (f *IntFlag) Required() bool { return f.required }

// IsSet returns whether the user explicitly provided this flag
func (f *IntFlag) IsSet() bool { return f.set }

// Parse sets the flag value from a string
func (f *IntFlag) Parse(value string) error {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("invalid integer value: %s", value)
	}
	f.value = parsed
	f.set = true
	return nil
}

// Value returns the current integer value
func (f *IntFlag) Value() int { return f.value }

// EnvVar returns the environment variable name for this flag
func (f *IntFlag) EnvVar() string { return f.envVar }

// String creates a new string flag with environment variable support
// If envVar is provided and set, it will be used as the default value
func String(name, usage, defaultValue, envVar string, required bool) *StringFlag {
	flag := &StringFlag{
		name:     name,
		usage:    usage,
		value:    defaultValue,
		envVar:   envVar,
		required: required,
	}

	// Check environment variable for default value
	if envVar != "" {
		if envValue := os.Getenv(envVar); envValue != "" {
			flag.value = envValue
			flag.set = true // Mark as set since env var was found
		}
	}

	return flag
}

// Bool creates a new boolean flag with environment variable support
func Bool(name, usage, envVar string, required bool) *BoolFlag {
	flag := &BoolFlag{
		name:     name,
		usage:    usage,
		envVar:   envVar,
		required: required,
	}

	// Check environment variable for default value
	if envVar != "" {
		if envValue := os.Getenv(envVar); envValue != "" {
			if parsed, err := strconv.ParseBool(envValue); err == nil {
				flag.value = parsed
				flag.set = true // Mark as set since env var was found
			}
		}
	}

	return flag
}

// Int creates a new integer flag with environment variable support
func Int(name, usage string, defaultValue int, envVar string, required bool) *IntFlag {
	flag := &IntFlag{
		name:     name,
		usage:    usage,
		value:    defaultValue,
		envVar:   envVar,
		required: required,
	}

	// Check environment variable for default value
	if envVar != "" {
		if envValue := os.Getenv(envVar); envValue != "" {
			if parsed, err := strconv.Atoi(envValue); err == nil {
				flag.value = parsed
				flag.set = true // Mark as set since env var was found
			}
		}
	}

	return flag
}
