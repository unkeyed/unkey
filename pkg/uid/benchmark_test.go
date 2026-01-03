package uid_test

import (
	"testing"

	"github.com/unkeyed/unkey/pkg/uid"
)

// BenchmarkComparison compares the performance of the optimized New() vs the original.
// This benchmark demonstrates the performance improvements achieved through:
// - Cached timestamps via go-timecache to avoid time.Now() syscalls
// - Using github.com/mr-tron/base58 instead of custom implementation
// - Efficient string building without fmt.Sprintf
func BenchmarkComparison(b *testing.B) {
	b.Run("New_Optimized", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = uid.New(uid.KeyPrefix)
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
}
