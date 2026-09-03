package deploy

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

// TestTrimBytes covers the truncation that stands between a long commit message
// and a rejected insert. The rune-boundary case is the one that matters: MySQL
// strict mode rejects malformed UTF-8, so a cut through a multi-byte rune fails
// the whole deployment rather than storing a shortened message.
func TestTrimBytes(t *testing.T) {
	// "🌯" is four bytes, so every limit from 5 to 8 lands inside it.
	burrito := "KEBAP🌯"
	require.Len(t, burrito, 9, "fixture must straddle the limits below")

	tests := []struct {
		name     string
		input    string
		bytesMax int
		want     string
	}{
		{name: "shorter than the limit", input: "KEBAP", bytesMax: 10, want: "KEBAP"},
		{name: "exactly the limit", input: "KEBAP", bytesMax: 5, want: "KEBAP"},
		{name: "cut between ascii runes", input: "KEBAP", bytesMax: 3, want: "KEB"},
		{name: "empty input", input: "", bytesMax: 5, want: ""},
		{name: "zero limit", input: "KEBAP", bytesMax: 0, want: ""},

		// Every one of these lands inside the four-byte rune and must fall back
		// to the boundary before it.
		{name: "cut at the rune's first byte", input: burrito, bytesMax: 5, want: "KEBAP"},
		{name: "cut one byte into the rune", input: burrito, bytesMax: 6, want: "KEBAP"},
		{name: "cut two bytes into the rune", input: burrito, bytesMax: 7, want: "KEBAP"},
		{name: "cut three bytes into the rune", input: burrito, bytesMax: 8, want: "KEBAP"},
		{name: "the whole rune fits", input: burrito, bytesMax: 9, want: burrito},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimBytes(tt.input, tt.bytesMax)
			require.Equal(t, tt.want, got)
			require.LessOrEqual(t, len(got), tt.bytesMax, "the result must fit the column")
			require.True(t, utf8.ValidString(got), "a cut mid-rune would fail the insert")
		})
	}
}

// TestTrimBytesKeepsMultiByteInputValid is the property the table above samples:
// no limit may ever produce invalid UTF-8, whatever the input.
func TestTrimBytesKeepsMultiByteInputValid(t *testing.T) {
	input := strings.Repeat("aä中🌯", 8)
	for bytesMax := 0; bytesMax <= len(input)+2; bytesMax++ {
		got := trimBytes(input, bytesMax)
		require.True(t, utf8.ValidString(got), "limit %d produced invalid UTF-8", bytesMax)
		require.LessOrEqual(t, len(got), bytesMax)
		require.True(t, strings.HasPrefix(input, got), "limit %d changed the content", bytesMax)
	}
}
