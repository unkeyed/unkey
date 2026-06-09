package cli

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
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
