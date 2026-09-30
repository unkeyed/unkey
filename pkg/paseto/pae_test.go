package paseto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPreAuthEncode_Examples reproduces the exact PAE outputs specified in RFC
// section 2.4.1.
func TestPreAuthEncode_Examples(t *testing.T) {
	tests := []struct {
		name     string
		pieces   [][]byte
		expected []byte
	}{
		{
			name:     "no pieces",
			pieces:   [][]byte{},
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name:     "one empty piece",
			pieces:   [][]byte{{}},
			expected: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name:     "test",
			pieces:   [][]byte{[]byte("test")},
			expected: append([]byte{1, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0}, []byte("test")...),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, preAuthEncode(test.pieces...))
		})
	}
}

// TestPreAuthEncode_SeparatesPieceBoundaries guarantees partially controlled
// fields cannot produce the same authenticated bytes through concatenation.
func TestPreAuthEncode_SeparatesPieceBoundaries(t *testing.T) {
	require.NotEqual(t,
		preAuthEncode([]byte("ab"), []byte("c")),
		preAuthEncode([]byte("a"), []byte("bc")),
	)
	require.NotEqual(t,
		preAuthEncode([]byte("test")),
		preAuthEncode([]byte("test"), []byte{}),
	)
}
