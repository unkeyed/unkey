package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStringFlag_BasicParsing(t *testing.T) {
	flag := String("test", "test flag")
	err := flag.Parse("hello")
	require.NoError(t, err)
	require.Equal(t, "hello", flag.Value())
	require.True(t, flag.IsSet())
	require.True(t, flag.HasValue())
}

func TestStringFlag_WithValidation_Failure(t *testing.T) {
	flag := String("url", "URL flag", Validate(validateURL))
	err := flag.Parse("invalid-url")
	require.Error(t, err)
	require.Contains(t, err.Error(), ErrValidationFailed.Error())
}

func TestStringFlag_WithEnvVar(t *testing.T) {
	os.Setenv("TEST_STRING", "env-value")
	defer os.Unsetenv("TEST_STRING")

	flag := String("test", "test flag", EnvVar("TEST_STRING"))
	require.Equal(t, "env-value", flag.Value())
	require.False(t, flag.IsSet())
	require.True(t, flag.HasValue())
}

func TestStringFlag_ValidationOnEnvVar(t *testing.T) {
	os.Setenv("INVALID_URL", "not-a-url")
	defer os.Unsetenv("INVALID_URL")

	exitCode, exitCalled, cleanup := mockExit()
	defer cleanup()

	// Capture the panic and validate the exit behavior
	defer func() {
		r := recover()
		require.NotNil(t, r, "Expected panic from mocked Exit")
		require.Equal(t, "exit called", r)
		require.True(t, *exitCalled, "Exit should have been called")
		require.Equal(t, 1, *exitCode, "Exit code should be 1")
	}()

	String("url", "URL flag", EnvVar("INVALID_URL"), Validate(validateURL))
}

// BoolFlag Tests
func TestBoolFlag_EmptyValue(t *testing.T) {
	flag := Bool("verbose", "verbose flag")
	err := flag.Parse("")
	require.NoError(t, err)
	require.True(t, flag.Value())
	require.True(t, flag.IsSet())
}

func TestBoolFlag_WithEnvVar_InvalidValue(t *testing.T) {
	os.Setenv("INVALID_BOOL", "maybe")
	defer os.Unsetenv("INVALID_BOOL")

	exitCode, exitCalled, cleanup := mockExit()
	defer cleanup()

	// Capture the panic and validate the exit behavior
	defer func() {
		r := recover()
		require.NotNil(t, r, "Expected panic from mocked Exit")
		require.Equal(t, "exit called", r)
		require.True(t, *exitCalled, "Exit should have been called")
		require.Equal(t, 1, *exitCode, "Exit code should be 1")
	}()

	Bool("test", "test flag", EnvVar("INVALID_BOOL"))
}

func TestBoolFlag_InvalidValue(t *testing.T) {
	flag := Bool("verbose", "verbose flag")
	err := flag.Parse("maybe")
	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidBoolValue.Error())
}

func TestBoolFlag_ValidationOnEmptyValue(t *testing.T) {
	flag := Bool("verbose", "verbose flag", Validate(func(s string) error {
		if s != "" && s != "true" && s != "false" {
			return fmt.Errorf("only 'true' or 'false' allowed")
		}
		return nil
	}))

	// Empty string should pass validation (gets converted to true)
	err := flag.Parse("")
	require.NoError(t, err)
	require.True(t, flag.Value())
}

func TestIntFlag_ValidInteger(t *testing.T) {
	flag := Int("count", "count flag")
	err := flag.Parse("42")
	require.NoError(t, err)
	require.Equal(t, 42, flag.Value())
}

func TestIntFlag_InvalidInteger(t *testing.T) {
	flag := Int("count", "count flag")
	err := flag.Parse("not-a-number")
	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidIntValue.Error())
}

func TestIntFlag_WithValidation_Failure(t *testing.T) {
	flag := Int("port", "port flag", Validate(validatePort))
	err := flag.Parse("70000")
	require.Error(t, err)
	require.Contains(t, err.Error(), ErrValidationFailed.Error())
}

func TestIntFlag_ZeroValueWithEnv(t *testing.T) {
	os.Setenv("COUNT", "0")
	defer os.Unsetenv("COUNT")

	flag := Int("count", "count flag", EnvVar("COUNT"))
	require.Equal(t, 0, flag.Value())
	require.True(t, flag.HasValue())
}

func TestFloatFlag_ValidFloat(t *testing.T) {
	flag := Float("rate", "rate flag")
	err := flag.Parse("3.14")
	require.NoError(t, err)
	require.Equal(t, 3.14, flag.Value())
}

func TestFloatFlag_InvalidFloat(t *testing.T) {
	flag := Float("rate", "rate flag")
	err := flag.Parse("not-a-number")
	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidFloatValue.Error())
}

func TestFloatFlag_ZeroValueWithEnv(t *testing.T) {
	os.Setenv("ZERO_RATE", "0.0")
	defer os.Unsetenv("ZERO_RATE")

	flag := Float("rate", "rate flag", EnvVar("ZERO_RATE"))
	require.Equal(t, 0.0, flag.Value())
	require.True(t, flag.HasValue())
}

func TestFloatFlag_ValidationOnEnvVar(t *testing.T) {
	os.Setenv("INVALID_RANGE", "2.5")
	defer os.Unsetenv("INVALID_RANGE")

	validateRange := func(s string) error {
		if val, err := strconv.ParseFloat(s, 64); err != nil || val < 0 || val > 1 {
			return fmt.Errorf("must be between 0 and 1")
		}
		return nil
	}

	exitCode, exitCalled, cleanup := mockExit()
	defer cleanup()

	// Capture the panic and validate the exit behavior
	defer func() {
		r := recover()
		require.NotNil(t, r, "Expected panic from mocked Exit")
		require.Equal(t, "exit called", r)
		require.True(t, *exitCalled, "Exit should have been called")
		require.Equal(t, 1, *exitCode, "Exit code should be 1")
	}()

	Float("rate", "rate flag", EnvVar("INVALID_RANGE"), Validate(validateRange))
}

func TestDurationFlag_BasicParsing(t *testing.T) {
	flag := Duration("timeout", "timeout flag")
	err := flag.Parse("5m30s")
	require.NoError(t, err)
	require.Equal(t, time.Duration(5*time.Minute+30*time.Second), flag.Value())
	require.True(t, flag.IsSet())
	require.True(t, flag.HasValue())
}

func TestDurationFlag_InvalidDuration(t *testing.T) {
	flag := Duration("timeout", "timeout flag")
	err := flag.Parse("invalid-duration")
	require.Error(t, err)
	require.Contains(t, err.Error(), "validation failed")
}

func TestDurationFlag_WithDefault(t *testing.T) {
	flag := Duration("timeout", "timeout flag", Default(time.Hour))
	require.Equal(t, time.Hour, flag.Value())
	require.False(t, flag.IsSet())
	require.True(t, flag.HasValue())
}

func TestDurationFlag_ZeroValue(t *testing.T) {
	flag := Duration("timeout", "timeout flag")
	require.Equal(t, time.Duration(0), flag.Value())
	require.False(t, flag.IsSet())
	require.False(t, flag.HasValue())
}

func TestDurationFlag_WithEnvVar(t *testing.T) {
	os.Setenv("TEST_DURATION", "2h30m")
	defer os.Unsetenv("TEST_DURATION")

	flag := Duration("timeout", "timeout flag", EnvVar("TEST_DURATION"))
	require.Equal(t, time.Duration(2*time.Hour+30*time.Minute), flag.Value())
	require.False(t, flag.IsSet())
	require.True(t, flag.HasValue())
}

func TestDurationFlag_CommandOverridesEnv(t *testing.T) {
	os.Setenv("TEST_DURATION", "1h")
	defer os.Unsetenv("TEST_DURATION")

	flag := Duration("timeout", "timeout flag", EnvVar("TEST_DURATION"))
	err := flag.Parse("30m")
	require.NoError(t, err)
	require.Equal(t, time.Duration(30*time.Minute), flag.Value())
	require.True(t, flag.IsSet())
}

func TestDurationFlag_ValidationOnEnvVar(t *testing.T) {
	os.Setenv("INVALID_DURATION", "not-a-duration")
	defer os.Unsetenv("INVALID_DURATION")

	exitCode, exitCalled, cleanup := mockExit()
	defer cleanup()

	defer func() {
		r := recover()
		require.NotNil(t, r, "Expected panic from mocked Exit")
		require.Equal(t, "exit called", r)
		require.True(t, *exitCalled, "Exit should have been called")
		require.Equal(t, 1, *exitCode, "Exit code should be 1")
	}()

	Duration("timeout", "timeout flag", EnvVar("INVALID_DURATION"))
}

func TestCommand_Duration_Integration(t *testing.T) {
	cmd := &Command{
		Name: "test",
		Flags: []Flag{
			Duration("timeout", "timeout for operation", Default(time.Minute*5)),
		},
	}

	err := cmd.parse(context.Background(), []string{"--timeout", "2h30m"})
	require.NoError(t, err)

	duration := cmd.Duration("timeout")
	require.Equal(t, time.Duration(2*time.Hour+30*time.Minute), duration)
}

func TestStringSliceFlag_CommaSeparated(t *testing.T) {
	flag := StringSlice("tags", "tags flag")
	err := flag.Parse("foo,bar,baz")
	require.NoError(t, err)
	require.Equal(t, []string{"foo", "bar", "baz"}, flag.Value())
}

func TestStringSliceFlag_FilterEmptyValues(t *testing.T) {
	flag := StringSlice("tags", "tags flag")
	err := flag.Parse("foo,,bar,")
	require.NoError(t, err)
	require.Equal(t, []string{"foo", "bar"}, flag.Value())
}

func TestStringSliceFlag_WithEnvVar(t *testing.T) {
	os.Setenv("TAGS", "web,api,service")
	defer os.Unsetenv("TAGS")

	flag := StringSlice("tags", "tags flag", EnvVar("TAGS"))
	require.Equal(t, []string{"web", "api", "service"}, flag.Value())
	require.False(t, flag.IsSet())
	require.True(t, flag.HasValue())
}

func TestStringSliceFlag_ValidationOnEnvVar(t *testing.T) {
	os.Setenv("INVALID_TAGS", "valid,invalid;;tag")
	defer os.Unsetenv("INVALID_TAGS")

	validateNoSemicolons := func(s string) error {
		if strings.Contains(s, ";;") {
			return fmt.Errorf("double semicolons not allowed")
		}
		return nil
	}

	exitCode, exitCalled, cleanup := mockExit()
	defer cleanup()

	// Capture the panic and validate the exit behavior
	defer func() {
		r := recover()
		require.NotNil(t, r, "Expected panic from mocked Exit")
		require.Equal(t, "exit called", r)
		require.True(t, *exitCalled, "Exit should have been called")
		require.Equal(t, 1, *exitCode, "Exit code should be 1")
	}()

	StringSlice("tags", "tags flag", EnvVar("INVALID_TAGS"), Validate(validateNoSemicolons))
}

func TestCommandLineOverrideEnvironment(t *testing.T) {
	os.Setenv("PORT", "3000")
	defer os.Unsetenv("PORT")

	portFlag := Int("port", "Server port", EnvVar("PORT"), Default(8080))
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{portFlag},
	}

	args := []string{"--port", "9000"}
	err := cmd.parse(context.Background(), args)
	require.NoError(t, err)
	require.Equal(t, 9000, cmd.Int("port"))
	require.True(t, portFlag.IsSet())
}

func TestRequiredFlagMissing(t *testing.T) {
	requiredFlag := String("required", "required flag", Required())
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{requiredFlag},
	}

	args := []string{}
	err := cmd.parse(context.Background(), args)
	require.Error(t, err)
	require.Contains(t, err.Error(), "required flag missing: required")
}

// Require Function Tests
func TestRequireString_Success(t *testing.T) {
	flag := String("api-key", "API key")
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{flag},
	}
	cmd.initFlagMap()

	err := flag.Parse("secret-key")
	require.NoError(t, err)

	result := cmd.RequireString("api-key")
	require.Equal(t, "secret-key", result)
}

func TestRequireString_FlagNotFound(t *testing.T) {
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{String("existing", "existing flag")},
	}
	cmd.initFlagMap()

	require.Panics(t, func() {
		cmd.RequireString("non-existent")
	})
}

func TestRequireString_WrongType(t *testing.T) {
	boolFlag := Bool("verbose", "verbose flag")
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{boolFlag},
	}
	cmd.initFlagMap()

	require.Panics(t, func() {
		cmd.RequireString("verbose")
	})
}

func TestRequireVsSafeAccessors(t *testing.T) {
	cmd := &Command{
		Name: "test",
		Flags: []Flag{
			String("existing", "existing flag"),
			Int("port", "port flag"),
		},
	}
	cmd.initFlagMap()

	// Safe accessors return zero values for missing flags
	require.Equal(t, "", cmd.String("missing"))
	require.Equal(t, false, cmd.Bool("missing"))
	require.Equal(t, 0, cmd.Int("missing"))
	require.Equal(t, 0.0, cmd.Float("missing"))
	require.Equal(t, []string{}, cmd.StringSlice("missing"))

	// Require accessors panic for missing flags
	require.Panics(t, func() { cmd.RequireString("missing") })
	require.Panics(t, func() { cmd.RequireBool("missing") })
}

func TestRequireBool_Success(t *testing.T) {
	flag := Bool("verbose", "verbose flag")
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{flag},
	}
	cmd.initFlagMap()

	err := flag.Parse("true")
	require.NoError(t, err)

	result := cmd.RequireBool("verbose")
	require.True(t, result)
}

func TestRequireBool_WrongType(t *testing.T) {
	stringFlag := String("config", "config flag")
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{stringFlag},
	}
	cmd.initFlagMap()

	require.Panics(t, func() {
		cmd.RequireBool("config")
	})
}

func TestRequireInt_Success(t *testing.T) {
	flag := Int("port", "port flag")
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{flag},
	}
	cmd.initFlagMap()

	err := flag.Parse("8080")
	require.NoError(t, err)

	result := cmd.RequireInt("port")
	require.Equal(t, 8080, result)
}

func TestRequireInt_WrongType(t *testing.T) {
	stringFlag := String("config", "config flag")
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{stringFlag},
	}
	cmd.initFlagMap()

	require.Panics(t, func() {
		cmd.RequireInt("config")
	})
}

func TestRequireFloat_Success(t *testing.T) {
	flag := Float("rate", "rate flag")
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{flag},
	}
	cmd.initFlagMap()

	err := flag.Parse("3.14")
	require.NoError(t, err)

	result := cmd.RequireFloat("rate")
	require.Equal(t, 3.14, result)
}

func TestRequireFloat_WrongType(t *testing.T) {
	intFlag := Int("port", "port flag")
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{intFlag},
	}
	cmd.initFlagMap()

	require.Panics(t, func() {
		cmd.RequireFloat("port")
	})
}

func TestRequireStringSlice_Success(t *testing.T) {
	flag := StringSlice("tags", "tags flag")
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{flag},
	}
	cmd.initFlagMap()

	err := flag.Parse("foo,bar,baz")
	require.NoError(t, err)

	result := cmd.RequireStringSlice("tags")
	require.Equal(t, []string{"foo", "bar", "baz"}, result)
}

func TestRequireStringSlice_WrongType(t *testing.T) {
	stringFlag := String("config", "config flag")
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{stringFlag},
	}
	cmd.initFlagMap()

	require.Panics(t, func() {
		cmd.RequireStringSlice("config")
	})
}

func TestBoolFlag_WithEnvVar(t *testing.T) {
	os.Setenv("VERBOSE", "true")
	defer os.Unsetenv("VERBOSE")

	flag := Bool("verbose", "verbose flag", EnvVar("VERBOSE"))
	require.True(t, flag.Value())
	require.False(t, flag.IsSet())
	require.True(t, flag.HasValue())
}

func TestBoolFlag_WithEnvVar_False(t *testing.T) {
	os.Setenv("QUIET", "false")
	defer os.Unsetenv("QUIET")

	flag := Bool("quiet", "quiet flag", EnvVar("QUIET"))
	require.False(t, flag.Value())
	require.False(t, flag.IsSet())
	require.True(t, flag.HasValue())
}

func validateURL(s string) error {
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") && !strings.HasPrefix(s, "postgres://") {
		return fmt.Errorf("invalid URL format")
	}
	return nil
}

func validatePort(s string) error {
	port, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid port number")
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1-65535")
	}
	return nil
}

func TestCommand_Run_NoArguments(t *testing.T) {
	cmd := &Command{Name: "test"}
	err := cmd.Run(context.Background(), []string{})

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNoArguments))
}

func TestCommand_RequireString_FlagNotFound(t *testing.T) {
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{String("existing", "existing flag")},
	}
	cmd.initFlagMap()

	// Capture the panic and verify the error structure
	defer func() {
		r := recover()
		require.NotNil(t, r, "Expected panic for missing flag")

		err, ok := r.(error)
		require.True(t, ok, "Panic value should be an error")
		require.True(t, errors.Is(err, ErrFlagNotFound))
		require.Contains(t, err.Error(), "non-existent")
		require.Contains(t, err.Error(), "command \"test\"")
		require.Contains(t, err.Error(), "available flags: existing")
	}()

	cmd.RequireString("non-existent")
}

func TestCommand_RequireString_WrongType(t *testing.T) {
	boolFlag := Bool("verbose", "verbose flag")
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{boolFlag},
	}
	cmd.initFlagMap()

	defer func() {
		r := recover()
		require.NotNil(t, r, "Expected panic for wrong type")

		err, ok := r.(error)
		require.True(t, ok, "Panic value should be an error")
		require.True(t, errors.Is(err, ErrWrongFlagType))
		require.Contains(t, err.Error(), "verbose")
		require.Contains(t, err.Error(), "BoolFlag")
		require.Contains(t, err.Error(), "expected StringFlag")
		require.Contains(t, err.Error(), "available stringflag flags: none")
	}()

	cmd.RequireString("verbose")
}

func TestCommand_RequireBool_FlagNotFound(t *testing.T) {
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{},
	}
	cmd.initFlagMap()

	defer func() {
		r := recover()
		require.NotNil(t, r)

		err, ok := r.(error)
		require.True(t, ok)
		require.True(t, errors.Is(err, ErrFlagNotFound))
		require.Contains(t, err.Error(), "missing")
		require.Contains(t, err.Error(), "available flags: none")
	}()

	cmd.RequireBool("missing")
}

func TestCommand_RequireBool_WrongType(t *testing.T) {
	stringFlag := String("config", "config flag")
	intFlag := Int("port", "port flag")
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{stringFlag, intFlag},
	}
	cmd.initFlagMap()

	defer func() {
		r := recover()
		require.NotNil(t, r)

		err, ok := r.(error)
		require.True(t, ok)
		require.True(t, errors.Is(err, ErrWrongFlagType))
		require.Contains(t, err.Error(), "config")
		require.Contains(t, err.Error(), "StringFlag")
		require.Contains(t, err.Error(), "expected BoolFlag")
		require.Contains(t, err.Error(), "available boolflag flags: none")
	}()

	cmd.RequireBool("config")
}

func TestCommand_RequireInt_FlagNotFound(t *testing.T) {
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{String("name", "name flag")},
	}
	cmd.initFlagMap()

	defer func() {
		r := recover()
		require.NotNil(t, r)

		err, ok := r.(error)
		require.True(t, ok)
		require.True(t, errors.Is(err, ErrFlagNotFound))
	}()

	cmd.RequireInt("count")
}

func TestCommand_RequireInt_WrongType(t *testing.T) {
	floatFlag := Float("rate", "rate flag")
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{floatFlag},
	}
	cmd.initFlagMap()

	defer func() {
		r := recover()
		require.NotNil(t, r)

		err, ok := r.(error)
		require.True(t, ok)
		require.True(t, errors.Is(err, ErrWrongFlagType))
		require.Contains(t, err.Error(), "rate")
		require.Contains(t, err.Error(), "FloatFlag")
		require.Contains(t, err.Error(), "expected IntFlag")
	}()

	cmd.RequireInt("rate")
}

func TestCommand_RequireFloat_FlagNotFound(t *testing.T) {
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{},
	}
	cmd.initFlagMap()

	defer func() {
		r := recover()
		require.NotNil(t, r)

		err, ok := r.(error)
		require.True(t, ok)
		require.True(t, errors.Is(err, ErrFlagNotFound))
	}()

	cmd.RequireFloat("missing")
}

func TestCommand_RequireFloat_WrongType(t *testing.T) {
	intFlag := Int("port", "port flag")
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{intFlag},
	}
	cmd.initFlagMap()

	defer func() {
		r := recover()
		require.NotNil(t, r)

		err, ok := r.(error)
		require.True(t, ok)
		require.True(t, errors.Is(err, ErrWrongFlagType))
		require.Contains(t, err.Error(), "port")
		require.Contains(t, err.Error(), "IntFlag")
		require.Contains(t, err.Error(), "expected FloatFlag")
	}()

	cmd.RequireFloat("port")
}

func TestCommand_RequireStringSlice_FlagNotFound(t *testing.T) {
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{},
	}
	cmd.initFlagMap()

	defer func() {
		r := recover()
		require.NotNil(t, r)

		err, ok := r.(error)
		require.True(t, ok)
		require.True(t, errors.Is(err, ErrFlagNotFound))
	}()

	cmd.RequireStringSlice("tags")
}

func TestCommand_RequireStringSlice_WrongType(t *testing.T) {
	stringFlag := String("config", "config flag")
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{stringFlag},
	}
	cmd.initFlagMap()

	defer func() {
		r := recover()
		require.NotNil(t, r)

		err, ok := r.(error)
		require.True(t, ok)
		require.True(t, errors.Is(err, ErrWrongFlagType))
		require.Contains(t, err.Error(), "config")
		require.Contains(t, err.Error(), "StringFlag")
		require.Contains(t, err.Error(), "expected StringSliceFlag")
	}()

	cmd.RequireStringSlice("config")
}

// Test error helpers work correctly
func TestCommand_ErrorHelpers_FlagsByType(t *testing.T) {
	cmd := &Command{
		Name: "test",
		Flags: []Flag{
			String("name", "name flag"),
			String("config", "config flag"),
			Bool("verbose", "verbose flag"),
			Int("port", "port flag"),
			Int("timeout", "timeout flag"),
			Float("rate", "rate flag"),
			StringSlice("tags", "tags flag"),
		},
	}

	// Test getFlagsByType returns correct flags
	require.Equal(t, "name, config", cmd.getFlagsByType("StringFlag"))
	require.Equal(t, "verbose", cmd.getFlagsByType("BoolFlag"))
	require.Equal(t, "port, timeout", cmd.getFlagsByType("IntFlag"))
	require.Equal(t, "rate", cmd.getFlagsByType("FloatFlag"))
	require.Equal(t, "tags", cmd.getFlagsByType("StringSliceFlag"))
	require.Equal(t, "none", cmd.getFlagsByType("NonexistentFlag"))
}

func TestCommand_ErrorHelpers_AvailableFlags(t *testing.T) {
	// Test with no flags
	cmd := &Command{Name: "test", Flags: []Flag{}}
	require.Equal(t, "none", cmd.getAvailableFlags())

	// Test with multiple flags
	cmd.Flags = []Flag{
		String("name", "name flag"),
		Bool("verbose", "verbose flag"),
		Int("port", "port flag"),
	}
	expected := "name, verbose, port"
	require.Equal(t, expected, cmd.getAvailableFlags())
}

func TestCommand_ErrorHelpers_GetFlagType(t *testing.T) {
	cmd := &Command{Name: "test"}

	require.Equal(t, "StringFlag", cmd.getFlagType(String("test", "test")))
	require.Equal(t, "BoolFlag", cmd.getFlagType(Bool("test", "test")))
	require.Equal(t, "IntFlag", cmd.getFlagType(Int("test", "test")))
	require.Equal(t, "FloatFlag", cmd.getFlagType(Float("test", "test")))
	require.Equal(t, "StringSliceFlag", cmd.getFlagType(StringSlice("test", "test")))
}

// Test that safe accessors still work as expected (no panics)
func TestCommand_SafeAccessors_MissingFlags(t *testing.T) {
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{},
	}
	cmd.initFlagMap()

	// All safe accessors should return zero values, no panics
	require.Equal(t, "", cmd.String("missing"))
	require.Equal(t, false, cmd.Bool("missing"))
	require.Equal(t, 0, cmd.Int("missing"))
	require.Equal(t, 0.0, cmd.Float("missing"))
	require.Equal(t, []string{}, cmd.StringSlice("missing"))
}

func TestCommand_SafeAccessors_WrongType(t *testing.T) {
	boolFlag := Bool("verbose", "verbose flag")
	cmd := &Command{
		Name:  "test",
		Flags: []Flag{boolFlag},
	}
	cmd.initFlagMap()

	// Safe accessors should return zero values for wrong types, no panics
	require.Equal(t, "", cmd.String("verbose"))              // Bool flag accessed as String
	require.Equal(t, 0, cmd.Int("verbose"))                  // Bool flag accessed as Int
	require.Equal(t, 0.0, cmd.Float("verbose"))              // Bool flag accessed as Float
	require.Equal(t, []string{}, cmd.StringSlice("verbose")) // Bool flag accessed as StringSlice
}

// Test successful Require* calls
func TestCommand_RequireSuccess(t *testing.T) {
	stringFlag := String("name", "name flag")
	boolFlag := Bool("verbose", "verbose flag")
	intFlag := Int("port", "port flag")
	floatFlag := Float("rate", "rate flag")
	sliceFlag := StringSlice("tags", "tags flag")

	cmd := &Command{
		Name:  "test",
		Flags: []Flag{stringFlag, boolFlag, intFlag, floatFlag, sliceFlag},
	}
	cmd.initFlagMap()

	// Set up flag values
	require.NoError(t, stringFlag.Parse("test-name"))
	require.NoError(t, boolFlag.Parse("true"))
	require.NoError(t, intFlag.Parse("8080"))
	require.NoError(t, floatFlag.Parse("3.14"))
	require.NoError(t, sliceFlag.Parse("foo,bar"))

	// All Require* calls should succeed
	require.Equal(t, "test-name", cmd.RequireString("name"))
	require.Equal(t, true, cmd.RequireBool("verbose"))
	require.Equal(t, 8080, cmd.RequireInt("port"))
	require.Equal(t, 3.14, cmd.RequireFloat("rate"))
	require.Equal(t, []string{"foo", "bar"}, cmd.RequireStringSlice("tags"))
}

// Test that error messages are informative
func TestCommand_ErrorMessage_Quality(t *testing.T) {
	cmd := &Command{
		Name: "deploy",
		Flags: []Flag{
			String("config", "config file"),
			Bool("verbose", "verbose output"),
			Int("timeout", "timeout seconds"),
		},
	}
	cmd.initFlagMap()

	defer func() {
		r := recover()
		require.NotNil(t, r)

		err, ok := r.(error)
		require.True(t, ok)

		errMsg := err.Error()
		// Should contain all the important context
		require.Contains(t, errMsg, "flag not found")
		require.Contains(t, errMsg, "missing-flag")
		require.Contains(t, errMsg, "deploy")
		require.Contains(t, errMsg, "available flags: config, verbose, timeout")
	}()

	cmd.RequireString("missing-flag")
}

func mockExit() (exitCode *int, exitCalled *bool, cleanup func()) {
	var code int
	var called bool

	originalExit := ExitFunc
	ExitFunc = func(c int) {
		code = c
		called = true
		panic("exit called") // Prevent actual exit
	}

	return &code, &called, func() {
		ExitFunc = originalExit
	}
}
