package fuzz

import (
	"encoding/binary"
	"math/rand/v2"
	"testing"
)

// Seed adds 256 deterministic pseudo-random byte slices to the fuzz corpus.
//
// The slices vary in length from 0 to 65,025 bytes (lengths follow i² pattern).
// Because the underlying RNG uses a fixed seed, the output is identical across runs,
// making fuzz test failures reproducible.
//
// Usage:
//
//	func FuzzSomething(f *testing.F) {
//	    fuzz.Seed(f)
//
//	    f.Fuzz(func(t *testing.T, data []byte) {
//	        c := fuzz.New(t, data)
//	        // ...
//	    })
//	}
func Seed(f *testing.F) {
	var seed [32]byte = [32]byte{
		0x1f, 0x2e, 0x3d, 0x4c, 0x5b, 0x6a, 0x79, 0x88,
		0x97, 0xa6, 0xb5, 0xc4, 0xd3, 0xe2, 0xf1, 0x00,
		0x10, 0x21, 0x32, 0x43, 0x54, 0x65, 0x76, 0x87,
		0x98, 0xa9, 0xba, 0xcb, 0xdc, 0xed, 0xfe, 0x0f,
	}

	rng := rand.New(rand.NewChaCha8(seed))

	for i := range 256 {
		n := i * i

		b := []byte{}
		for len(b) < n {
			b = binary.LittleEndian.AppendUint64(b, rng.Uint64())
		}
		f.Add(b[:n])
	}
}
