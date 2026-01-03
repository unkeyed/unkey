package clock

import (
	"testing"
	"time"
)

func BenchmarkCachedClockNow(b *testing.B) {
	resolutions := []time.Duration{
		time.Microsecond,
		500 * time.Microsecond,
		time.Millisecond,
		10 * time.Millisecond,
	}

	for _, resolution := range resolutions {
		b.Run(resolution.String(), func(b *testing.B) {
			clock := NewCachedClock(resolution)
			defer clock.Close()

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = clock.Now()
				}
			})
		})
	}
}

func BenchmarkRealClockNow(b *testing.B) {
	clock := New()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = clock.Now()
		}
	})
}
