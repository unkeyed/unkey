-- name: ListBuckets :many
-- ListBuckets returns one row per bucket of a fingerprint, newest first:
-- the totals of the pod `pod_id` and the totals of all other pods. A
-- decision needs only these, so a read grows with buckets, not with pods.
-- The base row counts as a peer in bucket 0. Rows with zero consumed_ns
-- are refused writes and count for nothing, so the query skips them before
-- grouping: a bucket of refused writes only must not count towards the
-- LIMIT. Grouping in key order lets the LIMIT stop the scan. If the result reaches the LIMIT, older buckets or the
-- base row may be missing, and the caller must not use the result for a
-- decision: a missing row can open the limit too early.
SELECT
    bucket_start_ns,
    CAST(SUM(IF(pod_id = sqlc.arg(pod_id), consumed_ns, 0)) AS SIGNED) AS own_consumed_ns,
    CAST(MAX(IF(pod_id = sqlc.arg(pod_id), reset_at_ns, 0)) AS SIGNED) AS own_reset_at_ns,
    CAST(SUM(IF(pod_id <> sqlc.arg(pod_id), consumed_ns, 0)) AS SIGNED) AS peer_consumed_ns,
    CAST(MAX(IF(pod_id <> sqlc.arg(pod_id), reset_at_ns, 0)) AS SIGNED) AS peer_reset_at_ns,
    CAST(MAX(compact_after_ns) AS SIGNED) AS compact_after_ns
FROM ratelimit_state
WHERE fingerprint = sqlc.arg(fingerprint)
    AND consumed_ns > 0
GROUP BY bucket_start_ns
ORDER BY bucket_start_ns DESC
LIMIT ?;

-- name: ListClosedRows :many
-- ListClosedRows returns the base row of a fingerprint and its rows whose
-- compact_after_ns is before before_ns, oldest bucket first. If the result
-- reaches the LIMIT, the rows of its newest bucket may be incomplete.
SELECT bucket_start_ns, pod_id, consumed_ns, reset_at_ns, compact_after_ns
FROM ratelimit_state
WHERE fingerprint = sqlc.arg(fingerprint)
    AND (bucket_start_ns = 0 OR compact_after_ns < sqlc.arg(before_ns))
ORDER BY bucket_start_ns, pod_id
LIMIT ?;
