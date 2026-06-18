# Frontline performance investigation

2026-07-02. Scope: `svc/frontline` plus the shared packages on its hot path (`pkg/zen`, `pkg/cache`, `pkg/batch`, `pkg/timing`, `internal/services/keys`, `internal/services/ratelimit`). Method: four parallel code-analysis passes (hot path, routing/cache/DB, policy engine, transport/runtime), every claim verified against source, top findings benchmarked in isolated worktrees. Benchmarks ran on an M4 Pro (darwin/arm64, go1.26.4); absolute numbers will differ on server hardware, the ratios are structural.

## TL;DR

Frontline's proxy path is architecturally sound in the places that are usually wrong (shared pooled transports, parsed cert cache, cached policy parsing, local in-memory ratelimit, no DB N+1). The problems are concentrated in five areas:

1. zen buffers every non-gRPC request body fully into heap, uncapped, before the handler runs. This is the dominant memory issue, an OOM/DoS vector, and it defeats request streaming.
2. Observability is doing megabytes of work per request before deciding whether anyone will see it: redaction copies bodies twice with zero matches, ClickHouse capture runs even when ClickHouse is disabled, the OpenAPI validator cache key copies the whole spec per request.
3. Two unbounded memory leaks tied to deployment churn: `firewall_matches_total{policy_id}` Prometheus series and the compiled-regex cache.
4. The SWR cache degrades badly under DB slowness: blocking non-deduplicated revalidation sends can stall every request goroutine, and certificate negative caching is broken so unknown-SNI handshakes always hit MySQL.
5. Missing runtime hardening: no GOMAXPROCS/GOMEMLIMIT awareness in containers, no ReadHeaderTimeout/IdleTimeout, no h2c keep-alive health checks.

## Critical

### 1. Unbounded request body buffering in zen (memory, latency, DoS)

`pkg/zen/session.go:82`: `Session.Init` does `io.ReadAll(s.r.Body)` for every request whose Content-Type is not gRPC/Connect. Frontline passes `MaxRequestBodySize: 0` (`svc/frontline/run.go:333`, `:371`), so `http.MaxBytesReader` is never installed. The full body sits in heap before any middleware or the proxy handler runs, and forwarding cannot start until the upload completes.

Measured (`BenchmarkSessionInit`):

| body | B/op | allocs/op |
|---|---|---|
| 1KB | 8.5KB | 20 |
| 1MB | 5.2MB | 46 |
| 16MB | 102.6MB | 61 |
| 16MB, gRPC skip path | 5.6KB | 15 |

`io.ReadAll` growth allocates ~5-6x the body size. A handful of concurrent large uploads is an OOM. Bodies are buffered even for requests that routing will reject.

Fix: the proxy route should not buffer at all. Give zen a per-server (or per-route) flag to skip body buffering and treat the catch-all proxy route like the streaming path (`s.requestBody = nil`, forward `r.Body` directly). The handler's existing 1MiB TeeReader (`routes/proxy/handler.go:105-113`) already provides the ClickHouse capture. Stopgap if the flag is contentious: set a real `MaxRequestBodySize`. Note `pkg/zen/middleware_logger.go` reads `s.requestBody`; on the proxy route that logging is redundant with ClickHouse capture (see finding 3).

### 2. SWR stale path: blocking, non-deduplicated revalidation sends (outage amplifier)

`pkg/cache/cache.go:332-339`: every stale hit allocates a closure and does a blocking send into a shared 1000-slot channel drained by 10 workers. Dedup happens only inside the worker; `SWRWithFallback` (`cache.go:497-507`) has the correct dedup-before-enqueue pattern, `SWR` does not. Route cache is `Fresh: 5s` (`internal/caches/caches.go:56`), so hot hostnames take the stale path almost every request.

Failure mode: DB slows down, 10 workers park in origin fetches, duplicate closures fill the channel, then every request goroutine blocks on the send. A DB slowdown becomes a proxy stall. Demonstrated in `TestSWR_StaleHitBlocksWhenRevalidationQueueFull`: stale-hit latency goes from 12µs to a max of 11.0ms (one full origin refresh) once the queue fills.

Measured per stale hit: current 540-844ns, 116B, 4 allocs, ~0.9 origin calls per hit; with dedup-before-enqueue (SWRWithFallback pattern) 305-358ns, 21B, 2 allocs, ~0.02 origin calls per hit.

Fix in `SWR`: check `inflightRefreshes` before enqueue, and make the send non-blocking (`select ... default:` drop plus a counter metric). Separately, add singleflight on the synchronous miss path (`cache.go:346-357`): today N concurrent misses on one cold key run N origin fetches (cold start, post-eviction, and certificates: DB query + vault Decrypt + X509KeyPair each).

### 3. Certificate negative caching is broken: unknown SNI always hits MySQL

`pkg/cache/cache.go:520-525` (`SWRWithFallback`, used only by certmanager): `operation := op(err)` is computed, then the function returns early on any error before the `WriteNull` switch. `DefaultFindFirstOp` maps `sql.ErrNoRows` to `WriteNull`, but the null entry is never written. The `hit == cache.Null` branch in `internal/certmanager/service.go:80` is dead code. Plain `SWR` applies `op(err)` correctly, so route lookups do negative-cache; certificates do not.

Effect: every TLS handshake with an unregistered SNI (scanners do this constantly) runs `FindBestCertificateByCandidates` against MySQL, forever. Cache-penetration DoS against the read replica plus a full DB round trip inside the handshake. Compounded by `svc/frontline/tls.go:51`: `GetCertificate` uses `context.Background()` with no timeout, so slow DB/vault holds handshake goroutines indefinitely.

Fix: apply `op(err)` before the early return, keyed per queried hostname (canonicalKey is empty on error, use the exact hostname candidate). Add `context.WithTimeout(..., ~2s)` in the `GetCertificate` closure.

## High

### 4. WithLogging redacts and copies full bodies before sampling decides anything

`pkg/zen/middleware_logger.go:76-77` + `pkg/zen/redact.go`: two `regexp.ReplaceAll` passes over the request body and two over the response body, per request, eagerly. Go's `ReplaceAll` allocates a full copy of the input even with zero matches, and the log sampler only decides later whether to emit. Combined with finding 1, the "request body" here is the unbounded buffer.

Measured (`BenchmarkRedact`, 1MB body, zero matches): 667µs, 2.1MB, per call, and it is called twice per request. A `bytes.Contains` pre-check drops that to 0 B/op.

Fix: on the proxy route, drop body logging from `WithLogging` entirely (ClickHouse already captures bodies). Generally: make body attrs lazy (`slog.LogValuer` evaluated only on emit) and pre-check with `bytes.Contains` before running the regexes.

### 5. ClickHouse capture: unpooled MiB buffers, runs when disabled, huge retention ceiling, unsafe aliasing

- Request tee (`routes/proxy/handler.go:105-113`) and response tee (`internal/proxy/forward.go:65,142-145`) each allocate an unpooled `bytes.Buffer` up to 1MiB per request. This runs even when ClickHouse is not configured (`run.go:213` wires `batch.NewNoop`), and there is no sampling: 100% capture.
- Retention: each buffered row can pin ~2MiB (request + response strings) until the 5s flush. Worst case live heap is `BufferSize x 2MiB`. Under ClickHouse slowness this is the first OOM you will see.
- `middleware/clickhouse_logging.go:87,90` uses `unsafe.String` over `buf.Bytes()`. Safe today only because the buffers are per-request and never reused. The moment someone pools them (the obvious fix), rows silently corrupt. No comment marks this.
- Row construction costs ~59 allocs / 2.5KB / 2.8µs per request (measured delta, `BenchmarkClickHouseLogging`): `req.URL.Query()` re-parse stored in the row, `formatHeaders` string per header plus `ToLower` churn.

Fix: gate the tee on a real capture decision made up front (processor is noop, `ShouldLogRequestToClickHouse`, or a sampling knob); pool the buffers and copy owned bytes into the row at emit time (removing the `unsafe.String`); cap retained bodies at 16-64KB per row rather than 1MiB; build `QueryParams` only when `RawQuery != ""`. On the non-streaming path the request tee is fully redundant with `s.requestBody` until finding 1 is fixed.

### 6. OpenAPI validator cache keyed by the full spec content

`svc/frontline/internal/policies/openapi/executor.go:74-77`: `e.cache.SWR(ctx, string(spec), ...)` copies the entire spec per request with an openapi policy, otter hashes and equality-compares the full string, the tracing middleware `fmt.Sprintf("%v", key)`s it into a span attribute (`pkg/cache/middleware/tracing.go:139`), and the cache retains up to 64 full spec copies as keys.

Measured (`BenchmarkGetOrCompile`, 513KB spec, cache hit): current 46µs, 516KB, 4 allocs per request. Precomputed hash key: 315ns, 52B. Hashing per request is not the fix (sha256 of 500KB is 171µs, slower than the status quo); the digest must be computed once when the policy is hydrated into the policy cache (`internal/router/lookup.go:120-122`) and stored alongside the spec.

### 7. Unbounded Prometheus cardinality: `firewall_matches_total{policy_id}`

`svc/frontline/internal/policies/metrics.go:36-44`, emitted unconditionally per firewall evaluation at `engine.go:177`. One never-released time series per firewall policy ever seen by the process; deployments churn continuously, so registry memory grows monotonically. Fix: drop `policy_id` (keep `action`); per-policy attribution belongs in ClickHouse/logs, which already have it. Same pattern to check elsewhere: `PanicsTotal` labeled with raw `URL.Path` (`pkg/zen/middleware_panic_recovery.go:36`, attacker-controlled, panic-path only) and ratelimit metrics labeled by `workspace_id`.

### 8. Unbounded compiled-regex cache

`svc/frontline/internal/policies/match.go:13-43`: process-lifetime `map[string]*regexp.Regexp` keyed by raw pattern from policy match expressions. No cap, no eviction; compiled regexes are KBs each and deployment churn grows it forever. Fix: bound it (otter or small LRU, ~1024 entries).

### 9. Missing server timeouts and runtime limits

- `pkg/zen/server.go:130-131` sets only Read/WriteTimeout, which frontline disables. `ReadHeaderTimeout` and `IdleTimeout` are never set: slowloris holds goroutine+conn+FD forever, idle keep-alives are never reaped, and `RequestTimeout` defaults to 15m (`config.go:106`). Fix: `ReadHeaderTimeout` 5-10s, `IdleTimeout` 60-120s in `zen.New`; both are safe for streaming.
- No `automaxprocs`/`GOMEMLIMIT` anywhere in the repo (checked `go.mod` and all entrypoints). Under k8s CPU limits Go defaults GOMAXPROCS to host cores, causing throttling and tail-latency inflation; without GOMEMLIMIT the GC is blind to the container ceiling and OOM-kills instead of collecting harder. Fix in the shared `build/util` entrypoint so every service benefits.
- h2c upstream transport (`internal/proxy/transport.go:39-48`) has no `ReadIdleTimeout`/`PingTimeout`: a dead upstream conn (node failure) hangs all multiplexed streams until the request deadline. Fix: `ReadIdleTimeout` 15-30s, `PingTimeout` 10-15s, which also feeds the existing dial-retry loop.

## Medium

### 10. `ReverseProxy` per request, no `BufferPool`

`internal/proxy/forward.go:96-163`: fresh `httputil.ReverseProxy` plus 3 closures plus `httptrace.ClientTrace` per attempt, and no `BufferPool`, so `copyResponse` allocates a 32KB buffer per request. Measured e2e (`BenchmarkProxyE2E`, 4KB bodies): shared proxy + `sync.Pool` BufferPool cuts 33KB/op (-40% B/op). Also per attempt: `url.Parse(fmt.Sprintf("http://%s", addr))` (`service.go:246`, build the `url.URL` struct directly) and `fmt.Sprintf("%s::%s", platform, region)` recomputed 2-3x per request (`forward.go:40`, `director.go:22,51`, precompute in `New`).

### 11. `ResolveField` JSON round-trips the whole Principal per ratelimit check

`internal/policies/principal/principal.go:215-242`: `json.Marshal` + `json.Unmarshal` into `map[string]any` to read one dotted field, per request for `PrincipalField`-keyed ratelimit policies. Measured: 7.5-8.1µs, 6.8KB, 159 allocs per call vs 5-23ns, 0 allocs for a typed walk (parity-tested across 12 paths in the benchmark worktree). Related: `Principal.Marshal` for the `X-Unkey-Principal` header plus the key/identity metadata JSON parses in `KeyPrincipalFromVerifier` are recomputed per request from cache-stable data; memoize the serialized principal on the cached key entry for the API-key path.

### 12. Cache observability overhead and byte-blind sizing

- Fresh-hit SWR is 234ns/2 allocs vs 47ns/0 allocs for the raw otter lookup (5x): `fmt.Sprintf("%t")` + labels lookup per get (`cache.go:224`), a `map[string]string` per `recordTiming` (`cache.go:110-119`), and an unconditional span + key-Sprintf in the tracing middleware. Pre-resolve hit/miss counters at construction, prebuild the timing entry, guard span attrs on `IsRecording`. 3-4 cache ops per request makes this worth it.
- otter cost is hardcoded 1 (`cache.go:69-71`): `MaxSize` counts entries, not bytes. Policy cache (10k entries) holds hydrated `SpecYaml`; route cache duplicates `SentinelConfig` bytes per hostname pointing at the same deployment. Use weighted eviction or move config bytes to the deployment-keyed cache.
- Route cache `Fresh: 5s` means steady per-node DB polling proportional to hot hostnames; routes rarely change, raise to 30-60s or add event-driven invalidation from the deploy path.
- Policy parse errors are `Noop`-cached (`lookup.go:78-83`): a tenant with a permanently broken config costs a DB query + parse per request. Cache failures for a few seconds.

### 13. OpenAPI validator body handling

libopenapi-validator drains `req.Body` via `io.ReadAll` with no cap (before the handler's TeeReader is installed), then re-buffers and re-marshals for schema validation: 2-3 heap copies for validated bodies, and the validator-side read is unbounded independent of `MaxBodyCapture`. Wrap with `io.LimitReader` before `Validate` and reuse the validator's buffered bytes for the ClickHouse capture instead of teeing again.

### 14. Small per-request cleanups (each verified, sum is ~90-130 gateway allocs/request)

- Middleware chain rebuilt per request (`pkg/zen/server.go:316-320`): hoist to route registration.
- Duplicate timing entries: `ModifyResponse` and the deferred writer in `forward.go` both write `frontline`/`total` (4 header values, 2 duplicated); each `timing.Write` allocates a map + sorted keys + builder. Write once, prebuild the constant `{scope: frontline}` attribute.
- `WithRequestStartTime` context value is redundant with `RequestTracking.StartTime` set one frame earlier.
- `SecretLocations` recomputed per request (`redaction.go:20-40`, 125ns/5 allocs, unconditional at `handler.go:75`): precompute when the policy slice is cached.
- `req.URL.Query()` re-parsed up to 3x per request (keyauth extract, match eval, ratelimit keyextract): parse once and thread through.
- Eager span attribute construction when the span isn't recording (`middleware/observability.go:53-59`).

## Verified healthy (do not spend time here)

- Transports are shared and pooled correctly (`TransportRegistry`, region transport); no per-request Transport anywhere. HTTP/2 with session cache to peers.
- Certificates cached parsed (post-`X509KeyPair`, post-vault); PEM/decrypt is per fill, not per handshake.
- Policies parsed once per deployment and cached; protojson does not run per request; the cached slice is read-only.
- No DB N+1: cold miss is at most 3 sequential single queries (route+deployment join, instances+regions join, spec only if needed).
- Route lookups negative-cache correctly (unlike certificates).
- Ratelimit hot path is lock-free local counters with a janitor; no network hop steady-state.
- Key lookups SWR-cached with roles/permissions parsed at fill time.
- Error page template parsed once; error path only.
- Instance selection copies before shuffling; cached slice not mutated.

## Correctness flags found along the way (not perf)

- `WithReservedHeaderStrip` deletes all `X-Unkey-*` headers at the edge, including `X-Unkey-Frontline-Hops` / `X-Unkey-Parent-*` set by a forwarding peer, which appears to defeat cross-region hop/loop counting (`director.go:72-79`). Owner of the cross-region path should confirm.
- An upstream that responds before draining the request body causes net/http to close the body and the proxy returns a silently truncated 200 (surfaced while building the e2e benchmark).
- `unsafe.String` aliasing in `clickhouse_logging.go:87,90` (see finding 5) is one refactor away from data corruption.

## Suggested order of work

| # | Change | Findings | Effort | Wins |
|---|---|---|---|---|
| 1 | Skip body buffering on proxy routes (zen flag), stopgap MaxRequestBodySize | 1 | S | memory ceiling, TTFB, DoS |
| 2 | SWR: dedup before enqueue + non-blocking send + miss singleflight | 2 | S | outage resilience, DB load |
| 3 | Fix SWRWithFallback negative caching + GetCertificate timeout | 3 | S | DB load, handshake p99, DoS |
| 4 | automaxprocs + GOMEMLIMIT in shared entrypoint; zen ReadHeader/IdleTimeout; h2c ping | 9 | S | CPU throttling, OOM behavior, hangs |
| 5 | Drop policy_id metric label; bound regexCache | 7, 8 | S | stops both leaks |
| 6 | Hash-key the OpenAPI validator cache | 6 | S | 516KB/req -> 52B/req on openapi routes |
| 7 | Gate/pool/cap ClickHouse capture, remove unsafe.String | 5 | M | GC + retention ceiling |
| 8 | Drop body logging from proxy WithLogging (or lazy + precheck) | 4 | S | 4MB copies/req at 1MB bodies |
| 9 | Shared ReverseProxy + BufferPool | 10 | S | 33KB/req |
| 10 | Typed ResolveField; memoize principal JSON | 11 | M | 8µs+159 allocs -> ~0 per rl check |
| 11 | Cache observability slimming; byte-weighted eviction; Fresh tuning | 12 | M | steady-state CPU, heap |
| 12 | Small per-request cleanups batch | 13, 14 | M | ~50-100 allocs/req |

## Reproducing the benchmarks

Benchmark code lives in two worktrees (not committed, not in the Bazel graph; run via `go test`):

- `/Users/florianeikel/Developer/unkeyed/unkey/.claude/worktrees/agent-a146e89a33be848b4` (zen + middleware + e2e proxy)
- `/Users/florianeikel/Developer/unkeyed/unkey/.claude/worktrees/agent-a0fee938a61689f7a` (cache + policies)

```bash
# worktree 1
cd /Users/florianeikel/Developer/unkeyed/unkey/.claude/worktrees/agent-a146e89a33be848b4
mise exec -- go test -bench='BenchmarkSessionInit' -benchmem -run='^$' -count=3 ./pkg/zen/
mise exec -- go test -bench='BenchmarkRedact' -benchmem -run='^$' -count=5 ./pkg/zen/
mise exec -- go test -bench='BenchmarkClickHouseLogging' -benchmem -run='^$' -count=5 ./svc/frontline/middleware/
mise exec -- go test -bench='BenchmarkProxyE2E' -benchmem -run='^$' -count=5 ./svc/frontline/middleware/

# worktree 2
cd /Users/florianeikel/Developer/unkeyed/unkey/.claude/worktrees/agent-a0fee938a61689f7a
mise exec -- go test -bench='BenchmarkSWR_FreshHit$|BenchmarkSWR_FreshHit_Parallel|BenchmarkOtterGet_Baseline|StaleHit_EnqueueCost' -benchmem -run='^$' -count=5 ./pkg/cache/
mise exec -- go test -bench='StaleHit_Stampede' -benchmem -run='^$' -count=5 -benchtime=200000x ./pkg/cache/
mise exec -- go test -run 'TestSWR_StaleHitBlocksWhenRevalidationQueueFull' -v ./pkg/cache/
mise exec -- go test -bench='BenchmarkGetOrCompile' -benchmem -run='^$' -count=5 ./svc/frontline/internal/policies/openapi/
mise exec -- go test -bench='BenchmarkResolveField' -benchmem -run='^$' -count=5 ./svc/frontline/internal/policies/principal/
mise exec -- go test -bench='BenchmarkSecretLocations' -benchmem -run='^$' -count=5 ./svc/frontline/internal/policies/
```

Caveats: e2e ns/op is noisy over loopback TCP; B/op and allocs/op are the stable signal. Fresh-hit cache numbers understate production overhead (no zen session in ctx, so timing-header formatting was skipped). The e2e benchmark omits `WithLogging` and observability middleware, so production per-request cost is higher than measured.
