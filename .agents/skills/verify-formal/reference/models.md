# Model inventory

The runner lives in [tools/formalverify](../../../../tools/formalverify/).
It calls production APIs directly. It does not copy their implementations into
a second state machine. Small independent predicates and schedule invariants
define the expected behavior.

## Circuit-breaker probe admission

Run `mise run verify-formal --check circuitbreaker-limit`.

Invariant: before a recovering breaker closes, the number of in-flight probes
must not exceed `WithMaxRequests`. Violating this limit can overload a downstream
service while it recovers.

Sources:

- [WithMaxRequests and Do](../../../../pkg/circuitbreaker/lib.go)
- [ErrTooManyRequests contract](../../../../pkg/circuitbreaker/errors.go)
- [State contracts](../../../../pkg/circuitbreaker/interface.go)
- [Executable schedule model](../../../../tools/formalverify/circuitbreaker.go)

The initial state is reached through public calls: construct a breaker with
threshold 1, timeout 1 second, and counter period 1 second; fail one request;
advance the injected clock by 2 seconds. The next request enters recovery.
No wall-clock sleep is used.

Each client has three model states: not started, in flight, and finished. A
start waits until `Do` either rejects or enters its callback. A finish releases
that callback and waits for `Do` to return. Thus `preflight` and `postflight`
execute as serialized mutex sections, with every permitted callback overlap
explored. The checker does not inspect private fields.

Bounds: two or three clients, one invocation per client, probe limits 1 and 2,
and successful probe completions. Enumerating every ordering that puts each
client's start before its finish gives 6 schedules for two clients and 90 for
three. Across two limits, the suite checks 192 schedules. It stops replaying a
schedule at its first violation, but still enumerates the remaining schedules.
Counts describe schedules, not unique failing prefixes.

The recovery model tracks successful completions. Once their count reaches the
probe limit, the breaker may close and the probe-only bound no longer applies.

### Open finding

The reviewed implementation violates the bound in 112 of 192 schedules:

```text
trip breaker; wait past cooldown and counter reset; probe limit=1
start client 0: admitted=true, callback remains blocked
start client 1: admitted=true, callback remains blocked
2 probes in flight before recovery, maximum is 1
```

`preflight` checks `requests`, but `postflight` increments it only after a
callback completes. The mutex protects the counter, not a reservation for an
in-flight probe. The verifier reports this finding and exits nonzero. Product
code is not patched or allowlisted by the suite.

Excluded: failures during probing, ratio thresholds, counter resets during
probing, old requests crossing recovery epochs, multiple recovery cycles,
cancellation, arbitrary clocks, and internal instruction interleavings. The
interface and implementation comments disagree about reopening on probe
failure, so that behavior is not silently chosen as a contract here. This
bounded schedule check does not establish liveness or general linearizability.

## Circuit-breaker recovery control

Run `mise run verify-formal --check circuitbreaker-recovery`.

After the same setup, complete the configured number of successful probes and
issue another request. All callbacks must run without an admission error. The
domain is probe limits 1, 2, and 3, with serial completions and no clock advance
during recovery. All three cases pass on the reviewed source.

Negative control: change the actual recovery comparison from `>=` to `>` in a
temporary Go overlay. All three cases fail because the next request remains
blocked. Without the overlay, all three pass again.

This is a finite recovery control, not exhaustive exploration of all breaker
states. It does not conceal the separate concurrent-admission finding.

## Deployment lifecycle admission

Run `mise run verify-formal --check deploygate`.

The [guard implementation](../../../../pkg/deploy/deploygate/deploygate.go)
is checked against independent predicates in the
[executable contract](../../../../tools/formalverify/deploygate.go):

- Promote: ready, desired running, production, and a current deployment exists;
  an already-current target is allowed only when confirming a rollback.
- Rollback: ready, desired running, production, a current deployment exists,
  and the target differs from current.
- Stop: ready, desired running, and not production.
- Start: desired stopped, not production, and not spend-suspended.

Both acceptance and rejection are checked, so rejecting everything cannot pass.
Incorrect acceptance can activate an unready deployment or stop production.
Incorrect rejection can block recovery or rollback.

The finite domain contains 15 statuses (13 declared values, empty, and an
unknown representative), 4 desired states (running, stopped, empty, unknown),
and 4 environment kinds (production, preview, empty, unknown). Promotion and
rollback include absent/equal/different current IDs and both rollback flags.
Start includes both spend-suspension values. Target IDs are nonempty generated
IDs; only their equality relation matters.

This gives 2,432 decisions: 240 stop, 720 rollback, 1,440 promote, and 32 start.
All pass on the reviewed source. Error codes, message wording, and rejection
precedence remain covered by ordinary package tests, not this model.

Unknown values are representatives, not enumeration of every string. The
reviewed guards use equality tests, making these input classes sufficient for
those predicates. This is a reviewed abstraction, not a machine-checked proof
of equivalence for every string. In particular, start and stop require
non-production, not strictly preview. The model preserves that contract rather
than inventing new validation rules.

Negative control: reverse the readiness comparison in `targetCore` through a
temporary Go overlay. The verifier reports 60 violations. The unmodified
implementation passes before and after.

Excluded: database transactions, stale reads, concurrent workflow transitions,
authorization, deployment existence, and actual traffic routing. Passing these
guards does not prove that their callers pass fresh or authorized state.

## Evidence and maintenance

The command uses the Go toolchain pinned in `.mise/config.toml` and
`.mise/mise.lock`. Fingerprints also cover the modeled source, relevant enum and
clock source, and Go dependency manifests. They do not prove that a model is
correct and do not track every transitive implementation file.

Run the maintained checker tests with:

```bash
mise exec -- rask ./tools/formalverify
```

These tests check CLI results and execute both real-source mutations. No test
requires the open probe-limit bug to remain present. Once it is fixed, the
normal suite can pass without weakening the invariant or changing an expected
failure assertion.

The project has no installed TLA+, SMT, or proof-assistant toolchain. This suite
uses exhaustive finite enumeration in Go. It is not an unbounded mathematical
proof, a complete Go scheduler, or a substitute for integration tests. Add a
separate model-checker backend when the chosen property needs one and its
installation is authorized.
