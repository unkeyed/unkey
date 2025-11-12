package base58

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/require"
)

// TestEncodeRandomBytes tests encoding with cryptographically random data of various sizes.
//
// This test is important because it exercises the full range of possible byte values
// and ensures our implementation handles edge cases correctly. Random data is more
// likely to reveal bugs than hand-picked test cases.
func TestEncodeRandom(t *testing.T) {
	// Test with random bytes of various sizes
	sizes := []int{1, 2, 4, 8, 16, 32, 64, 128, 256}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("random_%d_bytes", size), func(t *testing.T) {
			// Generate cryptographically secure random bytes
			input := make([]byte, size)
			_, err := rand.Read(input)
			require.NoError(t, err)

			// Test our implementation
			ourResult := Encode(input)

			// Test against reference implementation
			referenceResult := base58.Encode(input)
			require.Equal(t, referenceResult, ourResult, "Results should match reference implementation")

			// Test roundtrip to ensure correctness
			decoded, err := base58.Decode(ourResult)
			require.NoError(t, err)
			require.Equal(t, input, decoded)
		})
	}
}

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
