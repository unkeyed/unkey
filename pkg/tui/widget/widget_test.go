package widget

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWidthStripsANSI(t *testing.T) {
	require.Equal(t, 5, Width("hello"))
	require.Equal(t, 5, Width("\x1b[1mhello\x1b[0m"), "escapes do not count")
	require.Equal(t, 3, Width("a·b"), "middle dot is one cell")
}

func TestTruncateAndPad(t *testing.T) {
	require.Equal(t, "ab…", Truncate("abcdef", 3))
	require.Equal(t, "abc", Truncate("abc", 3))
	require.Equal(t, "ab   ", Pad("ab", 5))
	require.Equal(t, "abcde", Pad("abcdefgh", 5), "over-long is hard-cut")
}

func TestSplitWidthsTileFullWidth(t *testing.T) {
	l, d, split := SplitWidths(120)
	require.True(t, split)
	require.Equal(t, 120, l+d, "boxes tile the full width")

	_, _, split = SplitWidths(80)
	require.False(t, split, "no split below threshold")
}

// A themed table highlights the cursor row and stays within display width.
func TestTableRendersWithinWidth(t *testing.T) {
	th := NewTheme(Palette{Border: 237, Title: 110, Text: 250, Dim: 242, Selected: 238, SelText: 253, OK: 73, Mid: 180, Warn: 215, Err: 174, Key: 109, Accent: 110, TabActive: 253})
	out := th.Table(60,
		[]Column{{Title: "Name", Width: 10, Dim: false}, {Title: "ID", Width: 12, Dim: true}},
		[][]string{{"alpha", "id_1"}, {"beta", "id_2"}},
		0,
	)
	require.Contains(t, out, "NAME")
	require.Contains(t, out, "alpha")
	for _, line := range strings.Split(out, "\n") {
		require.LessOrEqual(t, Width(line), 60, "no row exceeds the pane width")
	}
}

func TestBoxIsExactlyWidthAndHeight(t *testing.T) {
	th := NewTheme(Palette{}) //nolint:exhaustruct
	out := th.Box(40, 6, "hi")
	lines := strings.Split(out, "\n")
	require.Len(t, lines, 6, "box fills the height")
	for _, l := range lines {
		require.Equal(t, 40, Width(l), "every row is exactly the width")
	}
}
