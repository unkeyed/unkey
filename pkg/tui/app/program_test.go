package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClampWidth(t *testing.T) {
	require.Equal(t, "abc", clampWidth("abc", 10), "short line is unchanged")
	require.Equal(t, "abc", clampWidth("abc", 3), "exact-width line is unchanged")

	// Overflowing plain text is cut to width (plus a trailing reset).
	require.Equal(t, "abcde\x1b[0m", clampWidth("abcdefghij", 5))

	// ANSI escapes don't count toward the width and are preserved.
	styled := "\x1b[1mhello world\x1b[0m"
	got := clampWidth(styled, 5)
	require.Equal(t, "\x1b[1mhello\x1b[0m", got, "keeps opening escape, cuts at 5 visible cells")
}
