---
name: verifying-usage-credits
description: "Exhaustively checks bounded credit admission schedules against Unkey's real DB fallback limiter. Use when changing usage-limit admission, credit SQL, or fallback wiring, or when investigating concurrent credit overspend."
---

# Verify usage credits

Property: the sum of costs accepted for one limited key must not exceed its
initial credits when there are no refills. A violation lets clients use more
paid operations than their credit budget permits.

This suite executes the real `usagelimiter.New(...).Limit` code with a small
exhaustive scheduler and an in-memory model of the two SQL operations. It is
bounded explicit-state exploration, not a proof of the full Go program, MySQL,
or the deployed service. Go is already pinned by mise. No verifier installation
or additional dependency is needed.

## Run

From the repository root, run:

```bash
mise exec -- go run ./.agents/skills/verifying-usage-credits/scripts
```

The checker makes no network or database calls and changes no product files.
Go may fetch pinned modules on the first build. The checker checks
[source fingerprints](sources.sha256), explores every bounded
schedule, prints counts and the first counterexample for each caller count,
then exits nonzero if the property fails. A source mismatch prints `STALE`.
A timeout, unexpected error, or extra callback is an incomplete check, not a
property violation. `go run` may collapse the checker's exit status to 1.

Keep output in a review artifact when needed. No services require cleanup.
Normal Go build caches remain. Do not install tooling if the pinned toolchain
is unavailable.

## Bounds and source mapping

The [checker](scripts/main.go) covers one existing, limited key, one to three
callers, initial credits 0 through 3, and each cost 0 through 3. All operations
succeed. Reads see the latest committed balance, without replica lag.

Each caller has two scheduling slots, one read and at most one decrement.
An early-return caller's unused slot does nothing. Enumerating all ordered
merges gives 1, 6, and 90 schedules for one, two, and three callers. Multiplying
by all balance and cost assignments gives 16, 384, and 23,040 executions.
Some early-return histories repeat. Counts are executions, not unique states.

The modeled atomic steps are:

| Step | Source | Meaning |
| --- | --- | --- |
| Read | `pkg/db/queries/key_find_credits.sql` | Read the shared balance atomically. |
| Check and return | `internal/services/usagelimiter/limit.go` | Execute real Go admission and response logic. No shared balance access between callbacks. |
| Decrement | `pkg/db/queries/key_update_credits_decrement.sql` | Atomically set balance to `max(0, balance - cost)`, without reporting whether sufficient credits existed. |
| Wiring | `svc/api/run.go` | Read through `RO()`, decrement through `RW()`, outside a shared transaction. |
| Fallback | `internal/services/usagelimiter/redis.go` | Counter errors call the direct DB limiter. The checker starts at that limiter, not at Redis. |

The fingerprints also cover the two generated sqlc implementations in `pkg/db`.
Their execution and argument binding must match the modeled SQL semantics.

One service instance per caller represents independent API processes sharing
one database. Callback channels let the scheduler order every database access.
Local calculations, metrics, tracing, and result publication do not change the
modeled balance. Results are collected in caller order, not completion order.
With nonnegative costs and no refill, any overspend persists to the final state.

Excluded: unlimited/missing keys, negative costs, int64 overflow, SQL failures,
replica lag, retries, ambiguous commits, Redis initialization and replay,
expiration, refills, cancellation, and liveness. The scheduler is repository
code, not an independently verified model checker. SQL behavior is modeled,
not run against MySQL. Do not call this an end-to-end production reproduction.

## Interpret and maintain

A counterexample identifies each caller's read, update, and final `Valid`
response. Check the trace against the source mapping before diagnosing a bug.
Preserve failing properties. Do not fix product behavior without authorization.

When a fingerprint changes, read the complete affected function and its callers.
Update the mapping, bounds, and scheduler if callback order, SQL semantics, or
atomicity changed. Only then regenerate the affected hashes with `sha256sum`.
Never refresh hashes merely to silence `STALE`.

The bounded run exposes an overspend race: two callers can both read the last
credit before either decrements it. The SQL clamps both writes to zero, while
both callers return `Valid: true`. See [the run record](reference/run.md) for
executed results and limitations.

## Check that it can fail

Use a scratch Go build overlay for `internal/services/usagelimiter/limit.go`.
Copy that file outside the checkout and change only the insufficient-credit
guard to `if false`. Map its absolute original path to the scratch file under
the overlay JSON `Replace` object. Run:

```bash
mise exec -- go run -overlay /tmp/usage-credit-overlay.json \
  ./.agents/skills/verifying-usage-credits/scripts
```

Choose unique temporary paths when running this procedure. The command above
illustrates the overlay file argument, not a checked-in file. Under the overlay,
the one-caller cases must now show overspend, not only the baseline concurrency
finding. Disk source hashes remain unchanged because only compilation uses the
overlay. This is intentional mutation validation, never the normal check.

Remove only your scratch overlay files, rerun the normal command, and confirm
the one-caller control passes again. A compiler error or fingerprint mismatch
does not satisfy this mutation check.
