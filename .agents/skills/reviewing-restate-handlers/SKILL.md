---
name: reviewing-restate-handlers
description: "Reviews restate handler code in svc/ctrl/worker for two classes of defect: restate client calls (service calls, state access, sleep, etc.) incorrectly placed inside restate.Run/RunVoid/RunAsync closures, and cron handlers whose invocation retry policy pauses instead of kills on max attempts. Use when reviewing restate handlers, adding a cron job, checking restate.Run usage, or auditing worker service code."
---

# Reviewing Restate Handlers

Two independent audits: [context misuse inside `Run` closures](#context-misuse-inside-run-closures) and [cron retry policy](#cron-retry-policy). Run both when reviewing worker code; run the retry-policy one whenever a cron handler is added or its policy is touched.

## Cron retry policy

**Every cron handler must set an explicit `restate.WithInvocationRetryPolicy` ending in `restate.KillOnMaxAttempts()`. Never `restate.PauseOnMaxAttempts()`, and never leave the policy unset.**

Cron handlers in `svc/ctrl/worker/cron/` are Virtual Object handlers on a fixed or period-derived key, so every tick under that key shares one queue. A paused invocation sits on the key forever waiting for an operator, and every later tick queues behind it, so a single bad run silently stops the whole job until someone notices (this is how the key_last_used_sync incident wedged: partitions hit a Restate 404, sat paused, and suspended the run). Killing lets the invocation finish terminally, the queue drains, and the next tick retries from a clean slate.

Killing is safe for crons specifically because they satisfy all three of:

- **Idempotent or cursor-driven.** A cron re-derives its work each tick (a fresh cutoff, a persisted cursor, an absolute month-to-date total), so dropping one run loses nothing that the next run will not redo.
- **No compensation to run.** Pausing exists so a handler can be re-entered and its Go `defer`-based compensation can fire; `KILL` tears the invocation down without re-entering the handler, so `defer compensation.Execute` never runs. Cron handlers have no compensations. Workflow-style services that do (e.g. `DeployService`) legitimately pause — do not "fix" those.
- **Failure is visible another way.** Crons report health through a heartbeat ping, so a missing end-of-run heartbeat surfaces the failure without needing a parked invocation as the signal.

Leaving the policy unset is also wrong: the SDK default retries forever, which parks the VO key indefinitely — the same wedge as pausing, with no attempt cap.

Note that "the DELETE is stateless, so pausing lets an operator inspect it" is not a valid exception. The inspection value is a journal that retention will drop anyway, and the cost is a wedged key.

### What to check

For each handler registered via `ConfigureHandler` on `hydrav1.NewCronServiceServer` in `svc/ctrl/worker/run.go`, and for each standalone service that a cron fans out to (per-partition, per-workspace children):

1. A `restate.WithInvocationRetryPolicy(...)` is passed.
2. It sets `restate.WithMaxAttempts(...)` — an unbounded policy retries forever.
3. It ends in `restate.KillOnMaxAttempts()`.

Report `PauseOnMaxAttempts()` on a cron handler, or a cron handler with no policy at all, as a violation.

## Context misuse inside Run closures

Audit restate handler code to ensure the outer restate context is never used inside `restate.Run`, `restate.RunVoid`, or `restate.RunAsync` closures.

### Background

Restate journals every interaction with its context for deterministic replay. The `restate.Run` / `restate.RunVoid` / `restate.RunAsync` functions wrap **non-deterministic side effects** (DB queries, HTTP calls, etc.) into a single journaled step. Inside these closures, the only context available is `restate.RunContext`, which is a plain `context.Context` — it has no restate capabilities.

If the **outer** restate context (the handler's `ctx` parameter, typed as `restate.Context`, `restate.ObjectContext`, `restate.WorkflowContext`, `restate.WorkflowSharedContext`, etc.) is used inside a `Run` closure, it can break replay determinism and cause subtle bugs that are very hard to diagnose.

### What to check

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

#### Violations — using the outer `ctx` inside the closure

These are **forbidden** inside `restate.Run` / `restate.RunVoid` / `restate.RunAsync`:

1. **Service calls via the restate context**: e.g., `hydrav1.NewVersioningServiceClient(ctx, ...)` — the first argument to generated service client constructors is the restate context
2. **State access**: `restate.Get(ctx, ...)`, `restate.Set(ctx, ...)`
3. **Sleep**: `restate.Sleep(ctx, ...)`
4. **Nested Run**: `restate.Run(ctx, ...)` inside another `restate.Run`
5. **Key access**: `restate.Key(ctx)`
6. **Async operations**: `restate.RunAsync(ctx, ...)`, sending messages via `.Send()`
7. **Any other method or function that takes the outer `ctx`** where `ctx` is a restate context type

#### Allowed — using the closure's `RunContext`

Inside the closure, code should use the `restate.RunContext` parameter (often named `rc`, `runCtx`, or `stepCtx`) for:
- Database queries: `db.Query.Something(runCtx, ...)`
- Transactions: `db.TxRetry(runCtx, ...)`
- External API calls: `s.vault.Encrypt(runCtx, ...)`
- Any operation that needs a `context.Context`

#### Not a violation

- Using the outer `ctx` **outside** of `restate.Run` closures is correct and expected
- Passing plain values (not the context) captured from outer scope into the closure is fine
- Using `restate.TerminalError(...)` inside a closure is fine (it doesn't take a context)

## Procedure

Context misuse:

1. Use `Grep` to find all files containing `restate.Run`, `restate.RunVoid`, or `restate.RunAsync` under `svc/ctrl/worker/`
2. Read each file
3. For each closure passed to `Run`/`RunVoid`/`RunAsync`, identify:
   - The outer restate context variable name and type (from the enclosing handler function signature)
   - The closure's `RunContext` parameter name
4. Check if the outer context variable appears anywhere in the closure body
5. Report each violation with file, line, and what the fix should be (either move the call outside the closure, or replace `ctx` with the closure's `RunContext` if the call only needs a `context.Context`)

Cron retry policy:

1. Read the `hydrav1.NewCronServiceServer(...)` binding in `svc/ctrl/worker/run.go` and list every handler name passed to `ConfigureHandler`, plus every handler declared on `CronService` in `svc/ctrl/proto/hydra/v1/cron.proto`
2. Any `CronService` handler with no `ConfigureHandler` entry, or whose entry passes no retry policy, is a violation (the SDK default retries forever)
3. For each policy, check it sets `WithMaxAttempts` and ends in `KillOnMaxAttempts()`
4. Repeat for the standalone services a cron fans out to (per-partition, per-workspace children bound separately in `run.go`)

## Output format

For each violation found, report:

```
FILE:LINE — `ctx` used inside restate.Run/RunVoid/RunAsync
  Outer context: `ctx restate.ObjectContext` (from handler function signature)
  Violation: `someFunction(ctx, ...)` should use `runCtx` or be moved outside the closure
```

```
FILE:LINE — cron handler pauses on retry exhaustion
  Handler: RunSomething (ConfigureHandler in run.go)
  Violation: `restate.PauseOnMaxAttempts()` wedges the VO key; use `restate.KillOnMaxAttempts()`
```

If no violations are found, report: "No restate context violations found." and/or "All cron handlers kill on max attempts."
