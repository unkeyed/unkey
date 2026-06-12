# Frontline benchmark harness

A process-level load harness for the frontline proxy hot path. It lives in
`harness/`; the optimization log lives in `todo/frontline-performance.md`.

Benchmark *results* are machine-dependent and are intentionally **not** committed
(only compare runs captured on the same hardware). Capture your own and compare.

## Process harness

`harness/run.sh` runs everything as separate OS processes so nothing shares a
Go runtime, scheduler, or GC with the proxy: hey (load), the proxy under test,
a minimal upstream, and a seeded MySQL in docker.

```bash
./benchmarks/frontline/harness/run.sh                   # stock + current
IMPLS="current" ./benchmarks/frontline/harness/run.sh
DURATION=20s WARMUP=5s CONC=64 ./benchmarks/frontline/harness/run.sh
```

### Performance knobs

The harness applies these to `stock` and `current` so the delta stays clean.
Numbers shift run to run, so only compare within a single run.

- `GOGC` (default `200`): higher trades resident memory for fewer GC cycles;
  helps the larger-body `tls16k` path most. Set `GOGC=100` for the old
  behavior.
- `GOMEMLIMIT` (default unset): e.g. `GOMEMLIMIT=768MiB` for a soft cap.
- `GOAMD64` (default `v3`): microarch level, applied at build time. No-op on
  arm64 (the M-series bench box); enables AVX2 crypto/memmove paths on amd64
  prod targets.
- `PGO` (default `auto`): captures a CPU profile from `current` under load
  via its loopback pprof endpoint, writes it to
  `build/frontline/default.pgo`, and rebuilds `current` with
  profile-guided optimization (`go build -pgo=auto` picks it up). `stock`
  stays unprofiled (the `origin/main` archive has no such file). Set
  `PGO=off` to skip. `PGO_SECONDS` (default `15`) controls profile length.
- `CPUPROFILE` (default `current`): records a CPU profile of the named
  impl over each measured load run (plain/tls/tls16k) into
  `results/<scenario>.cpu.prof`, for `go tool pprof -http=: <file>`. Use it
  to see where CPU goes (e.g. GC share vs crypto/tls vs the proxy copy
  loop). Profiling adds a few % overhead to *that* impl's rps, so set
  `CPUPROFILE=off` for a clean throughput comparison.

```bash
GOGC=300 GOMEMLIMIT=512MiB PGO=auto ./benchmarks/frontline/harness/run.sh
```

The captured `build/frontline/default.pgo` is left in the working tree.
Committing it makes every `go build` of frontline PGO-optimized. Bazel does
**not** auto-pick `default.pgo`; for the prod (`mise run build`) path add a
`pgoprofile = ":default.pgo"` attribute to the `go_binary` targets in
`build/frontline/BUILD.bazel`.

- `stock` builds the Go frontline from `origin/main` (via `git archive`)
- `current` builds from the working tree

Scenarios per implementation: plain HTTP, TLS, and TLS with a 16 KiB
upstream response. Results land in `/tmp/frontline-bench/results/` and are not
committed.
