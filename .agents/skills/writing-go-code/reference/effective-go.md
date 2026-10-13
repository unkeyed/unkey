# Effective Go (distilled)

Source: https://go.dev/doc/effective_go

Caveat: This document was written for Go's 2009 release and is not actively updated. It remains a guide to the core language, but predates generics, modules, and later library additions.

## Introduction

- Write Go as Go. Straight translations from languages such as C++ or Java rarely produce satisfactory Go programs.
- Learn Go's properties, idioms, naming, formatting, and construction conventions so other Go programmers can understand the result.
- Treat this guide as an addition to the language specification, the Tour of Go, and How to Write Go Code, not a replacement.

### Examples

- Use standard-library documentation, source, and runnable examples as models for approaches and implementations.

## Formatting

- Run `gofmt` or package-level `go fmt`; let the tool decide indentation, alignment, and comment layout.
- If formatted output looks wrong, rearrange the program or report a formatter bug. Do not work around `gofmt`.
- Indent with tabs. Use spaces only when necessary.
- Go has no line-length limit. If a line feels too long, wrap it and indent with an extra tab.
- Do not parenthesize `if`, `for`, or `switch` conditions. Go's smaller precedence hierarchy also reduces the need for grouping parentheses.

## Commentary

- Prefer `//` line comments. Use `/* */` mainly for package comments, comments inside expressions, or temporarily disabling large regions.
- A comment immediately before a top-level declaration, with no intervening blank line, documents that declaration.
- Doc comments are the primary package and command documentation. Follow the separate Go Doc Comments guide for their full conventions.

## Names

- Names have semantic effect: an identifier whose first character is uppercase is exported from its package.

### Package names

- Choose short, concise, evocative, lowercase, single-word package names. Avoid underscores and mixed caps.
- Prefer brevity because every importer types the package name.
- Do not preemptively avoid package-name collisions. Importers can rename the rare collision locally.
- Make the package name the base name of its source directory: `encoding/base64` declares `package base64`.
- Avoid dot imports except, when useful, in external tests of a package.
- Use package qualification to avoid repetition: `bufio.Reader`, not `bufio.BufReader`.
- Name a constructor `New` when the package exports one principal type, yielding names such as `ring.New`.
- Prefer a concise name such as `once.Do` over a verbose restatement. A useful doc comment can clarify more than a long name.

### Getters

- Do not prefix a getter with `Get`. For an unexported field `owner`, use `Owner()`.
- Name the corresponding setter `SetOwner` when one is needed.

### Interface names

- Name one-method interfaces with the method name plus `-er` or a similar agent suffix: `Reader`, `Writer`, `Formatter`, or `CloseNotifier`.
- Honor canonical method names and signatures such as `Read`, `Write`, `Close`, `Flush`, and `String`.
- Do not reuse a canonical name for a different signature or meaning.
- When the semantics match a well-known method, use its established name and signature, such as `String`, not `ToString`.

### MixedCaps

- Use `MixedCaps` or `mixedCaps` for multiword names, never underscores.

## Semicolons

- Go's lexer inserts semicolons. Source normally omits them.
- A newline inserts a semicolon after an identifier, basic literal, or `break`, `continue`, `fallthrough`, `return`, `++`, `--`, `)`, or `}`.
- Omit a semicolon immediately before a closing brace.
- Write explicit semicolons in `for` clauses and, rarely, between multiple statements on one line.
- Keep an opening control-structure brace on the same line. A newline before it triggers semicolon insertion:

```go
if i < f() {
    g()
}
```

## Control structures

- Go has only `for`, not `while` or `do-while`.
- `if` and `switch` can start with an initializer. `break` and `continue` can name labels, and `select` multiplexes communications.
- Conditions have no parentheses, and bodies always use braces.

### If

- Put even simple bodies on separate lines, especially bodies containing `return` or `break`.
- Use an initializer to keep setup local: `if err := file.Chmod(0664); err != nil { ... }`.
- Omit `else` when the preceding branch ends in `break`, `continue`, `goto`, or `return`.
- Let successful control flow proceed down the page while error cases return early.

### Redeclaration and reassignment

- A short declaration can redeclare a variable when it is in the same scope, the new value is assignable, and at least one other variable is newly declared: `d, err := f.Stat()`.
- A name declared only in an outer scope is shadowed rather than reassigned.
- Function parameters and named results have the same scope as the function body.
- Use this rule pragmatically to reuse a single `err` variable through sequential operations.

### For

- Use one of the three forms: `for init; condition; post`, `for condition`, or `for {}`.
- Use `range` over arrays, slices, strings, maps, and channels.
- Drop an unused second range value: `for key := range m`.
- Use `_` when only the second value matters: `for _, value := range values`.
- Ranging over a string decodes UTF-8 into runes and byte positions. An invalid encoding consumes one byte and yields U+FFFD.
- Go has no comma operator. `++` and `--` are statements, not expressions.
- Update multiple loop variables with parallel assignment:

```go
for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
    a[i], a[j] = a[j], a[i]
}
```

### Switch

- Switch expressions need not be constants or integers. Cases are tested top to bottom until one matches.
- An expressionless `switch` switches on `true` and idiomatically replaces an `if`/`else if` chain.
- Cases do not fall through automatically. Combine alternatives in a comma-separated case list.
- Use `break` to leave a switch early.
- Label an enclosing loop and `break` to that label when leaving the loop, not merely the switch.
- A labeled `continue` applies only to loops.

### Type switch

- Discover an interface value's dynamic type with `switch v := x.(type)`.
- In each case, the declared variable has that case's type.
- Reuse the original variable name idiomatically because each case receives a newly typed variable.

## Functions

### Multiple return values

- Return a result and an error rather than encoding failure in-band or modifying an output pointer.
- Return partial progress with its error, as `Write` returns `(n int, err error)`: a non-nil error is required when fewer than all bytes were written.
- Use multiple ordinary results when a function naturally computes both a value and continuation state.

### Named result parameters

- Name results when the names clarify otherwise indistinguishable return values.
- Named results are initialized to their zero values and act as variables in the function.
- A bare `return` returns their current values.
- Deferred functions can modify named results.

### Defer

- Use `defer` for cleanup that must happen on every return path, such as closing a file or unlocking a mutex.
- Place cleanup next to resource acquisition so later return paths cannot omit it.
- Deferred calls run immediately before the surrounding function returns, not at block exit.
- Arguments, including method receivers, are evaluated when the `defer` statement executes.
- Deferred calls execute in last-in, first-out order.
- Use argument evaluation and LIFO ordering for paired entry/exit tracing when useful.

## Data

### Allocation with `new`

- `new(T)` allocates zeroed storage for `T` and returns `*T`.
- The source notes that, starting with Go 1.26, `new(expression)` allocates a variable initialized to that expression and returns its address.
- Design types with useful zero values so declaration or `new` is enough to begin using them.
- Useful zero values compose: a struct containing `sync.Mutex` and `bytes.Buffer` can itself be immediately usable.

### Constructors and composite literals

- Provide a constructor when the zero value is insufficient.
- Prefer a composite literal over allocating and then assigning each field.
- Returning the address of a local variable is safe. Its storage survives as needed.
- `&T{...}` creates a fresh instance each time it is evaluated.
- Positional struct literals require every field in declaration order.
- Prefer keyed fields when omissions or ordering flexibility matter: `&File{fd: fd, name: name}`. Omitted fields retain zero values.
- `new(T)` and `&T{}` are equivalent for a struct's zero value.
- Arrays, slices, and maps also support composite literals, with indices or map keys as labels.

### Allocation with `make`

- Use `make` only for slices, maps, and channels. It initializes their internal data structures and returns `T`, not `*T`.
- `make([]int, 10, 100)` creates a slice of length 10 and capacity 100 backed by a 100-element array.
- `new([]int)` returns a pointer to a nil slice descriptor and is rarely useful. Prefer `make([]int, n)`.
- Use `new` or an explicit address operation when an actual pointer is required.

### Arrays

- Arrays are values. Assignment and parameter passing copy every element.
- Array length is part of the type, so `[10]int` and `[20]int` differ.
- Arrays can control memory layout or avoid allocation, but copies may be expensive.
- Although a pointer to an array avoids copying, sequence APIs idiomatically use slices instead.

### Slices

- Use slices for most sequence work except explicitly dimensioned data such as matrices.
- A slice references an underlying array. Copies of a slice share that array.
- Functions receiving slices can modify the caller-visible elements. The slice length itself bounds operations such as `Read`.
- Reslice up to capacity. `cap` gives the maximum available length before reallocation.
- `len` and `cap` of a nil slice are zero.
- When growing manually, allocate a larger slice, copy old elements, extend the length, and return the new slice.
- Return a changed slice descriptor because the descriptor itself, including pointer, length, and capacity, is passed by value.

### Two-dimensional slices

- Model two dimensions as an array of arrays or a slice of slices.
- Inner slices can have different lengths.
- Allocate rows independently when they may grow or shrink, preventing one row from overwriting another.
- For fixed row sizes, one backing allocation sliced into rows can be more efficient.

### Maps

- Map keys may be any comparable type. Slices cannot be keys because they have no equality operation.
- Maps are reference-like. Mutations made by a called function are visible to its caller.
- Initialize maps with `make` or key-value composite literals.
- Reading a missing key returns the element type's zero value.
- A `map[T]bool` can represent a set because a missing key reads as `false`.
- Distinguish absence from a stored zero with `value, ok := m[key]`.
- Use `_, ok := m[key]` when only presence matters.
- Delete with `delete(m, key)`; deleting a missing key is safe.

### Printing

- Formatted printing lives in `fmt`: `Printf`, `Fprintf`, `Sprintf`, and related functions. `Sprintf` returns a string.
- Use `Print`, `Fprint`, or `Sprint` for default formats, and `Println` variants for spaces plus a trailing newline.
- `Fprint` and related functions write to any `io.Writer`.
- Numeric verbs derive size and signedness from argument types rather than format modifiers.
- `%v` prints the default representation of any value. `%+v` adds struct field names, and `%#v` prints Go syntax.
- `%q` quotes strings and byte slices, and prints integers or runes as quoted rune constants. `%#q` uses backquotes when possible.
- `%x` prints integers, strings, byte arrays, and byte slices in hexadecimal; `% x` spaces bytes.
- `%T` prints a value's type.
- Map output from `fmt` is sorted lexicographically by key.
- Define `String() string` to control a custom type's string form. Use a value receiver if both values and pointers must format through that method.
- Avoid recursive `String` implementations. Convert the receiver to its underlying type before passing it to a string format.
- A variadic parameter acts as a slice inside the function. Pass it onward as separate arguments with `args...`.
- Variadic parameters can have a specific element type, such as `...int`, not only `...interface{}`.

### Append

- `append` adds elements and returns the resulting slice. Always retain the result because the backing array may change: `x = append(x, values...)`.
- Append one slice to another by expanding the second slice with `...`.

## Initialization

### Constants

- Constants are compile-time numbers, runes, strings, or booleans, including constants declared inside functions.
- Constant initializers must be compiler-evaluable constant expressions. Function calls are not constant expressions.
- Use `iota` for enumerations. It can participate in expressions, and omitted constant expressions repeat implicitly.
- Attach methods such as `String` to named scalar types when values need custom presentation.
- Calling `Sprintf` from `String` is nonrecursive when the chosen verb, such as `%f`, does not request string formatting.

### Variables

- Variable initializers may be arbitrary expressions evaluated at run time.
- Group related package-level initializers when that improves clarity.

### The init function

- A source file may define multiple niladic `init` functions.
- Imported packages initialize first, then package variable initializers run, then `init` functions run.
- Use `init` for setup that cannot be expressed as a declaration, or to verify or repair state before execution.

## Methods

### Pointers vs. values

- Define methods on any named type except a pointer or interface. A receiver need not be a struct.
- Use a pointer receiver when a method must replace or mutate the receiver value, such as updating a slice descriptor.
- Choose established method signatures when they provide useful interface compatibility, such as `Write([]byte) (int, error)` for `io.Writer`.
- Value-receiver methods can be invoked on values and pointers.
- Pointer-receiver methods belong only to the pointer method set. When an addressable value is used in a direct call, the compiler inserts `&` automatically.
- Interface satisfaction still depends on the method set: if only `*T` has the required method, pass `*T`, not `T`.

## Interfaces and other types

### Interfaces

- Interfaces specify behavior. A value can be used wherever its method set provides the required behavior.
- Small interfaces, often one or two methods, are common and usually named from their methods.
- A type can satisfy multiple interfaces without declarations.
- Avoid mutating a value merely to format it. Copy before sorting or otherwise changing it for presentation.

### Conversions

- Convert between named and unnamed types with the same underlying representation to reuse existing behavior, such as formatting a named slice as `[]int`.
- Such a conversion can avoid recursive `String` calls and does not necessarily create a new value.
- Convert an expression to another named type to access that type's methods, such as `sort.IntSlice(s).Sort()`.
- Use conversions to compose capabilities instead of implementing every interface directly when that is clearer.

### Interface conversions and type assertions

- A type switch can extract a concrete value or convert one interface to another.
- Extract one expected type with `value.(Type)`.
- The asserted type must be the value's concrete dynamic type or an interface it satisfies.
- A failed single-result assertion panics.
- Use `value, ok := x.(Type)` to test safely. On failure, `value` is the type's zero value and `ok` is false.

### Generality

- If an implementation type has no exported behavior beyond an interface, keep the type unexported and export the interface.
- In that case, return the interface from constructors. This communicates the supported behavior and localizes implementation substitutions.
- Design APIs around behavior so algorithms or modes can change without changing surrounding users.

### Interfaces and methods

- Methods can make structs, scalar named types, channels, and function types satisfy the same interface.
- Protect mutable handler state against concurrent access in a real HTTP server.
- A channel type can implement an HTTP handler by sending each request as a notification.
- Adapt a function to an interface by defining a named function type with the required method. `http.HandlerFunc` turns a correctly shaped function into `http.Handler`.
- The HTTP examples demonstrate that interfaces are method sets, not class hierarchies.

## The blank identifier

- `_` accepts a value of any type and discards it.

### The blank identifier in multiple assignment

- Use `_` where multiple assignment requires a position whose value is irrelevant.
- It is appropriate to discard a normal result when only the accompanying error matters.
- Never discard an error return. Always check errors.

### Unused imports and variables

- Unused imports and variables are compile errors because they often indicate waste or bugs.
- During unfinished development only, references such as `var _ = fmt.Printf`, `var _ io.Reader`, or `_ = fd` can temporarily keep code compiling.
- Put temporary package-level blank references immediately after imports, comment them, and remove them when no longer needed.

### Import for side effect

- Import a package solely for its initialization side effects with a blank name: `import _ "net/http/pprof"`.
- Use this form only when the side effect is the intended use. Otherwise, use or remove the import.

### Interface checks

- Interface implementation is implicit and most real conversions already provide compile-time checks.
- Test a runtime interface value with `_, ok := value.(Interface)` when only satisfaction matters.
- Add `var _ Interface = (*Type)(nil)` when a package must guarantee implementation but no static conversion elsewhere checks it.
- Do not add such declarations for every implementation. They are for the rare case with no existing static conversion.

## Embedding

- Embed interfaces in interfaces to form their union. Only interfaces can be embedded in interfaces.
- Embed types in structs without field names to promote their methods and avoid forwarding boilerplate.
- Initialize embedded pointers before use.
- Promoted methods can make the outer struct satisfy every interface implemented by the embedded fields.
- Embedding is not subclassing. An invoked promoted method receives the embedded inner value, not the outer value.
- Embedded values remain ordinary fields. Refer to one by its unqualified type name, such as `job.Logger`.
- Define an outer method with the same name to refine or override access to a promoted method.
- A shallower field or method hides a deeper one with the same name.
- Duplicate names at the same depth are errors only when referenced outside the type definition, protecting against irrelevant conflicts added to embedded external types.

## Concurrency

### Share by communicating

- Prefer passing shared values through channels so only one goroutine accesses a value at a time: "Do not communicate by sharing memory; instead, share memory by communicating."
- Treat communication as synchronization, analogous to a type-safe Unix pipeline.
- Do not apply the slogan absolutely. A mutex around a reference count can be the clearer solution.

### Goroutines

- Start concurrent execution by prefixing a function or method call with `go`.
- Goroutines share an address space, are multiplexed over OS threads, start with small stacks, and exit silently when their function returns.
- Invoke a function literal after `go`; closures keep referenced variables alive as needed.
- Give practical goroutines a completion-signaling mechanism, usually a channel.

### Channels

- Allocate channels with `make`. The optional integer sets buffer capacity; omitted or zero means unbuffered.
- Unbuffered sends synchronize sender and receiver. A sender waits until receipt.
- A buffered sender waits only until the value enters the buffer, blocking when the buffer is full.
- Use a channel to signal completion of background work.
- A buffered channel can act as a semaphore that bounds simultaneous work.
- Do not spawn an unlimited goroutine for every queued request merely to block most of them on a semaphore.
- Gate goroutine creation before launch, or start a fixed number of workers that consume a request channel.
- The source notes that its closure over a range variable is buggy before Go 1.22 because iterations shared the loop variable.

### Channels of channels

- Channels are first-class values and can be carried in structs or over other channels.
- Put a result channel in each request to give each client a distinct reply path.
- This pattern supports rate-limited, parallel, non-blocking RPC-style dispatch without shared-state locking.

### Parallelization

- Split an independent calculation into pieces, run each piece in a goroutine, and collect one completion signal per piece.
- A completion channel may be buffered to the number of workers.
- Use `runtime.GOMAXPROCS(0)` when choosing worker parallelism to honor the user's configured resource limit. `runtime.NumCPU()` reports hardware CPUs.
- Distinguish concurrency, structuring independent components, from parallelism, simultaneously executing work for efficiency. Go is a concurrent language, and not every parallel problem fits its model.

### A leaky buffer

- A buffered channel can serve as a bounded free list for reusable buffers.
- Use `select` with `default` to take a free buffer without blocking, allocating when none is ready.
- Return a processed buffer with another non-blocking `select`; if the free list is full, drop it for garbage collection.
- `default` runs when no communication case is ready, making these operations non-blocking.

## Errors

- Return the built-in `error` interface alongside normal results and make failures detailed enough to diagnose.
- Implement richer concrete errors, such as `os.PathError`, to retain operation, path, and underlying error details.
- When feasible, prefix error strings with their operation or package origin, such as `image: unknown format`.
- Callers needing details can use type assertions or switches and inspect concrete error fields to decide whether to recover or retry.

### Panic

- Report ordinary failure with an `error` result.
- Use `panic` for an unrecoverable or supposedly impossible state. Its argument may have any type.
- Real library functions must avoid panic when a problem can be reported or worked around.
- Panic during initialization can be reasonable when a package genuinely cannot initialize.

### Recover

- A panic stops the current function and unwinds that goroutine's stack, executing deferred functions. Reaching the stack top terminates the program.
- `recover` stops unwinding and returns the panic value. It is useful only when called directly from a deferred function.
- Recover at a goroutine boundary when one failed server task must not terminate unrelated goroutines.
- Internal parsing code may panic with a package-specific error to escape deeply nested calls, then recover at the public boundary and return an `error`.
- Deferred functions can clear or modify named result values during recovery.
- Verify that a recovered value is the package's expected panic type. An unexpected runtime panic must continue to fail rather than be hidden.
- Keep panic/recover error handling inside a package. Convert expected internal panics to errors before returning to clients.

## A web server

- The complete example combines flags, `html/template`, `net/http`, and logging in a small server.
- Parse flags, adapt a correctly shaped function with `http.HandlerFunc`, register it with `http.Handle`, then call blocking `http.ListenAndServe` — and check its returned error.
- A handler reads form data with `req.FormValue` and executes a parsed template directly to the `http.ResponseWriter`.
- Build static templates once at package initialization. `template.Must` turns an initialization-time parse failure into a panic.
- Template actions use `{{...}}`; `{{if .}}...{{end}}` conditionally emits content when the current data item `.` is nonempty.
- `{{.}}` renders the current value, and `html/template` applies context-appropriate escaping so submitted text is safe to display.
- The example's broader lesson is that interfaces, function adapters, templates, and package initialization let a useful server remain small and data-driven.
