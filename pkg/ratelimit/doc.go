// Package ratelimit is a global rate limiter: GCRA (Generic Cell Rate
// Algorithm) over the sessions of the `ratelimit_state` table, which all pods
// share through MySQL.
//
// rate.go and algorithm.go are pure functions: `summarize` reduces the
// sessions of one rate limit to a summary, `decide` decides a request against
// it in O(1), and `summaryAfter` updates it. An `entry` holds the sessions and
// their summary and is the only code that changes them.
//
// # Terms
//
//   - Consumed time: the time that units use, `⌈units × Duration / Limit⌉` ns.
//   - `reset_at`: when every admitted unit is paid back. GCRA papers call it
//     TAT (theoretical arrival time).
//   - Session: one row, written by one pod. It holds an anchor, which never
//     changes, and a consumed time, which only grows.
//   - Live session: the session a pod adds its units to. Its older sessions
//     are frozen: they keep counting but get no new units.
//   - Candidate: a session's anchor plus the consumed time of every session
//     anchored at or after it. `reset_at` is the largest candidate.
//
// All times are int64 nanoseconds since the Unix epoch, marked by `Ns`.
//
// # GCRA
//
// A request is allowed if `max(reset_at, now) + consumed time` stays at most
// one `Duration` ahead of now. Then that becomes the new `reset_at`; a denied
// request changes nothing. With 10 per 10 seconds:
//
//	now       request   reset_at before   check               result   reset_at after
//	12:00:00  1 unit    (past)            0s + 1s ≤ 10s       allow    12:00:01
//	12:00:00  9 units   12:00:01          1s + 9s ≤ 10s       allow    12:00:10
//	12:00:00  1 unit    12:00:10          10s + 1s > 10s      deny     12:00:10
//	12:00:01  1 unit    12:00:10          9s + 1s ≤ 10s       allow    12:00:11
//	12:00:30  1 unit    12:00:11 (past)   0s + 1s ≤ 10s       allow    12:00:31
//
// Each request rounds up on its own: at 3 units per 2 ns, one request of 3
// units fits at once, but only two requests of 1 unit.
//
// # Many pods
//
// One shared `reset_at` row would lose writes when two pods update it at the
// same time. So each pod writes only its own session rows, and `summarize`
// computes `reset_at` from all of them. A request goes into the live session
// only if that moves `reset_at` correctly; otherwise it starts a new session
// (see `decide`).
//
// Pods that have not seen each other yet can over-admit, and the excess is
// paid back. With 10 per 10 seconds, pods A and B each admit 10 at 12:00:00.
// Once both rows are visible, `reset_at` is 12:00:20, and the next unit fits
// at 12:00:11.
//
// `TestDecideReplaysTheDocumentationTables` replays both examples.
//
// # Exact time arithmetic
//
// Do not replace the 128-bit math in [Rate] with plain int64 or uint64
// multiplication. `units × Duration` does not fit into 64 bits for rates the
// API accepts:
//
//	rate                 Limit × Duration (ns)   fits into uint64 (max 1.8e19)
//	10,000 per 30 days   2.6e19                  no
//	400M per minute      2.4e19                  no
//
// A wrapped product admits too much. Dividing first loses precision: 162M per
// minute is 370.37 ns per unit, and 370 ns lets the full limit through in
// less than a minute. [bits.Div64] panics on a zero divisor or a quotient
// above 64 bits; [Rate.Validate] and `units ≤ Limit` in `decide` prevent both.
// `FuzzRateMatchesExactArithmetic` checks the math against [math/big].
package ratelimit
