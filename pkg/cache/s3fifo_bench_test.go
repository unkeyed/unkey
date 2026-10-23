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
// Output of go test -run '^$' -bench BenchmarkStore -cpu 1,16 -count 3 -benchmem:
//
//	goos: linux
//	goarch: amd64
//	pkg: github.com/unkeyed/unkey/pkg/cache
//	cpu: Intel(R) Xeon(R) Processor @ 2.60GHz
//	BenchmarkStore/get_hit/s3fifo            	23871847	        50.99 ns/op	       0 B/op	       0 allocs/op
//	BenchmarkStore/get_hit/s3fifo            	24101128	        47.79 ns/op	       0 B/op	       0 allocs/op
//	BenchmarkStore/get_hit/s3fifo            	23644620	        50.70 ns/op	       0 B/op	       0 allocs/op
//	BenchmarkStore/get_hit/s3fifo-16         	299990985	         3.349 ns/op	       0 B/op	       0 allocs/op
//	BenchmarkStore/get_hit/s3fifo-16         	360204813	         3.097 ns/op	       0 B/op	       0 allocs/op
//	BenchmarkStore/get_hit/s3fifo-16         	366911586	         3.056 ns/op	       0 B/op	       0 allocs/op
//	BenchmarkStore/get_hit/otter             	12172839	       103.5 ns/op	       0 B/op	       0 allocs/op
//	BenchmarkStore/get_hit/otter             	12013740	        97.46 ns/op	       0 B/op	       0 allocs/op
//	BenchmarkStore/get_hit/otter             	12169024	       107.9 ns/op	       0 B/op	       0 allocs/op
//	BenchmarkStore/get_hit/otter-16          	169920867	        18.17 ns/op	       0 B/op	       0 allocs/op
//	BenchmarkStore/get_hit/otter-16          	64057741	        18.44 ns/op	       0 B/op	       0 allocs/op
//	BenchmarkStore/get_hit/otter-16          	62563912	        17.59 ns/op	       0 B/op	       0 allocs/op
//	BenchmarkStore/set_evict/s3fifo          	 1615728	       740.4 ns/op	     155 B/op	       3 allocs/op
//	BenchmarkStore/set_evict/s3fifo          	 1640722	       703.0 ns/op	     155 B/op	       3 allocs/op
//	BenchmarkStore/set_evict/s3fifo          	 1718134	       716.0 ns/op	     155 B/op	       3 allocs/op
//	BenchmarkStore/set_evict/s3fifo-16       	 1725292	       696.4 ns/op	     153 B/op	       3 allocs/op
//	BenchmarkStore/set_evict/s3fifo-16       	 1752132	       750.7 ns/op	     153 B/op	       3 allocs/op
//	BenchmarkStore/set_evict/s3fifo-16       	 1640595	       807.0 ns/op	     154 B/op	       3 allocs/op
//	BenchmarkStore/set_evict/otter           	 1936186	       702.5 ns/op	      80 B/op	       1 allocs/op
//	BenchmarkStore/set_evict/otter           	 1968142	       559.4 ns/op	      80 B/op	       1 allocs/op
//	BenchmarkStore/set_evict/otter           	 2110152	       568.4 ns/op	      80 B/op	       1 allocs/op
//	BenchmarkStore/set_evict/otter-16        	  936218	      1415 ns/op	      81 B/op	       1 allocs/op
//	BenchmarkStore/set_evict/otter-16        	  967288	      1436 ns/op	      81 B/op	       1 allocs/op
//	BenchmarkStore/set_evict/otter-16        	  818215	      1263 ns/op	      81 B/op	       1 allocs/op
//	BenchmarkStore/get_or_set/s3fifo         	 3855166	       345.4 ns/op	         0.7964 hit_ratio	      40 B/op	       0 allocs/op
//	BenchmarkStore/get_or_set/s3fifo         	 3449515	       350.5 ns/op	         0.7964 hit_ratio	      40 B/op	       0 allocs/op
//	BenchmarkStore/get_or_set/s3fifo         	 3391858	       375.2 ns/op	         0.7963 hit_ratio	      40 B/op	       0 allocs/op
//	BenchmarkStore/get_or_set/s3fifo-16      	 8846104	       147.0 ns/op	         0.9461 hit_ratio	       9 B/op	       0 allocs/op
//	BenchmarkStore/get_or_set/s3fifo-16      	 9073567	       157.0 ns/op	         0.9433 hit_ratio	       9 B/op	       0 allocs/op
//	BenchmarkStore/get_or_set/s3fifo-16      	12637964	       132.9 ns/op	         0.9487 hit_ratio	       8 B/op	       0 allocs/op
//	BenchmarkStore/get_or_set/otter          	 4458972	       294.8 ns/op	         0.8018 hit_ratio	      16 B/op	       0 allocs/op
//	BenchmarkStore/get_or_set/otter          	 4536636	       298.6 ns/op	         0.8018 hit_ratio	      16 B/op	       0 allocs/op
//	BenchmarkStore/get_or_set/otter          	 4375852	       295.2 ns/op	         0.8018 hit_ratio	      16 B/op	       0 allocs/op
//	BenchmarkStore/get_or_set/otter-16       	 9487905	       138.1 ns/op	         0.9400 hit_ratio	       4 B/op	       0 allocs/op
//	BenchmarkStore/get_or_set/otter-16       	 5178060	       210.7 ns/op	         0.9157 hit_ratio	       6 B/op	       0 allocs/op
//	BenchmarkStore/get_or_set/otter-16       	 7647254	       152.2 ns/op	         0.9250 hit_ratio	       6 B/op	       0 allocs/op
//	PASS
//	ok  	github.com/unkeyed/unkey/pkg/cache	74.881s
//
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
		{
			name: "s3fifo",
			build: func(*testing.B) benchmarkStore {
				return s3fifoBenchmarkStore{store: newStore[string, int](benchmarkCapacity)}
			},
		},
		{
			name:  "otter",
			build: newOtterBenchmarkStore,
		},
	}
	workloads := []struct {
		name     string
		keyCount uint64
		run      func(s benchmarkStore, key string, hits, misses *atomic.Int64)
	}{
		{
			name:     "get_hit",
			keyCount: benchmarkCapacity / 2,
			run: func(s benchmarkStore, key string, _, _ *atomic.Int64) {
				s.get(key)
			},
		},
		{
			name:     "set_evict",
			keyCount: 1_000_000,
			run: func(s benchmarkStore, key string, _, _ *atomic.Int64) {
				s.set(key, 1)
			},
		},
		{
			name:     "get_or_set",
			keyCount: 100_000,
			run: func(s benchmarkStore, key string, hits, misses *atomic.Int64) {
				if _, ok := s.get(key); ok {
					hits.Add(1)
					return
				}
				misses.Add(1)
				s.set(key, 1)
			},
		},
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
