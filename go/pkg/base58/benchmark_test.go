package base58

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/mr-tron/base58"
)

// BenchmarkComparison runs a focused benchmark comparison between our implementation
// and the mr-tron/base58 reference implementation.
//
// This benchmark is useful for quickly verifying that performance characteristics
// remain identical between the two implementations.
func BenchmarkComparison(b *testing.B) {
	testCases := []struct {
		name  string
		input []byte
	}{
		{
			name:  "8_bytes",
			input: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		},
		{
			name:  "32_bytes",
			input: bytes.Repeat([]byte{0x01, 0x02, 0x03, 0x04}, 8),
		},
		{
			name:  "128_bytes",
			input: bytes.Repeat([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, 16),
		},
		{
			name:  "512_bytes",
			input: bytes.Repeat([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, 64),
		},
	}

	for _, tc := range testCases {
		// Test our implementation
		b.Run("our_"+tc.name, func(b *testing.B) {
			b.SetBytes(int64(len(tc.input)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Encode(tc.input)
			}
		})

		// Test mr-tron implementation
		b.Run("mrtron_"+tc.name, func(b *testing.B) {
			b.SetBytes(int64(len(tc.input)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				base58.Encode(tc.input)
			}
		})
	}
}

// ExampleEncode demonstrates basic usage of the base58 package.
func ExampleEncode() {
	// Encode a byte slice to base58
	data := []byte("Hello World")
	encoded := Encode(data)
	fmt.Printf("Encoded: %s\n", encoded)

	// To decode, you would need to implement a Decode function or use the reference
	decoded, err := base58.Decode(encoded)
	if err != nil {
		fmt.Printf("Decode error: %v\n", err)
		return
	}
	fmt.Printf("Decoded: %s\n", string(decoded))

	// Output:
	// Encoded: JxF12TrwUP45BMd
	// Decoded: Hello World
}
