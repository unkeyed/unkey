# Modern Go

Checked on 2026-10-13 against the official release history: Go 1.27.2 was the
newest stable release. This reference selects useful improvements from Go
1.21 through Go 1.27; it is not an exhaustive changelog. Run the skill's live
release check before using it. Minimum versions below are compatibility gates,
not instructions to upgrade a project.

## Language and collections

Prefer standard constructs over one-use helpers when the semantics match.

- [Go 1.21](https://go.dev/doc/go1.21): Use `min`, `max`, and `clear` rather
  than local equivalents. `clear(m)` deletes map entries; `clear(s)` zeroes
  slice elements without changing length or capacity. Use `slices`, `maps`,
  and `cmp` for common collection operations and ordered comparisons.
  Cloning a slice or map is shallow, not a copy of referenced objects.
- [Go 1.22](https://go.dev/doc/go1.22): Loop variables declared with `:=`
  are per-iteration. Do not add redundant `v := v` capture workarounds for
  these loops. Assignment to an existing variable with `=` still shares that
  variable. Do not copy mutexes or other non-copyable values in loop headers.
  Use `for i := range n` for a simple count from zero to `n-1`; keep a
  three-clause loop when it needs a different start, step, or condition.
- [Go 1.23](https://go.dev/doc/go1.23): Use range-over-function and
  `iter.Seq`/`iter.Seq2` when an API genuinely needs lazy iteration. A plain
  slice and ordinary range remain simpler for many callers. An iterator must
  stop when `yield` returns false. If using `iter.Pull`, call its returned
  `stop` function when abandoning iteration early. Use `slices.Collect`,
  `slices.Backward`, and `maps.Keys` instead of local equivalents.
  Map iteration is unordered: `slices.Sorted(maps.Keys(m))` produces ordered
  keys when deterministic output matters.
- [Go 1.24](https://go.dev/doc/go1.24): Use `strings.SplitSeq`,
  `strings.FieldsSeq`, and their `bytes` equivalents when consuming parts
  once without needing an intermediate slice. Do not claim zero allocations
  without measurement. Generic type aliases are available for real alias or
  migration needs, not as a reason to introduce another layer.
- [Go 1.26](https://go.dev/doc/go1.26): `new(expression)` creates a pointer
  to a variable initialized with that value. Prefer `new(value)` to a one-use
  pointer helper or temporary for optional scalar fields when clearer. Keep
  `new(T)` for a zero value and keyed literals for structured initialization.
- [Go 1.27](https://go.dev/doc/go1.27): Methods may declare their own type
  parameters. Use them only for real generic behavior belonging to a receiver;
  do not convert ordinary concrete methods merely because the feature exists.
  Interface methods cannot declare type parameters, and generic methods
  cannot implement interface methods. `strings.CutLast` and `bytes.CutLast`
  can replace last-index searches followed by manual slicing.

## Errors and concurrency

Keep ownership, cancellation, and error propagation explicit even when an API
removes bookkeeping.

- [Go 1.26](https://go.dev/doc/go1.26): Prefer
  `errors.AsType[*FieldError](err)` to a target variable plus `errors.As`
  when the minimum version permits it. The result is `(value, ok)`; choose
  the actual pointer or value error type and check `ok` before using it.
  Use `errors.Is` for sentinel identity; do not replace error matching with
  a direct type assertion that misses wrapped errors.
- [Go 1.25](https://go.dev/doc/go1.25): `sync.WaitGroup.Go` replaces the
  common `Add`/`go`/`defer Done` sequence. Start tasks before `Wait` when the
  group is empty; its callback must not panic. It does not propagate errors,
  bound concurrency, or cancel siblings. Use the repository's existing task
  group when those are required. Never put `WaitGroup.Add` inside the new
  goroutine.
- [Go 1.23](https://go.dev/doc/go1.23): With the new timer semantics,
  channel-based timers have no stale receives after `Stop` or `Reset` returns.
  Do not retain old unconditional stop-and-drain sequences that can block on
  an empty channel. The semantics depend on the main module's Go version and
  `GODEBUG` through Go 1.26. In Go 1.27, timer channels are always synchronous
  and the `asynctimerchan` fallback is removed. Check the actual version before
  simplifying old code; this does not describe `time.AfterFunc` callbacks.
- [Go 1.25](https://go.dev/doc/go1.25): The runtime's default `GOMAXPROCS`
  respects Linux container CPU limits and can update as limits change.
  Avoid setting it from `runtime.NumCPU()` or adding a helper solely to
  duplicate this behavior. An explicit setting disables automatic updates.
  Check version-dependent defaults and deployment configuration before
  removing an existing override.
- [Go 1.27](https://go.dev/doc/go1.27): The `goroutineleak` pprof profile
  is generally available without the Go 1.26 experiment. Use it to investigate
  permanently blocked goroutines. It misses some leaks, including primitives
  reachable from runnable goroutines or globals; a clean profile is not proof
  that every goroutine terminates. Keep diagnostics access-controlled.

## Tests and benchmarks

Use standard test lifecycle and synchronization APIs when they fit the
repository's testing rules.

- [Go 1.24](https://go.dev/doc/go1.24): Use `t.Context()` or `b.Context()`
  instead of an unrelated background context. They are canceled before cleanup
  functions run. Register cleanup near acquisition with `t.Cleanup`; cleanup
  work that must remain live needs its own deliberately bounded context.
- [Go 1.24](https://go.dev/doc/go1.24): Prefer `for b.Loop() { ... }` for
  new sequential benchmarks. Put setup before the loop and cleanup after it;
  both are excluded from timing. Do not combine a `B.Loop` loop with a `b.N`
  loop. Keep the exact `b.Loop()` loop condition for compiler protection
  against eliminating benchmark work. Go 1.26 removes the earlier inlining
  restriction; remeasure rather than assume old and new results are equivalent.
- [Go 1.25](https://go.dev/doc/go1.25): Consider `testing/synctest.Test`
  and `synctest.Wait` for self-contained concurrent tests with virtual time,
  not wall-clock sleeps. Real network and filesystem I/O are not virtualized.
  Keep the repository's clock and harness helpers where appropriate.
  [Go 1.27](https://go.dev/doc/go1.27) adds `synctest.Sleep` and
  `httptest.NewTestServer` for HTTP tests on an in-memory fake network.
- [Go 1.26](https://go.dev/doc/go1.26): Use `t.ArtifactDir()` for generated
  diagnostic files rather than source-tree writes. Retaining them requires the
  test runner's `-artifacts` support; otherwise the temporary directory is
  removed after the test. Use `t.TempDir()` for ordinary disposable fixtures.

## I/O, serialization, and security

Treat changed defaults as contract changes, not mechanical style improvements.

- [Go 1.24](https://go.dev/doc/go1.24): Use the JSON `omitzero` tag when
  the contract omits zero values, including zero `time.Time` values that
  `omitempty` does not omit. It uses the type's `IsZero() bool` method when
  present. Do not mechanically replace `omitempty`: empty and zero differ.
- [Go 1.24](https://go.dev/doc/go1.24): Use `os.OpenRoot` and `os.Root`
  for file access confined to a directory tree instead of string-prefix checks
  or check-then-open path validation. Close the root. Read the
  [platform limits](https://pkg.go.dev/os#Root): this is not a complete sandbox,
  and some platforms and operations retain race limitations.
- [Go 1.26](https://go.dev/doc/go1.26): `httputil.ReverseProxy.Director`
  is deprecated because malicious hop-by-hop headers can remove headers it
  adds. Use `Rewrite` (available since Go 1.20), with explicit forwarding-header
  policy. Do not build host:port strings with formatting; `net.JoinHostPort`
  handles IPv6 correctly, as the Go 1.25 `hostport` vet check enforces.
- [Go 1.27](https://go.dev/doc/go1.27): `encoding/json/v2` and
  `encoding/json/jsontext` are stable. Do not recommend the former experiment
  for this version. V2 rejects duplicate object names and invalid UTF-8 by
  default; other wire semantics differ too. Prefer v2 for new contracts when
  repository conventions permit it. The v1 API remains supported and preserves
  its semantics on the new implementation. Read the
  [migration differences](https://pkg.go.dev/encoding/json#hdr-Migrating_to_v2)
  and test the actual wire contract before switching APIs. Do not compare
  exact error text to establish error identity.
- [Go 1.27](https://go.dev/doc/go1.27): `net/url.URL.Clone` and
  `net/url.Values.Clone` provide deep copies. Use `URL.Clone` instead of a
  shallow struct copy and `Values.Clone` before modifying shared query maps.
  HTTP/1 response bodies drain a bounded amount on close for connection reuse;
  callers must still close the body and must not assume an unlimited drain.

## Tooling and performance

Use the pinned repository tools and review every suggested modernization.

- [Go 1.24](https://go.dev/doc/go1.24): `go.mod` tool directives track
  executable dependencies without a blank-import `tools.go`. Use existing
  project conventions; this repository also pins tools through mise.
- [Go 1.26](https://go.dev/doc/go1.26): `go fix` includes modernizers.
  Preview only the packages in scope with `mise exec -- go fix -diff ./path`.
  A nonempty diff returns a nonzero status; it is a proposal, not a test
  failure. Inspect behavior and compatibility before applying changes.
- [Go 1.27](https://go.dev/doc/go1.27): `go test` runs the `stdversion`
  vet check by default, catching standard-library APIs newer than the file's
  effective Go version. This is a useful check, not permission to ignore
  toolchain pins, supported builds, or downstream consumers.
- Go 1.24 changes map implementation; Go 1.26 enables the Green Tea GC and
  improves slice allocation and `io.ReadAll`; Go 1.27 improves small
  allocations. Rebenchmark real workloads with the pinned compiler before
  adding pooling, unsafe conversions, or custom replacements. Do not present
  release-note speedups as measurements of this application.
- Experimental APIs, including SIMD in Go 1.27, are not defaults for normal
  implementation. Verify current status in the release notes and require a
  concrete need before proposing an experiment or dependency change.
