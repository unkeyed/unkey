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
