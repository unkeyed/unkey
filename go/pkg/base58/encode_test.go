package base58

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollision(t *testing.T) {
	seen := map[string]bool{}

	for range 1_000_000 {
		input := make([]byte, 12)
		_, err := rand.Read(input)
		require.NoError(t, err)

		encoded := Encode(input)
		require.False(t, seen[encoded])
		seen[encoded] = true
	}

}

func TestEncodeDeterministic(t *testing.T) {
	input := make([]byte, 32)
	_, err := rand.Read(input)
	require.NoError(t, err)

	// Encode multiple times and ensure consistency
	results := make([]string, 1_000_000)
	for i := range len(results) {
		results[i] = Encode(input)
	}

	// All results should be identical
	for i := range len(results) {
		require.Equal(t, results[0], results[i])
	}
}

func BenchmarkEncode(b *testing.B) {

	byteLengths := []int{8, 32, 128, 512, 1024}

	for n := range byteLengths {
		input := make([]byte, byteLengths[n])
		_, err := rand.Read(input)
		require.NoError(b, err)

		b.Run(fmt.Sprintf("%d_bytes", byteLengths[n]), func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				Encode(input)
			}
		})
	}
}
