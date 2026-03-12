package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser_AcceptsArgs_FlagsAfterPositionalArg(t *testing.T) {
	// AcceptsArgs should allow flags after positional arguments
	// Example: deploy nginx:latest --project=local --env=preview
	var capturedArgs []string
	var projectValue string
	var envValue string

	cmd := &Command{
		Name:        "deploy",
		AcceptsArgs: true,
		Flags: []Flag{
			String("project", "project slug"),
			String("env", "environment"),
		},
		Action: func(ctx context.Context, c *Command) error {
			capturedArgs = c.Args()
			projectValue = c.String("project")
			envValue = c.String("env")
			return nil
		},
	}

	args := []string{"nginx:latest", "--project=local", "--env=preview"}
	err := cmd.parse(context.Background(), args)

	require.NoError(t, err)
	require.Equal(t, []string{"nginx:latest"}, capturedArgs)
	require.Equal(t, "local", projectValue)
	require.Equal(t, "preview", envValue)
}

func TestParser_AcceptsArgs_FlagsBeforeAndAfterPositionalArg(t *testing.T) {
	// Flags should work both before and after positional args
	var capturedArgs []string
	var projectValue string
	var envValue string

	cmd := &Command{
		Name:        "deploy",
		AcceptsArgs: true,
		Flags: []Flag{
			String("project", "project slug"),
			String("env", "environment"),
		},
		Action: func(ctx context.Context, c *Command) error {
			capturedArgs = c.Args()
			projectValue = c.String("project")
			envValue = c.String("env")
			return nil
		},
	}

	args := []string{"--project=local", "nginx:latest", "--env=preview"}
	err := cmd.parse(context.Background(), args)

	require.NoError(t, err)
	require.Equal(t, []string{"nginx:latest"}, capturedArgs)
	require.Equal(t, "local", projectValue)
	require.Equal(t, "preview", envValue)
}

func TestParser_AcceptsArgs_AllFlagsBefore(t *testing.T) {
	// All flags before positional arg should also work
	var capturedArgs []string
	var projectValue string

	cmd := &Command{
		Name:        "deploy",
		AcceptsArgs: true,
		Flags: []Flag{
			String("project", "project slug"),
		},
		Action: func(ctx context.Context, c *Command) error {
			capturedArgs = c.Args()
			projectValue = c.String("project")
			return nil
		},
	}

	args := []string{"--project=local", "nginx:latest"}
	err := cmd.parse(context.Background(), args)

	require.NoError(t, err)
	require.Equal(t, []string{"nginx:latest"}, capturedArgs)
	require.Equal(t, "local", projectValue)
}

func TestParser_AcceptsArgs_SpaceSeparatedFlagValue(t *testing.T) {
	// --flag value (space-separated) should work after positional args
	var capturedArgs []string
	var projectValue string

	cmd := &Command{
		Name:        "deploy",
		AcceptsArgs: true,
		Flags: []Flag{
			String("project", "project slug"),
		},
		Action: func(ctx context.Context, c *Command) error {
			capturedArgs = c.Args()
			projectValue = c.String("project")
			return nil
		},
	}

	args := []string{"nginx:latest", "--project", "local"}
	err := cmd.parse(context.Background(), args)

	require.NoError(t, err)
	require.Equal(t, []string{"nginx:latest"}, capturedArgs)
	require.Equal(t, "local", projectValue)
}

func TestParser_AcceptsArgs_MultiplePositionalArgs(t *testing.T) {
	var capturedArgs []string
	var verboseValue bool

	cmd := &Command{
		Name:        "run",
		AcceptsArgs: true,
		Flags: []Flag{
			Bool("verbose", "verbose output"),
		},
		Action: func(ctx context.Context, c *Command) error {
			capturedArgs = c.Args()
			verboseValue = c.Bool("verbose")
			return nil
		},
	}

	args := []string{"file1.txt", "--verbose", "file2.txt"}
	err := cmd.parse(context.Background(), args)

	require.NoError(t, err)
	require.True(t, verboseValue)
	require.Equal(t, []string{"file1.txt", "file2.txt"}, capturedArgs)
}

func TestParser_DoubleDash_StopsFlagParsing(t *testing.T) {
	// The -- separator should stop flag parsing entirely
	var capturedArgs []string

	cmd := &Command{
		Name:        "run",
		AcceptsArgs: true,
		Flags: []Flag{
			Bool("verbose", "verbose output"),
		},
		Action: func(ctx context.Context, c *Command) error {
			capturedArgs = c.Args()
			return nil
		},
	}

	// Simulate: run -- --verbose (the --verbose after -- is an arg, not a flag)
	args := []string{"--", "--verbose", "-x", "test"}
	err := cmd.parse(context.Background(), args)

	require.NoError(t, err)
	require.Equal(t, []string{"--verbose", "-x", "test"}, capturedArgs)
}

func TestParser_DoubleDash_WithFlagsBefore(t *testing.T) {
	var capturedArgs []string
	var verboseValue bool

	cmd := &Command{
		Name:        "run",
		AcceptsArgs: true,
		Flags: []Flag{
			Bool("verbose", "verbose output"),
		},
		Action: func(ctx context.Context, c *Command) error {
			capturedArgs = c.Args()
			verboseValue = c.Bool("verbose")
			return nil
		},
	}

	// Simulate: run --verbose -- -x test
	args := []string{"--verbose", "--", "-x", "test"}
	err := cmd.parse(context.Background(), args)

	require.NoError(t, err)
	require.True(t, verboseValue)
	require.Equal(t, []string{"-x", "test"}, capturedArgs)
}

func TestParser_DoubleDash_PassthroughExternalFlags(t *testing.T) {
	// Use -- to pass through args to external commands
	// Example: inject --debug -- /docker-entrypoint.sh nginx -g "daemon off;"
	var capturedArgs []string
	var debugValue bool

	cmd := &Command{
		Name:        "inject",
		AcceptsArgs: true,
		Flags: []Flag{
			String("provider", "provider type"),
			Bool("debug", "enable debug"),
		},
		Action: func(ctx context.Context, c *Command) error {
			capturedArgs = c.Args()
			debugValue = c.Bool("debug")
			return nil
		},
	}

	args := []string{"--debug", "--", "/docker-entrypoint.sh", "nginx", "-g", "daemon off;"}
	err := cmd.parse(context.Background(), args)

	require.NoError(t, err)
	require.True(t, debugValue)
	require.Equal(t, []string{"/docker-entrypoint.sh", "nginx", "-g", "daemon off;"}, capturedArgs)
}

func TestParser_UnknownFlag_WithoutAcceptsArgs(t *testing.T) {
	// When AcceptsArgs is false, unknown flags should error
	cmd := &Command{
		Name:        "test",
		AcceptsArgs: false,
		Flags: []Flag{
			Bool("verbose", "verbose output"),
		},
	}

	args := []string{"-x"}
	err := cmd.parse(context.Background(), args)

	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown flag: x")
}

func TestParser_SingleDashArg(t *testing.T) {
	// Single dash "-" is often used to mean stdin, should be treated as arg
	var capturedArgs []string

	cmd := &Command{
		Name:        "cat",
		AcceptsArgs: true,
		Flags:       []Flag{},
		Action: func(ctx context.Context, c *Command) error {
			capturedArgs = c.Args()
			return nil
		},
	}

	args := []string{"-"}
	err := cmd.parse(context.Background(), args)

	require.NoError(t, err)
	require.Equal(t, []string{"-"}, capturedArgs)
}
