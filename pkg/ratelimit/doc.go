// Package ratelimit is a global rate limiter: GCRA (Generic Cell Rate
// Algorithm) over the sessions of the `ratelimit_state` table, which all pods
// share through MySQL.
//
// The code has two layers. The math in rate.go and algorithm.go is pure
// functions with no I/O, no clock and no locks: `summarize` computes the
// summary of the sessions of one rate limit, `decide` decides a request
// against it in O(1), and `summaryAfter` updates it in O(1). An `entry`
// (entry.go) holds the sessions of one rate limit and their summary, and its
// methods are the only code that changes them. The limiter keeps one entry
// per fingerprint in memory.
//
// `TestDecideReplaysTheDocumentationTables` replays the tables below.
//
// # Terms
//
//   - Rate limit: one thing that is limited, for example one identifier in
//     one namespace with one duration. `Limit` is the number of units it
//     allows per `Duration`.
//   - Unit: what a request spends. A request asks for `units` units.
//   - Consumed time: the time that units use, `⌈units × Duration / Limit⌉`
//     nanoseconds (rounded up).
//   - `reset_at`: the time when every admitted unit is paid back and a
//     request of `Limit` units fits again. GCRA papers call it TAT
//     (theoretical arrival time).
//   - Session: a stretch of time in which one pod adds its admitted units to
//     one row. The row holds the session's anchor and its consumed time.
//   - Anchor: the fixed time from which a session's consumed time counts.
//   - Live session: the session a pod currently adds its admitted units to.
//     A pod has at most one per rate limit. All its older sessions are frozen:
//     they keep counting, but get no new units.
//   - Candidate: for one session, its anchor plus the consumed time of every
//     session anchored at or after it. `reset_at` is the largest candidate.
//   - Pause: a time in which `reset_at` was in the past, so every unit was
//     free.
//
// All times are int64 nanoseconds since the Unix epoch, which the suffix `Ns`
// marks.
//
// # GCRA on one machine
//
// GCRA is a token bucket written as time. It does not count tokens. It
// stores one point in time, `reset_at`.
//
// Every unit has a price in time. A rate of `Limit` units per `Duration`
// prices one unit at `Duration / Limit`, so 10 per 10 seconds makes one unit
// use 1 second.
//
// A request is allowed if, after paying for it, `reset_at` stays at most one
// `Duration` ahead of now. One `Duration` is the burst tolerance: a rate
// limit that was idle can spend all `Limit` units at once. GCRA papers write
// the tolerance as τ = `Duration` minus the price of one unit and check
// against the time of the last unit; the check here is the same thing.
//
// If the request is allowed, `reset_at` becomes
// `max(reset_at, now) + consumed time`. The max matters: a `reset_at` in the
// past means every unit is free, and idle time is not saved up beyond that. A
// denied request changes nothing.
//
// Time pays units back on its own. There is no refill job and no window
// boundary. With 10 per 10 seconds, where the check is
// `(how far reset_at is ahead of now, or 0) + consumed time ≤ Duration`:
//
//	now       request   reset_at before   check               result   reset_at after
//	12:00:00  1 unit    (past)            0s + 1s ≤ 10s       allow    12:00:01
//	12:00:00  9 units   12:00:01          1s + 9s ≤ 10s       allow    12:00:10
//	12:00:00  1 unit    12:00:10          10s + 1s > 10s      deny     12:00:10
//	12:00:01  1 unit    12:00:10          9s + 1s ≤ 10s       allow    12:00:11
//	12:00:30  1 unit    12:00:11 (past)   0s + 1s ≤ 10s       allow    12:00:31
//
// Each request rounds its consumed time up separately. So several small
// requests can use more time than one large request: at 3 units per 2 ns, an
// idle rate limit admits one request of 3 units (2 ns), but only two requests
// of 1 unit each (1 ns each) at the same instant.
//
// # Sharing a rate limit between pods
//
// Many pods decide on the same rate limit, and each pod sees the others a
// few seconds late. A single shared `reset_at` does not work: two pods read
// the same value, both add their own units, both write, and one write is
// lost. Lost units are over-admission that never evens out.
//
// So each pod writes only the rows of its own sessions. The anchor of a row
// never changes and its consumed time only grows. When `entry.mergeRows`
// sees two copies of one row, it keeps the larger consumed time, so a late or repeated
// write never undoes a newer one.
//
// `reset_at` is computed from all rows:
//
//	reset_at = max over sessions r of
//	           (r.anchor + sum of consumed of sessions anchored at or after r)
//
// Rows sorted newest first, any rate:
//
//	session   anchor     consumed   candidate
//	B         12:00:03   9s         12:00:03 + 9s             = 12:00:12
//	A         12:00:00   12s        12:00:00 + 9s + 12s       = 12:00:21   <- reset_at
//	C         11:58:40   3s         11:58:40 + 9s + 12s + 3s  = 11:59:04
//
// A candidate answers one question: if the rate limit had no pause from this
// anchor on, when would it be free? The formula is correct because each
// admitted request keeps `reset_at` equal to single-machine GCRA, by the rule
// in the next section. `FuzzDecideMatchesSingleMachineGCRA` checks this for
// random histories of sessions and requests.
//
// # Where a request goes
//
// A pod adds its admitted units to its live session when it can. Units added
// to the live session raise the candidates of the sessions anchored at or
// before it. They do not raise the candidates of sessions anchored after it.
// So the live session can take a request only if one of the candidates it
// raises already equals `reset_at` (a tie is enough), and `reset_at` is not
// in the past. Then `reset_at` moves to `reset_at + consumed time`, as GCRA
// requires. Two cases break this:
//
// After a pause, `reset_at` is in the past. Adding to the live session
// starts from the old, past `reset_at`, so the request is charged too little
// or not at all. GCRA starts from now.
//
// When a peer started a session after the live one and a candidate of the
// newer sessions is larger than every candidate the live session raises,
// adding to the live session moves `reset_at` too little or not at all.
//
// In both cases `decide` sets `createNewSessionAtNs` to
// `max(now, newest anchor)`. A session anchored there is counted by the
// candidate of every other session, and its anchor is never before now.
// So `reset_at` becomes exactly `max(reset_at, now) + consumed time`. The old
// sessions stay as they are and keep counting in full, so nothing is lost or
// counted twice. An extra session only costs one row.
//
// # Over-admission is paid back
//
// Two pods that have not seen each other yet can both admit. With 10 per 10
// seconds, pods A and B can each admit 10 units at 12:00:00, 20 in total.
// Once both rows are visible, `reset_at` is 12:00:00 + 10s + 10s = 12:00:20.
// A request of 1 unit next fits at 12:00:11, and a check of zero units at
// 12:00:10. Later decisions count the usage of both pods.
//
// # Exact time arithmetic
//
// Do not replace the 128-bit math in [Rate] with plain int64 or uint64
// multiplication. The product `units × Duration` can reach `Limit × Duration`,
// and that does not fit into 64 bits for rates the API accepts:
//
//	rate                 Limit × Duration (ns)   fits into uint64 (max 1.8e19)
//	10,000 per 30 days   2.6e19                  no
//	400M per minute      2.4e19                  no
//
// A product that wraps around gives a small or negative time, and the limiter
// admits too much. Dividing first, as in `Duration / Limit × units`, loses
// precision: 162M per minute is 370.37 ns per unit, and 370 ns would let the
// full limit through in less than a minute. [math/big] is exact but allocates
// on every decision.
//
// [bits.Mul64] keeps the full 128-bit product and [bits.Div64] divides it, so
// the result is exact for every `Limit` and `Duration` that fit into int64,
// with no allocation. `bits.Div64` panics if the divisor is zero or if the
// quotient does not fit into 64 bits; [Rate.Validate] and the rule
// `units ≤ Limit` in `decide` exclude both cases, so keep them in front
// of every call. `FuzzRateMatchesExactArithmetic` checks the result against
// [math/big] over the full int64 range.
package ratelimit
