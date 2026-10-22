package cache

import (
	"fmt"
	"math/rand/v2"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maypok86/otter"
)

const benchmarkCapacity = 10_000

// benchmarkStore is the part of a cache that the benchmarks exercise, so the
// S3-FIFO store and otter run the same workloads.
type benchmarkStore interface {
	get(key string) (int, bool)
	set(key string, value int)
}

type s3fifoBenchmarkStore struct{ store *store[string, int] }

func (s s3fifoBenchmarkStore) get(key string) (int, bool) { return s.store.get(key) }
func (s s3fifoBenchmarkStore) set(key string, value int)  { s.store.set(key, value) }

type otterBenchmarkStore struct{ cache otter.Cache[string, int] }

func (s otterBenchmarkStore) get(key string) (int, bool) { return s.cache.Get(key) }
func (s otterBenchmarkStore) set(key string, value int)  { s.cache.Set(key, value) }

// newOtterBenchmarkStore configures otter the way cache.New did before the
// S3-FIFO store replaced it.
func newOtterBenchmarkStore(b *testing.B) benchmarkStore {
	b.Helper()
	builder, err := otter.NewBuilder[string, int](benchmarkCapacity)
	if err != nil {
		b.Fatal(err)
	}
	c, err := builder.CollectStats().
		Cost(func(string, int) uint32 { return 1 }).
		WithTTL(time.Hour).
		DeletionListener(func(string, int, otter.DeletionCause) {}).
		Build()
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(c.Close)
	return otterBenchmarkStore{cache: c}
}

// zipfKeys returns a fixed sequence of keys drawn with a Zipf distribution
// from keyCount distinct keys, so a few keys are hot and most are rare.
func zipfKeys(keyCount uint64) []string {
	rng := rand.New(rand.NewPCG(1, 2))
	zipf := rand.NewZipf(rng, 1.01, 1, keyCount-1)
	keys := make([]string, 1<<20)
	for i := range keys {
		keys[i] = fmt.Sprintf("key_%d", zipf.Uint64())
	}
	return keys
}

// BenchmarkStore compares the S3-FIFO store with otter on three workloads
// over Zipf-distributed string keys and a capacity of 10,000:
//
//	get_hit     reads keys that are all cached
//	set_evict   writes keys from a set of 1,000,000, so almost every write evicts
//	get_or_set  reads a key from a set of 100,000 and writes it on a miss
//
// Results on a 16-core Intel Xeon at 2.60GHz, median of 3 runs of
// go test -run '^$' -bench BenchmarkStore -cpu 1,16 -count 3 -benchmem:
//
//	workload    cpus  s3fifo ns/op  otter ns/op  s3fifo B/op  otter B/op
//	get_hit        1          53          101            0           0
//	get_hit       16           3.2          8.9          0           0
//	set_evict      1         763          561          155          80
//	set_evict     16        1055         1550          154          81
//	get_or_set     1         331          342           40          16
//	get_or_set    16         145          119            8           4
//
// Hit ratio for get_or_set at -cpu 1 is 0.797 for s3fifo and 0.802 for otter.
// Compare hit ratios only at -cpu 1: with more goroutines the benchmark's own
// access order changes from run to run.
//
// Reads are where S3-FIFO wins, because a hit is a lock-free sync.Map lookup
// plus at most one atomic increment. Writes allocate a node and a sync.Map
// entry and take one mutex, so a single core writes slower than otter, which
// buffers writes and applies them on a background goroutine. Under contention
// that buffering costs otter more than the mutex costs the store.
func BenchmarkStore(b *testing.B) {
	stores := []struct {
		name  string
		build func(*testing.B) benchmarkStore
	}{
		{"s3fifo", func(*testing.B) benchmarkStore {
			return s3fifoBenchmarkStore{store: newStore[string, int](benchmarkCapacity)}
		}},
		{"otter", newOtterBenchmarkStore},
	}
	workloads := []struct {
		name     string
		keyCount uint64
		run      func(s benchmarkStore, key string, hits, misses *atomic.Int64)
	}{
		{"get_hit", benchmarkCapacity / 2, func(s benchmarkStore, key string, _, _ *atomic.Int64) {
			s.get(key)
		}},
		{"set_evict", 1_000_000, func(s benchmarkStore, key string, _, _ *atomic.Int64) {
			s.set(key, 1)
		}},
		{"get_or_set", 100_000, func(s benchmarkStore, key string, hits, misses *atomic.Int64) {
			if _, ok := s.get(key); ok {
				hits.Add(1)
				return
			}
			misses.Add(1)
			s.set(key, 1)
		}},
	}
	for _, workload := range workloads {
		keys := zipfKeys(workload.keyCount)
		for _, candidate := range stores {
			b.Run(workload.name+"/"+candidate.name, func(b *testing.B) {
				s := candidate.build(b)
				for i := range min(workload.keyCount, benchmarkCapacity) {
					s.set(fmt.Sprintf("key_%d", i), 1)
				}
				var hits, misses atomic.Int64
				var offset atomic.Uint64
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					i := int(offset.Add(7919) % uint64(len(keys)))
					for pb.Next() {
						workload.run(s, keys[i&(len(keys)-1)], &hits, &misses)
						i++
					}
				})
				if total := hits.Load() + misses.Load(); total > 0 {
					b.ReportMetric(float64(hits.Load())/float64(total), "hit_ratio")
				}
			})
		}
	}
}
