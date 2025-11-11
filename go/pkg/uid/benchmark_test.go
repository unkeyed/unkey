package uid_test

import (
	"testing"

	"github.com/unkeyed/unkey/go/pkg/uid"
)

// BenchmarkComparison compares the performance of the optimized New() vs the original NewV1().
// This benchmark demonstrates the performance improvements achieved through:
// - Using math/rand/v2 ChaCha8 instead of crypto/rand
// - Cached timestamps via go-timecache to avoid time.Now() syscalls
// - Efficient string building without fmt.Sprintf
func BenchmarkComparison(b *testing.B) {
	b.Run("New_Optimized", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = uid.New(uid.KeyPrefix)
		}
	})

	b.Run("NewV1_Original", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = uid.NewV1(uid.KeyPrefix)
		}
	})
}

// BenchmarkNew benchmarks the optimized New() implementation with different configurations.
func BenchmarkNew(b *testing.B) {
	b.Run("with_prefix", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = uid.New(uid.KeyPrefix)
		}
	})

	b.Run("without_prefix", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = uid.New("")
		}
	})

	b.Run("16_bytes", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = uid.New(uid.KeyPrefix, 16)
		}
	})

	b.Run("8_bytes", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = uid.New(uid.KeyPrefix, 8)
		}
	})
}

// BenchmarkNewV1 benchmarks the original NewV1() implementation for comparison.
func BenchmarkNewV1(b *testing.B) {
	b.Run("with_prefix", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = uid.NewV1(uid.KeyPrefix)
		}
	})

	b.Run("without_prefix", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = uid.NewV1("")
		}
	})
}

// BenchmarkParallel tests UID generation under concurrent load, simulating
// real-world scenarios where multiple goroutines generate IDs simultaneously.
func BenchmarkParallel(b *testing.B) {
	b.Run("New_Optimized", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = uid.New(uid.KeyPrefix)
			}
		})
	})

	b.Run("NewV1_Original", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = uid.NewV1(uid.KeyPrefix)
			}
		})
	})
}
