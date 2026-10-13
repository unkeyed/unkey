---
name: writing-go-code
description: Applies Effective Go and the Google Go Style Guide (guide, decisions, best practices) when writing, reviewing, or refactoring Go code. Use for any Go change, code review, or style audit.
---

# Writing Go Code

Faithful distillations of the authoritative Go style sources live in `reference/`.
Read the relevant file before writing or reviewing Go; for a full review or
style audit, read all four:

- `reference/effective-go.md` — https://go.dev/doc/effective_go. The community
  baseline: formatting, names, control structures, data, methods, interfaces,
  embedding, concurrency, errors. Predates generics and modules.
- `reference/google-style-guide.md` — https://google.github.io/styleguide/go/guide.
  Canonical principles: clarity, simplicity, concision, maintainability,
  consistency, least mechanism. Defines how the three Google documents relate.
- `reference/style-decisions.md` — https://google.github.io/styleguide/go/decisions.
  Normative rulings: naming, commentary, imports, errors, language features,
  common libraries, test failures and structure.
- `reference/best-practices.md` — https://google.github.io/styleguide/go/best-practices.
  Documented conventions: error structure and logging, documentation, variable
  declarations, option patterns, tests, string concatenation, global state,
  interfaces.

The rules below are a working subset drawn from those sources, for quick
recall. When a rule here seems to conflict with a reference file, the
reference file wins; when project conventions (AGENTS.md, local style guides)
conflict with either, the project wins — consistency within a codebase beats
external style.

## Blank lines and line breaks

Treat these as separate decisions:

- A **blank line** is an empty source line between declarations or statements.
  Use one to show a logical section boundary.
- A **line break** wraps one declaration, statement, expression, call, or
  comment. It does not create a new logical section. Never insert an empty line
  inside a wrapped construct.

`gofmt` is mandatory, but it cannot choose all semantic grouping. It normalizes
indentation and alignment while source structure, including meaningful blank
lines, remains the author's responsibility. Format first, then review the
resulting vertical structure.

When writing or reviewing a function:

1. Identify its phases, such as validate, acquire, transform, persist, and
   respond. Put one blank line between adjacent phases only when the boundary
   helps a reader scan the function.
2. Keep statements that form one operation together. Do not separate a call
   from its immediate error check, a resource acquisition from its `defer`, a
   value from the check that validates it, or consecutive guard clauses that
   serve the same purpose.
3. Do not add a blank line merely because the next statement is `if`, `for`,
   `switch`, `return`, or a function call. Add it only for a change of purpose.
4. Use at most one blank line at a boundary. Never use repeated empty lines,
   and do not pad the start or end of a block.
5. If a function still reads as a wall after its real phases are visible,
   simplify or extract a coherent responsibility. Do not use blank lines to
   disguise an overlong or mixed-purpose function.

Apply the same grouping rule elsewhere:

- Separate unrelated declaration groups; keep declarations that establish one
  concept together.
- Follow the required import groups, with one blank line between groups, then
  let `gofmt` sort them.
- Never put a blank line between a doc comment and its declaration, between a
  package comment and `package`, or inside a short literal solely to make it
  taller.
- Do not wrap to a fixed column. Keep function declarations and control headers
  on one line where practical. If a call or expression must wrap, break at
  semantic argument or operand groups and use ordinary line breaks, not blank
  lines.

Review every blank line in changed Go code with two checks:

- **Boundary check:** Can both neighboring sections be named by different
  purposes? If not, remove the blank line.
- **Coupling check:** Would removing the blank line make two distinct phases
  harder to find? If yes, keep exactly one; otherwise prefer the tighter form.

## Naming

- `MixedCaps`, never underscores. Initialisms keep consistent case: `userID`,
  `parseURL`, `APIClient` — never `Id`, `Url`, `Api`.
- No `Get` prefix on getters: `Owner()`, not `GetOwner()`; setter is `SetOwner`.
- No stutter with the package name: `widget.New`, not `widget.NewWidget`.
- Package names: short, lower-case, single word. Avoid `util`, `helpers`,
  `common`.
- Name length proportional to scope: `i` in a small loop is fine; wide-scope
  names carry meaning.

## Errors

- Handle every error; do not discard with `_` (a deliberate discard needs a
  comment saying why it is safe).
- Error strings: lower-case start (unless beginning with an exported name), no
  trailing punctuation.
- Give errors structure: sentinels or typed errors checked with
  `errors.Is`/`errors.As`. Never inspect `err.Error()` with string matching.
- Wrap with `%w` when callers may need the cause; use `%v` to deliberately
  hide it. Add context the caller does not already have; place `%w` at the
  end — except a categorizing sentinel, which may lead: `"%w: detail"`.
- Do not both log and return the same error by default — let one handler decide.
- Do not use panic for normal error handling; return errors. Call `Must*`
  functions only on fixed setup, usually early startup or package
  initialization.

## Functions, types, contexts

- Do not define an interface before a real consumer needs one; define it in
  the consuming package and return concrete types. Do not export interfaces
  just for mocking.
- Receiver type: be consistent per type; pointer if any method mutates or the
  type holds a lock or large state. Never copy a value containing a
  `sync.Mutex`.
- Make the zero value useful where practical. Use field names in composite
  literals for structs from other packages.
- `ctx context.Context` is the first parameter. Never store a context in a
  struct. Create `context.Background()` only at true entry points.
- Prefer synchronous APIs; let the caller add concurrency.

## Control flow

- Success path flows down the page, unindented; handle errors early and
  return. No `else` after a body ending in `return`.
- Prefer `switch` over long if-else chains.

## Concurrency

- Every goroutine must have an obvious answer to: when does it stop, and how
  does the parent wait for it? Use `context`, `sync.WaitGroup`, or channel
  close — no fire-and-forget.
- Bound simultaneous work (e.g. a buffered-channel semaphore) instead of
  spawning a goroutine per item only to block most of them.

## Comments and docs

- Every exported name has a doc comment starting with the name:
  `// Foo returns ...`. One package comment per package.
- Comments explain why, not what.

## Tests

- Table-driven tests for repeated cases; name the cases.
- Failure messages identify the function and input, and print `got` before
  `want`.
- Helpers call `t.Helper()`; keep pass/fail judgment in the `Test` function,
  not hidden inside helpers.
- Never call `t.Fatal` from a goroutine the test spawned.
- Note: the Google guide forbids assertion libraries; many repos use one
  (e.g. testify) by convention. Follow the repo.

## Final pass

Whenever this skill is used for Go implementation or review, loading and running
the `no-comments` skill is required before finalizing, including reviews with no
code changes. Apply the Go guidance above first, then load `no-comments` and run
its workflow on the scoped changed or reviewed Go files or diff. Loading it
without running its workflow does not satisfy this requirement.

Apply its accepted findings and rerun any verification affected by those
changes. Follow its exceptions for retained comments, including public API
contract docs; do not treat the Go comment guidance above as a blanket exemption.
Include its required report in the final response, and identify any justified
retained comments with the applicable exception and supporting evidence.
