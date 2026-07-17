package errors

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHumanizeBytes(t *testing.T) {
	cases := []struct {
		in   uint64
		want string
	}{
		{0, "0 B"},
		{5, "5 B"},
		{9, "9 B"},
		{10, "10 B"},
		{999, "999 B"},
		{1000, "1.0 kB"},
		{1500, "1.5 kB"},
		{16000, "16 kB"},
		{1000000, "1.0 MB"},
		{1048576, "1.0 MB"},
		{1500000, "1.5 MB"},
		{1000000000, "1.0 GB"},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, humanizeBytes(tc.in), "humanizeBytes(%d)", tc.in)
	}
}
