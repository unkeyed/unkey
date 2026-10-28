// Package ratelimit is a global rate limiter: an exponentially decaying
// counter over the rows of the `ratelimit_state` table.
//
// A limit of `Limit` units per `Duration` keeps a level. Each admitted
// request with cost c at time t adds c, and the level then decays with
// `Duration` as time constant:
//
//	level(now) = Σ c × e^(−(now − t) / Duration)
//
// A request fits if `effective level + cost ≤ Limit`, where the effective
// level is zero below one unit. Without that cut the level never reaches zero,
// and `remaining` would never return to `Limit`. Denied requests and requests
// for zero units change nothing. With 10 per 10 seconds:
//
//	now       request   level before   check            result   level after
//	12:00:00  10 units  0              0 + 10 ≤ 10      allow    10
//	12:00:00  1 unit    10             10 + 1 > 10      deny     10
//	12:00:01  1 unit    9.05           9.05 + 1 > 10    deny     9.05
//	12:00:02  1 unit    8.19           8.19 + 1 ≤ 10    allow    9.19
//	12:00:30  1 unit    0.56 (→ 0)     0 + 1 ≤ 10       allow    1.56
//
// From idle the burst is `Limit`, a steady stream is held to `Limit /
// Duration`, and `remaining` is back at `Limit` after `Duration × ln(Limit)`.
//
// Pods share the level exactly, in any order. Time is split into periods of
// `10 × Duration` on edges that all pods share, see `periodLengthMs`. Each
// pod writes one row per period that it used, with the scaled sum
//
//	S = Σ c × e^((t − period start) / Duration)
//
// over its admitted requests in the period, and `level(now) = Σ S ×
// e^(−(now − period start) / Duration)` over all rows. Addition does not
// care about order, and S only grows, so writes keep the larger value:
// retries and reordered writes are harmless, a lost write is carried by the
// next flush with the latest sum, and a pod writes at most one row per
// limit per flush, however many requests it allows. Once every write is
// visible, every pod computes the level of one machine that saw all admitted
// requests, up to float rounding. If two pods over-admit before they see
// each other, the excess decays like any other units: pods A and B each admit
// 75 of 100 per minute at 12:00:00, and once both rows are visible, the next
// unit fits at 12:00:25.
//
// A row's level falls below 0.01 units at a time that follows from its sum,
// its `delete_after_ms`, and cleanup then deletes it with one bounded delete.
// Pods may write any period at any time: a write that lands after its row was
// deleted inserts the pod's whole sum again, so nothing needs a write
// deadline or a merge.
package ratelimit
