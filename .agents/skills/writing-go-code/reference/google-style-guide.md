# Go style guide

Source: https://google.github.io/styleguide/go/guide

The canonical, normative page for Go style at Google: style principles and
core guidelines. The document-set overview and definitions in the next
section come from the companion overview page,
https://google.github.io/styleguide/go/index, and are included here because
they define the authority levels the other reference files rely on.

## The Go style document set (from the overview page)

The documents codify approaches for writing readable, idiomatic Go. They aim to
reduce guesswork, help newcomers avoid common mistakes, and unify guidance from
Go code reviewers at Google. Adherence is not absolute, the documents are not
exhaustive, and good judgment remains necessary.

The three documents have different roles and levels of authority:

1. The **Style Guide** defines the foundation of Go style at Google. It is
   canonical and normative, applies to everyone, and provides the definitive
   basis for the other two documents.
2. **Style Decisions** records specific style decisions and their reasoning. It
   is normative but not canonical, and primarily serves readability mentors.
   Decisions may change as evidence, the language, libraries, and common
   patterns change. Individual Go programmers are not expected to track every
   update.
3. **Best Practices** records familiar patterns that solve common problems,
   remain readable, and withstand maintenance. It is neither normative nor
   canonical. Go programmers are encouraged to use these practices where they
   fit so the codebase remains uniform.

Together, the documents establish principles for comparing styles, codify
settled style points, provide canonical examples of Go idioms, explain
tradeoffs, reduce surprises in readability reviews, and give readability
mentors consistent language. They are not an exhaustive review checklist, a
replacement for judgment, or a reason for sweeping churn to remove every style
difference. Apply updated guidance to new code and improve nearby code over
time.

### Definitions

- **Canonical** means prescriptive and enduring. A canonical standard applies
  to old and new code and is not expected to change substantially. Authors and
  reviewers should understand its principles. Because each rule must clear a
  high bar, canonical documents tend to be shorter and prescribe fewer style
  elements.
- **Normative** means intended to establish consistency. It gives reviewers
  agreed terminology, recommendations, and justifications. Normative guidance
  can change, and the documents change with it so reviews remain consistent.
  Authors are not expected to know all normative material, but reviewers often
  cite it during readability reviews.
- **Idiomatic** means common and familiar in Go. Prefer an idiomatic pattern to
  an unfamiliar one when both serve the same purpose in context, because the
  idiom is easier for readers to recognize.

Uniformity has substantial value even though style is subjective and involves
tradeoffs. Follow the guide even when personal preferences differ, but do not
nitpick every violation or use an update as a reason to churn existing code.

## Style principles

Readable Go code has these attributes, in descending order of importance:

1. **Clarity:** Its purpose and rationale are clear to the reader.
2. **Simplicity:** It accomplishes its goal in the simplest possible way.
3. **Concision:** It has a high signal-to-noise ratio.
4. **Maintainability:** It is easy to maintain correctly.
5. **Consistency:** It fits the broader codebase.

The ordering resolves conflicts. Consistency can break a tie, but it cannot
override the principles above it.

### Clarity

Readability's core goal is clarity for the reader. Effective names, useful
commentary, and efficient organization are the primary tools. Code being easy
to read matters more than it being easy to write.

Clarity has two distinct parts:

- What is the code doing?
- Why is it doing that?

#### Make the behavior clear

Go usually makes behavior straightforward to see. If understanding requires
uncertainty or prior knowledge, invest in making the purpose clearer. Depending
on the situation, use more descriptive names, add useful commentary, separate
ideas with whitespace and comments, or refactor work into focused functions or
methods. No single technique fits every case.

#### Make the rationale clear

Names for packages, types, functions, methods, and variables often communicate
rationale. Add commentary when they cannot. Explaining why is particularly
important for unfamiliar language details, such as a distant closure capturing
a loop variable, and for business rules, such as distinguishing a user from an
impersonator during an access-control check.

Some APIs require unusual care. Performance-sensitive implementation,
unexpected conversions in mathematical operations, and similar intricacies
need documentation so readers do not have to reverse-engineer them and future
maintainers do not accidentally break them.

Commentary can reduce clarity when it adds clutter, repeats the code,
contradicts it, or creates maintenance work. Prefer self-describing code and
comments that explain why rather than what.

Familiar patterns make ordinary code recede and unfamiliar code stand out.
Preserve that signal: unusual code often marks a real constraint, such as
performance, that deserves attention. Runnable examples can help both API
users and maintainers because they appear in documentation and run as tests. A
small helper can also improve clarity and correctness at every call site even
when its implementation is only a few lines.

### Simplicity

Code should be simple for its users, readers, and maintainers. Choose the
simplest implementation that meets both behavioral and performance goals.
Simple code:

- Reads naturally from top to bottom.
- Requires neither hidden prior knowledge nor memorization of all preceding
  code.
- Avoids unnecessary abstraction and conspicuous names for mundane concepts.
- Makes the flow of values and decisions visible.
- Uses comments to explain why and discourage incorrect future deviation.
- Has documentation that stands on its own.
- Produces useful errors and useful test failures.
- Avoids cleverness when cleverness conflicts with simplicity.

Implementation simplicity and API simplicity can conflict. Internal complexity
may be worthwhile when it materially helps users call an API correctly. In
other cases, asking the caller to do a little more keeps the implementation
clearer. Make that tradeoff deliberately.

Add complexity for a concrete need, typically performance or multiple
disparate clients. Explain and test necessary complexity, and provide examples,
especially when an API offers both simple and advanced paths. Complexity is not
forbidden. Avoiding needless complexity ensures that complexity signals code
that requires extra care.

Performance techniques such as preallocating and reusing a buffer can justify
complexity, but their rationale and maintenance constraints must be apparent.
Without a demonstrated need, that machinery only burdens future readers and
editors. If a simple purpose leads to very complex code, revisit the design.

#### Least mechanism

When several mechanisms express the same idea, prefer the most standard one.
Complexity is easy to add later and difficult to remove after it spreads.

1. First use a sufficient core language construct, such as a channel, slice,
   map, loop, or struct.
2. Otherwise, use a standard-library facility, such as an HTTP client or
   template engine.
3. Before adding a dependency or building custom machinery, look for a
   sufficient established core library in the codebase.

For example, tests that need to override a flag-bound variable should assign
the variable directly unless they are testing the command-line interface
itself. Calling `flag.Set` adds unnecessary mechanism. Likewise, a
Boolean-valued map usually supplies set membership. Use a set library only for
operations that a map cannot express or would make unduly complicated.

### Concision

Concise code has a high signal-to-noise ratio. Its names and structure guide the
reader toward the relevant details. Repetition, extraneous syntax, opaque
names, unnecessary abstractions, and distracting whitespace can hide those
details.

Repetition is especially harmful because readers must visually compare nearly
identical blocks to find differences. Table-driven tests can factor common code
away from meaningful inputs and outcomes, but the selected table fields must
keep the result understandable. In every structure, prefer the form that makes
the important details most apparent.

Common Go constructions and idioms are quickly recognizable and therefore
concise. A subtle variation can be easy to miss, such as checking `err == nil`
where readers expect `err != nil`. Intentionally boost the signal with a comment
when a familiar shape has an important reversal or variation.

### Maintainability

Code is edited far more often than it is first written. Maintainable code is
clear to both the reader learning it and the programmer changing it. It:

- Lets a future programmer modify behavior correctly.
- Provides APIs that can grow gracefully.
- States assumptions and maps abstractions to the problem, not merely to the
  current code structure.
- Avoids unnecessary coupling and unused features.
- Tests promised behavior and important logic comprehensively, with clear,
  actionable failure diagnostics.

Abstractions such as interfaces and custom types remove contextual information,
so use them only when their benefit exceeds that cost. An editor can navigate a
concrete method to its implementation, but an interface may hide details that a
maintainer needs for correct use. Explain required implementation knowledge in
the interface documentation or at the call site.

Do not conceal critical behavior in one easy-to-miss character, such as `=`
instead of `:=` or a `!` inside a complex expression. Rewrite the operation
explicitly or call attention to it with a comment. A helper that hides a vital
edge case creates the same risk.

Use predictable names. Identical concepts should generally use the same
parameter and receiver names across related functions and methods. This keeps
documentation coherent and reduces refactoring overhead.

Minimize explicit and implicit dependencies. Every dependency adds code that
can affect behavior, while reliance on internal or undocumented behavior makes
future changes more costly. Consider likely evolution when choosing a design.
A slightly more complex design can be a good tradeoff when it makes future
changes meaningfully easier and safer.

### Consistency

Consistent code looks, feels, and behaves like similar code across the broader
codebase, team, package, and file. Package consistency is often the most
immediately important: multiple solutions to one problem or multiple names for
one concept are particularly disruptive in a single package.

Consistency does not override clarity, simplicity, concision, maintainability,
documented principles, or global consistency. Use it to choose among otherwise
equivalent options.

## Core guidelines

The most important requirements for all Go code. They should be learned and
followed by the time readability is granted, change infrequently, face a high
bar for additions, and expand on Effective Go — the community-wide baseline.

### Formatting

- Every Go source file must match the output of `gofmt` (enforced in
  presubmit at Google).
- Generated code should generally also be formatted (e.g. with
  `format.Source`), since it is browsable in code search.

### MixedCaps

- Multi-word names use `MixedCaps` or `mixedCaps`, never underscores.
- This applies even where other languages' conventions differ: exported
  constants are `MaxLength`, unexported ones `maxLength`.
- Treat local variables as unexported when choosing initial capitalization.

### Line length

- Go source has no fixed line length.
- If a line feels too long, refactor rather than split it; if it is already as
  short as practical, let it remain long.
- Do not split a line before an indentation change (function declaration,
  conditional) and do not split long strings such as URLs merely to fit.

### Naming

- Naming is more art than science. Go names tend to be shorter than in many
  other languages; general identifier guidance still applies.
- Names should not feel repetitive at the use site, should account for
  context, and should not repeat concepts that are already clear.
- The Decisions document gives more specific naming guidance.

### Local consistency

- Where the guide has nothing to say, authors may choose their preferred
  style unless nearby code (usually the file or package, sometimes team or
  project directory) has a consistent stance on the point.
- Valid local-style examples: `%s` vs `%v` for formatting errors, buffered
  channels in place of mutexes.
- Invalid local-style considerations: line-length restrictions and
  assertion-based testing libraries — the guide's rules prevail regardless of
  local precedent.
- A one-file disagreement with the guide is generally raised in review; if a
  consistent fix exceeds the change's scope, file a bug to track it.
- Local consistency never justifies new code that worsens a deviation,
  exposes it through more APIs, spreads it to more files, or introduces a
  bug: clean up in the same change, refactor first, or pick an alternative
  that does not make the local problem worse.
