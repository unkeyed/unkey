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
- Cron handlers and their independently bound children need a bounded invocation
  retry policy ending in `restate.KillOnMaxAttempts()`. Pausing or retrying forever
  blocks subsequent ticks on the same key. See the bindings in [run.go](run.go).
- Do not apply that kill policy to workflows that rely on deferred compensation.
  Killing skips handler re-entry and its compensation. `Deploy` deliberately
  pauses on exhaustion; `Build` kills so its caller can compensate.

For tests of Restate scheduling and flow control, use a disposable Restate
instance, as in [deploy/flow_control_test.go](deploy/flow_control_test.go).
Ordinary dependency mocks do not exercise the runtime's behavior.
