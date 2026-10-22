// Package ratelimit is a global rate limiter: GCRA (Generic Cell Rate
// Algorithm) over the session rows of the `ratelimit_state` table.
//
// GCRA stores one time, `reset_at`. A request is allowed if
// `max(reset_at, now) + consumed time` stays at most one `Duration` ahead of
// now, and that becomes the new `reset_at`. With 10 per 10 seconds:
//
//	now       request   reset_at before   check               result   reset_at after
//	12:00:00  1 unit    (past)            0s + 1s ≤ 10s       allow    12:00:01
//	12:00:00  9 units   12:00:01          1s + 9s ≤ 10s       allow    12:00:10
//	12:00:00  1 unit    12:00:10          10s + 1s > 10s      deny     12:00:10
//	12:00:01  1 unit    12:00:10          9s + 1s ≤ 10s       allow    12:00:11
//	12:00:30  1 unit    12:00:11 (past)   0s + 1s ≤ 10s       allow    12:00:31
//
// Each pod writes only its own session rows, and `summarize` computes
// `reset_at` from all of them. If two pods over-admit before they see each
// other, the excess is paid back: pods A and B each admit 10 at 12:00:00, and
// once both rows are visible, the next unit fits at 12:00:11.
//
// Do not replace the 128-bit math in [Rate] with plain integer math.
// `units × Duration` overflows 64 bits for rates the API accepts:
//
//	rate                 Limit × Duration (ns)   uint64 max
//	10,000 per 30 days   2.6e19                  1.8e19
//	400M per minute      2.4e19                  1.8e19
package ratelimit
