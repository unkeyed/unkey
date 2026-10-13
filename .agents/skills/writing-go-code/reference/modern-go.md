# Modern Go

Useful changes from Go 1.21 through Go 1.27. Checked on 2026-10-13, when
Go 1.27.2 was the latest stable release. Run the skill's live release check
before use. Each entry gives the minimum version; it does not require an upgrade.

## Language and collections

Use built-ins and standard collection packages before adding helpers.

- [Go 1.21](https://go.dev/doc/go1.21): `min` and `max` replace comparison
  helpers. `clear(m)` deletes map entries. `clear(s)` zeroes slice elements
  without changing length or capacity. Use `slices`, `maps`, and `cmp` for
  collection operations and ordered comparisons. Slice and map clones are
  shallow: referenced objects are still shared.
- [Go 1.22](https://go.dev/doc/go1.22): Loop variables declared with `:=` are
  separate for each iteration. Remove redundant `v := v` capture workarounds
  in these loops. Variables assigned with `=` are still shared. Do not put
  mutexes or other non-copyable values in loop headers.
  Use `for i := range n` to count from zero to `n-1`. Keep a three-clause loop
  for a different start, step, or condition.
- [Go 1.23](https://go.dev/doc/go1.23): For lazy iteration, use
  range-over-function with `iter.Seq` or `iter.Seq2`. Otherwise, keep ordinary
  slice iteration. Stop when `yield` returns false. With `iter.Pull`, call
  the returned `stop` function if you stop reading early.
  Use `slices.Collect`, `slices.Backward`, and `maps.Keys` instead of local
  equivalents. To sort map keys, use `slices.Sorted(maps.Keys(m))`.
- [Go 1.24](https://go.dev/doc/go1.24): Use `strings.SplitSeq`,
  `strings.FieldsSeq`, and their `bytes` equivalents to read parts without
  building an intermediate slice. Measure before claiming zero allocations.
  Generic type aliases support aliases and migrations; do not add an alias
  where a direct type reference works.
- [Go 1.26](https://go.dev/doc/go1.26): `new(value)` returns a pointer to a
  variable initialized with that value. Use it instead of a pointer helper
  or temporary for optional scalar fields. `new(T)` still creates a zero
  value. Use keyed literals to initialize structs.
- [Go 1.27](https://go.dev/doc/go1.27): Methods can declare type parameters.
  Use them when the receiver owns an operation that needs multiple types.
  Interface methods cannot declare type parameters, and generic methods
  cannot implement them. `strings.CutLast` and `bytes.CutLast` replace
  last-index searches followed by manual slicing.

## Errors and concurrency

A wait group counts tasks. It does not handle errors or cancellation.

- [Go 1.26](https://go.dev/doc/go1.26): Prefer
  `errors.AsType[*FieldError](err)` to a target variable plus `errors.As`.
  Check the returned `ok` before using the value. Match the actual pointer
  or value error type. Use `errors.Is` for sentinels. A direct type assertion
  does not search wrapped errors.
- [Go 1.25](https://go.dev/doc/go1.25): `sync.WaitGroup.Go` replaces the
  `Add`/`go`/`defer Done` sequence. For an empty group, start tasks before
  calling `Wait`. The callback must not panic. Use the repository's task group
  for error propagation, concurrency limits, or sibling cancellation. Never
  put `WaitGroup.Add` inside the goroutine it counts.
- [Go 1.23](https://go.dev/doc/go1.23): Channel-based timers cannot deliver
  stale values after `Stop` or `Reset` returns. Old unconditional
  stop-and-drain code can block on an empty channel. Through Go 1.26, check
  the main module's Go version and `GODEBUG` before removing it. In Go 1.27,
  timer channels are always synchronous and `asynctimerchan` is removed.
  These rules do not apply to `time.AfterFunc` callbacks.
- [Go 1.25](https://go.dev/doc/go1.25): Default `GOMAXPROCS` accounts for
  Linux container CPU limits and updates when limits change. Setting it
  explicitly disables automatic updates. Do not set it from
  `runtime.NumCPU()` or add a helper that repeats the runtime's work.
  Check deployment settings before removing an existing override.
- [Go 1.27](https://go.dev/doc/go1.27): The `goroutineleak` pprof profile
  needs no experiment flag. Use it to find permanently blocked goroutines.
  It can miss leaks involving primitives reachable from runnable goroutines
  or globals. A clean profile does not prove all goroutines terminate.
  Restrict access to diagnostics.

## Tests and benchmarks

Keep the repository's clock and harness helpers where tests already use them.

- [Go 1.24](https://go.dev/doc/go1.24): Use `t.Context()` or `b.Context()`
  instead of `context.Background()`. They are canceled before cleanup runs.
  Register `t.Cleanup` near resource acquisition. If cleanup needs a live
  context, give it a separate context with a timeout.
- [Go 1.24](https://go.dev/doc/go1.24): Use `for b.Loop() { ... }` for
  new sequential benchmarks. Setup before the loop and cleanup after it are
  not timed. Do not also loop over `b.N`. Write the condition exactly as
  `b.Loop()` so the compiler keeps benchmark work alive. Go 1.26 removes the
  earlier inlining restriction. Remeasure when converting an old benchmark.
- [Go 1.25](https://go.dev/doc/go1.25): `testing/synctest.Test` and
  `synctest.Wait` support concurrent tests with virtual time. They do not
  virtualize real network or filesystem I/O.
  [Go 1.27](https://go.dev/doc/go1.27) adds `synctest.Sleep` and
  `httptest.NewTestServer`, which uses an in-memory network.
- [Go 1.26](https://go.dev/doc/go1.26): Store diagnostic files in
  `t.ArtifactDir()`, not the source tree. The runner must pass `-artifacts`
  to retain them. Otherwise, they are removed after the test.
  Use `t.TempDir()` for disposable fixtures.

## I/O, serialization, and security

Test changes to parsers and serialization against existing inputs and outputs.

- [Go 1.24](https://go.dev/doc/go1.24): Use the JSON `omitzero` tag when
  zero values must be omitted. Unlike `omitempty`, it omits zero `time.Time`
  values. It uses `IsZero() bool` if the type defines that method.
  Do not replace `omitempty` blindly: empty and zero differ.
- [Go 1.24](https://go.dev/doc/go1.24): `os.OpenRoot` and `os.Root` confine
  file access to a directory tree. Use them instead of string-prefix checks
  or check-then-open validation. Close the root. Read the
  [platform limits](https://pkg.go.dev/os#Root), including race limitations.
  A root is not a complete sandbox.
- [Go 1.26](https://go.dev/doc/go1.26): `httputil.ReverseProxy.Director` is
  deprecated. A malicious client's hop-by-hop headers can remove headers
  added by `Director`. Use `Rewrite` (Go 1.20+) and set forwarding headers
  explicitly. Use `net.JoinHostPort` for addresses, not formatted strings,
  so IPv6 works. Go 1.25's `hostport` vet check catches this mistake.
- [Go 1.27](https://go.dev/doc/go1.27): `encoding/json/v2` and
  `encoding/json/jsontext` are stable. No experiment flag is needed.
  V2 rejects duplicate object names and invalid UTF-8 by default. Prefer v2
  for new APIs if repository conventions permit it. V1 remains supported and
  preserves its behavior. Before migrating, read the
  [migration differences](https://pkg.go.dev/encoding/json#hdr-Migrating_to_v2)
  and test request and response compatibility. Do not match errors by text.
- [Go 1.27](https://go.dev/doc/go1.27): `net/url.URL.Clone` and
  `net/url.Values.Clone` make deep copies. Use them before modifying shared
  URLs or query maps. HTTP/1 response bodies drain a limited amount on close
  to reuse the connection. You must still close the body.

## Tooling and performance

Use the pinned tools. Limit automated edits to the packages in scope.

- [Go 1.24](https://go.dev/doc/go1.24): `go.mod` tool directives replace
  blank imports in `tools.go`. Follow project conventions. This repository
  also pins tools through mise.
- [Go 1.26](https://go.dev/doc/go1.26): `go fix` includes modernizers.
  Preview changes with `mise exec -- go fix -diff ./path`. A nonempty diff
  returns a nonzero status; that is not a test failure. Review behavior and
  version compatibility before applying fixes.
- [Go 1.27](https://go.dev/doc/go1.27): `go test` runs the `stdversion` vet
  check by default. It flags standard-library APIs newer than the file's
  effective Go version. Keep testing all supported builds.
- Go 1.24 changes the map implementation. Go 1.26 enables Green Tea GC and
  improves slice allocation and `io.ReadAll`. Go 1.27 improves small
  allocations. Benchmark with the pinned compiler before adding pools,
  unsafe conversions, or custom replacements. Release-note speedups are not
  measurements of this application.
- SIMD remains experimental in Go 1.27. Check the release notes before
  proposing an experiment, and explain what the application needs it for.
