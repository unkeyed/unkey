// Package ratelimit is a global rate limiter: GCRA (Generic Cell Rate
// Algorithm) over the rows of the `ratelimit_state` table.
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
// Each limit splits time into buckets of width `max(1s, Duration / 1000)`, on
// edges that all pods share. Each pod writes one row per bucket that it used:
// the consumed time and the GCRA `reset_at` of its own requests in that
// bucket. Writes keep the larger values, so retries and reordered writes are
// harmless, and a pod writes at most one row per bucket per flush, however
// many requests it allows. A read returns per bucket this pod's row and one
// row that sums its peers', so it grows with buckets, not with pods.
// `computeResetTimes` computes `reset_at` from those rows. If two pods
// over-admit before they see each other, the excess is paid back: pods A and
// B each admit 10 at 12:00:00, and once both rows are visible, the next unit
// fits at 12:00:20.
//
// Once every write is visible, every pod computes the same `reset_at` from
// the same rows. It is never later than the `reset_at` of plain GCRA over the
// same requests, and earlier by less than one bucket width. It is exact when
// one pod wrote each bucket. When several pods write one bucket, the rows do
// not keep the order of their requests within the bucket, and that order can
// change `reset_at` by up to a bucket width. See `bucketTotals.resetAtNs`.
//
// MySQL refuses writes to a bucket from `width + maxWriteDelay +
// maxClockSkew` after its start, by the MySQL clock. Then no write can change
// it, and cleanup merges all closed buckets into one base row without
// changing `reset_at`, or deletes them once the limit is idle; see
// `planCleanup`. A pod that cannot get a write acknowledged in time moves its
// units into its current bucket, see `entry.carryLostUnits`. A pod that
// crashes loses the units it has not written.
//
// Do not replace the 128-bit math in [Rate] with plain integer math.
// `units × Duration` overflows 64 bits for rates the API accepts:
//
//	rate                 Limit × Duration (ns)   uint64 max
//	10,000 per 30 days   2.6e19                  1.8e19
//	400M per minute      2.4e19                  1.8e19
package ratelimit
