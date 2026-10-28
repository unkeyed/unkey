-- name: ListPeriods :many
-- ListPeriods returns one row per period of a fingerprint, newest first:
-- the scaled sum of the pod `pod_id` and the summed scaled sums of all other
-- pods. A decision needs only these, so a read grows with periods, not with
-- pods. Grouping in key order lets the LIMIT stop the scan. If the result
-- reaches the LIMIT, the oldest periods are missing; they have faded.
SELECT
    period_start_ns,
    CAST(SUM(IF(pod_id = sqlc.arg(pod_id), scaled_sum, 0)) AS DOUBLE) AS own_scaled_sum,
    CAST(SUM(IF(pod_id <> sqlc.arg(pod_id), scaled_sum, 0)) AS DOUBLE) AS peer_scaled_sum,
    CAST(MAX(delete_after_ns) AS SIGNED) AS delete_after_ns
FROM ratelimit_state
WHERE fingerprint = sqlc.arg(fingerprint)
GROUP BY period_start_ns
ORDER BY period_start_ns DESC
LIMIT ?;
