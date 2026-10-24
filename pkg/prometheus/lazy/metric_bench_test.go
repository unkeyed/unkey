package lazy

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// These benchmarks put a number on what one metric update costs on a hot
// path, so a change to the lazy wrapper or a new hot-path metric can be
// compared against the plain client_golang cost.

func BenchmarkCounterVecInc(b *testing.B) {
	opts := prometheus.CounterOpts{Name: "bench_counter_total", Help: "h"}
	labels := []string{"resource", "outcome"}

	b.Run("prometheus", func(b *testing.B) {
		c := prometheus.NewCounterVec(opts, labels)
		b.ReportAllocs()
		for b.Loop() {
			c.WithLabelValues("key", "hit").Inc()
		}
	})

	b.Run("lazy", func(b *testing.B) {
		resetState(b)
		c := NewCounterVec(opts, labels)
		b.ReportAllocs()
		for b.Loop() {
			c.WithLabelValues("key", "hit").Inc()
		}
	})

	b.Run("lazy_bound", func(b *testing.B) {
		resetState(b)
		child := NewCounterVec(opts, labels).Bind("key", "hit")
		b.ReportAllocs()
		for b.Loop() {
			child.Inc()
		}
	})

	b.Run("lazy_bound_parallel", func(b *testing.B) {
		resetState(b)
		child := NewCounterVec(opts, labels).Bind("key", "hit")
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				child.Inc()
			}
		})
	})

	b.Run("lazy_parallel", func(b *testing.B) {
		resetState(b)
		c := NewCounterVec(opts, labels)
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				c.WithLabelValues("key", "hit").Inc()
			}
		})
	})
}

func BenchmarkHistogramVecObserve(b *testing.B) {
	opts := prometheus.HistogramOpts{Name: "bench_seconds", Help: "h", Buckets: prometheus.ExponentialBuckets(100e-9, 2, 24)}
	labels := []string{"resource", "operation", "status"}

	b.Run("prometheus", func(b *testing.B) {
		h := prometheus.NewHistogramVec(opts, labels)
		b.ReportAllocs()
		for b.Loop() {
			h.WithLabelValues("key", "get", "fresh").Observe(0.000001)
		}
	})

	b.Run("lazy", func(b *testing.B) {
		resetState(b)
		h := NewHistogramVec(opts, labels)
		b.ReportAllocs()
		for b.Loop() {
			h.WithLabelValues("key", "get", "fresh").Observe(0.000001)
		}
	})

	b.Run("lazy_bound_parallel", func(b *testing.B) {
		resetState(b)
		child := NewHistogramVec(opts, labels).Bind("key", "get", "fresh")
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				child.Observe(0.000001)
			}
		})
	})

	b.Run("lazy_parallel", func(b *testing.B) {
		resetState(b)
		h := NewHistogramVec(opts, labels)
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				h.WithLabelValues("key", "get", "fresh").Observe(0.000001)
			}
		})
	})
}
