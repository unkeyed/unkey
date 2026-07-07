package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseKeysCoversAppKeys(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		want string
	}{
		{"up arrow", []byte{27, '[', 'A'}, "up"},
		{"down arrow", []byte{27, '[', 'B'}, "down"},
		{"right arrow", []byte{27, '[', 'C'}, "right"},
		{"left arrow", []byte{27, '[', 'D'}, "left"},
		{"shift+tab", []byte{27, '[', 'Z'}, "shift+tab"},
		{"lone esc", []byte{27}, "esc"},
		{"enter CR", []byte{13}, "enter"},
		{"enter LF", []byte{10}, "enter"},
		{"tab", []byte{9}, "tab"},
		{"ctrl+c", []byte{3}, "ctrl+c"},
		{"ctrl+p", []byte{16}, "ctrl+p"},
		{"ctrl+n", []byte{14}, "ctrl+n"},
		{"letter q", []byte("q"), "q"},
		{"letter d", []byte("d"), "d"},
		{"question", []byte("?"), "?"},
		{"bracket open", []byte("["), "["},
		{"bracket close", []byte("]"), "]"},
		{"digit 5", []byte("5"), "5"},
		{"space", []byte(" "), " "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			keys := parseKeys(tc.in)
			require.Len(t, keys, 1, "exactly one key")
			require.Equal(t, tc.want, keys[0].String())
		})
	}
}

func TestParseKeysMultipleInOneRead(t *testing.T) {
	// A burst like "jk" then down-arrow in a single read must split cleanly.
	keys := parseKeys([]byte{'j', 'k', 27, '[', 'B'})
	require.Len(t, keys, 3)
	require.Equal(t, "j", keys[0].String())
	require.Equal(t, "k", keys[1].String())
	require.Equal(t, "down", keys[2].String())
}

func TestKeyTypesForNavHelpers(t *testing.T) {
	require.Equal(t, KeyUp, parseKeys([]byte{27, '[', 'A'})[0].Type)
	require.Equal(t, KeyEnter, parseKeys([]byte{13})[0].Type)
	require.Equal(t, KeyEsc, parseKeys([]byte{27})[0].Type)
	require.Equal(t, KeyRunes, parseKeys([]byte("k"))[0].Type)
}
