package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser_AcceptsArgs_StopsFlagParsing(t *testing.T) {
	// This test ensures that when a command accepts args, the first positional
	// argument stops flag parsing. This is critical for wrapper commands like
	// "inject" that exec other programs with their own flags.
	//
	// Example: inject /docker-entrypoint.sh nginx -g "daemon off;"
	// The "-g" should NOT be parsed as an inject flag.

	var capturedArgs []string
	cmd := &Command{
		Name:        "inject",
		AcceptsArgs: true,
		Flags: []Flag{
			String("provider", "provider type"),
			Bool("debug", "enable debug"),
		},
		Action: func(ctx context.Context, c *Command) error {
			capturedArgs = c.Args()
			return nil
		},
	}

	// Simulate: inject /docker-entrypoint.sh nginx -g "daemon off;"
	args := []string{"/docker-entrypoint.sh", "nginx", "-g", "daemon off;"}
	err := cmd.parse(context.Background(), args)

	require.NoError(t, err)
	require.Equal(t, []string{"/docker-entrypoint.sh", "nginx", "-g", "daemon off;"}, capturedArgs)
}

func TestParser_AcceptsArgs_FlagsBeforeCommand(t *testing.T) {
	// Flags before the command should still be parsed
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

	// Simulate: inject --debug /bin/sh -c "echo hello"
	args := []string{"--debug", "/bin/sh", "-c", "echo hello"}
	err := cmd.parse(context.Background(), args)

	require.NoError(t, err)
	require.True(t, debugValue)
	require.Equal(t, []string{"/bin/sh", "-c", "echo hello"}, capturedArgs)
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

func TestParser_NginxExample(t *testing.T) {
	// Real-world example that was failing before the fix
	var capturedArgs []string

	cmd := &Command{
		Name:        "inject",
		AcceptsArgs: true,
		Flags: []Flag{
			String("provider", "provider type"),
			String("endpoint", "provider endpoint"),
			String("deployment-id", "deployment ID"),
			String("environment-id", "environment ID"),
			String("secrets-blob", "encrypted secrets"),
			String("token", "auth token"),
			String("token-path", "path to token file"),
			Bool("debug", "enable debug"),
		},
		Action: func(ctx context.Context, c *Command) error {
			capturedArgs = c.Args()
			return nil
		},
	}

	// This is the exact command that was failing:
	// /docker-entrypoint.sh nginx -g daemon off;
	args := []string{"/docker-entrypoint.sh", "nginx", "-g", "daemon off;"}
	err := cmd.parse(context.Background(), args)

	require.NoError(t, err)
	require.Equal(t, []string{"/docker-entrypoint.sh", "nginx", "-g", "daemon off;"}, capturedArgs)
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
