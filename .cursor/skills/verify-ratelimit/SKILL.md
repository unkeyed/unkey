---
name: verify-ratelimit
description: "Check that the local sliding-window rate limiter never commits a request above its limit and that a denial or a failed batch leaves the counter unchanged. Use when a change touches internal/services/ratelimit decision, commit, or rollback code."
---

# Verify the local rate-limit commit

An allowed request must leave the committed sliding-window count within the limit. A denied request must not change that count. `RatelimitMany` commits every request in the batch or none of them. A break lets an identifier spend past its limit, or burns quota on a request the caller was told was denied.

The check calls the real `NewLocal` service. No verifier in this repo's toolchain checks Go for every input, so this enumerates every sequence inside the bounds below instead of modeling the function in another language. A separate model would stay green if `Ratelimit` changed.

## When to run

Run it when a change touches any of these:

- `internal/services/ratelimit/ratelimit.go`: `Ratelimit`, `RatelimitMany`, `slidingWindowCount`, `prepareCheck`
- `internal/services/ratelimit/sequence.go`: `calculateSequence`
- `internal/services/ratelimit/service.go`: `NewLocal`

Ignore the check for metrics, logging, the cross-region MySQL push, and the Redis replay buffer. Those paths are outside the bound.

## Run

From the repo root, with mise and Go 1.27.3:

```bash
mise exec -- go test -tags verifyratelimit -count=1 -timeout 180s ./internal/services/ratelimit/ -run TestFormalCommittedCount
```

Pass prints `pass: <count> cases` in the test log and exits 0. A failure prints a short sequence of `Ratelimit` calls and the committed current a later zero-cost read observed. That sequence is the counterexample. Do not weaken the test to match a new failure.

## How to read a failure

A real bug is a sequence the service itself accepted while a later zero-cost `Ratelimit` on the same identifier, namespace, duration, and timestamp reports a current above the limit, or a denial whose later zero-cost read moved. The same reading covers a batch: if any response has `Success: false`, every identifier in the batch must still show its earlier current.

A modeling error is a rejected request the test should not have sent: empty workspace, namespace, or identifier, a limit below 1, a negative cost, or a duration under 1 second. Those return an error from `Ratelimit` before any counter changes. Fix the test inputs. Do not treat that error as a product bug.

If the test no longer compiles because a field or function moved, read the functions named above and update the test only so that it still asks the same question. Then move the sync commit forward.

## What is in bounds

`NewLocal` only. One process. No Redis and no MySQL. Limits 1 through 4. Costs 0, 1, and 2 for single-call sequences of length 1 through 4 at one window start, and length 3 across offsets 0, 500ms, 999ms, 1s, 1500ms, and 2s for limits 1 and 3. Times in one sequence only move forward. A later read of an earlier moment is not a promise this service makes, because the weight depends on the read time while the counter is stored on the fixed window. Batches of two requests use costs 0 through 3 and limits 1 through 3, either on one identifier or on two. Duration is 1 second. Counts stay far below the point where `float64` stops representing integers exactly.

The committed current is a zero-cost `Ratelimit` at the same timestamp. At one timestamp the current window has weight 1, so a committed cost increases that reading by the cost. The test does not reimplement the previous-window weight except at a window boundary: at the first instant of the next window the whole previous window still counts, and two windows later an empty previous window reads as 0. The code's weight is `int64(float64(previous) * (1 - elapsed))`, which truncates. A previous count of 1 therefore adds nothing for every read after the exact boundary. That truncation is the implemented rule. This check does not replace it with real-number rounding.

## What is out of bounds

Concurrent `Ratelimit` calls and the CAS retry cap. Redis origin fetches, strict mode, and cross-region `globalCount`. The global push of `RatelimitMany` speculative increments. Limits or costs above the sets above. Windows other than 1 second. The mathematical real-number weight at offsets other than a window boundary. `slidingWindowCount`'s `float64` truncation for large counters.

A pass covers this bound on `NewLocal`, not every region and not every `int64`.

## Sync

Sync commit: `8eb54b6907b935f01986db8cf39a3fe3fe4d5abc`

Confirmed at that commit against `Ratelimit`, `RatelimitMany`, and `slidingWindowCount` in `internal/services/ratelimit/ratelimit.go`, `calculateSequence` in `internal/services/ratelimit/sequence.go`, and `NewLocal` in `internal/services/ratelimit/service.go`.

Move this commit forward only after you have read those functions at the new commit and confirmed the test still describes them. A green run is not that confirmation when the functions changed and the test was not reread.

## Known way it fails

Removing the deny check in `Ratelimit` (`effectiveCount > req.Limit`) lets a cost of 2 through on limit 1 at the window start. The test then reports that call as allowed and the following zero-cost read saw committed current 2. Restore the check. Do not change the test to allow that current.
