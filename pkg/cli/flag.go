package cli

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	ErrValidationFailed  = errors.New("validation failed")
	ErrInvalidBoolValue  = errors.New("invalid boolean value")
	ErrInvalidIntValue   = errors.New("invalid integer value")
	ErrInvalidInt64Value = errors.New("invalid int64 value")
	ErrInvalidFloatValue = errors.New("invalid float value")
)

type Flag interface {
	Name() string             // The flag name (without dashes)
	Usage() string            // Help text describing the flag
	Required() bool           // Whether this flag is mandatory
	Parse(value string) error // Parse string value into the flag's type
	IsSet() bool              // Whether the flag was explicitly set by user
	HasValue() bool           // Whether the flag has any value (from user, env, or default)
}

// ValidateFunc represents a function that validates a flag value
type ValidateFunc func(value string) error

// baseFlag contains common fields and methods shared by all flag types
type baseFlag struct {
	name     string       // Flag name
	usage    string       // Help description
	envVar   string       // Environment variable to check for default
	required bool         // Whether flag is mandatory
	set      bool         // Whether user explicitly provided this flag
	validate ValidateFunc // Optional validation function
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
	value       string // Current value
	hasEnvValue bool   // Track if value came from environment
}

// Parse sets the flag value from a string
func (f *StringFlag) Parse(value string) error {
	// Run validation if provided
	if f.validate != nil {
		if err := f.validate(value); err != nil {
			return newValidationError(f.name, err)
		}
	}
	f.value = value
	f.set = true
	return nil
}

// Value returns the current string value
func (f *StringFlag) Value() string { return f.value }

// HasValue returns true if the flag has any non-empty value or came from environment
func (f *StringFlag) HasValue() bool { return f.value != "" || f.hasEnvValue }

// DurationFlag represents a duration command line flag
type DurationFlag struct {
	baseFlag
	value       time.Duration // Current value
	hasEnvValue bool          // Track if value came from environment
}

// Parse sets the flag value from a string
func (f *DurationFlag) Parse(value string) error {
	// Run validation if provided
	if f.validate != nil {
		if err := f.validate(value); err != nil {
			return newValidationError(f.name, err)
		}
	}
	var err error
	f.value, err = time.ParseDuration(value)
	if err != nil {
		return newValidationError(f.name, err)
	}
	f.set = true
	return nil
}

// Value returns the current string value
func (f *DurationFlag) Value() time.Duration { return f.value }

// HasValue returns true if the flag has any non-empty value or came from environment
func (f *DurationFlag) HasValue() bool { return f.value != 0 || f.hasEnvValue }

// BoolFlag represents a boolean command line flag
type BoolFlag struct {
	baseFlag
	value       bool
	hasEnvValue bool // Track if value came from environment
}

// Parse sets the flag value from a string
// Empty string means the flag was provided without a value (--flag), which sets it to true
// Otherwise parses as boolean: "true", "false", "1", "0", etc.
func (f *BoolFlag) Parse(value string) error {
	var parsed bool
	if value == "" {
		parsed = true
	} else {
		var err error
		parsed, err = strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidBoolValue, value)
		}
	}

	// Run validation if provided - validate the original input string, not the parsed boolean
	if f.validate != nil {
		if err := f.validate(value); err != nil {
			return newValidationError(f.name, err)
		}
	}

	f.value = parsed
	f.set = true
	return nil
}

// Value returns the current boolean value
func (f *BoolFlag) Value() bool { return f.value }

// HasValue returns true - boolean flags always have a meaningful value
func (f *BoolFlag) HasValue() bool { return true }

// IntFlag represents an integer command line flag
type IntFlag struct {
	baseFlag
	value       int  // Current value
	hasEnvValue bool // Track if value came from environment
}

// Parse sets the flag value from a string
func (f *IntFlag) Parse(value string) error {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidIntValue, value)
	}

	// Run validation if provided
	if f.validate != nil {
		if err := f.validate(value); err != nil {
			return newValidationError(f.name, err)
		}
	}

	f.value = parsed
	f.set = true
	return nil
}

// Value returns the current integer value
func (f *IntFlag) Value() int { return f.value }

// HasValue returns true if the flag has a non-zero value or came from environment
func (f *IntFlag) HasValue() bool { return f.value != 0 || f.hasEnvValue }

// Int64Flag represents an int64 command line flag
type Int64Flag struct {
	baseFlag
	value       int64 // Current value
	hasEnvValue bool  // Track if value came from environment
}

// Parse sets the flag value from a string
func (f *Int64Flag) Parse(value string) error {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidInt64Value, value)
	}

	// Run validation if provided
	if f.validate != nil {
		if err := f.validate(value); err != nil {
			return newValidationError(f.name, err)
		}
	}

	f.value = parsed
	f.set = true
	return nil
}

// Value returns the current int64 value
func (f *Int64Flag) Value() int64 { return f.value }

// HasValue returns true if the flag has a non-zero value or came from environment
func (f *Int64Flag) HasValue() bool { return f.value != 0 || f.hasEnvValue }

// FloatFlag represents a float64 command line flag
type FloatFlag struct {
	baseFlag
	value       float64 // Current value
	hasEnvValue bool    // Track if value came from environment
}

// Parse sets the flag value from a string
func (f *FloatFlag) Parse(value string) error {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidFloatValue, value)
	}

	// Run validation if provided
	if f.validate != nil {
		if err := f.validate(value); err != nil {
			return newValidationError(f.name, err)
		}
	}

	f.value = parsed
	f.set = true
	return nil
}

// Value returns the current float64 value
func (f *FloatFlag) Value() float64 { return f.value }

// HasValue returns true if the flag has a non-zero value or came from environment
func (f *FloatFlag) HasValue() bool { return f.value != 0.0 || f.hasEnvValue }

// StringSliceFlag represents a string slice command line flag
type StringSliceFlag struct {
	baseFlag
	value       []string // Current value
	hasEnvValue bool     // Track if value came from environment
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
	parsed := f.parseCommaSeparated(value)

	// Run validation if provided (validate the original comma-separated string)
	if f.validate != nil {
		if err := f.validate(value); err != nil {
			return newValidationError(f.name, err)
		}
	}

	f.value = parsed
	f.set = true
	return nil
}

// Value returns the current string slice value
func (f *StringSliceFlag) Value() []string { return f.value }

// HasValue returns true if the slice is not empty or came from environment
func (f *StringSliceFlag) HasValue() bool { return len(f.value) > 0 || f.hasEnvValue }

// FlagOption represents an option for configuring flags
type FlagOption func(flag any)

// Required marks a flag as mandatory
func Required() FlagOption {
	return func(f any) {
		switch flag := f.(type) {
		case *StringFlag:
			flag.required = true
		case *BoolFlag:
			flag.required = true
		case *IntFlag:
			flag.required = true
		case *Int64Flag:
			flag.required = true
		case *FloatFlag:
			flag.required = true
		case *StringSliceFlag:
			flag.required = true
		case *DurationFlag:
			flag.required = true
		}
	}
}

// EnvVar sets an environment variable to check for default values
func EnvVar(envVar string) FlagOption {
	return func(f any) {
		switch flag := f.(type) {
		case *StringFlag:
			flag.envVar = envVar
		case *BoolFlag:
			flag.envVar = envVar
		case *IntFlag:
			flag.envVar = envVar
		case *Int64Flag:
			flag.envVar = envVar
		case *FloatFlag:
			flag.envVar = envVar
		case *StringSliceFlag:
			flag.envVar = envVar
		case *DurationFlag:
			flag.envVar = envVar
		}
	}
}

// Validate sets a validation function for the flag
func Validate(fn ValidateFunc) FlagOption {
	return func(f any) {
		switch flag := f.(type) {
		case *StringFlag:
			flag.validate = fn
		case *BoolFlag:
			flag.validate = fn
		case *IntFlag:
			flag.validate = fn
		case *Int64Flag:
			flag.validate = fn
		case *FloatFlag:
			flag.validate = fn
		case *StringSliceFlag:
			flag.validate = fn
		case *DurationFlag:
			flag.validate = fn
		}
	}
}

// Default sets a default value for the flag
func Default(value any) FlagOption {
	return func(f any) {
		var err error
		switch flag := f.(type) {
		case *StringFlag:
			if v, ok := value.(string); ok {
				flag.value = v
			} else {
				err = fmt.Errorf("default value for string flag '%s' must be string, got %T", flag.name, value)
			}
		case *BoolFlag:
			if v, ok := value.(bool); ok {
				flag.value = v
			} else {
				err = fmt.Errorf("default value for bool flag '%s' must be bool, got %T", flag.name, value)
			}
		case *IntFlag:
			if v, ok := value.(int); ok {
				flag.value = v
			} else {
				err = fmt.Errorf("default value for int flag '%s' must be int, got %T", flag.name, value)
			}
		case *Int64Flag:
			if v, ok := value.(int64); ok {
				flag.value = v
			} else {
				err = fmt.Errorf("default value for int64 flag '%s' must be int64, got %T", flag.name, value)
			}
		case *FloatFlag:
			if v, ok := value.(float64); ok {
				flag.value = v
			} else {
				err = fmt.Errorf("default value for float flag '%s' must be float64, got %T", flag.name, value)
			}
		case *StringSliceFlag:
			if v, ok := value.([]string); ok {
				flag.value = v
			} else {
				err = fmt.Errorf("default value for string slice flag '%s' must be []string, got %T", flag.name, value)
			}
		case *DurationFlag:
			if v, ok := value.(time.Duration); ok {
				flag.value = v
			} else {
				err = fmt.Errorf("default value for duration flag '%s' must be time.Duration, got %T", flag.name, value)
			}
		}

		if err != nil {
			_ = Exit(fmt.Sprintf("Configuration error: %s", err.Error()), 1)
		}
	}
}

// String creates a new string flag with optional configuration
func String(name, usage string, opts ...FlagOption) *StringFlag {
	// nolint: exhaustruct
	flag := &StringFlag{
		// nolint: exhaustruct
		baseFlag: baseFlag{
			name:     name,
			usage:    usage,
			required: false, // Default to not required
		},
		value: "", // Default to empty string
	}

	// Apply options
	for _, opt := range opts {
		opt(flag)
	}

	// Check environment variable for default value if specified
	if flag.envVar != "" {
		if envValue := os.Getenv(flag.envVar); envValue != "" {
			// Apply validation to environment variable values
			if flag.validate != nil {
				if err := flag.validate(envValue); err != nil {
					_ = Exit(fmt.Sprintf("Environment variable error: validation failed for %s=%q: %v",
						flag.envVar, envValue, err), 1)
				}
			}
			flag.value = envValue
			flag.hasEnvValue = true
			// Don't mark as explicitly set - this is from environment
		}
	}

	return flag
}

// Duration creates a new duration flag with optional configuration
func Duration(name, usage string, opts ...FlagOption) *DurationFlag {
	// nolint: exhaustruct
	flag := &DurationFlag{
		// nolint: exhaustruct
		baseFlag: baseFlag{
			name:     name,
			usage:    usage,
			required: false, // Default to not required
		},
		value: time.Duration(0), // Default to false
	}

	// Apply options
	for _, opt := range opts {
		opt(flag)
	}

	// Check environment variable for default value if specified
	if flag.envVar != "" {
		if envValue := os.Getenv(flag.envVar); envValue != "" {
			parsed, err := time.ParseDuration(envValue)
			if err != nil {
				_ = Exit(fmt.Sprintf("Environment variable error: invalid duration value in %s=%q: %v",
					flag.envVar, envValue, err), 1)
			}
			// Apply validation to environment variable values
			if flag.validate != nil {
				if err := flag.validate(envValue); err != nil {
					_ = Exit(fmt.Sprintf("Environment variable error: validation failed for %s=%q: %v",
						flag.envVar, envValue, err), 1)
				}
			}
			flag.value = parsed
			flag.hasEnvValue = true
			// Don't mark as explicitly set - this is from environment
		}
	}
	return flag
}

// Bool creates a new boolean flag with optional configuration
func Bool(name, usage string, opts ...FlagOption) *BoolFlag {
	// nolint: exhaustruct
	flag := &BoolFlag{
		// nolint: exhaustruct
		baseFlag: baseFlag{
			name:     name,
			usage:    usage,
			required: false, // Default to not required
		},
		value: false, // Default to false
	}

	// Apply options
	for _, opt := range opts {
		opt(flag)
	}

	// Check environment variable for default value if specified
	if flag.envVar != "" {
		if envValue := os.Getenv(flag.envVar); envValue != "" {
			parsed, err := strconv.ParseBool(envValue)
			if err != nil {
				_ = Exit(fmt.Sprintf("Environment variable error: invalid boolean value in %s=%q: %v",
					flag.envVar, envValue, err), 1)
			}
			// Apply validation to environment variable values
			if flag.validate != nil {
				if err := flag.validate(envValue); err != nil {
					_ = Exit(fmt.Sprintf("Environment variable error: validation failed for %s=%q: %v",
						flag.envVar, envValue, err), 1)
				}
			}
			flag.value = parsed
			flag.hasEnvValue = true
			// Don't mark as explicitly set - this is from environment
		}
	}
	return flag
}

// Int creates a new integer flag with optional configuration
func Int(name, usage string, opts ...FlagOption) *IntFlag {
	// nolint: exhaustruct
	flag := &IntFlag{
		// nolint: exhaustruct
		baseFlag: baseFlag{
			name:     name,
			usage:    usage,
			required: false, // Default to not required
		},
		value: 0, // Default to zero
	}

	// Apply options
	for _, opt := range opts {
		opt(flag)
	}

	// Check environment variable for default value if specified
	if flag.envVar != "" {
		if envValue := os.Getenv(flag.envVar); envValue != "" {
			parsed, err := strconv.Atoi(envValue)
			if err != nil {
				_ = Exit(fmt.Sprintf("Environment variable error: invalid integer value in %s=%q: %v",
					flag.envVar, envValue, err), 1)
			}
			// Apply validation to environment variable values
			if flag.validate != nil {
				if err := flag.validate(envValue); err != nil {
					_ = Exit(fmt.Sprintf("Environment variable error: validation failed for %s=%q: %v",
						flag.envVar, envValue, err), 1)
				}
			}
			flag.value = parsed
			flag.hasEnvValue = true
			// Don't mark as explicitly set - this is from environment
		}
	}

	return flag
}

// Float creates a new float flag with optional configuration
func Float(name, usage string, opts ...FlagOption) *FloatFlag {
	// nolint: exhaustruct
	flag := &FloatFlag{
		// nolint: exhaustruct
		baseFlag: baseFlag{
			name:     name,
			usage:    usage,
			required: false, // Default to not required
		},
		value: 0.0, // Default to zero
	}

	// Apply options
	for _, opt := range opts {
		opt(flag)
	}

	// Check environment variable for default value if specified
	if flag.envVar != "" {
		if envValue := os.Getenv(flag.envVar); envValue != "" {
			parsed, err := strconv.ParseFloat(envValue, 64)
			if err != nil {
				_ = Exit(fmt.Sprintf("Environment variable error: invalid float value in %s=%q: %v",
					flag.envVar, envValue, err), 1)
			}
			// Apply validation to environment variable values
			if flag.validate != nil {
				if err := flag.validate(envValue); err != nil {
					_ = Exit(fmt.Sprintf("Environment variable error: validation failed for %s=%q: %v",
						flag.envVar, envValue, err), 1)
				}
			}
			flag.value = parsed
			flag.hasEnvValue = true
			// Don't mark as explicitly set - this is from environment
		}
	}

	return flag
}

// StringSlice creates a new string slice flag with optional configuration
func StringSlice(name, usage string, opts ...FlagOption) *StringSliceFlag {
	// nolint: exhaustruct
	flag := &StringSliceFlag{
		// nolint: exhaustruct
		baseFlag: baseFlag{
			name:     name,
			usage:    usage,
			required: false, // Default to not required
		},
		value: []string{}, // Default to empty slice
	}

	// Apply options
	for _, opt := range opts {
		opt(flag)
	}

	// Check environment variable for default value if specified
	if flag.envVar != "" {
		if envValue := os.Getenv(flag.envVar); envValue != "" {
			// Apply validation to environment variable values
			if flag.validate != nil {
				if err := flag.validate(envValue); err != nil {
					_ = Exit(fmt.Sprintf("Environment variable error: validation failed for %s=%q: %v",
						flag.envVar, envValue, err), 1)
				}
			}
			flag.value = flag.parseCommaSeparated(envValue)
			flag.hasEnvValue = true
			// Don't mark as explicitly set - this is from environment
		}
	}

	return flag
}

// Int64 creates a new int64 flag with optional configuration
func Int64(name, usage string, opts ...FlagOption) *Int64Flag {
	// nolint: exhaustruct
	flag := &Int64Flag{
		// nolint: exhaustruct
		baseFlag: baseFlag{
			name:     name,
			usage:    usage,
			required: false, // Default to not required
		},
		value: 0, // Default to zero
	}

	// Apply options
	for _, opt := range opts {
		opt(flag)
	}

	// Check environment variable for default value if specified
	if flag.envVar != "" {
		if envValue := os.Getenv(flag.envVar); envValue != "" {
			parsed, err := strconv.ParseInt(envValue, 10, 64)
			if err != nil {
				_ = Exit(fmt.Sprintf("Environment variable error: invalid int64 value in %s=%q: %v",
					flag.envVar, envValue, err), 1)
			}
			// Apply validation to environment variable values
			if flag.validate != nil {
				if err := flag.validate(envValue); err != nil {
					_ = Exit(fmt.Sprintf("Environment variable error: validation failed for %s=%q: %v",
						flag.envVar, envValue, err), 1)
				}
			}
			flag.value = parsed
			flag.hasEnvValue = true
			// Don't mark as explicitly set - this is from environment
		}
	}

	return flag
}

func newValidationError(flagName string, err error) error {
	return fmt.Errorf("%w for flag %s: %w", ErrValidationFailed, flagName, err)
}
