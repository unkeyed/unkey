// Package bucket holds the math of the global rate limiter: GCRA (Generic Cell
// Rate Algorithm) over the sessions of the `ratelimit_state` table.
//
// The package has no I/O, no clock and no locks. The caller reads the
// sessions, passes the time, and serializes access, so the same code runs in
// the limiter and in simulations.
//
//	b, err := bucket.New(sessions, liveSessionID)
//	if err != nil {
//		return err
//	}
//	decision, err := b.Check(nowNs, bucket.Rate{Limit: 100, Duration: time.Minute}, 1)
//	if err != nil {
//		return err
//	}
//	b.Apply(decision)
//	if decision.CreateNewSessionAtNs != 0 {
//		// Write decision.ConsumedNs into a new session anchored at
//		// decision.CreateNewSessionAtNs.
//	}
//
// # GCRA on one machine
//
// GCRA is a token bucket written as time. It does not count tokens. It stores
// one point in time, `reset_at`: the time when the bucket is full again and
// every unit is free.
//
// Every unit has a price in time. A rate of `Limit` units per `Duration`
// prices one unit at `Duration / Limit`, so 10 per 10 seconds makes one unit
// use 1 second. A request of `units` units uses `⌈units × Duration / Limit⌉`
// nanoseconds.
//
// A request is allowed if, after paying for it, `reset_at` stays at most one
// `Duration` ahead of now. One `Duration` is the burst tolerance: a limit that
// was idle can spend all `Limit` units at once. If the request is allowed,
// `reset_at` becomes `max(reset_at, now) + time of the units`. The max matters: a
// `reset_at` in the past means the bucket is full, and a full bucket cannot
// get fuller, so idle time is not saved up. A denied request changes nothing.
//
// Time refills the bucket on its own. There is no refill job and no window
// boundary. With 10 per 10 seconds:
//
//	now       request   reset_at before   check               result   reset_at after
//	12:00:00  1 unit    (past)            0s + 1s ≤ 10s       allow    12:00:01
//	12:00:00  9 units   12:00:01          1s + 9s ≤ 10s       allow    12:00:10
//	12:00:00  1 unit    12:00:10          10s + 1s > 10s      deny     12:00:10
//	12:00:01  1 unit    12:00:10          9s + 1s ≤ 10s       allow    12:00:11
//	12:00:30  1 unit    12:00:11 (past)   0s + 1s ≤ 10s       allow    12:00:31
//
// # Sharing a limit between pods
//
// Many pods decide on the same limit, and each pod sees the others about a
// second late. A single shared `reset_at` does not work: two pods read the
// same value, both add their own units, both write, and one write is lost. Lost
// units are over-admission that never evens out.
//
// So each pod writes only its own rows. A session is one stretch of time in
// which one pod admits requests for one limit. Each session has one row with
// two values: its anchor, when it started, and its consumed time, the sum of
// the time of the units it admitted. The anchor never changes and the consumed time only
// grows, so writes that arrive late or twice cannot undo a newer write.
//
// `reset_at` is computed from all rows. The candidate of a session r is r's
// anchor plus the consumed time of every session anchored at or after r:
//
//	reset_at = max over sessions r of
//	           (r.anchor + sum of consumed of sessions anchored at or after r)
//
// A candidate answers one question: "if the limit was never idle since this
// anchor, when is it free again?" For the session that started the current
// busy stretch, the answer is exact. For older sessions the assumption is
// false, because the limit was idle at some point after their anchor, so their
// candidate is too early. No candidate is ever too late, so the largest one is
// right. This is the single-machine rule `max(reset_at, now) + time of the units`, written
// over sessions instead of over requests:
//
//	session   anchor     consumed   candidate
//	B         12:00:03   9s         12:00:03 + 9s             = 12:00:12
//	A         12:00:00   12s        12:00:00 + 9s + 12s       = 12:00:21   <- reset_at
//	C         11:58:40   3s         11:58:40 + 9s + 12s + 3s  = 11:59:04
//
// # Where a request goes
//
// A pod adds its admitted units to its live session. That is only correct if
// it gives the same `reset_at` as single-machine GCRA. Units added to the
// live session raise the candidates of the live session and of every session
// anchored before it. It does not raise the candidates of sessions anchored
// after it. Two cases break this:
//
// After a pause, `reset_at` is in the past. Adding to the live session would
// raise a candidate that is still in the past, so the request would be free.
// The request must start a new session at now, the same as `max(reset_at, now)`
// on one machine.
//
// When a peer started a session after the live one and that newer session
// holds `reset_at`, adding to the live session moves `reset_at` too little or
// not at all. The request must start a new session at the newest anchor or later, so
// that it counts toward every other session.
//
// [Bucket.Check] detects both cases and sets [Decision.CreateNewSessionAtNs].
// Old sessions stay as they are and still count in full. An extra session
// costs one row and never changes `reset_at`.
//
// # Over-admission is paid back
//
// Two pods that have not seen each other yet can both admit. With 10 per 10
// seconds, pods A and B can each admit 10 at 12:00:00, 20 in total. Once each
// sees the other's row, `reset_at` is 12:00:00 + 10s + 10s = 12:00:20, so every
// request is denied until 12:00:11. The extra 10 are paid back right after,
// and over time the total matches a single-machine limiter.
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
// the result is exact for every `Limit` and `Duration` that fit into int64, with
// no allocation. It costs about 1 ns more per decision than plain division on
// amd64. bits.Div64 panics if the divisor is zero or if the quotient does not
// fit into 64 bits; [Rate.Validate] and the rule `units ≤ Limit` in
// [Bucket.Check] exclude both cases, so keep them in front of every call.
// `FuzzRateMatchesExactArithmetic` checks the result against [math/big] over
// the full int64 range.
package bucket
