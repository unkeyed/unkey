package cli

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestEnumFlag_AcceptsAllowedValue(t *testing.T) {
	flag := Enum("bump", "bump kind", []string{"patch", "minor", "major"})
	require.NoError(t, flag.Parse("minor"))
	require.Equal(t, "minor", flag.Value())
	require.True(t, flag.IsSet())
	require.True(t, flag.HasValue())
}

func TestEnumFlag_RejectsDisallowedValue(t *testing.T) {
	flag := Enum("bump", "bump kind", []string{"patch", "minor", "major"})
	err := flag.Parse("sideways")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidEnumValue)
	require.Contains(t, err.Error(), "patch, minor, major")
}

func TestEnumFlag_Allowed(t *testing.T) {
	flag := Enum("bump", "bump kind", []string{"patch", "minor", "major"})
	require.Equal(t, []string{"patch", "minor", "major"}, flag.Allowed())
}

func TestEnumFlag_Default(t *testing.T) {
	flag := Enum("bump", "bump kind", []string{"patch", "minor", "major"}, Default("patch"))
	require.Equal(t, "patch", flag.Value())
	require.False(t, flag.IsSet())
	require.True(t, flag.HasValue())
}

func TestEnumFlag_EnvVar(t *testing.T) {
	require.NoError(t, os.Setenv("TEST_ENUM", "major"))
	t.Cleanup(func() { require.NoError(t, os.Unsetenv("TEST_ENUM")) })

	flag := Enum("bump", "bump kind", []string{"patch", "minor", "major"}, EnvVar("TEST_ENUM"))
	require.Equal(t, "major", flag.Value())
	require.False(t, flag.IsSet())
	require.True(t, flag.HasValue())
}

func TestCommand_EnumAccessorAndHelpLabel(t *testing.T) {
	var got string
	cmd := &Command{
		Name:        "release",
		Usage:       "test",
		Description: "",
		Examples:    []string{},
		Version:     "",
		Commands:    []*Command{},
		AcceptsArgs: false,
		Aliases:     []string{},
		Flags: []Flag{
			Enum("bump", "bump kind", []string{"patch", "minor", "major"}, Default("patch")),
		},
		Action: func(_ context.Context, c *Command) error {
			got = c.Enum("bump")
			return nil
		},
	}

	require.NoError(t, cmd.Run(context.Background(), []string{"release", "--bump", "minor"}))
	require.Equal(t, "minor", got)

	// The allowed values appear in the rendered flag signature.
	require.Equal(t, "--bump (patch|minor|major)", cmd.buildFlagName(cmd.Flags[0]))
}

func TestEnumInvalidEnvironmentIsDeferredUntilExecution(t *testing.T) {
	secret := uid.New(uid.KeyPrefix)
	t.Setenv("TEST_ENUM", secret)
	for _, test := range []struct {
		name string
		args []string
		runs int
		fail bool
	}{
		{name: "help", args: []string{"--help"}},
		{name: "invalid environment", fail: true},
		{name: "explicit override", args: []string{"--output=json"}, runs: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			runs, handled := 0, 0
			cmd := &Command{
				Name:  "test",
				Flags: []Flag{Enum("output", "", []string{"human", "json"}, Default("human"), EnvVar("TEST_ENUM"))},
				Prepare: func(_ context.Context, cmd *Command) error {
					require.Equal(t, "json", cmd.Enum("output"))
					require.True(t, cmd.FlagIsSet("output"))
					runs++
					return nil
				},
				OnError: func(_ context.Context, _ *Command, args []string, err error) error {
					handled++
					require.Nil(t, args)
					return err
				},
			}
			output := captureCommandOutput(t, func() {
				err := cmd.Run(t.Context(), append([]string{"test"}, test.args...))
				if test.fail {
					var input *InputError
					require.ErrorAs(t, err, &input)
					require.Equal(t, InputErrorInvalidFlagValue, input.Kind)
					require.Equal(t, []string{"output"}, input.Flags)
					require.ErrorIs(t, err, ErrInvalidEnumValue)
					require.NotContains(t, err.Error(), secret)
					require.Equal(t, 1, handled)
				} else {
					require.NoError(t, err)
					require.Zero(t, handled)
				}
			})
			require.Equal(t, test.runs, runs)
			require.NotContains(t, output, secret)
		})
	}
}

func TestEnumEnvironmentValidationPreservesCauseWithoutEchoingIt(t *testing.T) {
	secret := uid.New(uid.KeyPrefix)
	problem := fmt.Errorf("rejected credential: %s", secret)
	t.Setenv("TEST_ENUM", "json")
	flag := Enum("output", "", []string{"human", "json"}, EnvVar("TEST_ENUM"), Validate(func(string) error { return problem }))
	cmd := &Command{Name: "test", Flags: []Flag{flag}}
	err := cmd.Run(t.Context(), []string{"test"})
	require.ErrorIs(t, err, problem)
	require.ErrorIs(t, err, ErrValidationFailed)
	require.ErrorContains(t, err, "validation failed for TEST_ENUM")
	require.NotContains(t, err.Error(), secret)
}
