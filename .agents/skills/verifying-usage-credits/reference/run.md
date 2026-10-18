# Usage-credit verification run

Run on 2026-10-18 against Unkey source revision
`4b9345c7403c7ce6e1bcd947a7f0b4e7c7b50f4f`, with Go 1.27.2 from mise.
Generated using the Amp adaptation of pstack v0.15.27's formal verification
skill. No product code was changed.

## Result

The direct DB limiter can accept more credits than the key has. The minimal
observed counterexample uses two callers, one credit, and a cost of one each:

1. Caller A reads one credit.
2. Caller B reads one credit.
3. A's SQL decrement changes the balance from one to zero.
4. B's SQL decrement leaves the balance at zero and reports no error.
5. Both real `Limit` invocations return `Valid: true` and `Remaining: 0`.

Atomic SQL decrements do not make the preceding admission checks atomic.
The modeled reads are strongly consistent, so this failure does not require
replica lag. Redis errors can enter this fallback according to the source.
The run did not contact Redis, MySQL, the public API, or production telemetry.
It does not establish incident frequency or production impact.

## Executed bounds

The normal command was:

```bash
mise exec -- go run ./.agents/skills/verifying-usage-credits/scripts
```

It completed all 23,440 executions and exited nonzero for the finding:

```plaintext
callers=1 schedules=1 cases=16 overspend=0
callers=2 schedules=6 cases=384 overspend=40
callers=3 schedules=90 cases=23040 overspend=5040
```

Balances and costs each ranged from zero through three. Counts include
equivalent histories caused by unused scheduling slots. These are not
probabilities or counts of unique bugs. An independent Python enumeration
using permutations produced the same schedule and violation counts.

The same command with `-race` completed with identical counts and no Go
data-race diagnostic. The overspend is a protocol race, not an unsynchronized
Go memory access.

## Mutation validation

A scratch Go build overlay replaced the insufficient-credit guard in the real
`limit.go` with `if false`. The checker then found six overspends in the 16
one-caller cases, including accepting cost one with zero credits. It also
reported 264 and 19,890 violations for two and three callers.

The mutation changed the compiled product method, not the scheduler or its
property. Product files on disk stayed unchanged. Running without the overlay
restored zero one-caller violations and the original concurrency findings.

## Tool checks

`mise run build` passed, including repository lint. Explicit lint of the hidden
skill package also reported `0 issues.` The Rask invocation for that hidden
package was rejected as outside the module. Rask on the production limiter
reported zero tests, so neither is evidence for the property. The executed
checker and its mutation control provide that evidence.

`mise run test` failed in the unchanged logdrain integration test
`TestEngine_Integration/one_replica_processes_drains_in_parallel_up_to_the_limit`.
It expected two concurrent drains, observed one, and timed out. Rask reported
166 skipped suites after that failure. The full suite is not green. This run
did not investigate or change the logdrain code.

Independent comment review found no comments or suppressions in the checker:
zero deletions, restorations, reruns, structural flags, encoding offers,
encodings, or unenforced constraints. Self-review required no code refactor.
No Markdown formatter applies to these skill files.

See [the skill](../SKILL.md) for assumptions, exclusions, source-drift handling,
and rerun instructions. The finding remains open. Do not weaken the invariant
to make the command pass.
