package batchrand_test

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/batchrand"
)

// TestUniqueness verifies that Read generates unique random bytes
func TestUniqueness(t *testing.T) {
	batchrand.Reset()

	seen := make(map[string]bool)
	iterations := 10000

	for i := 0; i < iterations; i++ {
		buf := make([]byte, 16)
		err := batchrand.Read(buf)
		require.NoError(t, err)

		key := hex.EncodeToString(buf)
		require.False(t, seen[key], "Duplicate random bytes found at iteration %d: %s", i, key)
		seen[key] = true
	}
}

// TestConcurrentUniqueness verifies that concurrent reads produce unique bytes
func TestConcurrentUniqueness(t *testing.T) {
	batchrand.Reset()

	const goroutines = 10
	const readsPerGoroutine = 1000

	var wg sync.WaitGroup
	mu := sync.Mutex{}
	seen := make(map[string]bool)
	duplicates := 0

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < readsPerGoroutine; j++ {
				buf := make([]byte, 16)
				err := batchrand.Read(buf)
				require.NoError(t, err)

				key := hex.EncodeToString(buf)

				mu.Lock()
				if seen[key] {
					duplicates++
				}
				seen[key] = true
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	require.Equal(t, 0, duplicates, "Found %d duplicates in concurrent execution", duplicates)
	require.Equal(t, goroutines*readsPerGoroutine, len(seen), "Expected %d unique values", goroutines*readsPerGoroutine)
}

// TestDifferentSizes verifies that Read works correctly with various buffer sizes
func TestDifferentSizes(t *testing.T) {
	sizes := []int{1, 4, 8, 12, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 4097, 8192}

	for _, size := range sizes {
		t.Run(hex.EncodeToString([]byte{byte(size >> 8), byte(size)}), func(t *testing.T) {
			batchrand.Reset()

			buf := make([]byte, size)
			err := batchrand.Read(buf)
			require.NoError(t, err)

			// Verify all bytes were written (not all zeros unless extremely unlucky)
			allZeros := true
			for _, b := range buf {
				if b != 0 {
					allZeros = false
					break
				}
			}
			require.False(t, allZeros, "Buffer of size %d is all zeros (extremely unlikely)", size)
		})
	}
}

// TestLargeRequest verifies that requests larger than pool size work correctly
func TestLargeRequest(t *testing.T) {
	batchrand.Reset()

	// Request larger than 4KB pool
	buf := make([]byte, 8192)
	err := batchrand.Read(buf)
	require.NoError(t, err)

	// Verify randomness
	allZeros := true
	for _, b := range buf {
		if b != 0 {
			allZeros = false
			break
		}
	}
	require.False(t, allZeros, "Large buffer is all zeros")
}

// TestMultipleSmallReads verifies pool refilling logic
func TestMultipleSmallReads(t *testing.T) {
	batchrand.Reset()

	// Make enough small reads to exhaust the pool multiple times
	// Pool is 4KB, so 4096 / 12 = 341 reads will exhaust it once
	// Do 1000 reads to ensure multiple refills
	seen := make(map[string]bool)

	for i := 0; i < 1000; i++ {
		buf := make([]byte, 12)
		err := batchrand.Read(buf)
		require.NoError(t, err)

		key := hex.EncodeToString(buf)
		require.False(t, seen[key], "Duplicate found at read %d after pool refill", i)
		seen[key] = true
	}
}

// TestReset verifies that Reset forces pool refill
func TestReset(t *testing.T) {
	batchrand.Reset()

	// Read some bytes
	buf1 := make([]byte, 16)
	err := batchrand.Read(buf1)
	require.NoError(t, err)

	// Reset and read again - should get different bytes
	batchrand.Reset()
	buf2 := make([]byte, 16)
	err = batchrand.Read(buf2)
	require.NoError(t, err)

	require.NotEqual(t, buf1, buf2, "Reset should cause new random bytes to be generated")
}

// TestEmptyBuffer verifies that zero-length reads are handled
func TestEmptyBuffer(t *testing.T) {
	buf := make([]byte, 0)
	err := batchrand.Read(buf)
	require.NoError(t, err)
}

// TestRace runs tests with race detector enabled
func TestRace(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race test in short mode")
	}

	batchrand.Reset()

	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				buf := make([]byte, 16)
				_ = batchrand.Read(buf)
			}
		}()
	}

	wg.Wait()
}

// BenchmarkRead benchmarks the batched approach
func BenchmarkRead(b *testing.B) {
	b.Run("batchrand_12bytes", func(b *testing.B) {
		b.ReportAllocs()
		buf := make([]byte, 12)
		for i := 0; i < b.N; i++ {
			_ = batchrand.Read(buf)
		}
	})

	b.Run("crypto_rand_12bytes", func(b *testing.B) {
		b.ReportAllocs()
		buf := make([]byte, 12)
		for i := 0; i < b.N; i++ {
			_, _ = rand.Read(buf)
		}
	})
}

// BenchmarkReadParallel benchmarks concurrent access
func BenchmarkReadParallel(b *testing.B) {
	b.Run("batchrand_12bytes", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			buf := make([]byte, 12)
			for pb.Next() {
				_ = batchrand.Read(buf)
			}
		})
	})

	b.Run("crypto_rand_12bytes", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			buf := make([]byte, 12)
			for pb.Next() {
				_, _ = rand.Read(buf)
			}
		})
	})
}

// BenchmarkReadSizes benchmarks different buffer sizes
func BenchmarkReadSizes(b *testing.B) {
	sizes := []int{8, 12, 16, 32, 64, 128, 256, 4096, 8192}

	for _, size := range sizes {
		b.Run(hex.EncodeToString([]byte{byte(size >> 8), byte(size)}), func(b *testing.B) {
			b.ReportAllocs()
			buf := make([]byte, size)
			for i := 0; i < b.N; i++ {
				_ = batchrand.Read(buf)
			}
		})
	}
}
