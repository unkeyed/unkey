package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrapText_ShortText(t *testing.T) {
	lines := wrapText("short text", 80)
	require.Equal(t, []string{"short text"}, lines)
}

func TestWrapText_ExactWidth(t *testing.T) {
	text := "exactly ten"
	lines := wrapText(text, len(text))
	require.Equal(t, []string{"exactly ten"}, lines)
}

func TestWrapText_WrapsAtWordBoundary(t *testing.T) {
	lines := wrapText("the quick brown fox jumps over the lazy dog", 20)
	require.Equal(t, []string{
		"the quick brown fox",
		"jumps over the lazy",
		"dog",
	}, lines)
}

func TestWrapText_NoSpaces(t *testing.T) {
	lines := wrapText("abcdefghij", 5)
	require.Equal(t, []string{"abcde", "fghij"}, lines)
}

func TestWrapText_EmptyString(t *testing.T) {
	lines := wrapText("", 80)
	require.Equal(t, []string{""}, lines)
}

func TestBuildFlagUsage_MutuallyExclusive(t *testing.T) {
	cmd := &Command{Flags: []Flag{
		String("request", "Request input.", Required(), MutuallyExclusive("body")),
		String("body", "Body input."),
	}}

	require.Equal(
		t,
		"Request input. (required unless --body is set; mutually exclusive with --body)",
		cmd.buildFlagUsage(cmd.Flags[0]),
	)
	require.Equal(
		t,
		"Body input. (mutually exclusive with --request)",
		cmd.buildFlagUsage(cmd.Flags[1]),
	)
}
