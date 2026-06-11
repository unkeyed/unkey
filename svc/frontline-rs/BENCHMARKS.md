# Frontline benchmarks: Go vs Rust

Comparative load tests between the Go frontline (`svc/frontline`) and the
Rust port (`svc/frontline-rs`), measuring the proxy hot path: route lookup,
TLS termination, and forwarding to a deployment instance. Run 2026-06-10.

## TL;DR

The Rust gateway (tokio + hyper) beats the Go service in every scenario on
the same hardware, same database, same upstream, and same load generator:

| Scenario (c=64, 20s) | Go rps | Rust rps | Rust speedup | Go p50/p99 | Rust p50/p99 |
| --- | ---: | ---: | ---: | --- | --- |
| Plain HTTP proxy | 22,315 | 36,179 | 1.62× | 2.6ms / 8.8ms | 1.6ms / 4.7ms |
| TLS termination | 22,141 | 36,354 | 1.64× | 2.5ms / 9.3ms | 1.6ms / 4.9ms |
| TLS, 16 KB response | 13,460 | 29,176 | 2.17× | 4.2ms / 14.2ms | 2.0ms / 6.1ms |

All runs returned 100% HTTP 200 — no errors in either implementation.

Memory after sustained load (RSS):

| Process | Go | Rust |
| --- | ---: | ---: |
| TLS instance | 76.3 MB | 31.9 MB |
| Plain-HTTP instance | 47.8 MB | 21.7 MB |


## Methodology

Everything runs on one machine (load generator included), so absolute
numbers are not production numbers — the comparison between the two
implementations is the point.

- Hardware: Apple M1 Max, 10 cores, 32 GB RAM, macOS 15.7.1.
- Toolchains: Go 1.25.1 (`/tmp/bench/frontline-go` built from
  `./build/frontline`), Rust 1.90.0 (`cargo build --release`).
- Load generator: [hey](https://github.com/rakyll/hey) v0.1.5,
  64 concurrent keep-alive connections, 20s measured runs, each preceded
  by a discarded 5s warmup. Scenarios run sequentially, never in parallel.
- Database: MySQL 8 from `web/apps/dashboard/dev/docker-compose.yaml`
  (`mysql:3306`), seeded with one `frontline_routes` row
  (`localhost` → `dep_bench`), one region (`bench.local`), and one running
  instance pointing at the upstream. Both proxies were configured
  identically (`platform=local`, `region=bench`) except for ports.
- Upstream: a minimal Go `net/http` server on `127.0.0.1:38080` returning a
  38-byte JSON body at `/` and 16 KB at `/16k`. The "direct to upstream" row
  is the ceiling imposed by the upstream + load generator themselves.
- TLS: the same self-signed RSA-2048 `localhost` certificate served by both
  proxies from static files. TLS scenarios use
  `hey -host localhost https://127.0.0.1:<port>/` so the client sends valid
  SNI (see "Lessons" below).
- Route caches were warm in both implementations (5s fresh / 5m stale SWR
  on `frontline_routes` in both): the measured path is cache-hit routing +
  proxying, which is the steady state in production.
- Per-request access logging was directed to /dev/null for both proxies.
- Metric extraction: `Requests/sec` and the latency distribution straight
  from hey's report; RSS via `ps -o rss=` immediately after the last run.

Reproduce:

```bash
# upstream + both proxies, then per scenario:
hey -z 5s -c 64 -host localhost <url>   # warmup, discarded
hey -z 20s -c 64 -host localhost <url>  # measured
```

## History: how the Rust numbers got there

The first Rust implementation was a hand-rolled thread-per-connection HTTP
stack (project goal at the time: minimal dependencies). Benchmarks drove
three rounds of changes:

| Iteration | Plain-HTTP rps | Notes |
| --- | ---: | --- |
| v1: hand-rolled, no upstream pooling | ~860, mostly errors | Per-request upstream dials exhausted macOS ephemeral ports (`EADDRNOTAVAIL`); 502 storms. Unusable under load. |
| v2: + keep-alive pool, buffered head writes | 22,511 | Single-syscall head writes (was one `write(2)` per header fragment) took it from 3,969 to parity with Go — but intermittent 502s remained under sustained load (pooled-conn race). |
| v3: tokio + hyper rewrite (current) | 36,179 | hyper client pooling, no 502s, all scenarios clean, and the latency distribution tightened (p99 halved vs Go). |

The v3 rewrite replaced only the request hot path; routing, caching,
certificate management, config, metrics, logging, and the vault/ctrl/
ClickHouse clients are unchanged.

## Lessons / caveats

- **The earlier "tls: illegal parameter" failures were a load-tool
  artifact, not a server bug.** hey sets the TLS ServerName to the URL
  host *including the port* (`localhost:29443`), which is invalid SNI per
  RFC 6066. Go's TLS server silently tolerates it; rustls rejects it with
  `ServerNameMustContainOneHostName`. Real browsers and HTTP clients send
  valid SNI. Strictness difference worth knowing about, not a defect.
- The Go binary carries subsystems the Rust port doesn't (policy engine,
  Redis, pprof, OTEL wiring), which may account for part of the base memory
  difference; the hot path measured here does not execute any of them.
- The Rust port is HTTP/1.1-only on both sides (h2c upstreams fall back);
  the Go service can speak h2c to upstreams. Not exercised here.
- Single-machine localhost benchmarking understates network effects (real
  RTTs, congestion) and ignores multi-tenant cache churn. Treat the
  relative numbers as the signal: **~1.6× throughput on small responses,
  ~2.2× on larger bodies, roughly half the latency at every percentile,
  and ~2.3× less memory.**

## Raw results (final run)

```
name        rps          p50s    p90s    p99s    status
baseline    108046.8115  0.0005  0.0011  0.0026  100% 200
go-plain    22314.6224   0.0026  0.0051  0.0088  100% 200
rust-plain  36178.9914   0.0016  0.0024  0.0047  100% 200
go-tls      22140.7648   0.0025  0.0053  0.0093  100% 200
rust-tls    36354.0194   0.0016  0.0024  0.0049  100% 200
go-16k      13459.5105   0.0042  0.0087  0.0142  100% 200
rust-16k    29175.6328   0.0020  0.0031  0.0061  100% 200

RSS after load:
frontline-go   (tls)    76,336 KB
frontline-go   (plain)  47,776 KB
unkey-frontline (tls)   31,936 KB
unkey-frontline (plain) 21,680 KB
```
