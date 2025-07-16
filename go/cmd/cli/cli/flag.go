package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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

// baseFlag contains common fields and methods shared by all flag types
type baseFlag struct {
	name     string // Flag name
	usage    string // Help description
	envVar   string // Environment variable to check for default
	required bool   // Whether flag is mandatory
	set      bool   // Whether user explicitly provided this flag
}

// Name returns the flag name
func (b *baseFlag) Name() string { return b.name }

// Usage returns the flag's help text
func (b *baseFlag) Usage() string { return b.usage }

// Required returns whether this flag is mandatory
func (b *baseFlag) Required() bool { return b.required }

// IsSet returns whether the user explicitly provided this flag
func (b *baseFlag) IsSet() bool { return b.set }

// EnvVar returns the environment variable name for this flag
func (b *baseFlag) EnvVar() string { return b.envVar }

// StringFlag represents a string command line flag
type StringFlag struct {
	baseFlag
	value string // Current value
}

// Parse sets the flag value from a string
func (f *StringFlag) Parse(value string) error {
	f.value = value
	f.set = true
	return nil
}

// Value returns the current string value
func (f *StringFlag) Value() string { return f.value }

// BoolFlag represents a boolean command line flag
type BoolFlag struct {
	baseFlag
	value bool // Current value
}

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

// IntFlag represents an integer command line flag
type IntFlag struct {
	baseFlag
	value int // Current value
}

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

// FloatFlag represents a float64 command line flag
type FloatFlag struct {
	baseFlag
	value float64 // Current value
}

// Parse sets the flag value from a string
func (f *FloatFlag) Parse(value string) error {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("invalid float value: %s", value)
	}
	f.value = parsed
	f.set = true
	return nil
}

// Value returns the current float64 value
func (f *FloatFlag) Value() float64 { return f.value }

// StringSliceFlag represents a string slice command line flag
type StringSliceFlag struct {
	baseFlag
	value []string // Current value
}

// parseCommaSeparated splits a comma-separated string into a slice of trimmed non-empty strings
func (f *StringSliceFlag) parseCommaSeparated(value string) []string {
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// Parse sets the flag value from a string (comma-separated values)
func (f *StringSliceFlag) Parse(value string) error {
	f.value = f.parseCommaSeparated(value)
	f.set = true
	return nil
}

// Value returns the current string slice value
func (f *StringSliceFlag) Value() []string { return f.value }

// String creates a new string flag with environment variable support
// If envVar is provided and set, it will be used as the default value
func String(name, usage, defaultValue, envVar string, required bool) *StringFlag {
	flag := &StringFlag{
		baseFlag: baseFlag{
			name:     name,
			usage:    usage,
			envVar:   envVar,
			required: required,
		},
		value: defaultValue,
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
		baseFlag: baseFlag{
			name:     name,
			usage:    usage,
			envVar:   envVar,
			required: required,
		},
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
		baseFlag: baseFlag{
			name:     name,
			usage:    usage,
			envVar:   envVar,
			required: required,
		},
		value: defaultValue,
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

// Float creates a new float flag with environment variable support
func Float(name, usage string, defaultValue float64, envVar string, required bool) *FloatFlag {
	flag := &FloatFlag{
		baseFlag: baseFlag{
			name:     name,
			usage:    usage,
			envVar:   envVar,
			required: required,
		},
		value: defaultValue,
	}
	// Check environment variable for default value
	if envVar != "" {
		if envValue := os.Getenv(envVar); envValue != "" {
			if parsed, err := strconv.ParseFloat(envValue, 64); err == nil {
				flag.value = parsed
				flag.set = true // Mark as set since env var was found
			}
		}
	}
	return flag
}

// StringSlice creates a new string slice flag with environment variable support
func StringSlice(name, usage string, defaultValue []string, envVar string, required bool) *StringSliceFlag {
	flag := &StringSliceFlag{
		baseFlag: baseFlag{
			name:     name,
			usage:    usage,
			envVar:   envVar,
			required: required,
		},
		value: defaultValue,
	}
	// Check environment variable for default value
	if envVar != "" {
		if envValue := os.Getenv(envVar); envValue != "" {
			flag.value = flag.parseCommaSeparated(envValue)
			flag.set = true // Mark as set since env var was found
		}
	}
	return flag
}
