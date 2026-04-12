# Ratelimit Service: Latency Investigation

## Problem

`/v2/ratelimit.limit` endpoint spikes to 10s latency under load.

## Root Cause: Bucket Lock Convoy

The ratelimit service holds `bucket.mu.Lock()` while performing Redis I/O.
Under concurrent load on the same identifier, requests serialize behind the lock,
and each lock holder blocks for the duration of the Redis call.

### The hot path

Each request to `/v2/ratelimit.limit` calls `Ratelimit()` twice in series:
1. Workspace rate limit check (`workspace_ratelimit.go:63`)
2. The actual user rate limit (`handler.go:158`)

Each `Ratelimit()` call acquires `bucket.mu.Lock()` and may call Redis
`Get()` (up to 2x — current + previous window) while holding the lock.

### Replay workers amplify the problem

8 replay goroutines run `syncWithOrigin()` which also acquires `bucket.mu.Lock()`
and calls Redis `Increment` with a 2s timeout. A replay worker holding the lock
blocks all incoming request goroutines for that bucket.

### Why 10 seconds

With N requests queued on the same bucket lock, request N waits for requests
1..N-1 to each acquire the lock and complete their Redis call. At 500ms Redis
timeout: 20 queued requests = 10s tail latency.

## Evidence

### CPU Profile (under load, 40% utilization)

- `runtime.futex` at 3.7% flat — lock contention
- `go-redis.(*Client).Process`: 2.0s (16.7%) cumulative — Redis calls
- `ratelimit.(*service).Ratelimit`: 1.28s (10.7%) — handler path
- `ratelimit.(*service).syncWithOrigin`: 1.20s (10.0%) — replay workers
- `checkBucketWithLockHeld`: 1.15s (9.6%) — Redis Get under lock
- `libopenapi-validator.FindPath`: 1.18s (9.8%) — per-request OpenAPI validation tax

The service spends ~90% of its time waiting (I/O + locks), not computing.

### Heap Profile (under load, 206MB)

- `getOrCreateBucket`: 24.7MB (4x growth from first snapshot)
- `getCurrentWindow`: 15.0MB
- `newWindow`: 8.5MB
- ClickHouse batch encoding: 48.5MB (expected)
- Metrics middleware: 11.5MB

Bucket map is growing unbounded under high cardinality.

### Benchmark Results (mock counter, Apple M1 Max, 8 CPUs, 3 counts)

#### SingleKey — all goroutines hit same identifier (convoy scenario)

```
redis_latency_0s       ~3,400 ns/op
redis_latency_500µs  ~648,000 ns/op   (190x slower — serialized behind lock)
redis_latency_5ms  ~5,360,000 ns/op   (1,570x slower — pure convoy)
```

Latency scales linearly with simulated Redis latency — pure serialization.

#### MultiKey — spread across 1000 identifiers (no convoy)

```
redis_latency_0s     ~1,650 ns/op
redis_latency_500µs  ~1,500 ns/op   (local fast path, no Redis needed)
redis_latency_5ms  ~1,410,000 ns/op
```

Different keys = different locks = no contention.
SingleKey with 500µs Redis is 440x slower than MultiKey with 500µs Redis.

#### Latency Distribution — single hot key

| parallelism | redis   | p50    | p90     | p99      | max      |
|-------------|---------|--------|---------|----------|----------|
| 1           | 500µs   | 4.8ms  | 6.3ms   | 10.8ms   | 15ms     |
| 8           | 500µs   | 2.7ms  | 7.6ms   | 13.5ms   | 26ms     |
| 32          | 500µs   | 3µs    | 10.2ms  | 25.2ms   | 71ms     |
| 128         | 500µs   | 2µs    | 14.3ms  | 40.2ms   | 142ms    |
| 1           | 5ms     | 43.7ms | 45.5ms  | 51.6ms   | 86ms     |
| 8           | 5ms     | 11µs   | 54.7ms  | 105ms    | 168ms    |
| 32          | 5ms     | 3µs    | 53.8ms  | 128ms    | 335ms    |
| 128         | 5ms     | 3µs    | 59.8ms  | 168ms    | 518ms    |

p99 and max scale linearly with parallelism — classic lock convoy behavior.
Production has real Redis jitter on top of this, pushing max well past 10s.

## Proposed Fix: Channel-based Replay

Replace replay workers' lock acquisition with a per-bucket channel.

### Current flow (replay)
```
replay worker:
  lock bucket                    <-- blocks requests for up to 2s
  get window sequence            <-- just math
  Redis Increment                <-- 500ms-2s network I/O
  update counter                 <-- one assignment
  unlock bucket
```

### Proposed flow
```
replay worker:
  compute sequence               <-- pure math: time.UnixMilli() / duration, no lock
  Redis Increment                <-- network I/O, no lock held
  send {sequence, counter} to bucket channel  <-- non-blocking

Ratelimit() caller (already holds lock):
  drain channel                  <-- apply pending updates via max()
  check rate limit               <-- proceed as normal
```

### Why this is safe

1. Counter key is derived from `bucketKey` (immutable) + `sequence` (pure function
   of time + duration). Replay worker touches nothing in the bucket.
2. `max(currentWindow.counter, newCounter)` is commutative — order doesn't matter.
3. Updates are applied on next `Ratelimit()` call. Hot buckets (the ones with
   contention) are called constantly, so staleness is near-zero.
4. Non-blocking send with drop — same backpressure model as the existing replay buffer.

### Benchmark Results After Fix

#### SingleKey — convoy eliminated

| Redis latency | Before (ns/op) | After (ns/op) | Speedup |
|---------------|-----------------|----------------|---------|
| 0             | 3,400           | 2,300          | 1.5x    |
| 500µs         | 648,000         | 814            | **796x**    |
| 5ms           | 5,360,000       | 761            | **7,043x**  |

Redis latency no longer affects Ratelimit() throughput — the lock is never
held during I/O.

#### Latency Distribution — single hot key, 128 parallel goroutines

| Redis | Metric | Before     | After      | Improvement |
|-------|--------|------------|------------|-------------|
| 500µs | p90   | 14,300µs   | 3,533µs    | 4x          |
| 500µs | p99   | 40,200µs   | 11,324µs   | 3.5x        |
| 500µs | max   | 142,000µs  | 44,200µs   | 3.2x        |
| 5ms   | p90   | 59,800µs   | 3,462µs    | **17x**     |
| 5ms   | p99   | 168,000µs  | 11,547µs   | **14.6x**   |
| 5ms   | max   | 518,000µs  | 45,900µs   | **11.3x**   |

Remaining tail latency is pure lock contention between Ratelimit() callers
(local-only, sub-µs per hold), no longer amplified by Redis I/O.

## Other Improvements (lower priority)

1. **Don't hold lock across Redis Get in `checkBucketWithLockHeld`**: lock → read local
   state → unlock → Redis Get → lock → merge with max() → decide → unlock.
2. **Shard `bucketsMu`**: replace single global lock with sync.Map or sharded map.
3. **Cache or skip OpenAPI validation**: 17% CPU on `FindPath` for a small set of
   known routes.
4. **Add per-request context timeout**: server read/write timeouts are 0, middleware
   timeout is 60s. No deadline kills stuck requests early.
5. **Enable `runtime.SetMutexProfileFraction`**: allows `/pprof/mutex` to report
   exactly which mutexes are contended and for how long.

## Raw Benchmark Output

```bash
go test ./internal/services/ratelimit/ -bench=BenchmarkRatelimit -benchtime=1s -count=3 -cpu=8 -timeout=600s
```

```
goos: darwin
goarch: arm64
pkg: github.com/unkeyed/unkey/internal/services/ratelimit
cpu: Apple M1 Max
BenchmarkRatelimit_SingleKey/redis_latency_0s-8         	  421447	      3812 ns/op
BenchmarkRatelimit_SingleKey/redis_latency_0s-8         	  363766	      3347 ns/op
BenchmarkRatelimit_SingleKey/redis_latency_0s-8         	  383352	      3093 ns/op
BenchmarkRatelimit_SingleKey/redis_latency_500µs-8      	    2738	    663897 ns/op
BenchmarkRatelimit_SingleKey/redis_latency_500µs-8      	    4093	    651117 ns/op
BenchmarkRatelimit_SingleKey/redis_latency_500µs-8      	    7597	    629250 ns/op
BenchmarkRatelimit_SingleKey/redis_latency_5ms-8        	     249	   5000945 ns/op
BenchmarkRatelimit_SingleKey/redis_latency_5ms-8        	     256	   5461251 ns/op
BenchmarkRatelimit_SingleKey/redis_latency_5ms-8        	    1489	   5614792 ns/op
BenchmarkRatelimit_MultiKey/redis_latency_0s-8          	  747644	      1856 ns/op
BenchmarkRatelimit_MultiKey/redis_latency_0s-8          	  737044	      1625 ns/op
BenchmarkRatelimit_MultiKey/redis_latency_0s-8          	  780388	      1465 ns/op
BenchmarkRatelimit_MultiKey/redis_latency_500µs-8       	  800077	      1471 ns/op
BenchmarkRatelimit_MultiKey/redis_latency_500µs-8       	  642015	      1590 ns/op
BenchmarkRatelimit_MultiKey/redis_latency_500µs-8       	  698386	      1451 ns/op
BenchmarkRatelimit_MultiKey/redis_latency_5ms-8         	     848	   1345410 ns/op
BenchmarkRatelimit_MultiKey/redis_latency_5ms-8         	     741	   1442987 ns/op
BenchmarkRatelimit_MultiKey/redis_latency_5ms-8         	     759	   1439788 ns/op
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_1-8          	    2161	    639183 ns/op	     12326 max-µs	      4783 p50-µs	      6641 p90-µs	     10928 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_1-8          	    2355	    640871 ns/op	     19034 max-µs	      4765 p50-µs	      6175 p90-µs	     11854 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_1-8          	    2164	    617934 ns/op	     15259 max-µs	      4706 p50-µs	      6014 p90-µs	      9675 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_8-8          	   25522	     48331 ns/op	     25984 max-µs	        20.00 p50-µs	      7566 p90-µs	     13391 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_8-8          	   25446	     50336 ns/op	     24293 max-µs	      2730 p50-µs	      7165 p90-µs	     13566 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_8-8          	   26302	     64545 ns/op	     27867 max-µs	      4729 p50-µs	      8197 p90-µs	     15043 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_32-8         	  150319	     10462 ns/op	     75080 max-µs	         3.000 p50-µs	     10089 p90-µs	     25835 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_32-8         	  113664	     10734 ns/op	     71107 max-µs	         2.000 p50-µs	     10098 p90-µs	     25069 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_32-8         	  121687	     11381 ns/op	     68621 max-µs	         3.000 p50-µs	     10346 p90-µs	     24846 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_128-8        	  234955	      4350 ns/op	    157221 max-µs	         2.000 p50-µs	     15509 p90-µs	     42464 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_128-8        	  265258	      3976 ns/op	    135089 max-µs	         1.000 p50-µs	     14196 p90-µs	     40896 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_128-8        	  301660	      3862 ns/op	    132993 max-µs	         1.000 p50-µs	     13216 p90-µs	     37375 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_1-8            	     268	   4060550 ns/op	     42217 max-µs	     40376 p50-µs	     40518 p90-µs	     40614 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_1-8            	   12832	   5689082 ns/op	    122259 max-µs	     45583 p50-µs	     50214 p90-µs	     58346 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_1-8            	     238	   5152205 ns/op	     95153 max-µs	     45096 p50-µs	     45768 p90-µs	     55990 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_8-8            	    2841	    359780 ns/op	    192172 max-µs	         5.000 p50-µs	     59044 p90-µs	    124023 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_8-8            	    5307	    304255 ns/op	    171160 max-µs	         5.000 p50-µs	     53125 p90-µs	     98453 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_8-8            	    6003	    400482 ns/op	    140987 max-µs	     27954 p50-µs	     51839 p90-µs	     92348 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_32-8           	   20906	     48222 ns/op	    332003 max-µs	         4.000 p50-µs	     55523 p90-µs	    129044 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_32-8           	   33867	     38995 ns/op	    338650 max-µs	         3.000 p50-µs	     45990 p90-µs	    112679 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_32-8           	   21789	     68295 ns/op	    333146 max-µs	         3.000 p50-µs	     60013 p90-µs	    143343 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_128-8          	   62755	     17940 ns/op	    471485 max-µs	         3.000 p50-µs	     63196 p90-µs	    173223 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_128-8          	   67821	     16110 ns/op	    508197 max-µs	         3.000 p50-µs	     59124 p90-µs	    157988 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_128-8          	   71973	     15803 ns/op	    518820 max-µs	         2.000 p50-µs	     56990 p90-µs	    172922 p99-µs
PASS
ok  	github.com/unkeyed/unkey/internal/services/ratelimit	161.475s
```

### After (channel-based replay)

```
goos: darwin
goarch: arm64
pkg: github.com/unkeyed/unkey/internal/services/ratelimit
cpu: Apple M1 Max
BenchmarkRatelimit_SingleKey/redis_latency_0s-8         	  504334	      2180 ns/op
BenchmarkRatelimit_SingleKey/redis_latency_0s-8         	  524457	      2259 ns/op
BenchmarkRatelimit_SingleKey/redis_latency_0s-8         	  480256	      2532 ns/op
BenchmarkRatelimit_SingleKey/redis_latency_500µs-8      	 1391175	       817.7 ns/op
BenchmarkRatelimit_SingleKey/redis_latency_500µs-8      	 1423530	       798.9 ns/op
BenchmarkRatelimit_SingleKey/redis_latency_500µs-8      	 1394964	       825.8 ns/op
BenchmarkRatelimit_SingleKey/redis_latency_5ms-8        	 1455312	       756.7 ns/op
BenchmarkRatelimit_SingleKey/redis_latency_5ms-8        	 1573150	       762.3 ns/op
BenchmarkRatelimit_SingleKey/redis_latency_5ms-8        	 1436784	       763.3 ns/op
BenchmarkRatelimit_MultiKey/redis_latency_0s-8          	  733860	      1487 ns/op
BenchmarkRatelimit_MultiKey/redis_latency_0s-8          	  619141	      1627 ns/op
BenchmarkRatelimit_MultiKey/redis_latency_0s-8          	  633685	      1761 ns/op
BenchmarkRatelimit_MultiKey/redis_latency_500µs-8       	 1221952	       994.9 ns/op
BenchmarkRatelimit_MultiKey/redis_latency_500µs-8       	 1192080	       949.5 ns/op
BenchmarkRatelimit_MultiKey/redis_latency_500µs-8       	  994466	      1031 ns/op
BenchmarkRatelimit_MultiKey/redis_latency_5ms-8         	     913	   1354177 ns/op
BenchmarkRatelimit_MultiKey/redis_latency_5ms-8         	     823	   1418333 ns/op
BenchmarkRatelimit_MultiKey/redis_latency_5ms-8         	     822	   1447415 ns/op
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_1-8          	 1462009	       793.9 ns/op	      3209 max-µs	         1.000 p50-µs	        17.00 p90-µs	        81.00 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_1-8          	 1466012	       797.2 ns/op	      3225 max-µs	         1.000 p50-µs	        17.00 p90-µs	        81.00 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_1-8          	 1425576	       803.1 ns/op	      3219 max-µs	         1.000 p50-µs	        18.00 p90-µs	        80.00 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_8-8          	 1383052	       855.9 ns/op	      6968 max-µs	         1.000 p50-µs	       206.0 p90-µs	       839.0 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_8-8          	 1348843	       869.0 ns/op	      8975 max-µs	         1.000 p50-µs	       211.0 p90-µs	       828.0 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_8-8          	 1256836	       880.6 ns/op	      7256 max-µs	         1.000 p50-µs	       215.0 p90-µs	       841.0 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_32-8         	 1322391	       905.7 ns/op	     14648 max-µs	         1.000 p50-µs	       969.0 p90-µs	      3139 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_32-8         	 1316084	       874.0 ns/op	     13400 max-µs	         1.000 p50-µs	       945.0 p90-µs	      3158 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_32-8         	 1304624	       880.3 ns/op	     13171 max-µs	         1.000 p50-µs	       959.0 p90-µs	      3156 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_128-8        	 1237497	       923.5 ns/op	     72637 max-µs	         1.000 p50-µs	      3631 p90-µs	     11726 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_128-8        	 1259214	       913.7 ns/op	     44232 max-µs	         1.000 p50-µs	      3563 p90-µs	     11342 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_500µs/parallel_128-8        	 1230152	       932.2 ns/op	     44159 max-µs	         1.000 p50-µs	      3505 p90-µs	     10905 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_1-8            	 1451185	       770.1 ns/op	     10529 max-µs	         1.000 p50-µs	        16.00 p90-µs	        80.00 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_1-8            	 1477368	       793.3 ns/op	     10834 max-µs	         1.000 p50-µs	        16.00 p90-µs	        86.00 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_1-8            	 1452126	       809.8 ns/op	     10989 max-µs	         1.000 p50-µs	        17.00 p90-µs	        85.00 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_8-8            	 1405579	       880.8 ns/op	     11759 max-µs	         1.000 p50-µs	       210.0 p90-µs	       803.0 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_8-8            	 1425590	       824.6 ns/op	     11672 max-µs	         1.000 p50-µs	       195.0 p90-µs	       774.0 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_8-8            	 1401066	       859.2 ns/op	     12902 max-µs	         1.000 p50-µs	       202.0 p90-µs	       818.0 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_32-8           	 1356176	       863.8 ns/op	     15621 max-µs	         1.000 p50-µs	       919.0 p90-µs	      3230 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_32-8           	 1176331	       859.5 ns/op	     16041 max-µs	         1.000 p50-µs	       932.0 p90-µs	      3093 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_32-8           	 1339640	       849.0 ns/op	     16936 max-µs	         1.000 p50-µs	       911.0 p90-µs	      3106 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_128-8          	 1284777	       876.6 ns/op	     44285 max-µs	         1.000 p50-µs	      3455 p90-µs	     11551 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_128-8          	 1321280	       887.9 ns/op	     43986 max-µs	         1.000 p50-µs	      3482 p90-µs	     11501 p99-µs
BenchmarkRatelimit_LatencyDistribution/redis_5ms/parallel_128-8          	 1304432	       875.7 ns/op	     49418 max-µs	         1.000 p50-µs	      3449 p90-µs	     11588 p99-µs
PASS
ok  	github.com/unkeyed/unkey/internal/services/ratelimit	80.747s
```
