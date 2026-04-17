package ratelimit

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockCounter simulates a Redis counter with configurable latency.
// It uses atomic operations internally so the mock itself doesn't
// introduce lock contention that would skew the benchmark.
type mockCounter struct {
	latency time.Duration
	data    sync.Map // string -> *atomic.Int64
}

func (m *mockCounter) load(key string) *atomic.Int64 {
	val, _ := m.data.LoadOrStore(key, &atomic.Int64{})
	return val.(*atomic.Int64)
}

func (m *mockCounter) Increment(_ context.Context, key string, value int64, _ ...time.Duration) (int64, error) {
	if m.latency > 0 {
		time.Sleep(m.latency)
	}
	return m.load(key).Add(value), nil
}

func (m *mockCounter) Get(_ context.Context, key string) (int64, error) {
	if m.latency > 0 {
		time.Sleep(m.latency)
	}
	return m.load(key).Load(), nil
}

func (m *mockCounter) MultiGet(_ context.Context, keys []string) (map[string]int64, error) {
	if m.latency > 0 {
		time.Sleep(m.latency)
	}
	out := make(map[string]int64, len(keys))
	for _, k := range keys {
		out[k] = m.load(k).Load()
	}
	return out, nil
}

func (m *mockCounter) Decrement(_ context.Context, key string, value int64, ttl ...time.Duration) (int64, error) {
	return m.Increment(context.Background(), key, -value, ttl...)
}

func (m *mockCounter) DecrementIfExists(_ context.Context, key string, value int64) (int64, bool, bool, error) {
	if m.latency > 0 {
		time.Sleep(m.latency)
	}
	v := m.load(key)
	cur := v.Load()
	if cur == 0 {
		return 0, false, false, nil
	}
	if cur < value {
		return cur, true, false, nil
	}
	return v.Add(-value), true, true, nil
}

func (m *mockCounter) SetIfNotExists(_ context.Context, key string, value int64, _ ...time.Duration) (bool, error) {
	v := &atomic.Int64{}
	v.Store(value)
	_, loaded := m.data.LoadOrStore(key, v)
	return !loaded, nil
}

func (m *mockCounter) Delete(_ context.Context, key string) error {
	m.data.Delete(key)
	return nil
}

func (m *mockCounter) Close() error { return nil }

// newBenchService creates a ratelimit service with a mock counter.
func newBenchService(b *testing.B, latency time.Duration) *service {
	b.Helper()

	ctr := &mockCounter{latency: latency}

	svc, err := New(Config{Counter: ctr})
	if err != nil {
		b.Fatal(err)
	}

	return svc
}

// BenchmarkRatelimit_SingleKey measures throughput when all goroutines hit the
// same identifier — the worst-case convoy scenario for bucket.mu.
func BenchmarkRatelimit_SingleKey(b *testing.B) {
	for _, latency := range []time.Duration{0, 500 * time.Microsecond, 5 * time.Millisecond} {
		b.Run(fmt.Sprintf("redis_latency_%s", latency), func(b *testing.B) {
			svc := newBenchService(b, latency)
			b.Cleanup(func() { _ = svc.Close() })

			ctx := context.Background()
			req := RatelimitRequest{
				Name:       "benchmark",
				Identifier: "user-1",
				Limit:      1_000_000,
				Duration:   time.Minute,
				Cost:       1,
			}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, err := svc.Ratelimit(ctx, req)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

// BenchmarkRatelimit_MultiKey measures throughput when requests spread across
// many identifiers — tests bucketsMu (global map lock) contention plus
// cold-window Redis origin fetches.
func BenchmarkRatelimit_MultiKey(b *testing.B) {
	for _, latency := range []time.Duration{0, 500 * time.Microsecond, 5 * time.Millisecond} {
		b.Run(fmt.Sprintf("redis_latency_%s", latency), func(b *testing.B) {
			svc := newBenchService(b, latency)
			b.Cleanup(func() { _ = svc.Close() })

			ctx := context.Background()
			var keyIdx atomic.Uint64

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					id := keyIdx.Add(1)
					req := RatelimitRequest{
						Name:       "benchmark",
						Identifier: fmt.Sprintf("user-%d", id%1000),
						Limit:      1_000_000,
						Duration:   time.Minute,
						Cost:       1,
					}
					_, err := svc.Ratelimit(ctx, req)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

// BenchmarkRatelimit_LatencyDistribution measures p50/p90/p99/max latency
// under concurrent load on a single key at varying parallelism levels.
// This is the scenario that produces tail latency spikes in production.
func BenchmarkRatelimit_LatencyDistribution(b *testing.B) {
	// 500µs simulates a healthy Redis, 5ms simulates Redis under load
	for _, latency := range []time.Duration{500 * time.Microsecond, 5 * time.Millisecond} {
		for _, parallelism := range []int{1, 8, 32, 128} {
			name := fmt.Sprintf("redis_%s/parallel_%d", latency, parallelism)
			b.Run(name, func(b *testing.B) {
				svc := newBenchService(b, latency)
				b.Cleanup(func() { _ = svc.Close() })

				ctx := context.Background()
				req := RatelimitRequest{
					Name:       "benchmark",
					Identifier: "hot-key",
					Limit:      1_000_000,
					Duration:   time.Minute,
					Cost:       1,
				}

				var mu sync.Mutex
				latencies := make([]time.Duration, 0, b.N)

				b.SetParallelism(parallelism)
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					local := make([]time.Duration, 0, 256)
					for pb.Next() {
						start := time.Now()
						_, err := svc.Ratelimit(ctx, req)
						elapsed := time.Since(start)
						if err != nil {
							b.Fatal(err)
						}
						local = append(local, elapsed)
					}
					mu.Lock()
					latencies = append(latencies, local...)
					mu.Unlock()
				})
				b.StopTimer()

				if len(latencies) == 0 {
					return
				}
				sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

				p50 := latencies[len(latencies)*50/100]
				p90 := latencies[len(latencies)*90/100]
				p99 := latencies[len(latencies)*99/100]
				pMax := latencies[len(latencies)-1]

				b.ReportMetric(float64(p50.Microseconds()), "p50-µs")
				b.ReportMetric(float64(p90.Microseconds()), "p90-µs")
				b.ReportMetric(float64(p99.Microseconds()), "p99-µs")
				b.ReportMetric(float64(pMax.Microseconds()), "max-µs")
			})
		}
	}
}
