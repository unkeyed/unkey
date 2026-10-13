# Go Style Decisions

Source: https://google.github.io/styleguide/go/decisions

These are normative rulings, but they are not canonical. They are subordinate
to the core Go style guide, which takes precedence if the documents conflict.

## About

- These decisions unify guidance, explanations, and examples used by Go
  readability mentors.
- They are not exhaustive and may grow over time.
- MixedCaps, general formatting, and line length rulings now live in the core
  guide rather than on this page.

## Naming

- Apply the core guide's overarching naming guidance as well as these specific
  rulings.

### Underscores

- In general, do not use underscores in Go identifiers.
- Exceptions are package names imported only by generated code;
  `Test`, `Benchmark`, and `Example` functions in `*_test.go`; and rare
  low-level OS or cgo interoperability names.
- Source filenames are not identifiers and may contain underscores.

### Package names

- Use concise package names containing only lowercase letters and numbers.
- Keep multiword names unbroken and lowercase: `tabwriter`, not `tab_writer`.
- Avoid names likely to be shadowed by common locals; `usercount` is clearer
  than a package named `count`.
- Imported packages whose declared names contain underscores must be renamed to
  valid local package names.
- Generated-only packages may use underscores. Valid cases include
  `linkedlist_test` for black-box tests, descriptive integration packages such
  as `linked_list_service_test`, and package-level documentation examples.
- Avoid uninformative names such as `util`, `utility`, `common`, `helper`,
  `model`, and `testhelper`; choose a name that describes the capability.
- A renamed import's local name follows package-name rules. Use the same local
  name for the same import across nearby files when possible.

### Receiver names

- Receiver names must be short, usually one or two letters, abbreviate the
  receiver type, and remain consistent across that type's methods.
- Do not use `this`, `self`, a long descriptive name, or `_`.
- Omit the receiver name when it is unused.

### Constant names

- Use MixedCaps for constants, including when another language's convention
  differs. Exported constants start with uppercase, unexported with lowercase.
  Do not use SCREAMING_SNAKE_CASE or a `k`/`K` prefix.
- Name a constant for the role its value plays, not as a derivative of its
  literal value.
- If a value has no role beyond being its literal value, a named constant is
  unnecessary.

### Initialisms

- Keep all letters of an initialism or acronym in the same case: `URL` or
  `url`, `ID` or `id`, and `DB` or `db`, never `Url`, `Id`, or `Db`.
- In a name containing multiple initialisms, each initialism is internally
  consistent, though separate initialisms may differ: exported `XMLAPI`,
  unexported `xmlAPI`.
- Preserve standard prose spelling for forms with lowercase letters, such as
  `DDoS`, `iOS`, and `gRPC`, unless exportedness forces the first letter to
  change. Then normalize the whole initialism, for example `IOS` or `GRPC`.
- Follow normal English usage for abbreviations such as `Txn`; do not
  mechanically turn every abbreviation into all capitals.

### Getters

- Do not prefix accessors with `Get` or `get` unless “get” is part of the
  domain, such as HTTP GET. Prefer `Counts` to `GetCounts`.
- For expensive computation or remote access, use a verb such as `Compute` or
  `Fetch` to communicate that the call can take time, block, or fail.

### Variable names

- Make name length proportional to scope size and inversely proportional to
  frequency of use; adjust for ambiguity and the number of similar variables.
- Treat roughly 1-7 lines as small scope, 8-15 as medium, 15-25 as large, and
  more than a page as very large. These ranges are guidance, not strict rules.
- Familiar, specific abbreviations such as `db` can remain clear in large
  scopes; do not lengthen names mechanically.
- Name a local for what it contains and how it is used now, not merely for the
  field or source from which it came.
- Start with a single word such as `count` or `options`; add words only to
  disambiguate values such as `userCount` and `projectCount`.
- Do not delete letters merely to save typing, especially in exported names.
- Usually omit type words: prefer `users` over `userSlice`, and `userCount`
  over `numUsers` or `usersInt`.
- A representation qualifier is useful when two forms coexist, such as
  `ageString` and `age`.
- Omit words supplied by the surrounding package, type, method, or function.

#### Single-letter variable names

- Use single-letter names only when the full word is obvious and spelling it
  repeatedly would add noise.
- One- or two-letter method receivers are preferred.
- Familiar names include `r` for a reader or request, `w` for a writer or
  response writer, `i` for an index, and `x`/`y` for coordinates.
- A short abbreviation such as `n` in a brief loop is acceptable.

### Repetition

- Avoid unnecessary repetition in names and in nearby code.
- Evaluate repetition from the symbol user's context, not in isolation.

#### Package vs. exported symbol name

- Remove words already supplied by package qualification.
- If a package primarily exports one type named for the package, call its
  constructor `New`: prefer `widget.New()` to `widget.NewWidget()`.
- Likewise prefer `widget.NewWithName`, `db.Load`, or `goatteleport.Count` to
  names that restate the package or implementation source.

#### Variable name vs. type

- Do not restate a variable's obvious type: prefer `users`, `name`, and
  `primary` to `numUsers`, `nameString`, and `primaryProject`.
- When the same value exists in multiple representations, qualify it with
  words such as `raw`/`parsed` or `Str`, for example `limitRaw` and `limit`.

#### External context vs. local names

- Package, method, type, function, import path, and filename already qualify
  local names; do not repeat that context.
- Prefer `reporting.Report`, `(*Project).Name`, and `sqldb.Connection` to names
  that restate their containers.
- Remove repetitive result names, local type suffixes, and error prefixes when
  context and use already make them clear.

## Commentary

- Write commentary to support users reading public API documentation.
- Preview generated docs and runnable examples during development and review.
- Godoc uses little decoration; indent lists and code snippets and otherwise
  keep formatting plain.

### Comment line length

- Go comments have no fixed line-length limit.
- Wrap prose when needed for readability in tools without automatic wrapping;
  80 or 100 columns are common choices, not hard cutoffs.
- Keep long URLs, literals, or other text intact when wrapping harms clarity.
- Avoid packing large prose paragraphs onto one source line, and stay
  consistent within a file.

### Doc comments

- Every top-level exported name must have a doc comment. Add one for unexported
  types and functions whose behavior or meaning is unobvious.
- Write a full sentence beginning with the declared name; an article may come
  first when it reads naturally.
- Write for package users because Godoc and IDEs surface these comments.
- A comment before a group of struct fields documents that whole group.
- Apply the same name-first convention to unexported declarations so later
  export requires only changing the name.

### Comment sentences

- Capitalize and punctuate complete sentences as standard English.
- An uncapitalized identifier may begin a sentence when clear, preferably only
  at the start of a paragraph.
- Fragments need not be capitalized or punctuated.
- Documentation comments must always be complete sentences.
- Short end-of-line comments, especially on fields, may be phrases that treat
  the field name as their subject.

### Examples

- Clearly document intended package use and try to provide runnable examples.
- Put runnable examples in test files, not production source files.
- If a runnable example is infeasible, use a comment code sample formatted by
  normal code and command-line conventions.

### Named result parameters

- Name results only when names improve the Godoc signature, such as
  distinguishing multiple same-typed results or telling callers an action is
  required.
- Do not name results when function name and result types are already clear, or
  when the names merely repeat the function or type.
- Do not name results just to avoid local declarations or enable naked returns.
- Naked returns are acceptable only in small functions; use explicit values in
  medium and larger functions.
- A result may be named when a deferred closure must modify it.
- A named function type can sometimes communicate better than a result name.

### Package comments

- Put the package comment immediately above `package`, with no blank line.
- Provide exactly one package comment per package.
- Start a library comment with `Package name`.
- For `main`, use the exact binary/build-target name, commonly in forms such as
  `The seed_generator command ...`; preserve its spelling even when it starts
  the sentence and therefore must be capitalized.
- Indent example invocations and API snippets for Godoc.
- Use `doc.go` when no primary file is obvious or package documentation is
  unusually long.
- A multiline block comment is acceptable when copyable sections benefit.
- Put file-maintainer comments after imports so Godoc does not treat them as
  package comments.

## Imports

### Import renaming

- Normally do not rename imports. Rename when required or when it materially
  improves readability.
- Local import names obey package naming: lowercase, no underscores or capitals,
  and consistent for the same package.
- Resolve import collisions by renaming the more local or project-specific
  package.
- Rename generated protobuf packages to remove underscores and end the local
  name in `pb`, for example `foosvcpb`.
- A non-generated package with a vague name such as `util` or `v1` may be
  renamed sparingly; prefer fixing its package name when possible.
- For a package colliding with a useful local such as `url`, prefer `urlpkg` if
  both must be referenced in the same scope. Shadowing alone does not require a
  rename when the package is no longer needed there.

### Import grouping

- Group imports in this order, with blank lines between groups: standard
  library; other project and vendored packages; protocol buffers; side-effect
  imports.

### Import "blank" (`import _`)

- Side-effect-only imports belong only in `main` packages or tests that need
  them, not in library packages.
- Exceptions are blank imports used to bypass a disallowed-import static check
  and `embed` imported in a file containing `//go:embed`.
- If production use of a library indirectly requires a side-effect import,
  document that intended usage for callers.

### Import "dot" (`import .`)

- Never use dot imports; qualification must reveal where functionality comes
  from.

## Errors

### Returning errors

- Signal fallible operations with `error`, conventionally the final result.
- Return `nil` error on success.
- After a non-nil error, callers must treat all other results as unspecified
  unless documentation says otherwise; they are often zero, but do not assume.
- Exported functions return the `error` interface, not a concrete error pointer,
  which can produce a non-nil interface around a nil pointer.
- A function accepting `context.Context` should usually return an error so the
  caller can detect cancellation.

### Error strings

- Begin error strings with lowercase unless the first word is an exported name,
  proper noun, or acronym; do not end them with punctuation.
- Full displayed messages in logs, tests, APIs, and UIs usually begin with a
  capital according to that output's style.

### Handle errors

- Deliberately handle every error: address it immediately, return it, or in an
  exceptional situation terminate with `log.Fatal` or, only if necessary,
  `panic`.
- Do not normally discard errors with `_`.
- Ignoring a documented-impossible error, such as `bytes.Buffer.Write`, is
  acceptable only with a comment explaining why it is safe.

### In-band errors

- Do not normally encode failure or absence in values such as `-1`, `nil`, or
  `""`; return a final validity value instead.
- Use a final `bool` when no explanation is needed, or `error` when it is.
- Some standard-library string functions use in-band values because they make
  string manipulation much simpler; this tradeoff demands caller diligence and
  is not the general rule.

### Indent error flow

- Handle errors and other terminal conditions before the normal path.
- Put normal code after a terminating `if`; do not nest it in `else`.
- If an initialized value is used for more than a few lines, declare it before
  the `if` rather than keeping a large normal path in `else`.

## Language

### Literal formatting

- Prefer composite literals to field-by-field construction.
- Keep literals readable according to the following specific rules.

#### Field names

- Struct literals of types from another package must name fields, avoiding
  coupling to field order and the complete field set.
- Package-local positional literals are allowed when clear, but name fields
  when that improves readability; large structs almost always need names.

#### Matching braces

- Place a closing brace at the same indentation as its opening brace.
- In multiline literals, end the last value with a comma and put the closing
  brace on its own aligned line.

#### Cuddled braces

- Cuddle slice or array literal braces only when indentation matches and inner
  values are themselves literals or proto builders.
- Do not cuddle variables or other expressions with literal braces.

#### Repeated type names

- Omit repeated element type names in slice and map literals to reduce clutter.
- Repeat an unfamiliar complex type when occurrences are far apart and the
  repetition restores context.

#### Zero-value fields

- Omit zero-valued fields when clarity is preserved so specified options stand
  out.
- In nontrivial table-driven tests, name fields so irrelevant zero-valued fields
  can be omitted.
- Explicitly include a zero or nil field when that value is essential to the
  test case; explicit rows are also acceptable when all values matter.

### Nil slices

- Treat nil and empty slices equivalently in API design; `len`, `cap`, range,
  and append already work with nil slices.
- Prefer `var s []T` for an initially empty local slice, especially one that may
  be returned.
- Do not use nil versus empty as an error signal or force clients to distinguish
  them.
- Test emptiness with `len(s) == 0`, not `s == nil`.

### Indentation confusion

- Avoid line breaks that align a wrapped header with the indented body below.
- If unavoidable, use visible spacing to separate the wrapped line from body
  code, and follow the rules for functions, conditions, loops, and literals.

### Function formatting

- Keep function and method declarations on one line; wrapped signatures make
  body indentation ambiguous.
- Do not split calls solely because they are long. Shorten with local variables
  or improved APIs instead.
- Avoid inline argument-label comments; prefer an options struct or better API
  documentation.
- If an API cannot change or semantic grouping genuinely helps, break a call by
  meaning rather than at an arbitrary column.
- Do not split long string literals for line length. If useful, break after a
  format string and group arguments semantically on following lines.

### Conditionals and loops

- Keep `if`, `for`, `switch`, and `case` headers on one line rather than
  introducing indentation confusion.
- If short-circuit behavior is unnecessary, extract long boolean operands into
  named locals; also extract repeated expressions.
- Ensure braces remain visibly matched when an initializer contains a closure
  or multiline literal.
- Let a line remain long if no clear refactoring exists.
- When a long `for` condition can be clearer as a body guard and `break`, use
  that refactoring.
- If a long switch requires multiline clauses, indent complete cases and
  separate them clearly rather than ambiguously wrapping a case header.
- Put the variable on the left and constant on the right in comparisons.

### Copying

- In general, do not copy a value of type `T` when its methods are on `*T`.
- Never copy synchronization values such as `sync.Mutex`; beware aliasing types
  such as `bytes.Buffer` too.
- API structs containing non-copyable fields should generally be accepted,
  returned, and received by pointer.

### Don't panic

- Do not use panic for normal error handling; return errors and multiple values.
- In `main` and initialization, an exit without a stack trace can be appropriate
  for invalid configuration where a trace does not help. Remember that
  `os.Exit`-style termination skips deferred functions.
- For genuinely impossible bug states, returning an error or calling a fatal
  logger can be reasonable; panic is covered by the separate acceptable-panic
  guidance.

### Must functions

- Name setup helpers that stop on failure `MustXYZ` or `mustXYZ`.
- Call them only on fixed setup, usually early startup or package initialization,
  not on user input, request paths, or anywhere normal error checking belongs.
- Test-only must helpers may stop only the current test; mark them with
  `t.Helper()` and use `t.Fatal`.
- Their value is use in value contexts with inputs whose validity is itself
  tested; use ordinary error propagation whenever feasible.

### Goroutine lifetimes

- Make clear when and why every spawned goroutine exits.
- Prevent leaks from blocked sends/receives, races from touching inputs after
  ownership ends, sends on closed channels, and unbounded resource lifetime.
- Keep synchronization in the narrowest function scope, factor core logic into
  synchronous functions, and join owned goroutines before returning.
- Use contexts, signal channels, synchronized state, or condition variables so
  termination is evident; document a lifecycle that code cannot make obvious.

### Interfaces

- Do not create an interface until a real consumer needs one; model required
  behavior rather than abstract “service” or “repository” patterns.
- Do not wrap RPC clients in manual interfaces only for tests, and do not export
  back doors or test doubles solely to test an interface; prefer real transports
  and public APIs.
- Keep interfaces small and document contracts, edge cases, and expected errors.
- Keep internally used interfaces unexported and define them in the consuming
  package. A producer may export one when the common protocol is itself the
  product.
- Accept interfaces and return concrete types by default. Returning an interface
  is acceptable for deliberate encapsulation and suitable patterns such as
  command, chaining, factory, or strategy.

### Generics

- Use generics when they fulfill real requirements, not prematurely when slices,
  maps, interfaces, or concrete code are simpler.
- Thoroughly document exported generic APIs and strongly consider motivating
  runnable examples.
- If only one concrete instantiation exists, implement that first; do not add
  generics merely because an algorithm is theoretically type-independent.
- Do not invent DSLs, error-handling frameworks, or test assertion frameworks
  with generics.
- Prefer a useful common interface when several types already share one; prefer
  generics over `any` plus excessive type switches for genuine parametric work.

### Pass values

- Do not pass pointers merely to save a few bytes when the function only reads
  the dereferenced value.
- Pass strings and interfaces as values, not `*string` or pointers to interfaces.
- This does not apply to large or growth-prone structs. Pass protocol buffer
  messages by pointer because they can be large and pointer types implement
  `proto.Message`.

### Receiver type

- Choose receiver type for correctness and method-set semantics before speed.
- Use a value receiver for a slice not resliced/reallocated, a map, function,
  channel, immutable built-in-derived type, or small naturally immutable value.
- Use a pointer receiver to mutate the receiver or for a struct containing
  non-copyable fields.
- Prefer pointers for large structs/arrays and for structs whose pointee fields
  may mutate, making mutable semantics clear.
- For concurrent mutation, choose a value when modifications should be hidden
  from the method and a pointer when they should be visible.
- When in doubt, use a pointer receiver; generally keep a type consistently
  pointer-received or value-received.
- Do not rely on folklore about pointer performance. Profile realistic
  benchmarks when it matters.

### `switch` and `break`

- Do not put an unlabeled `break` at the end of a switch clause; Go switches
  already break automatically.
- Explain an intentionally empty clause with a comment.
- Inside a loop, an unlabeled break exits only the switch. Label the loop and
  break to that label when the loop must end.

### Synchronous functions

- Prefer synchronous functions that return results directly and finish all
  callbacks and channel operations before returning.
- This localizes goroutines, makes lifetimes and tests simpler, and avoids leaks
  and races. A caller can add concurrency; removing forced concurrency is hard.

### Type aliases

- Define distinct types with `type T1 T2`.
- Reserve `type T1 = T2` aliases primarily for source migrations; aliases are
  rare and should not be used without need.

### Use %q

- Use `%q` instead of manually putting quotes around `%s`.
- Prefer `%q` in human-readable output when a string might be empty or contain
  control characters, making those values visible.

### Use any

- Prefer the `any` alias to `interface{}` in new code.

## Common libraries

### Flags

- Use snake_case command-line flag names and mixedCaps Go variable names.
- Define flags only in `package main` or its equivalent.
- Configure general-purpose libraries through Go APIs, not import-time flags.
  An extremely rare library flag must clearly identify its package.
- Prefer explicit arguments or struct fields over exported flag-backed
  globals; use such globals rarely and under strict scrutiny.
- Put global flag variables in their own `var` group after imports.

### Logging

- Google logging distinguishes fatal termination with a stack trace from exit
  without one; choose deliberately. Its names are not the standard `log` API:
  `log.Fatal` aborts with a stack trace, `log.Exit` without one, and there is
  no `log.Panic`. Open-source Google projects may use `glog`.
- Prefer non-formatting methods such as `log.Info(v)` when no formatting is
  needed instead of `log.Infof("%v", v)`.

### Contexts

- Pass `context.Context` explicitly through the entire call chain and place it
  first in function and method parameter lists.
- Obtain handler contexts from `req.Context()`, streaming RPC contexts from the
  stream, and test contexts from `testing.TB.Context()`.
- Create `context.Background()` only at true entry points such as `main` and
  `init`, except for framework-level request-context construction. Prefer a
  caller's context in normal library code.
- Test helpers follow context-first, then `*testing.T`, then other parameters.
- Do not store context in a struct; pass it to each method that needs it. A
  forced external interface signature is a very rare exception requiring
  special review.
- Prefer `testing.TB.Context()` over `context.Background()` in tests and pass it
  explicitly into setup and helpers.
- Background operations that outlive cancellation need a proper detachment
  mechanism rather than an unrelated base context.
- Reuse one context across calls sharing cancellation, deadline, credentials,
  and trace; contexts are immutable.

#### Custom contexts

- Never create custom context types or substitute another context interface in
  function signatures.
- Pass ordinary application data explicitly, on a receiver, globally when
  justified, or in `context.Context` only when it truly belongs there.

### crypto/rand

- Never use `math/rand` for keys, including throwaway keys; seeding does not
  provide sufficient entropy.
- Read key bytes from `crypto/rand`, encode text as hexadecimal or base64, and
  handle random-source failures.

## Useful test failures

- A failure should be diagnosable without opening the test source: identify the
  cause, inputs, actual result, and expected result.

### Assertion libraries

- Do not create assertion libraries or testing DSLs that combine comparison and
  reporting; they hide context, often stop too early, and fragment failure style.
- Use Go control flow plus `testing`, `cmp`, and `fmt`.
- Domain comparison helpers should return a value or error for the test to
  report, rather than accept `*testing.T`.
- Nontrivial helpers deserve direct tests of their own.

### Identify the function

- Usually name the failing function in the message:
  `YourFunc(%v) = %v, want %v`, not only `got ..., want ...`.

### Identify the input

- Include short inputs in failures.
- For large or opaque inputs, give the case a meaningful description and print
  that description in the failure.

### Got before want

- Print the actual value before the expected value; prefer the words `got` and
  `want` to “actual” and “expected”.
- Diff argument order has no universal ruling, so always print a direction key.

### Full structure comparisons

- When all fields matter, construct the complete expected struct, slice, array,
  or map and compare it deeply rather than checking fields by hand.
- Compare only relevant fields when irrelevant data would obscure intent.
- Configure `cmp` options for semantic/approximate equality or uncomparable
  fields.
- Do not wrap multiple function results in a struct solely to compare them;
  check each result directly.

### Compare stable results

- Do not assert dependency output bytes or formatting that the dependency does
  not promise to keep stable.
- Parse formatted or serialized output and compare stable semantic content, as
  with decoded JSON.

### Keep going

- Continue after mismatches when later checks remain meaningful so one run
  reports all defects; prefer `t.Error` in that case.
- Use `t.Fatal` when later checks would be meaningless or misleading.
- In table tests, prefer subtests with `t.Fatal` to `t.Error` followed by
  `continue`.

### Equality comparison and diffs

- Use language `==` only where its semantics are the intended equality.
- Use `cmp.Equal` and `cmp.Diff` for complex values; use
  `protocmp.Transform()` for protobuf messages.
- Prefer `cmp` for new tests. Existing `pretty` use may remain for consistency,
  but it has weaker semantics, scales poorly, sees unexported details, and is
  inappropriate for protobufs.
- Replace `reflect.DeepEqual` in tests because it is brittle around unexported
  fields and implementation details.
- Do not use panic-prone `cmp` in production code; it is designed for tests.

### Level of detail

- Default to `YourFunc(%v) = %v, want %v`, adding call identity and relevant
  state for complex interactions.
- For boilerplate-heavy structures, print important parts without hiding the
  data needed to diagnose failure.
- Setup failures can be shorter because the intended test input may be
  irrelevant.
- Exercise failure paths during development to review message usefulness.
- Use `%q` for strings, `%+v` for small structs, and diffs for large values.

### Print diffs

- For large outputs, print a diff rather than dumping got and want.
- Prefer `cmp.Diff` in new tests; multiline-string diff tools may also be useful.
- State direction, for example `(-want +got)` when calling `cmp.Diff(want, got)`
  or `(-got +want)` for the reverse.
- Print a newline before the multiline diff.

### Test error semantics

- Do not compare full error text to establish error type; wording changes make
  such tests brittle.
- Comparing text is allowed when the text itself is the contract under test,
  such as containing a parameter name.
- Test semantic control-flow information with `errors.Is` or
  `cmpopts.EquateErrors`.
- When a dependency's errors expose insufficient semantics, prefer requesting
  an API improvement over parsing its message text.
- If only error presence matters, use a `wantErr bool`; `cmp` is unnecessary
  when expectations are only nil versus any error.

## Test structure

### Subtests

- Subtests are useful, especially with tables, but not mandatory.
- Keep cases independent so each can run by itself under test filtering.

#### Subtest names

- Choose concise, identifier-like names useful both in logs and command-line
  filters.
- Avoid spaces, nonprinting characters, and especially `/`, all of which are
  escaped or have filter semantics.
- Put a long prose description in a separate field for logging or failures.
- Include actual function inputs in failure messages even though `t.Run`
  identifies the case.

### Table-driven tests

- Use table-driven tests when many cases share the same equality check or set
  of invariants.
- Failure messages must identify function and inputs; a numeric row index is
  unnecessary.
- Split cases into separate tests when they require materially different logic.
- A simple error-presence case may share a table only while it keeps the loop
  straightforward.
- Prefer separate success and error tables when that removes conditional logic;
  identical setup can also support a sequence of distinct subtests.

#### Data-driven test cases

- Do not let table fields select complicated setup modes or conditional test
  behavior.
- Duplicate a simple test loop across clearly named tests when that makes setup
  and intent obvious; the duplication is preferable to a data-driven mini-DSL.

#### Identifying the row

- Never use the table index as the only case identity.
- Print a meaningful description or inputs, or use an effective subtest name.
- Even with `t.Run`, always identify the input in the failure, and follow the
  subtest-name restrictions.

### Test helpers

- Test helpers perform setup or cleanup; their failures should represent
  environment/setup failures, not hidden assertions about code under test.
- A helper accepting `*testing.T` must call `t.Helper()` so failures point to
  the caller.
- Put `*testing.T` after context, if present, and before remaining arguments.
- Call such helpers from the test's goroutine unless their contract says
  otherwise.
- Do not use `t.Helper()` to disguise assertion libraries or obscure the
  conditions causing failure. The advice also applies to benchmark and fuzz
  helpers.

### Test package

- Choose same-package or external-package tests according to the access and
  dependency needs below.

#### Tests in the same package

- Put tests in `foo_test.go` with `package foo` and do not import `foo` itself.
- Same-package tests can access unexported identifiers and may be more concise.
- Examples in same-package tests omit the package qualification users may need.

#### Tests in a different package

- Use `package foo_test` when tests cannot or should not live in `foo`, such as
  integration ownership or avoiding an import cycle.
- The `_test` suffix is an explicit exception to the package underscore rule.

### Use package `testing`

- The standard `testing` package is the only permitted test framework.
- Do not use assertion libraries or third-party testing frameworks.
- Use its tests, benchmarks, runnable examples, subtests, logging, and
  error/fatal reporting together with ordinary Go syntax.

## Non-decisions

- Either `var i int` or `i := 0` is acceptable zero-value initialization.
- Either `&File{}` or `new(File)` is acceptable, as are
  `map[string]bool{}` and `make(map[string]bool)`.
- Either got/want order in `cmp.Diff` is acceptable; stay locally consistent
  and include a direction legend.
- `errors.New("foo")` and `fmt.Errorf("foo")` are interchangeable when no
  formatting is needed.
- Mentors generally should not enforce a preference on these non-decisions.
