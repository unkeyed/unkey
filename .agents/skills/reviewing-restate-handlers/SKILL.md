---
name: reviewing-restate-handlers
description: "Reviews restate handler code in svc/ctrl/worker to find restate client calls (service calls, state access, sleep, etc.) incorrectly placed inside restate.Run, restate.RunVoid, or restate.RunAsync closures. Use when reviewing restate handlers, checking restate.Run usage, or auditing worker service code."
---

# Reviewing Restate Handlers

Audit restate handler code to ensure the outer restate context is never used inside `restate.Run`, `restate.RunVoid`, or `restate.RunAsync` closures.

## Background

Restate journals every interaction with its context for deterministic replay. The `restate.Run` / `restate.RunVoid` / `restate.RunAsync` functions wrap **non-deterministic side effects** (DB queries, HTTP calls, etc.) into a single journaled step. Inside these closures, the only context available is `restate.RunContext`, which is a plain `context.Context` — it has no restate capabilities.

If the **outer** restate context (the handler's `ctx` parameter, typed as `restate.Context`, `restate.ObjectContext`, `restate.WorkflowContext`, `restate.WorkflowSharedContext`, etc.) is used inside a `Run` closure, it can break replay determinism and cause subtle bugs that are very hard to diagnose.

## What to check

Scan all `.go` files under `svc/ctrl/worker/` for closures passed to:
- `restate.Run(ctx, func(rc restate.RunContext) ...)`
- `restate.RunVoid(ctx, func(rc restate.RunContext) ...)`
- `restate.RunAsync(ctx, func(rc restate.RunContext) ...)`

Inside each closure body, flag any reference to the **outer** restate context variable. The outer context is typically the handler function's first parameter (often named `ctx`) whose type is one of:
- `restate.Context`
- `restate.ObjectContext`
- `restate.ObjectSharedContext`
- `restate.WorkflowContext`
- `restate.WorkflowSharedContext`

### Violations — using the outer `ctx` inside the closure

These are **forbidden** inside `restate.Run` / `restate.RunVoid` / `restate.RunAsync`:

1. **Service calls via the restate context**: e.g., `hydrav1.NewVersioningServiceClient(ctx, ...)` — the first argument to generated service client constructors is the restate context
2. **State access**: `restate.Get(ctx, ...)`, `restate.Set(ctx, ...)`
3. **Sleep**: `restate.Sleep(ctx, ...)`
4. **Nested Run**: `restate.Run(ctx, ...)` inside another `restate.Run`
5. **Key access**: `restate.Key(ctx)`
6. **Async operations**: `restate.RunAsync(ctx, ...)`, sending messages via `.Send()`
7. **Any other method or function that takes the outer `ctx`** where `ctx` is a restate context type

### Allowed — using the closure's `RunContext`

Inside the closure, code should use the `restate.RunContext` parameter (often named `rc`, `runCtx`, or `stepCtx`) for:
- Database queries: `db.Query.Something(runCtx, ...)`
- Transactions: `db.TxRetry(runCtx, ...)`
- External API calls: `s.vault.Encrypt(runCtx, ...)`
- Any operation that needs a `context.Context`

### Not a violation

- Using the outer `ctx` **outside** of `restate.Run` closures is correct and expected
- Passing plain values (not the context) captured from outer scope into the closure is fine
- Using `restate.TerminalError(...)` inside a closure is fine (it doesn't take a context)

## Procedure

1. Use `Grep` to find all files containing `restate.Run`, `restate.RunVoid`, or `restate.RunAsync` under `svc/ctrl/worker/`
2. Read each file
3. For each closure passed to `Run`/`RunVoid`/`RunAsync`, identify:
   - The outer restate context variable name and type (from the enclosing handler function signature)
   - The closure's `RunContext` parameter name
4. Check if the outer context variable appears anywhere in the closure body
5. Report each violation with file, line, and what the fix should be (either move the call outside the closure, or replace `ctx` with the closure's `RunContext` if the call only needs a `context.Context`)

## Output format

For each violation found, report:

```
FILE:LINE — `ctx` used inside restate.Run/RunVoid/RunAsync
  Outer context: `ctx restate.ObjectContext` (from handler function signature)
  Violation: `someFunction(ctx, ...)` should use `runCtx` or be moved outside the closure
```

If no violations are found, report: "No restate context violations found."
