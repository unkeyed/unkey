---
title: "Testing"
description: "Testing standards and patterns for Unkey"
notion:
  owners:
    - andreas
  tags:
    - Quality
    - Testing
---

## Why we test

Tests exist to give confidence. Confidence to ship changes quickly, confidence that refactoring will not break production, confidence that the system behaves as expected. A test suite with 90 percent coverage that misses critical edge cases is less valuable than one with 60 percent coverage that catches real bugs.

We prioritize quality over quantity. A single well-designed test that validates complex business logic is worth more than a dozen tests that exercise trivial code paths. When writing tests, ask what could go wrong in production that this test would catch.

Treat test code as production code. Tests are part of the system contract, not a safety net added after implementation. They must be readable, typed, deterministic, reviewed with the same care as application code, and maintained when behavior changes.

Every test must make its guarantee clear to the reader. The guarantee is the production behavior that would break if the test failed. A future reader must be able to understand what risk the test covers without reverse engineering setup, mocks, or fixtures.

Document non-obvious guarantees with docstrings or comments. Use them when a test protects a security property, regression, invariant, concurrency condition, failure mode, or business rule that the test name cannot fully explain.

```go
// TestTokenParserRejectsAlgorithmConfusion guarantees that tokens signed with
// one algorithm cannot be accepted under another algorithm. This protects the
// verifier from accepting attacker-controlled headers as trusted policy.
func TestTokenParserRejectsAlgorithmConfusion(t *testing.T) {
    // ...
}
```

Do not document obvious mechanics. A comment that says "creates a workspace" above `createWorkspace(t)` adds noise. A comment that explains why cross-workspace access must return 404 instead of 403 documents a guarantee.

## What to test

Invest testing effort where bugs would hurt most.

**High value targets:** Business logic with complex conditionals, error handling paths, concurrent code with race potential, security sensitive operations, data transformations that could silently corrupt.

**Lower value targets:** Simple getters and setters, straightforward pass-through functions, code that delegates to well-tested libraries.

**Skip entirely:** Tests that verify the programming language works.

Ask what bug this test would catch that the compiler, a code review, or a more meaningful test would not.

### Establish an independent expectation

Before writing assertions, record the contract source, a plausible wrong
implementation, and an input where the expected and wrong results differ. Use
an accepted requirement, public protocol, independently derived calculation, or
reference model. Reading the implementation can reveal branches to exercise,
but it cannot be the sole authority for expected values.

Use asymmetric inputs and both sides of a relevant boundary. Include a positive
or unaffected control so an implementation that rejects everything cannot pass.
Reviewers must check the contract and the expected values. A linter, coverage
percentage, or killed mutation cannot make that decision.

For a regression or security feature, run the same named test against unfixed
production code or a targeted source mutation, then against the correct code.
Keep test assertions unchanged. Inspect the red failure: it must expose the
claimed wrong behavior, not a build error, unavailable service, or broken setup.
If the base cannot compile with the new test, undo only the fix in a disposable
copy of the head revision. Record any exception and obtain review rather than
claiming red evidence that you do not have.

### Verify a test with a temporary source mutation

A source mutation is a temporary change that introduces one plausible bug. It
checks whether the test detects that bug. It is not code to keep or ship, and it
does not replace an independent requirement for the expected behavior.

1. Copy the implementation and test under review into a disposable checkout,
   including relevant uncommitted changes. Save the correct implementation before
   changing it.
2. Change one condition related to the test's contract. For example, if access
   requires both grants, replace the authorization `AND` with `OR`. A credential
   with only one grant must still be expected to receive `403`, not `200`.
3. Run the named test without changing its assertions. Inspect the failure and
   confirm that it exposes the introduced bug, not a build or setup failure.
   Keep the valid-grant positive control in the test.
4. Restore the saved implementation without discarding unrelated changes. Run
   the same test and its positive control again, with caching disabled. Both must
   pass. Use repeated runs and recorded seeds when checking reliability.
5. Inspect the final diff to confirm that no mutation remains. Keep the relevant
   test, mutation diff, commands, and red/green results as review evidence. Never
   commit, push, or deploy the mutated implementation.

Use the [named witness commands](#fresh-reliability-and-named-witnesses) to record
the failing and passing runs. A mutation check is incomplete until the correct
implementation is restored and the unchanged test passes.

### Separate validation from permanent tests

Temporary probes, repetitions, and mutations establish evidence for this change.
Remove them after use. Keep a test when it protects observable behavior from a
plausible future regression that types and existing tests do not catch. Extend
an existing case before adding a file. A refactor needs existing tests to pass,
not a new test per helper. Do not retain tests of private fields, call order, or
mock interactions unless those observations are themselves the public contract.

### Testing observability

Do not verify every log line or metric increment. Test metrics that drive alerts or SLOs. Test that error conditions produce the logs operators need for debugging.

```go
func TestRateLimiter_EmitsRejectionMetric(t *testing.T) {
    collector := &testMetricCollector{}
    limiter := NewRateLimiter(Config{Limit: 1}, collector)

    limiter.Allow()
    limiter.Allow()

    require.Equal(t, 1, collector.Count("rate_limit_rejected_total"))
}
```

## Go testing

All Go tests use `github.com/stretchr/testify/require` for assertions.

Run normal Go test suites with Rask through mise. Rask keeps package-level test
runs fast. Cache misses run with a random shuffle seed; cache hits reuse a
passing result. A cached pass does not establish fresh reliability or changed
external service state. Rask does not forward arbitrary `go test` flags.

## Test organization

Tests live alongside the code they test. A file `cache.go` has its tests in `cache_test.go` in the same directory.

For integration tests that require substantial setup or external dependencies, create an `integration/` subdirectory when it improves clarity.

Organize tests around guarantees, not implementation details. File names, test names, table cases, and helper names must help a reader answer which behavior is protected and why it matters.

Use package structure, test names, and helper names to communicate test scope.
Keep unit tests close to the package under test. Put expensive cross-service
tests in an `integration/` subdirectory when that makes the boundary clearer.

## Writing test helpers

Every helper function must call `t.Helper()` as its first line.

```go
func createTestWorkspace(t *testing.T) *Workspace {
    t.Helper()
    ws, err := db.CreateWorkspace(ctx, "test-workspace")
    require.NoError(t, err)
    return ws
}
```

## Resource cleanup

Tests that acquire resources must clean them up. Use `t.Cleanup()` instead of `defer`.

```go
func TestWithDatabase(t *testing.T) {
    db := setupTestDatabase(t)
    t.Cleanup(func() {
        db.Close()
    })
}
```

## Running tests

During development, run tests for the package you are working on:

```bash
mise exec -- rask ./pkg/cache
```

Before pushing, run the full test suite:

```bash
mise run test
```

### Fresh reliability and named witnesses

Use `test-evidence` when fresh execution or red/green evidence is required. It
runs `go test -json` with result caching disabled, validates terminal events,
prints test output to stderr, and emits a JSON receipt to stdout.

```bash
mise run test-evidence -- --seeds=7,19 --repeat=3 ./pkg/cache
mise run test-evidence -- --race --seeds=7 ./pkg/cache
mise run test-evidence -- --base=origin/main --seeds=7,19
```

`--base` selects tracked changed Go directories that contain test files and
excludes `testdata`. It does not select reverse dependencies. For shared code,
select affected callers explicitly or use the full suite. An empty changed
scope is reported as no evidence. Explicit package selections must execute at
least one passing test per package. Existing intentional skips can remain in a
package run; an exact named witness must not skip or disappear.

For a red/green pair, substitute the real package and exact test or subtest name:

```bash
# Run in the disposable unfixed or mutated checkout.
mise run test-evidence -- --witness=TestBehavior/boundary \
  --expect=fail --seeds=7 --receipt=/tmp/red.json ./path/to/package
# Run with the same test assertions and the correct production code.
mise run test-evidence -- --witness=TestBehavior/boundary \
  --seeds=7 --receipt=/tmp/green.json ./path/to/package
```

The task accepts red only when the named test fails and the package exits with
failure. It cannot distinguish an assertion failure from setup that calls
`t.Fatal`, or verify the mutation's relevance. Review the full log, source diff,
independent expectation, and positive control. Receipts include the revision,
tracked diff hash, tool version, commands, exit codes, and observed outcomes.
Stage new source files with `git add -N` before recording evidence so their
contents appear in the tracked diff hash. Receipts are not signed attestations.

Repetitions are not retries. The task preserves any failure and stops before the
next seed. Record simulation/data seeds and fuzz corpus inputs separately from
shuffle seeds. Race detection does not prove all schedules are safe; run it on
concurrency changes, not every expensive suite by default. Shared test
containers use an isolated Compose project, which the task removes on exit.

### CI test rules

Depot retains the normal Rask suite and runs changed Go packages fresh with
shuffle seeds 1 and 2. `lint-test-go <base>` checks new-line findings from
selected `testifylint` checks, helper placement, and specific explained lint
suppressions. `lint-test-web <base>` checks changed TypeScript test files for
focused/skipped tests, explicit `any`, and non-null assertions, even where the
general Biome test override disables those rules. It checks the whole touched
test file, not all legacy tests. Intentional exceptions need a narrow explained
suppression and review. These static checks do not replace TypeScript typechecks.

CI verifies fresh execution and mechanical rules. Reviewers still require
red/green evidence for regressions and security features; CI does not infer
requirements or accept a PR checklist as proof.

## What is next

Use these guides for deeper patterns:

- [Unit tests](./unit-tests.md)
- [Integration tests](./integration-tests.md)
- [Anti-patterns](./anti-patterns.md)
