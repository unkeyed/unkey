# Restate workers

- Handler bodies replay. Journal external I/O and nondeterministic results with
  `restate.Run`, `RunVoid`, or `RunAsync`. External writes still need idempotency:
  an effect can succeed before its result is journaled and then run again.
- Keep Restate service calls, state operations, sleeps, and nested Runs outside
  Run closures. Inside a closure, pass its `restate.RunContext` to ordinary I/O,
  not the outer handler context.
- Use `restateutil.Now(ctx)` for time in replayed handler bodies. An injected
  wall clock does not make replay deterministic. Keep journal order stable,
  including when starting work from a map.

## Retry exhaustion: kill or pause

Give every handler a bounded invocation retry policy. Choose its exhaustion
action from what happens to a stuck invocation. See the bindings in
[run.go](run.go).

- A killed invocation ends with a terminal error. Its handler does not run
  again, so deferred compensation in that handler never runs. A caller that
  awaits it receives the error.
- A paused invocation stays in Restate until an operator resumes or kills it.
  On a keyed service or workflow, it blocks every later invocation on the same
  key. A caller that awaits it waits until then.

Use `restate.KillOnMaxAttempts()` when:

- The handler is a cron handler or an independently bound child. A paused tick
  blocks subsequent ticks on the same key, and the next tick retries the work.
- The work is best effort and nothing waits for it, such as
  `OpenapiService.ScrapeSpec`, which `Deploy` sends without awaiting. A pause
  only leaves an invocation to clean up by hand.
- An awaiting caller owns the compensation. `Build` kills, so `Deploy` gets the
  error and compensates.

Use `restate.PauseOnMaxAttempts()` only when:

- The handler relies on deferred compensation that must run on re-entry.
  `Deploy` pauses because a kill would skip the Go `defer` that runs its
  compensation. Only a resume re-enters the handler.
- An operator must fix the cause and resume the invocation, and losing its
  journaled progress costs more than a stuck invocation.

Do not choose pause "for future compensation". If the handler has no
compensation now, kill.

The invocation policy is a safety net. For a handler with compensation, bound
each `restate.Run` with `restate.WithMaxRetryDuration` so that exhaustion
returns a terminal error into Go and compensation runs normally.

For tests of Restate scheduling and flow control, use a disposable Restate
instance, as in [deploy/flow_control_test.go](deploy/flow_control_test.go).
Ordinary dependency mocks do not exercise the runtime's behavior.
