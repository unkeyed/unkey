# Go Style Best Practices

Source: https://google.github.io/styleguide/go/best-practices

These are documented conventions at the "idiomatic" level. The source is neither normative nor canonical; it supplements Google's core Go Style Guide and Style Decisions.

## About

- Apply this guidance to common cases; use judgment when constraints differ.
- Treat the core Go Style Guide and Go Style Decisions as authoritative over this summary.
- Optimize for readable call sites, explicit contracts, useful errors, hermetic tests, and replaceable dependencies.

## Naming

### Function and method names

#### Avoid repetition

- Read a proposed name at its call site before accepting it.
- Omit the receiver type from a method name.
- Omit pointer-ness from function and method names.
- Do not repeat the package name in a package function.
- Do not repeat parameter names or types in the function name.
- Do not repeat return names or types unless they disambiguate sibling operations.

```go
// Bad: yamlconfig.ParseYAMLConfig(input)
// Good: yamlconfig.Parse(input)

// Bad: cfg.WriteConfigTo(w)
// Good: cfg.WriteTo(w)
```

- Add qualifiers when operations genuinely differ, such as `WriteTextTo` and `WriteBinaryTo`.

#### Naming conventions

- Give accessors and other value-returning operations noun-like names: `JobName`, not `GetJobName`.
- Give operations that perform actions verb-like names: `WriteDetail`.
- Put a distinguishing type suffix last: `ParseInt`, `ParseInt64`, `AppendInt64`.
- Omit the suffix from the clear primary form: use `Marshal` beside `MarshalText`.

### Test double and helper packages

#### Creating test helper packages

- Name a test support package by appending `test` to the production package: `creditcardtest`.
- Mark a Bazel test-helper `go_library` as `testonly = True`.
#### Simple case

- Use a concise double name when the package doubles only one production type: `creditcardtest.Stub`, not `StubService` or `StubCreditCardService`.
- A dumb data type, like a protobuf message, ordinarily needs no double.
- This conventional arrangement is usually readily understood by other engineers.

#### Multiple test double behaviors

- Name multiple doubles by behavior when behavior distinguishes them: `AlwaysCharges`, `AlwaysDeclines`.

#### Multiple doubles for multiple types

- Name doubles by production type when one package doubles several types: `StubService`, `StubStoredValue`.

#### Local variables in tests

- Name local doubles to distinguish them from production objects: `spyCC`, not an ambiguous `cc`.

### Shadowing

- Reassign with `:=` only when stomping a value in the same scope and the old value is no longer needed.
- Avoid accidentally shadowing a variable in an inner scope.
- Predeclare additional outputs and use `=` when a replacement must survive the block.

```go
// Bad: inner ctx disappears after the if block.
if shorten {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
}

// Good: update the outer ctx.
if shorten {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
}
```

- Shadowing may intentionally introduce a different type; use a new name when that is clearer.
- Do not shadow standard package names except in tiny, unmistakable scopes.
- Avoid package names that force clients to rename imports or shadow natural variables.

### Util packages

- Name a package for the focused capability it provides.
- Naming a package just `util`, `helper`, `common`, or similar is usually a
  poor choice; such words are fine as part of a longer name (`spannertest`,
  `creditcardtest`).
- Judge the package name together with exported identifiers at call sites.
- Prefer informative expressions such as `spannertest.NewDatabaseFromFile`, `io.SeekStart`, and `elliptic.Marshal`.

## Package size

- Keep types together when client code normally needs them to interact.
- Keep tightly coupled implementations together when splitting would expose internals merely to reconnect them.
- Combine two packages when meaningful use of either almost always requires importing both.
- Split conceptually distinct capabilities into packages that stand on their own.
- Let package and exported names combine meaningfully: `bytes.Buffer`, `ring.New`.
- Do not put an entire project in one package.
- Do not enforce one type per file.
- Avoid both many-thousand-line files and swarms of tiny files.
- Group related implementation into predictably named files within a package.
- Put long package documentation in `doc.go` when useful, but do not require it.
- Prefer one package per directory when future open sourcing is plausible.

## Imports

### Protocol Buffer Messages and Stubs

- Rename generated proto imports with descriptive whole-word prefixes.
- Suffix message imports with `pb` and gRPC stub imports with `grpc`.
- Prefer `foopb` and `foogrpc` over generic `pb`, `xpb`, or ambiguous abbreviations.
- When uncertain, derive the alias from the proto package name through `_go`, then add `pb`.

```go
import (
	foopb "path/to/foo_service_go_proto"
	foogrpc "path/to/foo_service_go_grpc"
)
```

- Do not churn old short aliases solely to conform; use descriptive aliases in new code.
### Import ordering

- Follow the Go Style Decisions import-grouping rules.

## Error handling

- Treat errors as values consumed by code and humans.
- Handle, enrich, translate, or propagate each error deliberately; consider
  whether the current call frame is the one best positioned to handle it.
- Ignore secondary errors only when orchestration makes the first error authoritative; consider `errgroup`.

### Error structure

- Give errors structure whenever callers need to distinguish conditions.
- Export a sentinel for an unparameterized condition.
- Compare direct sentinels with `==` only when wrapping cannot occur.
- Prefer `errors.Is` when an error may be wrapped.
- Never inspect `err.Error()` with string equality or regular expressions.

```go
var ErrDuplicate = errors.New("duplicate")

func Add(name string) error {
	if alreadyPresent(name) {
		return ErrDuplicate
	}
	return nil
}

if err := Add(name); errors.Is(err, ErrDuplicate) {
	// Handle the documented condition.
}
```

- Define a custom error type when callers need fields such as operation, path, code, or invalid value; `os.PathError` is the model.
- Implement `Unwrap() error` when a custom type preserves an underlying cause.
- Use `errors.As` to extract custom errors; match pointer versus value form correctly.

```go
type FieldError struct {
	Field string
	Err   error
}

func (e *FieldError) Error() string { return e.Field + ": " + e.Err.Error() }
func (e *FieldError) Unwrap() error { return e.Err }

var fieldErr *FieldError
if errors.As(err, &fieldErr) {
	handleInvalidField(fieldErr.Field)
}
```

- A project error struct with code and detail or gRPC `status` may be appropriate, but is not obligatory. If using status, use canonical codes and first decide whether status codes fit.

### Adding information to errors

- Add context available at the current layer that neither caller nor callee can provide.
- Do not repeat paths, operations, or other details already present in the underlying error.
- Return the original error instead of adding empty phrases such as `failed` or `operation failed`.
- Use `%v` when intentionally creating a human-readable independent error and hiding the original structure.
- Use `%w` when callers should inspect the original error with `errors.Is` or `errors.As`.
- Treat wrapping behavior as part of a package's API contract; document and test exposed causes.
- Translate internal errors at RPC, IPC, storage, or other system boundaries when callers need canonical domain results.

```go
// Bad: duplicates details from os.Open and preserves no useful contract.
return fmt.Errorf("could not open settings.txt: %v", err)

// Good: adds domain context but intentionally hides the underlying error.
return fmt.Errorf("launch codes unavailable: %v", err)

// Good: adds operation context and preserves a documented cause.
return fmt.Errorf("load launch settings: %w", err)
```

### Placement of `%w` in errors

- Put `%w` at the end as `context: %w` for ordinary wrapping.
- Make printed order mirror chain order: newest context first, oldest cause last.
- Do not place ordinary `%w` first or in the middle of prose.

```go
// Good: newest-to-oldest chain in text and structure.
err = fmt.Errorf("read profile: %w", err)

// Bad: text reverses the chain.
err = fmt.Errorf("%w: read profile", err)

// Bad: text interleaves layers.
err = fmt.Errorf("read %w profile", err)
```

#### Sentinel error placement

- Exception: put a sentinel `%w` first when it is the primary category readers should see immediately, such as not found, invalid argument, or an OS/package sentinel.
- Preserve detailed causes with `%v` after the leading category when they are not part of the inspectable chain.

```go
var ErrParse = errors.New("parse error")
var ErrInvalidHeader = fmt.Errorf("%w: invalid header", ErrParse)

return fmt.Errorf("%w: invalid character: %v", ErrInvalidHeader, cause)
```

### Logging errors

- Log enough relevant context to diagnose what failed.
- Do not both log and return the same error by default; let the eventual handler choose logging, recovery, rate limiting, or exit. The tradeoff: caller-side logging reports the caller's source coordinates.
- Keep PII out of logs and other unsuitable sinks.
- Reserve error-level logging for actionable events; account for flush and performance cost.
- Use monitoring and metrics for alerting instead of hoping someone reads logs.
#### Custom verbosity levels

- Define consistent verbosity levels, for example small extra detail at V(1), traces at V(2), and large state dumps at V(3).
- Guard expensive verbose-log arguments explicitly so they are not evaluated when disabled.

```go
if log.V(2) {
	log.Infof("query plan: %v", query.Explain())
}
```

### Program initialization

- Propagate initialization and configuration errors to `main`.
- Exit from `main` (the source uses `log.Exit`) with an actionable message that tells the operator how to fix the problem.
- Prefer that message over a stack trace from a deep `Fatal` call.

### Program checks and panics

- Return errors from libraries, especially for transient or recoverable failures.
- Terminate only when a violated invariant makes internal state unrecoverable; in Google's environment, prefer `log.Fatal` because panic cleanup may deadlock or further corrupt state.
- Do not broadly recover unexpected panics and continue with potentially corrupted state.

#### When to panic

- Panic for clear programmer API misuse or impossible production conditions, only when tests and review should catch them.
- Use panic/recover internally only in tightly coupled code where the public boundary always converts owned panic types to errors.
- Re-panic values not created by that internal mechanism.
- Never let implementation-detail panics cross package boundaries.
- Ensure deferred cleanup releases resources around panic-based control flow.
- Use a final panic only to satisfy the compiler after a truly non-returning call.
- Avoid logging before flags are parsed; panic may be acceptable in `init` or `Must` construction.

## Documentation

### Conventions

- Document APIs in familiar Godoc style and add runnable examples for important usage.
#### Parameters and configuration

- Explain non-obvious constraints, defaults, side effects, and failure modes.
- Do not mechanically enumerate every function parameter, method parameter, struct field, or option; explain why error-prone or non-obvious ones matter.
- Write at a depth suitable for maintainers, newcomers, external users, and your future self.

#### Contexts

- Assume cancellation interrupts a context-taking function and returns `ctx.Err()` when it can return an error.
- Do not restate conventional cancellation behavior.
- Document deviations: returning nil or another error, asynchronous cancellation, alternate stop mechanisms, lifetime assumptions, lineage, or required values.
- Avoid APIs that demand unusual context properties, such as no deadline.

#### Concurrency

- Assume conceptually read-only operations are safe for concurrent use.
- Assume visibly mutating operations require caller synchronization.
- Document surprising mutation, such as an LRU lookup that updates recency.
- Document synchronization supplied by the API.
- Put whole-type concurrency guarantees on the type when they cover every method.
- Document concurrency requirements imposed on user implementations of an interface.

#### Cleanup

- State every caller-owned cleanup obligation.
- Name the cleanup method and when to call it.
- Explain response/body ownership and nil guarantees where ambiguity could leak resources.
- Document when and how the caller must perform cleanup.

#### Errors

- Document significant sentinels and custom error types callers can handle.
- State whether a returned custom error is a pointer or value type.
- Keep the function's return type as `error` when needed to avoid typed-nil interface traps.
- Put package-wide error conventions in package documentation.

### Preview

- Preview rendered documentation before and during review to validate Godoc formatting.

### Godoc formatting

- Separate paragraphs with blank comment lines.
- Put runnable examples in test files so godoc attaches them to the symbol.
- Indent comment text by two additional spaces for verbatim blocks; use
  verbatim formatting as the workaround for unsupported lists and tables.
- A standalone capitalized line without punctuation, followed by a paragraph,
  renders as an anchored heading.

### Signal boosting

- Add a signal-boosting comment to visually surprising logic, such as `err == nil`.

## Variable declarations

### Initialization

- Use `:=` for a new local initialized to a non-zero value.

### Declaring variables with zero values

- Use `var` to communicate an empty zero value ready for later use.
- Do not spell out zero-valued fields or elements in composite literals.
- Initialize maps before mutation; permit reads from nil maps.
- Use pointer protobuf messages and pointer values for objects that must not be copied.
- Store mutexes as values and use pointer receivers on containing types.

### Composite literals

- Use a composite literal when initial members are known.
- Use `new(T)` or `&T{}` for a pointer to a zero value; choose the clearer signal.

### Size hints

- Most code should let the runtime grow slices and maps; preallocate when the
  final size is known or measurements justify it.
- Avoid speculative oversized allocations; benchmark performance-sensitive hints.

### Channel direction

- Specify `<-chan T` or `chan<- T` in parameters whenever ownership permits.
- Let channel direction communicate ownership and make invalid sends, receives, or closes fail at compile time.

## Function argument lists

- Keep signatures short enough that parameter roles remain memorable and adjacent same-typed arguments cannot be swapped easily.
- Split use-case-specific operations before adding configuration machinery.
### Option structure

- Use an options struct when all callers set options, many callers set many options, configurations are shared, or field-level documentation helps.
- Pass an options struct last; export it only for exported APIs.
- Never put `context.Context` inside an options struct.

### Variadic options

- Use functional options only when most callers need none, options are numerous or uncommon, or options need rich validation and documentation.
- Variadic options add substantial implementation code; use them only when their call-site and extensibility benefits outweigh that overhead.
- Options are functions returning closures that mutate an options value, usually a pointer to an unexported struct.
- They can take several arguments, be shared or accumulated, have a named Godoc-grouped type, return errors, and optionally permit third-party options.
- Prefer one option accepting an enum constant over proliferating format-specific option functions.
- Accept values in option functions: prefer `FailFast(enabled bool)` over separate enable/disable options.
- Process options in order and let the last non-cumulative option win.
- Keep the mutable options representation unexported unless third-party options are a deliberate extension point.
- Prefer the least mechanism that keeps expected call sites clear.

## Complex command-line interfaces

- Prefer `google/subcommands` when its simple Go-style flags and feature set suffice.
- Use Cobra when its additional features justify its complexity and usage pitfalls.
- In Cobra handlers, derive context from `cmd.Context()`; do not create a new root context.
- Use the context parameter supplied by `subcommands`.
- Do not create one package per subcommand without ordinary package-boundary reasons.
- Separate reusable library logic from CLI wiring; make the CLI a client of the library.

## Tests

### Leave testing to the `Test` function

- Keep correctness checks and failure messages in the `Test` function.
- Distinguish setup/cleanup helpers from assertion helpers.
- Use test helpers for environmental setup and cleanup failures, not to hide assertions about the system under test.
- Avoid assertion helpers that accept `testing.T` and decide correctness remotely.
- Inline simple validation even when mildly repetitive.
- Convert similar cases into a table-driven test while retaining validation in the loop.
- Make reusable validation return an error, diff, option, or value; let the test decide how to fail.

```go
func TestParse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Result
	}{
		{name: "empty", input: "", want: Result{}},
		{name: "one item", input: "x", want: Result{Items: []string{"x"}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.input)
			if err != nil {
				t.Fatalf("Parse(%q) returned unexpected error: %v", tc.input, err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Parse(%q) mismatch (-want +got):\n%s", tc.input, diff)
			}
		})
	}
}
```

### Designing extensible validation APIs

#### Acceptance testing

- Acceptance facilities invert control: users hand an implementation to a black-box validator that checks conformance to the library contract.

#### Writing an acceptance test

- Put acceptance-test support in a sibling `test` package, such as `chesstest`.
- Accept the implementation under test and return structured validation errors.
- Fail fast for cheap checks when one sentinel or custom error is enough.
- Aggregate errors when validation is expensive and users benefit from all failures.
- Reserve `t.Fatal` inside acceptance support for broken setup invariants, not ordinary conformance failures.
- Keep the end user's test in control of reporting pass or fail.

### Use real transports

- Test HTTP and RPC integrations through real production clients and transports connected to test servers.
- Do not hand-implement a client double when a server fake can exercise the real client stack.
- Prefer testing facilities supplied by the service authors.

### `t.Error` vs. `t.Fatal`

- Prefer `t.Error` so a test can report independent failures.
- Use `t.Fatal` when failed setup makes further execution meaningless.
- Before a table loop, use `t.Fatal` for setup required by every case.
- Without subtests, report a case-blocking failure with `t.Error`, then `continue`.
- Within `t.Run`, use `t.Fatal` to stop only the current subtest.

### Error handling in test helpers

- Call `t.Helper()` in helpers that may report test failures.
- Use `t.Fatalf` when setup failure means the test cannot continue.
- Include what failed and why in helper failure messages.
- Register resource release with `t.Cleanup` near acquisition.
- Remove `testing.T` from helpers that cannot fail or register cleanup.

```go
func mustWriteFixture(t *testing.T, dir string, data []byte) string {
	t.Helper()
	name := filepath.Join(dir, "fixture.json")
	if err := os.WriteFile(name, data, 0o600); err != nil {
		t.Fatalf("write fixture %q: %v", name, err)
	}
	return name
}
```

### Don't call `t.Fatal` from separate goroutines

- Keep `t.FailNow`, `t.Fatal`, and `t.Fatalf` on the goroutine running the test or subtest.
- A spawned goroutine may use `t.Error` and return, or communicate a result for the test goroutine to report.
- Adding `t.Parallel` does not itself make `t.Fatal` unsafe.

### Use field names in struct literals

- Use field names in table-case struct literals, especially for long cases, repeated field types, and omitted zero values.

### Keep setup code scoped to specific tests

- Scope setup to the test cases that need it.
- Avoid `init`-time test setup and unrelated package globals.

#### When to use a custom `TestMain` entrypoint

- Use `TestMain` only when every package test needs expensive setup that also requires teardown.
- Put `m.Run()` behind a helper that can defer teardown; call `os.Exit` only after deferred cleanup has run. Report setup errors to stderr.
- `TestMain` is not the first choice; ordinary helpers or amortized setup are usually simpler.
- Keep each test hermetic and reset any global or external state it changes.

#### Amortizing common test setup

- Use `sync.Once` only for expensive setup needed by some tests that requires no teardown.
- Consider cancellation problems before combining `sync.Once` with context-dependent setup.

## String concatenation

### Prefer “+” for simple cases

- Use `+` for a few simple strings.

### Prefer `fmt.Sprintf` when formatting

- Use `fmt.Sprintf` when formatting several values makes the result clearer.
- Write directly with `fmt.Fprintf` when the destination is an `io.Writer`.
- Use `text/template` or a safe HTML template package for complex templating.

### Prefer `strings.Builder` for constructing a string piecemeal

- Use `strings.Builder` for piecemeal incremental construction: it is
  amortized linear, while repeated `+` or `Sprintf` construction is quadratic.

### Constant strings

- Use raw backtick literals for constant multiline strings.

## Global state

- Make stateful library APIs instance-based and pass instances as explicit dependencies.
- Do not force clients through package variables, global registries, callbacks, service locators, or thick-client singletons.
- Do not let a global API nudge the design toward `Must`-style abort-on-error behavior; general-purpose libraries must not assume they run only during program initialization.

```go
// Bad: one registry shared by every client and test.
var plugins = map[string]*Plugin{}
func Register(name string, p *Plugin) error { /* mutate plugins */ return nil }

// Good: each client owns an isolated registry.
type Registry struct {
	plugins map[string]*Plugin
}

func NewRegistry() *Registry {
	return &Registry{plugins: make(map[string]*Plugin)}
}

func (r *Registry) Register(name string, p *Plugin) error {
	/* mutate r.plugins */
	return nil
}
```

- Inject dependencies through constructors, functions, methods, or struct fields.
- Audit global state for order-dependent tests, replacement by doubles, registration collisions, lifecycle restrictions, and mismatched concurrency needs.
- Reject legacy and standard-library globals as precedent for new APIs.

### Major forms of package state APIs

- Scrutinize all top-level variables, whether exported or private.
- Avoid global service locators, callback registries, and thick-client singletons for backends, storage, data access, and other system resources.
- These patterns can be useful on instances; making their locator or storage global causes the problem.
- Existing Google or standard-library globals are legacy, not precedent.

### Litmus tests

- Reject global APIs when otherwise independent functions or tests can interact through them.
- Reject them when tests need to swap their values, or callers must reason about initialization, flag parsing, registration, or other ordering.
- Revisit these tests as requirements change; global state deserves extreme scrutiny because its effects cascade.

- Permit logically constant state.
- Permit private caches only when callers cannot observe hit versus miss and behavior remains stateless.
- Keep global effects process-local; do not silently affect shared files, sidecars, networks, or distributed systems.
- Permit nondeterministic global behavior only where predictability is explicitly not promised.
- A safe registry case such as image decoders has independent calls, little reason to replace implementations, and stateless, idempotent, pure entries; registration collision remains possible.
- Sidecars may be shared across processes and contact distributed systems, so they are not necessarily process-local; apply stateless, idempotent, and local requirements inside them too.

### Providing a default instance

- A package-level default is acceptable, though not recommended, and only
  alongside isolated instances.
- Offer isolated instances first.
- Make convenience package functions thin proxies to a documented default instance.
- Restrict default-global usage to binaries; do not make imported infrastructure libraries depend on it.
- Document and enforce lifecycle and concurrency invariants.
- Provide a way to reset the default to a known-good state for tests.
- Maximize convenience this way only when needed and after applying the litmus tests.

## Interfaces

- Interfaces are implicitly satisfied structural tools; do not introduce one merely because a concept is called a service or repository.

### Avoid unnecessary interfaces

- Create an interface only for a real substitution, decoupling, or capability need.
- Reuse existing and generated interfaces; do not wrap generated RPC clients merely for tests.
- Prefer real transports over client interfaces invented solely for mocking.
- Do not export a double or interface back door solely for tests; every exported abstraction adds cognitive and maintenance cost.
- Material reasons include multiple implementations handled by common logic, package decoupling, or narrowing a huge API to the few capabilities a consumer needs.
- An interface used only to break a cycle may indicate packages that should be consolidated.

### Interface ownership and visibility

- Define interfaces at the consumer and include only methods the consumer uses.
- Keep internal interfaces unexported.
- Let producers own interfaces when the protocol itself is the product, such as `io.Writer` or generated RPC contracts.
- Producer ownership may also avoid importing a huge implementation package, avoid burdensome duplication of one canonical protocol, or resolve an import cycle.

### Designing effective interfaces

- Keep interfaces small and compose them when broader behavior is required.
- Document contracts, edge cases, expected errors, and concurrency obligations.
- Usually document a single-method interface's contract on the type; document every method of a multi-method interface. Consider documenting important unexported interfaces too.
- Accept interfaces and return concrete types by default.
- Return an interface when it intentionally limits unsafe capabilities, selects among implementations at runtime (including factory, strategy, command, or chaining patterns), or breaks an unavoidable import cycle.
- Do not use an interface to conceal harmless implementation details without a maintenance or integrity benefit.
- Returning an interface only guides the normal API surface; callers can still use type assertions for exported methods.
- Prefer one robust concrete type when it can contain implementation variation, as `database/sql.DB` does.
- Reconsider package boundaries before introducing interfaces solely to break cycles.
